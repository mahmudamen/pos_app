package identity

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is the minimal SQL surface shared by the identity stores. Both
// *pgxpool.Pool and pgx.Tx satisfy it, so every store can run either inside a
// caller-managed transaction (atomicity) or standalone.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

// AuditRecorder appends security-relevant events to audit_log. The table is
// append-only from the application's point of view: the app role has
// INSERT+SELECT grants and no DELETE, and no store in this package ever
// updates or deletes existing rows.
type AuditRecorder struct{}

func NewAuditRecorder() AuditRecorder { return AuditRecorder{} }

// Record inserts one audit_log row inside the caller's transaction (or
// standalone when passed the pool). Sensitive values (passwords, OTPs,
// tokens, refresh tokens) must never be placed in Before/After/Reason.
func (AuditRecorder) Record(ctx context.Context, q Querier, e AuditEntry) error {
	before := encodeAuditJSON(e.Before)
	after := encodeAuditJSON(e.After)
	ip := e.IP
	if ip == "" {
		ip = ""
	}
	_, err := q.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, account_id, tenant_id, action, entity_type, entity_id, before, after, reason, ip, user_agent)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::uuid, $7, $8, $9, NULLIF($10, '')::inet, $11)`,
		nullable(e.ActorUserID), nullable(e.AccountID), nullable(e.TenantID),
		e.Action, e.EntityType, nullable(e.EntityID),
		before, after, e.Reason, ip, e.UserAgent)
	return err
}

func nullable(id string) any {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	return id
}

func encodeAuditJSON(m map[string]any) []byte {
	if len(m) == 0 {
		return nil
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return encoded
}
