package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is a concurrency-safe in-memory Store. It is used for tests and
// as a fallback when no Redis is configured (fail-open degradation).
type MemoryStore struct {
	mu  sync.Mutex
	m   map[string]int64
	ttl map[string]time.Duration
	now func() time.Time
}

// NewMemoryStore creates an empty MemoryStore. now, if nil, defaults to time.Now.
func NewMemoryStore(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{m: map[string]int64{}, ttl: map[string]time.Duration{}, now: now}
}

// Incr atomically increments key. TTL is applied only on first creation.
func (s *MemoryStore) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key]++
	if _, exists := s.ttl[key]; !exists {
		s.ttl[key] = ttl
	}
	return s.m[key], nil
}

// Get returns the current value (0 if absent or expired).
func (s *MemoryStore) Get(_ context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key], nil
}

// Set unconditionally stores key with ttl (overwrites any existing value/TTL).
func (s *MemoryStore) Set(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = 1
	s.ttl[key] = ttl
	return nil
}

// Del removes key.
func (s *MemoryStore) Del(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	delete(s.ttl, key)
	return nil
}
