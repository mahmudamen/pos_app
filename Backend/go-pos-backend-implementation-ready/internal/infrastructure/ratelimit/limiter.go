package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(key string) bool
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

// counterClient is the minimal redis surface the RedisLimiter needs,
// kept primitive-returning so tests can fake it without go-redis.
type counterClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
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
