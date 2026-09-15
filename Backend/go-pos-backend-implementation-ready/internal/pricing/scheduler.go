package pricing

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Scheduler runs the price refresh once at start and then on a fixed interval.
// It lives inside the API process; Stop must be called on shutdown.
type Scheduler struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	stopped bool
}

// Start launches the refresh loop. interval is the delay between runs;
// variationPct clamps per-item price variation (0 disables it). rng may be nil
// for a pure catalog refresh (no jitter).
func Start(pool *pgxpool.Pool, interval time.Duration, variationPct int, rng *rand.Rand, logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{cancel: cancel, done: make(chan struct{})}
	go s.loop(ctx, pool, interval, variationPct, rng, logger)
	return s
}

func (s *Scheduler) loop(ctx context.Context, pool *pgxpool.Pool, interval time.Duration, variationPct int, rng *rand.Rand, logger *slog.Logger) {
	defer close(s.done)
	run := func() {
		if _, err := RunOnce(ctx, pool, variationPct, rng, logger); err != nil {
			logger.Warn("price refresh failed", "error", err)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// Stop cancels the loop and waits for the in-flight run to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	s.cancel()
	<-s.done
}
