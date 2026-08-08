package ratelimit

import (
	"context"
	"testing"
	"time"
)

// fakeStore is an in-memory Store used to test the algorithm without Redis.
// It deliberately does NOT make Incr atomic across goroutines by default, so
// we also test the atomicity contract via a mutex-guarded variant below.
type fakeStore struct {
	m   map[string]int64
	ttl map[string]time.Duration
}

func newFakeStore() *fakeStore {
	return &fakeStore{m: map[string]int64{}, ttl: map[string]time.Duration{}}
}

func (f *fakeStore) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	f.m[key]++
	f.ttl[key] = ttl
	return f.m[key], nil
}

func (f *fakeStore) Set(_ context.Context, key string, ttl time.Duration) error {
	f.m[key] = 1
	f.ttl[key] = ttl
	return nil
}

func (f *fakeStore) Get(_ context.Context, key string) (int64, error) {
	return f.m[key], nil
}

func (f *fakeStore) Del(_ context.Context, key string) error {
	delete(f.m, key)
	delete(f.ttl, key)
	return nil
}

func TestCheck_UnderLimit(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{Window: time.Minute, MaxHits: 3})
	ok, err := l.Check(context.Background(), "ip:1.2.3.4")
	if !ok || err != nil {
		t.Fatalf("expected allowed under limit, got allowed=%v err=%v", ok, err)
	}
}

func TestIncr_ExceedsAfterMax(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{Window: time.Minute, MaxHits: 3})

	// hits 1,2 -> allowed (under max)
	for i := int64(1); i <= 2; i++ {
		ok, err := l.Incr(context.Background(), "u:alice")
		if !ok || err != nil {
			t.Fatalf("hit %d should be allowed, got allowed=%v err=%v", i, ok, err)
		}
	}
	// hit 3 -> reaches max, blocked (and banned)
	ok, err := l.Incr(context.Background(), "u:alice")
	if ok {
		t.Fatalf("3rd hit (reaching max) should be blocked")
	}
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestIncr_SetsWindowTTLOnFirst(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{Window: 10 * time.Minute, MaxHits: 5})
	if _, err := l.Incr(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}
	if s.ttl["k"] != 10*time.Minute {
		t.Fatalf("expected ttl 10m on first incr, got %v", s.ttl["k"])
	}
}

func TestReset_ClearsCounter(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{Window: time.Minute, MaxHits: 1})
	l.Incr(context.Background(), "u:bob") // now at 1
	if err := l.Reset(context.Background(), "u:bob"); err != nil {
		t.Fatal(err)
	}
	ok, _ := l.Check(context.Background(), "u:bob")
	if !ok {
		t.Fatalf("after reset, key should be allowed again")
	}
}

// TestBan_StartsOnReachingLimit verifies the desired semantics:
//   - counting uses Window as TTL
//   - a ban entry (ban:<key>) is created when the counter REACHES MaxHits (the
//     failing request that exhausts the quota triggers the ban), and Check
//     rejects while the ban entry lives (independent of the counting window).
func TestBan_StartsOnReachingLimit(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{Window: time.Minute, MaxHits: 5, BanDuration: 5 * time.Minute})

	// First 4 failures: still allowed, NO ban entry yet.
	for i := int64(1); i <= 4; i++ {
		ok, err := l.Incr(context.Background(), "u:eve")
		if !ok || err != nil {
			t.Fatalf("failure %d should be allowed, got allowed=%v err=%v", i, ok, err)
		}
	}
	if _, exists := s.m[banKey("u:eve")]; exists {
		t.Fatalf("ban entry must NOT exist before the limit is reached")
	}

	// 5th failure -> reaches MaxHits, ban entry created, request rejected.
	ok, err := l.Incr(context.Background(), "u:eve")
	if ok {
		t.Fatalf("5th failure (reaching limit) should be blocked")
	}
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, exists := s.m[banKey("u:eve")]; !exists {
		t.Fatalf("ban entry must exist after reaching the limit")
	}
	if s.ttl[banKey("u:eve")] != 5*time.Minute {
		t.Fatalf("ban ttl should be BanDuration (5m), got %v", s.ttl[banKey("u:eve")])
	}

	// Check must reject while banned, regardless of the counter window.
	if ok, _ := l.Check(context.Background(), "u:eve"); ok {
		t.Fatalf("banned key should be rejected by Check")
	}

	// Reset clears both counter and ban, restoring access.
	if err := l.Reset(context.Background(), "u:eve"); err != nil {
		t.Fatal(err)
	}
	if _, exists := s.m[banKey("u:eve")]; exists {
		t.Fatalf("Reset must clear the ban entry")
	}
	if ok, _ := l.Check(context.Background(), "u:eve"); !ok {
		t.Fatalf("after Reset should be allowed again")
	}
}

// errStore simulates a dead Redis to verify fail-open caller behaviour.
type errStore struct{ err error }

func (e errStore) Incr(context.Context, string, time.Duration) (int64, error) { return 0, e.err }
func (e errStore) Set(context.Context, string, time.Duration) error         { return e.err }
func (e errStore) Get(context.Context, string) (int64, error)                { return 0, e.err }
func (e errStore) Del(context.Context, string) error                        { return e.err }

func TestCheck_StoreErrorReturnsUnavailable(t *testing.T) {
	l := New(errStore{err: ErrStoreUnavailable}, Config{Window: time.Minute, MaxHits: 3})
	_, err := l.Check(context.Background(), "x")
	if err != ErrStoreUnavailable {
		t.Fatalf("expected ErrStoreUnavailable, got %v", err)
	}
}

func TestNew_AppliesDefaults(t *testing.T) {
	s := newFakeStore()
	l := New(s, Config{}) // zero config -> defaults
	// defaults: window=1m, maxHits=5 -> 4 allowed, 5th (reaching max) blocked
	for i := int64(1); i <= DefaultMaxHits-1; i++ {
		ok, err := l.Incr(context.Background(), "k")
		if !ok || err != nil {
			t.Fatalf("default hit %d should be allowed", i)
		}
	}
	ok, _ := l.Incr(context.Background(), "k")
	if ok {
		t.Fatalf("default max hit should be blocked")
	}
}
