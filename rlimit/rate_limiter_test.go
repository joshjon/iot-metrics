package rlimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRateLimiter_allowsRequestsWithinLimit(t *testing.T) {
	limiter := NewRateLimiter(10, 5, 0, 0) // 10 req/sec, burst 5
	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.Wait(t.Context(), "foo"))
	}
}

func TestRateLimiter_blocksOnExceedingBurst(t *testing.T) {
	ctx := t.Context()

	interval := 20 * time.Millisecond
	limiter := NewRateLimiter(rate.Every(interval), 1, 0, 0) // 10 req/sec, burst 1
	require.NoError(t, limiter.Wait(ctx, "foo"))             // consume burst

	start := time.Now()
	require.NoError(t, limiter.Wait(ctx, "foo"))
	elapsed := time.Since(start)

	require.GreaterOrEqual(t, elapsed, interval)
	require.Less(t, elapsed, 2*interval)
}

func TestRateLimiter_contextCancel(t *testing.T) {
	limiter := NewRateLimiter(1, 1, 0, 0)
	_ = limiter.Wait(t.Context(), "foo") // consume burst

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	err := limiter.Wait(ctx, "foo")
	require.Error(t, err)
}

func TestRateLimiter_gc(t *testing.T) {
	ttl := 50 * time.Millisecond
	gcInterval := 20 * time.Millisecond

	limiter := NewRateLimiter(1, 1, ttl, gcInterval)
	_ = limiter.Wait(t.Context(), "expired")

	time.Sleep(2 * ttl) // wait for GC to remove entry
	require.NotContains(t, limiter.limiters, "expired")
}
