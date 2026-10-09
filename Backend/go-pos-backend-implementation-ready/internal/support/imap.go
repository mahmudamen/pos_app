package support

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
)

// IMAPConfig is the inbound mailbox connection.
type IMAPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	Mailbox     string
	TLS         bool
	DialTimeout time.Duration
}

// IMAPFetcher reads new mail from a real IMAP server. It keys off the mailbox
// UID so a restart never re-ingests old messages, and detects a UIDVALIDITY
// change (server-side mailbox rebuild) to force a clean re-sync.
type IMAPFetcher struct {
	cfg    IMAPConfig
	store  Store
	logger *slog.Logger
}

// NewIMAPFetcher builds a fetcher over the given store.
func NewIMAPFetcher(cfg IMAPConfig, store Store, logger *slog.Logger) *IMAPFetcher {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Mailbox == "" {
		cfg.Mailbox = "INBOX"
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 30 * time.Second
	}
	return &IMAPFetcher{cfg: cfg, store: store, logger: logger}
}

// Fetch connects, selects the mailbox, and returns every message with a UID
// greater than the stored cursor. The cursor in the result must only be
// persisted after all returned messages are stored.
func (f *IMAPFetcher) Fetch(ctx context.Context) (FetchResult, error) {
	res := FetchResult{Mailbox: f.cfg.Mailbox}
	addr := net.JoinHostPort(f.cfg.Host, strconv.Itoa(f.cfg.Port))

	var c *imapclient.Client
	var err error
	if f.cfg.TLS {
		dialer := &net.Dialer{Timeout: f.cfg.DialTimeout}
		conn, derr := dialer.DialContext(ctx, "tcp", addr)
		if derr != nil {
			return res, fmt.Errorf("support: imap dial: %w", derr)
		}
		tlsConn := tls.Client(conn, &tls.Config{ServerName: f.cfg.Host, MinVersion: tls.VersionTLS12})
		if derr = tlsConn.HandshakeContext(ctx); derr != nil {
			_ = conn.Close()
			return res, fmt.Errorf("support: imap tls: %w", derr)
		}
		c, err = imapclient.New(tlsConn)
	} else {
		c, err = imapclient.Dial(addr)
	}
	if err != nil {
		return res, fmt.Errorf("support: imap connect: %w", err)
	}
	defer func() { _ = c.Logout() }()

	if err = c.Login(f.cfg.Username, f.cfg.Password); err != nil {
		return res, fmt.Errorf("support: imap login: %w", err)
	}
	mbox, err := c.Select(f.cfg.Mailbox, false)
	if err != nil {
		return res, fmt.Errorf("support: imap select %s: %w", f.cfg.Mailbox, err)
	}

	cursor, err := f.store.Cursor(ctx, f.cfg.Mailbox)
	if err != nil {
		return res, fmt.Errorf("support: imap cursor: %w", err)
	}
	uidValidity := int64(mbox.UidValidity)
	lastUID := cursor.LastUID
	if cursor.UIDValidity != uidValidity {
		// First run, or the server rebuilt the mailbox: resync from the start.
		lastUID = 0
	}

	criteria := imap.NewSearchCriteria()
	seq := new(imap.SeqSet)
	seq.AddRange(uint32(lastUID+1), 0) // last+1:* (0 is "*")
	criteria.Uid = seq
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return res, fmt.Errorf("support: imap search: %w", err)
	}
	if len(uids) == 0 {
		res.Cursor = SyncCursor{UIDValidity: uidValidity, LastUID: lastUID}
		return res, nil
	}

	fetchSet := new(imap.SeqSet)
	fetchSet.AddNum(uids...)
	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid, imap.FetchInternalDate}

	messages := make(chan *imap.Message, 16)
	done := make(chan error, 1)
	go func() { done <- c.UidFetch(fetchSet, items, messages) }()

	maxUID := lastUID
	for msg := range messages {
		if int64(msg.Uid) > maxUID {
			maxUID = int64(msg.Uid)
		}
		literal := msg.GetBody(section)
		if literal == nil {
			continue
		}
		parsed, perr := ParseMessage(literal)
		if perr != nil {
			f.logger.Warn("support: imap parse failed", "uid", msg.Uid, "err", perr)
			continue
		}
		received := msg.InternalDate
		if received.IsZero() {
			received = parsed.Date
		}
		res.Messages = append(res.Messages, InboundMessage{
			Mailbox:     f.cfg.Mailbox,
			ProviderRef: strconv.FormatUint(uint64(msg.Uid), 10),
			MessageID:   parsed.MessageID,
			InReplyTo:   parsed.InReplyTo,
			FromName:    parsed.FromName,
			FromEmail:   parsed.FromEmail,
			ToEmail:     parsed.ToEmail,
			Subject:     parsed.Subject,
			BodyText:    parsed.BodyText,
			BodyHTML:    parsed.BodyHTML,
			ReceivedAt:  received,
		})
	}
	if err := <-done; err != nil {
		return res, fmt.Errorf("support: imap fetch: %w", err)
	}
	res.Cursor = SyncCursor{UIDValidity: uidValidity, LastUID: maxUID}
	return res, nil
}

// Ensure IMAPFetcher satisfies Fetcher.
var _ Fetcher = (*IMAPFetcher)(nil)
