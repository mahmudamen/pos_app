package support

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore is the Postgres-backed inbox store used by the poller.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore wraps a pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// Ingest stores one inbound message. It reports stored=false when the message
// was already present (same mailbox + provider_ref), which makes polling
// replay-safe. The thread is resolved from In-Reply-To first, then from a
// matching normalized subject + sender, and otherwise started fresh.
func (s *PGStore) Ingest(ctx context.Context, m InboundMessage) (bool, error) {
	if s.pool == nil {
		return false, errors.New("support: nil pool")
	}
	mailbox := m.Mailbox
	if mailbox == "" {
		mailbox = DefaultMailbox
	}
	key := ThreadKey(m.Subject)
	received := m.ReceivedAt
	if received.IsZero() {
		received = time.Now()
	}

	threadID, tenantID := s.resolveThread(ctx, m, key)

	tag, err := s.pool.Exec(ctx, `
		INSERT INTO support_messages (
			thread_id, thread_key, direction, mailbox, tenant_id, message_id, in_reply_to,
			from_email, from_name, to_email, subject, body_text, body_html,
			status, provider_ref, received_at
		) VALUES (
			$1::uuid, $2, 'inbound', $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			'unread', $13, $14
		)
		ON CONFLICT DO NOTHING`,
		threadID, key, mailbox, tenantID, nullIfEmpty(m.MessageID), nullIfEmpty(m.InReplyTo),
		m.FromEmail, m.FromName, m.ToEmail, m.Subject, m.BodyText, m.BodyHTML,
		m.ProviderRef, received)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// resolveThread returns the thread the message belongs to and the tenant it is
// attributed to. A failure to look up an existing thread is non-fatal: the
// message is still stored under a fresh thread.
func (s *PGStore) resolveThread(ctx context.Context, m InboundMessage, key string) (string, any) {
	var tenantID any
	if m.FromEmail != "" {
		var id string
		err := s.pool.QueryRow(ctx,
			`SELECT id::text FROM tenants WHERE contact_email <> '' AND lower(contact_email) = lower($1) LIMIT 1`,
			m.FromEmail).Scan(&id)
		if err == nil {
			tenantID = id
		}
	}

	if reply := normalizeMessageID(m.InReplyTo); reply != "" {
		var thread string
		err := s.pool.QueryRow(ctx,
			`SELECT thread_id::text FROM support_messages
			 WHERE message_id IS NOT NULL
			   AND lower(trim(both '<>' from message_id)) = lower($1)
			 ORDER BY created_at LIMIT 1`, reply).Scan(&thread)
		if err == nil {
			return thread, tenantID
		}
	}
	if key != "" && m.FromEmail != "" {
		var thread string
		err := s.pool.QueryRow(ctx,
			`SELECT thread_id::text FROM support_messages
			 WHERE thread_key = $1 AND lower(from_email) = lower($2)
			 ORDER BY created_at DESC LIMIT 1`, key, m.FromEmail).Scan(&thread)
		if err == nil {
			return thread, tenantID
		}
	}
	return NewThreadID(), tenantID
}

// Cursor returns the resume position for a mailbox; a missing row is a zero
// cursor (start from the beginning).
func (s *PGStore) Cursor(ctx context.Context, mailbox string) (SyncCursor, error) {
	var c SyncCursor
	if s.pool == nil {
		return c, errors.New("support: nil pool")
	}
	err := s.pool.QueryRow(ctx,
		`SELECT uid_validity, last_uid FROM support_sync_state WHERE mailbox = $1`, mailbox).
		Scan(&c.UIDValidity, &c.LastUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SyncCursor{}, nil
	}
	return c, err
}

// SaveCursor upserts the resume position for a mailbox.
func (s *PGStore) SaveCursor(ctx context.Context, mailbox string, c SyncCursor) error {
	if s.pool == nil {
		return errors.New("support: nil pool")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO support_sync_state (mailbox, uid_validity, last_uid, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (mailbox) DO UPDATE
		SET uid_validity = EXCLUDED.uid_validity, last_uid = EXCLUDED.last_uid, updated_at = now()`,
		mailbox, c.UIDValidity, c.LastUID)
	return err
}

// normalizeMessageID strips the RFC 5322 angle brackets and whitespace.
func normalizeMessageID(id string) string {
	return strings.Trim(strings.TrimSpace(id), "<>")
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
