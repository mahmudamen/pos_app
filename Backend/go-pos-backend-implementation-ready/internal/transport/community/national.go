package community

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// National community slice (docs/24_NATIONAL_COMMUNITY.md, slice A).
//
// community_members / community_invitations are deliberately NOT tenant-RLS'd
// (the community is Egypt-wide). Every handler here therefore enforces, inside
// its own transactions:
//   - bearer auth (h.authenticate)
//   - Egypt locality of the actor's tenant (requireEgypt)
//   - membership where the route is member-only
//   - national moderation role / saas_admin for moderator actions
//
// National queries never join FORCE-RLS tenant tables (users, tenants data is
// only projected at join time into community_members.display_name).

const inviteCodeAlphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789" // no I/O/0/1/L/U collisions

// Member is the public wire shape of one national community member.
type Member struct {
	UserID         string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	OriginTenantID string `json:"origin_tenant_id"`
	JoinedVia      string `json:"joined_via"`
	InvitedBy      string `json:"invited_by"`
	InvitedByName  string `json:"invited_by_name,omitempty"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	Level          string `json:"level"`
	ExpertiseScore int    `json:"expertise_score"`
	JoinedAt       string `json:"joined_at"`
}

// Invitation is the wire shape of one invitation code. Code is populated ONLY
// on create (the plaintext is never stored; the DB keeps a SHA-256).
type Invitation struct {
	ID        string `json:"id"`
	Code      string `json:"code,omitempty"`
	InviterID string `json:"inviter_id"`
	Email     string `json:"email"`
	Note      string `json:"note"`
	MaxUses   int    `json:"max_uses"`
	UsedCount int    `json:"used_count"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type joinRequest struct {
	Code string `json:"code"`
}

type createInvitationRequest struct {
	Email     string  `json:"email"`
	Note      string  `json:"note"`
	MaxUses   int     `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"`
}

type updateMemberRequest struct {
	Status *string `json:"status"`
	Role   *string `json:"role"`
}

// --- pure helpers ---------------------------------------------------------

// newInviteCode returns the canonical 12-char code and its EG- display form.
func newInviteCode() (canonical, display string) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand failure is essentially unreachable; fall back to a time-seeded value.
		seed := uint64(time.Now().UnixNano())
		for i := range raw {
			raw[i] = inviteCodeAlphabet[seed%uint64(len(inviteCodeAlphabet))]
			seed /= uint64(len(inviteCodeAlphabet))
		}
		return string(raw), "EG-" + string(raw)
	}
	// #nosec G404 -- non-deterministic truncation of crypto/rand bytes, not math/rand.
	for i := range raw {
		raw[i] = inviteCodeAlphabet[int(raw[i])%len(inviteCodeAlphabet)]
	}
	canonical = string(raw)
	return canonical, "EG-" + canonical
}

// normalizeCode uppercases, keeps only A-Z0-9 and strips a leading EG- prefix so
// "eg-1234-abcd", "EG1234ABCD" and "EG-1234-ABCD" all resolve to the same code.
func normalizeCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	return strings.TrimPrefix(out, "EG")
}

func codeHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// levelForScore maps an expertise score to the cached level (docs/24).
func levelForScore(score int) string {
	switch {
	case score >= 1000:
		return "platinum"
	case score >= 500:
		return "gold"
	case score >= 200:
		return "silver"
	default:
		return "bronze"
	}
}

// --- shared guards --------------------------------------------------------

