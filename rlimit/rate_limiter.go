package rlimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	limiter *rate.Limiter
	seen    time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
	limit    rate.Limit
	burst    int

	ttl        time.Duration
	gcInterval time.Duration
}

func NewRateLimiter(limit rate.Limit, burst int, ttl, gcInterval time.Duration) *RateLimiter {
	m := &RateLimiter{
		limiters:   make(map[string]*entry),
		limit:      limit,
		burst:      burst,
		ttl:        ttl,
		gcInterval: gcInterval,
	}
	if ttl > 0 && gcInterval > 0 {
		go m.startGC()
	}
	return m
}

func (m *RateLimiter) Wait(ctx context.Context, key string) error {
	getLimiter := func() *rate.Limiter {
		m.mu.Lock()
		defer m.mu.Unlock()

		e, ok := m.limiters[key]
		if !ok {
			e = &entry{
				limiter: rate.NewLimiter(m.limit, m.burst),
				seen:    time.Now(),
			}
			m.limiters[key] = e
			return e.limiter
		}

		e.seen = time.Now()
		return e.limiter
	}

	return getLimiter().Wait(ctx)
}

func (m *RateLimiter) startGC() {
	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()
	for now := range ticker.C {
		m.mu.Lock()
		for key, e := range m.limiters {
			if now.Sub(e.seen) > m.ttl {
				delete(m.limiters, key)
			}
		}
		m.mu.Unlock()
	}
}
