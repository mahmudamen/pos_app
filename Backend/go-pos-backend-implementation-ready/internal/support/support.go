// Package support is the platform support inbox: the inbound IMAP ingest that
// stores mail into `support_messages`, the poller that drives it, and the pure
// threading/parsing helpers shared by the poller and the console API.
//
// The inbox is platform-level (no RLS): one cross-tenant queue read by
// saas_admins, and an inbound message from an unknown sender must be stored
// before any tenant context exists. `tenant_id` is attribution only.
package support

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	gomail "github.com/emersion/go-message/mail"
)

// Direction and status vocabularies (mirror the CHECK constraints).
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"

	StatusUnread   = "unread"
	StatusRead     = "read"
	StatusArchived = "archived"
)

// DefaultMailbox is the mailbox name used when a message is not tied to a
// configured IMAP mailbox (e.g. an outbound reply composed in the console).
const DefaultMailbox = "support"

// maxBodyBytes caps a single decoded body so a pathological (or malicious)
// message cannot exhaust memory. Bodies are truncated, never rejected.
const maxBodyBytes = 512 * 1024

// InboundMessage is a message pulled from an inbound mailbox and about to be
// stored. ProviderRef is the transport's stable identifier (the IMAP UID as a
// string) and is the idempotency key for ingest.
type InboundMessage struct {
	Mailbox     string
	ProviderRef string
	MessageID   string
	InReplyTo   string
	FromName    string
	FromEmail   string
	ToEmail     string
	Subject     string
	BodyText    string
	BodyHTML    string
	ReceivedAt  time.Time
}

// SyncCursor resumes a mailbox after a restart.
type SyncCursor struct {
	UIDValidity int64
	LastUID     int64
}

// FetchResult is one poll's worth of mail plus the cursor to resume from once
// every message in the batch is stored.
type FetchResult struct {
	Mailbox  string
	Messages []InboundMessage
	Cursor   SyncCursor
}

// Fetcher pulls not-yet-seen messages from the configured mailbox. The IMAP
// implementation talks to a real server; tests use an in-memory fake.
type Fetcher interface {
	Fetch(ctx context.Context) (FetchResult, error)
}

// Store persists inbound mail and the per-mailbox sync cursor. It is small on
// purpose: the console API reads the table directly, while the poller only
// needs ingest + resume.
type Store interface {
	Ingest(ctx context.Context, m InboundMessage) (stored bool, err error)
	Cursor(ctx context.Context, mailbox string) (SyncCursor, error)
	SaveCursor(ctx context.Context, mailbox string, c SyncCursor) error
}

// ParsedMail is the decoded shape of an RFC 5322 message.
type ParsedMail struct {
	MessageID string
	InReplyTo string
	FromName  string
	FromEmail string
	ToEmail   string
	Subject   string
	Date      time.Time
	BodyText  string
	BodyHTML  string
}

// ParseMessage decodes a raw RFC 5322 message, flattening multipart bodies into
// the first text/plain and text/html alternatives. It is deliberately lenient:
// a message with an unknown charset still yields whatever could be decoded.
func ParseMessage(r io.Reader) (ParsedMail, error) {
	var out ParsedMail
	mr, err := gomail.CreateReader(r)
	if err != nil {
		return out, fmt.Errorf("support: parse message: %w", err)
	}
	defer func() { _ = mr.Close() }()

	if subject, err := mr.Header.Subject(); err == nil {
		out.Subject = strings.TrimSpace(subject)
	}
	if id, err := mr.Header.MessageID(); err == nil {
		out.MessageID = strings.TrimSpace(id)
	}
	if reply := mr.Header.Get("In-Reply-To"); reply != "" {
		out.InReplyTo = strings.TrimSpace(reply)
	}
	if date, err := mr.Header.Date(); err == nil {
		out.Date = date
	}
	if list, err := mr.Header.AddressList("From"); err == nil && len(list) > 0 {
		out.FromName = list[0].Name
		out.FromEmail = strings.ToLower(strings.TrimSpace(list[0].Address))
	}
	if list, err := mr.Header.AddressList("To"); err == nil && len(list) > 0 {
		out.ToEmail = strings.ToLower(strings.TrimSpace(list[0].Address))
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Keep whatever header decoding produced; an unreadable body part
			// must not discard an otherwise valid message.
			break
		}
		contentType := "text/plain"
		if raw := part.Header.Get("Content-Type"); raw != "" {
			if mediaType, _, err := mime.ParseMediaType(raw); err == nil {
				contentType = mediaType
			}
		}
		body, _ := io.ReadAll(io.LimitReader(part.Body, maxBodyBytes))
		switch {
		case out.BodyText == "" && strings.EqualFold(contentType, "text/plain"):
			out.BodyText = strings.TrimSpace(string(body))
		case out.BodyHTML == "" && strings.EqualFold(contentType, "text/html"):
			out.BodyHTML = strings.TrimSpace(string(body))
		}
	}
	if out.BodyText == "" {
		out.BodyText = stripHTMLToText(out.BodyHTML)
	}
	return out, nil
}

var (
	replyPrefix = regexp.MustCompile(`(?i)^\s*((re|fw|fwd|sv|aw)\s*(\[\d+\])?\s*:\s*(\[\d+\])?\s*)+`)
	spaceRun    = regexp.MustCompile(`\s+`)
	htmlTag     = regexp.MustCompile(`(?s)<[^>]*>`)
	htmlEntity  = strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'",
	)
)

// ThreadKey normalizes a subject into a conversation key: reply/forward
// prefixes stripped, whitespace collapsed, lowercased. Two messages about the
// same problem share a key even when one is a reply.
func ThreadKey(subject string) string {
	s := strings.TrimSpace(subject)
	for {
		trimmed := replyPrefix.ReplaceAllString(s, "")
		if trimmed == s {
			break
		}
		s = trimmed
	}
	s = strings.ToLower(spaceRun.ReplaceAllString(s, " "))
	return strings.TrimSpace(s)
}

// NewThreadID returns a random RFC 4122 v4 UUID string for a new conversation.
func NewThreadID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is unrecoverable; fall back to a time-based value
		// so ingest still produces a usable (if not random) identifier.
		now := time.Now().UnixNano()
		for i := 0; i < 8; i++ {
			b[i] = byte(now >> (8 * i))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}

// stripHTMLToText is the plain-text fallback for an HTML-only message.
func stripHTMLToText(html string) string {
	if html == "" {
		return ""
	}
	text := htmlTag.ReplaceAllString(html, " ")
	text = htmlEntity.Replace(text)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}
	return strings.TrimSpace(spaceRun.ReplaceAllString(text, " "))
}
