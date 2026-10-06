package community

import (
	"errors"
	"net/http"
	"strings"

	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EndorsementGiver is one peer inside an endorsement group.
type EndorsementGiver struct {
	ID        string `json:"id"`
	GiverID   string `json:"giver_id"`
	GiverName string `json:"giver_name"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

// EndorsementGroup is one skill's endorsement tally for a receiver.
type EndorsementGroup struct {
	Skill           string             `json:"skill"`
	Count           int                `json:"count"`
	Givers          []EndorsementGiver `json:"givers"`
	MyEndorsementID string             `json:"my_endorsement_id"`
}

// Endorsement is the public wire shape of one endorsement.
type Endorsement struct {
	ID         string `json:"id"`
	Skill      string `json:"skill"`
	Note       string `json:"note"`
	GiverID    string `json:"giver_id"`
	GiverName  string `json:"giver_name"`
	ReceiverID string `json:"receiver_id"`
	CreatedAt  string `json:"created_at"`
}

type createEndorsementRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
	Skill      string `json:"skill" binding:"required"`
	Note       string `json:"note"`
}

// normalizeEndorsementSkill lowercases + trims a skill label; empty means
// the input carried no usable skill.
func normalizeEndorsementSkill(skill string) string {
	return strings.ToLower(strings.TrimSpace(skill))
}

func (h *Handler) listEndorsements(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	receiverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var userExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, receiverID).Scan(&userExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify user")
		return
	}
	if !userExists {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT e.id::text, e.skill, e.giver_id::text, u.display_name, e.note,
			e.created_at::text, (e.giver_id = $2::uuid)
		FROM skill_endorsements e
		JOIN users u ON u.id = e.giver_id
		WHERE e.receiver_id = $1
		ORDER BY e.skill, e.created_at DESC`, receiverID, claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load endorsements")
		return
	}
	defer rows.Close()

	groups := make([]EndorsementGroup, 0)
	indexBySkill := map[string]int{}
	for rows.Next() {
		var id, skill, giverID, giverName, note, createdAt string
		var mine bool
		if err := rows.Scan(&id, &skill, &giverID, &giverName, &note, &createdAt, &mine); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load endorsements")
			return
		}
		pos, seen := indexBySkill[skill]
		if !seen {
			pos = len(groups)
			indexBySkill[skill] = pos
			groups = append(groups, EndorsementGroup{Skill: skill, Givers: []EndorsementGiver{}})
		}
		groups[pos].Count++
		groups[pos].Givers = append(groups[pos].Givers, EndorsementGiver{
			ID: id, GiverID: giverID, GiverName: giverName, Note: note, CreatedAt: createdAt,
		})
		if mine {
			groups[pos].MyEndorsementID = id
		}
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load endorsements")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load endorsements")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": groups,
		"meta": gin.H{"request_id": c.GetString("request_id"), "total": len(groups)},
	})
}

func (h *Handler) createEndorsement(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	// Any staff member may endorse a peer: this rides on community read.
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	var request createEndorsementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid endorsement request")
		return
	}
	skill := normalizeEndorsementSkill(request.Skill)
	if skill == "" || len(skill) > 60 {
		writeError(c, http.StatusBadRequest, "validation_error", "skill is required and must be at most 60 characters")
		return
	}
	if len(request.Note) > 500 {
		writeError(c, http.StatusBadRequest, "validation_error", "note is too long")
		return
	}
	receiverID, err := uuid.Parse(request.ReceiverID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid receiver id")
		return
	}
	if request.ReceiverID == claims.UserID {
		writeError(c, http.StatusBadRequest, "cannot_endorse_self", "you cannot endorse yourself")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var receiverExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, receiverID).Scan(&receiverExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify user")
		return
	}
	if !receiverExists {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	endorsementID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO skill_endorsements (id, tenant_id, giver_id, receiver_id, skill, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at::text`,
		endorsementID, claims.TenantID, claims.UserID, receiverID, skill,
		strings.TrimSpace(request.Note)).Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(c, http.StatusConflict, "already_endorsed", "you already endorsed this skill")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save endorsement")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save endorsement")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": Endorsement{
			ID: endorsementID.String(), Skill: skill, Note: strings.TrimSpace(request.Note),
			GiverID: claims.UserID, ReceiverID: receiverID.String(), CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) deleteEndorsement(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	endorsementID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid endorsement id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var giverID string
	if err = tx.QueryRow(ctx, `SELECT giver_id::text FROM skill_endorsements WHERE id = $1`, endorsementID).Scan(&giverID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "endorsement_not_found", "endorsement not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load endorsement")
		return
	}
	// Withdraw is for the giver; anyone with community write may moderate.
	if giverID != claims.UserID && !httptransport.HasPermission(claims.Role, "community", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM skill_endorsements WHERE id = $1`, endorsementID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete endorsement")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete endorsement")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}
