package support

import (
	"context"
	"log/slog"
	"time"
)

// Poller periodically pulls new inbound mail from a Fetcher and stores it. It
// is deliberately sequential and idempotent: a store failure leaves the cursor
// unmoved, so the next poll retries the same batch (deduped by provider_ref).
type Poller struct {
	fetcher  Fetcher
	store    Store
	interval time.Duration
	logger   *slog.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

// NewPoller builds a poller. An interval <= 0 defaults to two minutes.
func NewPoller(fetcher Fetcher, store Store, interval time.Duration, logger *slog.Logger) *Poller {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	return &Poller{fetcher: fetcher, store: store, interval: interval, logger: logger}
}

// Start launches the poll loop in the background and returns immediately.
func (p *Poller) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	go p.run(ctx)
}

// Stop cancels the loop and waits for it to finish.
func (p *Poller) Stop() {
	if p.cancel == nil {
		return
	}
	p.cancel()
	<-p.done
	p.cancel = nil
}

func (p *Poller) run(ctx context.Context) {
	defer close(p.done)
	// Poll once immediately so a fresh deploy does not wait a full interval.
	p.pollAndLog(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollAndLog(ctx)
		}
	}
}

func (p *Poller) pollAndLog(ctx context.Context) {
	if err := p.PollOnce(ctx); err != nil {
		p.logger.Warn("support: inbox poll failed", "err", err)
	}
}

// PollOnce runs a single fetch + ingest cycle. It stores every message first
// and only advances the cursor when the whole batch is persisted, so a partial
// failure cannot skip mail.
func (p *Poller) PollOnce(ctx context.Context) error {
	result, err := p.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}
	stored := 0
	for _, msg := range result.Messages {
		ok, ierr := p.store.Ingest(ctx, msg)
		if ierr != nil {
			return ierr
		}
		if ok {
			stored++
		}
	}
	if err := p.store.SaveCursor(ctx, result.Mailbox, result.Cursor); err != nil {
		return err
	}
	if stored > 0 {
		p.logger.Info("support: inbox ingested", "mailbox", result.Mailbox, "stored", stored)
	}
	return nil
}