// requireEgypt denies requests from tenants whose country is not EG. tenants is
// not RLS'd, so this works without tenant context.
func (h *Handler) requireEgypt(c *gin.Context, ctx context.Context, tx pgx.Tx, tenantID string) bool {
	var countryCode string
	if err := tx.QueryRow(ctx, `SELECT country_code FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&countryCode); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return false
	}
	if countryCode != "EG" {
		writeError(c, http.StatusForbidden, "egypt_only", "the national community is available in Egypt only")
		return false
	}
	return true
}

// loadMemberRole returns the actor's national role ("" when they are not an
// active member). National tables carry no tenant policy, so no context needed.
func (h *Handler) loadMemberRole(ctx context.Context, tx pgx.Tx, userID string) (string, error) {
	var role string
	err := tx.QueryRow(ctx,
		`SELECT role FROM community_members WHERE user_id = $1::uuid AND is_active = TRUE`, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
}

// validIdentity rejects tokens whose UUID claims would crash the $n::uuid casts
// with a driver syntax error. Claims come from the app's own JWT (aleays well
// formed in practice), but a hand-forged token must get a clean 401, not a 500.
func validIdentity(c *gin.Context, claims security.Claims) bool {
	if _, err := uuid.Parse(claims.UserID); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return false
	}
	if _, err := uuid.Parse(claims.TenantID); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return false
	}
	return true
}

// requireMember 403s when the actor has no active national membership.
func (h *Handler) requireMember(c *gin.Context, ctx context.Context, tx pgx.Tx, userID string) bool {
	role, err := h.loadMemberRole(ctx, tx, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify membership")
		return false
	}
	if role == "" {
		writeError(c, http.StatusForbidden, "not_a_member", "press join the national community first")
		return false
	}
	return true
}

// requireInviter is the invitation-granting check: any active member, plus
// saas_admin (platform staff) who bootstrap the very first launch codes
// before any member exists (docs/24 slice A bootstrap flow).
func (h *Handler) requireInviter(c *gin.Context, ctx context.Context, tx pgx.Tx, claims security.Claims) bool {
	if claims.Role == "saas_admin" {
		return true
	}
	return h.requireMember(c, ctx, tx, claims.UserID)
}

// requireModeration allows national moderator/admin members and saas_admin.
func (h *Handler) requireModeration(c *gin.Context, claims security.Claims, actorMemberRole, targetMemberRole string) bool {
	if claims.Role == "saas_admin" || actorMemberRole == "moderator" || actorMemberRole == "admin" {
		_ = targetMemberRole // reserved: mods manage below admin, admins manage all
		return true
	}
	writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
	return false
}

func (h *Handler) beginTx(c *gin.Context) (context.Context, pgx.Tx, bool) {
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return ctx, nil, false
	}
	return ctx, tx, true
}

// --- handlers -------------------------------------------------------------

func (h *Handler) nationalMe(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var member Member
	err := tx.QueryRow(ctx, `
		SELECT m.user_id::text, m.display_name, COALESCE(m.origin_tenant_id::text, ''),
			m.joined_via, COALESCE(m.invited_by::text, ''), m.role, m.status, m.level,
			m.expertise_score, m.joined_at::text
		FROM community_members m
		WHERE m.user_id = $1::uuid`, claims.UserID).Scan(
		&member.UserID, &member.DisplayName, &member.OriginTenantID,
		&member.JoinedVia, &member.InvitedBy, &member.Role, &member.Status, &member.Level,
		&member.ExpertiseScore, &member.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "not_a_member", "not a national community member")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load membership")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load membership")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": member, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) joinCommunity(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}
	ip := c.ClientIP()
	if !h.joinLimiter.Allow("community:join:" + ip) {
		writeError(c, http.StatusTooManyRequests, "too_many_requests", "too many join attempts, try again later")
		return
	}
	var request joinRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid join request")
		return
	}
	canonical := normalizeCode(request.Code)
	if canonical == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "an invitation code is required")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireEgypt(c, ctx, tx, claims.TenantID) {
		return
	}
	// Read the caller's users row inside their tenant context (FORCE RLS) so we
	// can (a) prove they belong to the tenant and (b) take the display name.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var displayName string
	if err := tx.QueryRow(ctx,
		`SELECT display_name FROM users WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		claims.UserID, claims.TenantID).Scan(&displayName); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	// The code row is locked so concurrent claims cannot both consume it.
	var id, originTenant, inviterID, email, note string
	var maxUses, usedCount int
	var status string
	var expiresAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(origin_tenant_id::text, ''), inviter_id::text,
			COALESCE(email, ''), COALESCE(note, ''), max_uses, used_count, status,
			expires_at
		FROM community_invitations
		WHERE code_hash = $1
		FOR UPDATE`, codeHash(canonical)).Scan(
		&id, &originTenant, &inviterID, &email, &note, &maxUses, &usedCount, &status, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusForbidden, "invalid_code", "that invitation code is not valid")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify invitation")
		return
	}
	if status == "revoked" || status == "expired" {
		writeError(c, http.StatusGone, "expired", "that invitation is no longer active")
		return
	}
	if expiresAt != nil && time.Now().After(*expiresAt) {
		writeError(c, http.StatusGone, "expired", "that invitation has expired")
		return
	}
	if usedCount >= maxUses {
		writeError(c, http.StatusGone, "expired", "that invitation has been used up")
		return
	}

	// Join: idempotency-safe insert + consume one use, same tx. The member is
	// re-selected so the response reflects the stored row exactly.
	if _, err = tx.Exec(ctx, `
		INSERT INTO community_members (user_id, origin_tenant_id, display_name, joined_via, invited_by,
			expertise_score, level)
		VALUES ($1::uuid, $2::uuid, $3, 'invitation', $4::uuid, 10, $5)`,
		claims.UserID, claims.TenantID, displayName, inviterID, levelForScore(10)); err != nil {
		writeError(c, http.StatusConflict, "already_member", "you are already a community member")
		return
	}
	if _, err = tx.Exec(ctx,
		`UPDATE community_invitations SET used_count = used_count + 1 WHERE id = $1::uuid`, id); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to mark invitation used")
		return
	}
	var member Member
	err = tx.QueryRow(ctx, `
		SELECT user_id::text, display_name, $1::text, 'invitation', COALESCE(invited_by::text, ''),
			role, status, level, expertise_score, joined_at::text
		FROM community_members WHERE user_id = $2::uuid`, originTenant, claims.UserID).Scan(
		&member.UserID, &member.DisplayName, &member.OriginTenantID, &member.JoinedVia,
		&member.InvitedBy, &member.Role, &member.Status, &member.Level,
		&member.ExpertiseScore, &member.JoinedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load membership")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to join community")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": member, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createInvitation(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	if !h.can(c, claims.Role, "community", "invite") {
		return
	}
	var request createInvitationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid invitation request")
		return
	}
	request.Note = strings.TrimSpace(request.Note)
	request.Email = strings.TrimSpace(request.Email)
	if request.Email != "" && !strings.Contains(request.Email, "@") {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid recipient email")
		return
	}
	if len(request.Note) > 500 || len(request.Email) > 320 {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid invitation fields")
		return
	}
	if request.MaxUses < 1 || request.MaxUses > 100 {
		request.MaxUses = 1
	}
	var expiresAt *time.Time
	if request.ExpiresAt != nil {
		when, perr := time.Parse(time.RFC3339, *request.ExpiresAt)
		if perr != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "expires_at must be an RFC3339 timestamp")
			return
		}
		expiresAt = &when
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireInviter(c, ctx, tx, claims) {
		return
	}
	canonical, display := newInviteCode()
	code := codeHash(canonical)
	var createdAt, id string
	err := tx.QueryRow(ctx, `
		INSERT INTO community_invitations (code_hash, inviter_id, origin_tenant_id, email, note, max_uses, expires_at)
		VALUES ($1, $2::uuid, $3::uuid, NULLIF($4, ''), $5, $6, $7)
		RETURNING id::text, created_at::text`, code, claims.UserID, claims.TenantID, request.Email, request.Note,
		request.MaxUses, expiresAt).Scan(&id, &createdAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create invitation")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create invitation")
		return
	}
	var expires string
	if expiresAt != nil {
		expires = expiresAt.UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": Invitation{
			ID: id, Code: display, InviterID: claims.UserID, Email: request.Email, Note: request.Note,
			MaxUses: request.MaxUses, UsedCount: 0, Status: "active", ExpiresAt: expires, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) listInvitations(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	if !h.can(c, claims.Role, "community", "invite") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireInviter(c, ctx, tx, claims) {
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, inviter_id::text, COALESCE(email, ''), COALESCE(note, ''), max_uses,
			used_count, status, expires_at, created_at::text
		FROM community_invitations
		WHERE inviter_id = $1::uuid OR (origin_tenant_id = $2::uuid AND origin_tenant_id IS NOT NULL)
		ORDER BY created_at DESC
		LIMIT 200`, claims.UserID, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invitations")
		return
	}
	defer rows.Close()
	items := make([]Invitation, 0)
	for rows.Next() {
		var item Invitation
		var expiresAt *time.Time
		if err := rows.Scan(&item.ID, &item.InviterID, &item.Email, &item.Note,
			&item.MaxUses, &item.UsedCount, &item.Status, &expiresAt, &item.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invitations")
			return
		}
		if expiresAt != nil {
			item.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invitations")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invitations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) revokeInvitation(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	if !h.can(c, claims.Role, "community", "invite") {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid invitation id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireInviter(c, ctx, tx, claims) {
		return
	}
	tag, err := tx.Exec(ctx, `
		UPDATE community_invitations SET status = 'revoked'
		WHERE id = $1::uuid AND (inviter_id = $2::uuid OR origin_tenant_id = $3::uuid)
			AND status <> 'revoked'`, id, claims.UserID, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke invitation")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "invitation_not_found", "invitation not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke invitation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) listMembers(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	q := strings.TrimSpace(c.Query("q"))
	level := strings.TrimSpace(c.Query("level"))
	memberRole := strings.TrimSpace(c.Query("role"))

	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireMember(c, ctx, tx, claims.UserID) {
		return
	}
	where := "m.is_active = TRUE AND m.status <> 'suspended'"
	whereArgs := make([]any, 0)
	if q != "" {
		whereArgs = append(whereArgs, "%"+q+"%")
		where += " AND m.display_name ILIKE $" + strconv.Itoa(len(whereArgs))
	}
	if level != "" {
		whereArgs = append(whereArgs, level)
		where += " AND m.level = $" + strconv.Itoa(len(whereArgs))
	}
	if memberRole != "" {
		whereArgs = append(whereArgs, memberRole)
		where += " AND m.role = $" + strconv.Itoa(len(whereArgs))
	}
	var total int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM community_members m WHERE `+where, whereArgs...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count members")
		return
	}
	pageArgs := append(append([]any{}, whereArgs...), limit, int64((page-1)*limit))
	rows, err := tx.Query(ctx, `
		SELECT m.user_id::text, m.display_name, COALESCE(m.origin_tenant_id::text, ''),
			m.joined_via, COALESCE(m.invited_by::text, ''), COALESCE(ib.display_name, ''),
			m.role, m.status, m.level, m.expertise_score, m.joined_at::text
		FROM community_members m
		LEFT JOIN community_members ib ON ib.user_id = m.invited_by
		WHERE `+where+` ORDER BY m.expertise_score DESC, m.display_name ASC
		LIMIT $`+strconv.Itoa(len(whereArgs)+1)+` OFFSET $`+strconv.Itoa(len(whereArgs)+2), pageArgs...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load members")
		return
	}
	defer rows.Close()
	items := make([]Member, 0)
	for rows.Next() {
		var item Member
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.OriginTenantID, &item.JoinedVia,
			&item.InvitedBy, &item.InvitedByName, &item.Role, &item.Status, &item.Level,
			&item.ExpertiseScore, &item.JoinedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load members")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load members")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load members")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) getMember(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid member id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if !h.requireMember(c, ctx, tx, claims.UserID) {
		return
	}
	var member Member
	err = tx.QueryRow(ctx, `
		SELECT m.user_id::text, m.display_name, COALESCE(m.origin_tenant_id::text, ''),
			m.joined_via, COALESCE(m.invited_by::text, ''), COALESCE(ib.display_name, ''),
			m.role, m.status, m.level, m.expertise_score, m.joined_at::text
		FROM community_members m
		LEFT JOIN community_members ib ON ib.user_id = m.invited_by
		WHERE m.user_id = $1::uuid`, userID).Scan(
		&member.UserID, &member.DisplayName, &member.OriginTenantID, &member.JoinedVia,
		&member.InvitedBy, &member.InvitedByName, &member.Role, &member.Status, &member.Level,
		&member.ExpertiseScore, &member.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "member_not_found", "member not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load member")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": member, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) updateMember(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !validIdentity(c, claims) {
		return
	}

	var request updateMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid member update")
		return
	}
	if request.Status == nil && request.Role == nil {
		writeError(c, http.StatusBadRequest, "validation_error", "nothing to update")
		return
	}
	if request.Status != nil && !validMemberStatus(*request.Status) {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be active, muted or suspended")
		return
	}
	if request.Role != nil && !validMemberRole(*request.Role) {
		writeError(c, http.StatusBadRequest, "validation_error", "role must be member, moderator or admin")
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid member id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx, tx, ok := h.beginTx(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	targetRole, err := h.loadMemberRole(ctx, tx, userID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify membership")
		return
	}
	if targetRole == "" {
		writeError(c, http.StatusNotFound, "member_not_found", "member not found")
		return
	}
	actorMemberRole, err := h.loadMemberRole(ctx, tx, claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify membership")
		return
	}
	if !h.requireModeration(c, claims, actorMemberRole, targetRole) {
		return
	}

	switch {
	case request.Status != nil && request.Role != nil:
		_, err = tx.Exec(ctx, `UPDATE community_members SET status = $1, role = $2 WHERE user_id = $3::uuid`,
			*request.Status, *request.Role, userID)
	case request.Status != nil:
		_, err = tx.Exec(ctx, `UPDATE community_members SET status = $1 WHERE user_id = $2::uuid`,
			*request.Status, userID)
	default:
		_, err = tx.Exec(ctx, `UPDATE community_members SET role = $1 WHERE user_id = $2::uuid`,
			*request.Role, userID)
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update member")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}

func validMemberStatus(v string) bool {
	return v == "active" || v == "muted" || v == "suspended"
}

func validMemberRole(v string) bool {
	return v == "member" || v == "moderator" || v == "admin"
}
