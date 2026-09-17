package identity

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Counter is the Redis-backed abuse-prevention primitive. It is intentionally
// narrow (increment + expiry) so implementations are trivial to swap. It never
// holds authoritative state: PostgreSQL remains the source of truth for trial
// entitlements; Redis only throttles velocity.
type Counter interface {
	// Take increments the counter under key and reports whether the caller
	// may proceed (count <= limit within the window). The first increment for
	// a key sets its TTL. On storage failure it fails open so a Redis outage
	// cannot lock every legitimate signup out.
	Take(ctx context.Context, key string, limit int, window time.Duration) bool
}

// counterClient is the minimal Redis surface Counter needs.
type counterClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// goRedisCounter adapts *redis.Client to counterClient.
type goRedisCounter struct{ client *redis.Client }

func (g goRedisCounter) Incr(ctx context.Context, key string) (int64, error) {
	return g.client.Incr(ctx, key).Result()
}

func (g goRedisCounter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return g.client.Expire(ctx, key, expiration).Err()
}

// RedisCounter implements Counter on top of Redis. Keys are namespaced by
// prefix so a shared Redis instance can host unrelated counters.
type RedisCounter struct {
	client counterClient
	prefix string
}

func NewRedisCounter(client *redis.Client, prefix string) *RedisCounter {
	if prefix == "" {
		prefix = "abuse:"
	}
	return &RedisCounter{client: goRedisCounter{client: client}, prefix: prefix}
}

func (c *RedisCounter) Take(ctx context.Context, key string, limit int, window time.Duration) bool {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Hour
	}
	count, err := c.client.Incr(ctx, c.prefix+key)
	if err != nil {
		return true // fail open: Redis outage must not block signups
	}
	if count == 1 {
		_ = c.client.Expire(ctx, c.prefix+key, window)
	}
	return count <= int64(limit)
}

// MemoryCounter is the single-instance fallback Counter (mirrors the
// existing in-memory login limiter). Same fail-open semantics.
type MemoryCounter struct {
	mu       sync.Mutex
	counters map[string][]time.Time
	prefix   string
}

func NewMemoryCounter(prefix string) *MemoryCounter {
	if prefix == "" {
		prefix = "abuse:"
	}
	return &MemoryCounter{counters: make(map[string][]time.Time), prefix: prefix}
}

func (c *MemoryCounter) Take(ctx context.Context, key string, limit int, window time.Duration) bool {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Hour
	}
	now := time.Now()
	cutoff := now.Add(-window)
	c.mu.Lock()
	defer c.mu.Unlock()
	key = c.prefix + key
	entries := c.counters[key]
	valid := entries[:0]
	for _, t := range entries {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= limit {
		c.counters[key] = valid
		return false
	}
	c.counters[key] = append(valid, now)
	return true
}
