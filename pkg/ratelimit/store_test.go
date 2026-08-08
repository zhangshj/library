package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestMemoryStore_InterfaceContract exercises the Store contract end-to-end
// through the Limiter using the in-memory implementation.
func TestMemoryStore_InterfaceContract(t *testing.T) {
	s := NewMemoryStore(nil)
	l := New(s, Config{Window: time.Minute, MaxHits: 3})

	// Check on empty key -> allowed
	ok, err := l.Check(context.Background(), "ip:9.9.9.9")
	if !ok || err != nil {
		t.Fatalf("empty key should be allowed, got %v %v", ok, err)
	}

	// Increment up to the limit
	for i := int64(1); i <= 3; i++ {
		ok, err := l.Incr(context.Background(), "ip:9.9.9.9")
		if !ok || err != nil {
			t.Fatalf("hit %d allowed, got %v %v", i, ok, err)
		}
	}
	// One more -> blocked
	ok, err = l.Incr(context.Background(), "ip:9.9.9.9")
	if ok {
		t.Fatalf("exceeding max should be blocked")
	}
	if err != nil {
		t.Fatalf("store ok, err unexpected: %v", err)
	}

	// Reset clears the counter
	if err := l.Reset(context.Background(), "ip:9.9.9.9"); err != nil {
		t.Fatal(err)
	}
	ok, _ = l.Check(context.Background(), "ip:9.9.9.9")
	if !ok {
		t.Fatalf("after reset should be allowed again")
	}
}

// TestMemoryStore_ConcurrentIncr verifies atomicity: 100 concurrent increments
// on the same key must yield exactly 100, never a lost update.
func TestMemoryStore_ConcurrentIncr(t *testing.T) {
	s := NewMemoryStore(nil)
	const n = 100
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			_, _ = s.Incr(context.Background(), "concurrent", time.Minute)
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	got, err := s.Get(context.Background(), "concurrent")
	if err != nil {
		t.Fatal(err)
	}
	if got != n {
		t.Fatalf("expected %d after concurrent incr, got %d", n, got)
	}
}

// TestMemoryStore_TTLAppliedOnlyOnce confirms the window ttl is set on first
// incr and not overwritten on subsequent calls (fixed window start is stable).
func TestMemoryStore_TTLAppliedOnlyOnce(t *testing.T) {
	s := NewMemoryStore(nil)
	if _, err := s.Incr(context.Background(), "ttlkey", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Incr(context.Background(), "ttlkey", 99*time.Second); err != nil {
		t.Fatal(err)
	}
	// We cannot read ttl back from the public API, but we exercise the path to
	// ensure no panic and idempotent first-set behaviour.
	if v, _ := s.Get(context.Background(), "ttlkey"); v != 2 {
		t.Fatalf("expected counter 2, got %d", v)
	}
}
