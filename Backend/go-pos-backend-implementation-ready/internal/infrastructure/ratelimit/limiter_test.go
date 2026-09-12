package ratelimit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryAllowsUpToLimit(t *testing.T) {
	rl := NewMemory(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !rl.Allow("key") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if rl.Allow("key") {
		t.Fatal("expected request after limit to be blocked")
	}
}

func TestMemorySeparatesKeys(t *testing.T) {
	rl := NewMemory(1, time.Minute)
	if !rl.Allow("a") {
		t.Fatal("first request for 'a' should be allowed")
	}
	if !rl.Allow("b") {
		t.Fatal("'b' should not share the counter with 'a'")
	}
	if rl.Allow("a") {
		t.Fatal("'a' should be blocked after its own limit")
	}
}

func TestMemoryResetsAfterWindow(t *testing.T) {
	rl := NewMemory(1, 30*time.Millisecond)
	if !rl.Allow("key") {
		t.Fatal("first request should be allowed")
	}
	if rl.Allow("key") {
		t.Fatal("second request within the window should be blocked")
	}
	time.Sleep(40 * time.Millisecond)
	if !rl.Allow("key") {
		t.Fatal("request after the window should be allowed")
	}
}

func TestRedisAllowsWithinLimit(t *testing.T) {
	client := &fakeCounter{}
	rl := NewRedisTest(client, 2, time.Minute)
	if !rl.Allow("ip") {
		t.Fatal("first request should be allowed")
	}
	if !rl.Allow("ip") {
		t.Fatal("second request should be allowed")
	}
	if rl.Allow("ip") {
		t.Fatal("third request should be blocked")
	}
}

func TestRedisFailsOpenOnClientError(t *testing.T) {
	client := &fakeCounter{errOnIncr: true}
	rl := NewRedisTest(client, 1, time.Minute)
	if !rl.Allow("ip") {
		t.Fatal("limiter should allow when redis is down")
	}
}

func TestRedisSetsExpiryOnFirstRequest(t *testing.T) {
	client := &fakeCounter{}
	rl := NewRedisTest(client, 3, 90*time.Second)
	if !rl.Allow("ip") {
		t.Fatal("first request should be allowed")
	}
	if !client.tenured("ratelimit:ip") {
		t.Fatal("expected expiry to be set on first request")
	}
}

func NewRedisTest(client counterClient, limit int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{client: client, limit: int64(limit), window: window, prefix: "ratelimit:"}
}

type fakeCounter struct {
	mu        sync.Mutex
	counts    map[string]int64
	expired   map[string]time.Duration
	errOnIncr bool
}

func (f *fakeCounter) Incr(_ context.Context, key string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.errOnIncr {
		return 0, errors.New("connection refused")
	}
	if f.counts == nil {
		f.counts = map[string]int64{}
	}
	f.counts[key]++
	return f.counts[key], nil
}

func (f *fakeCounter) Expire(_ context.Context, key string, expiration time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.expired == nil {
		f.expired = map[string]time.Duration{}
	}
	f.expired[key] = expiration
	return nil
}

func (f *fakeCounter) tenured(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.expired[key] != 0
}
