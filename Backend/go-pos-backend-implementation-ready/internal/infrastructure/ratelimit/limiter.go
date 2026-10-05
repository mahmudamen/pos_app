package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(key string) bool
	// Reset drops the recorded attempts for key. Used to refund a token when the
	// request that consumed it was not actually a failed attempt, so only real
	// credential guesses accumulate.
	Reset(key string)
}

type MemoryLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewMemory(limit int, window time.Duration) *MemoryLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &MemoryLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *MemoryLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.attempts[key]
	valid := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts[key] = valid

	if len(valid) >= rl.limit {
		return false
	}
	rl.attempts[key] = append(rl.attempts[key], now)
	return true
}

func (rl *MemoryLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, key)
}

// counterClient is the minimal redis surface the RedisLimiter needs,
// kept primitive-returning so tests can fake it without go-redis.
type counterClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// goRedisClient adapts *redis.Client to counterClient.
type goRedisClient struct {
	client *redis.Client
}

func (g goRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return g.client.Incr(ctx, key).Result()
}

func (g goRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return g.client.Expire(ctx, key, expiration).Err()
}

func (g goRedisClient) Del(ctx context.Context, keys ...string) error {
	return g.client.Del(ctx, keys...).Err()
}

type RedisLimiter struct {
	client counterClient
	limit  int64
	window time.Duration
	prefix string
}

func NewRedis(client *redis.Client, limit int, window time.Duration) *RedisLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RedisLimiter{
		client: goRedisClient{client: client},
		limit:  int64(limit),
		window: window,
		prefix: "ratelimit:",
	}
}

func (rl *RedisLimiter) Allow(key string) bool {
	ctx := context.Background()
	counter, err := rl.client.Incr(ctx, rl.prefix+key)
	if err != nil {
		return true
	}
	if counter == 1 {
		_ = rl.client.Expire(ctx, rl.prefix+key, rl.window)
	}
	return counter <= rl.limit
}

// Reset drops the counter. A redis error is swallowed on purpose: a limiter that
// cannot refund a token should fail towards allowing the request, exactly like
// the Incr error path above.
func (rl *RedisLimiter) Reset(key string) {
	_ = rl.client.Del(context.Background(), rl.prefix+key)
}
