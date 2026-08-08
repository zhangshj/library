// Package ratelimit provides a pluggable, framework-agnostic rate limiter
// with pluggable storage backends. The core algorithm lives here and only
// depends on the Store interface, so it can be tested with an in-memory fake.
package ratelimit

import (
	"context"
	"errors"
	"time"
)

// ErrStoreUnavailable is returned by Limiter methods when the underlying store
// fails. Callers decide between fail-open (ignore) and fail-closed (reject).
var ErrStoreUnavailable = errors.New("ratelimit: store unavailable")

// Store is the minimal KV contract the rate limiting algorithm needs.
// Implementations must make Incr atomic: concurrent calls for the same key
// must each observe a strictly increasing counter, not a lost update.
type Store interface {
	// Incr atomically increments key by 1 and returns the new value.
	// If the key did not exist, it is created with value 1 and the given ttl.
	// If the key already existed, ttl is ignored (sliding window start is fixed).
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// Get returns the current counter value for key (0 if missing).
	Get(ctx context.Context, key string) (int64, error)

	// Del removes key. Used to reset a counter after a successful auth.
	Del(ctx context.Context, key string) error
}

// Limiter enforces a fixed-window limit for a logical bucket identified by key.
type Limiter struct {
	store    Store
	window   time.Duration
	maxHits  int64
	onStoreErr func(error) // optional hook for alerting on store failures
}

// New constructs a Limiter. Zero-valued Config fields fall back to defaults
// (see DefaultWindow / DefaultMaxHits).
func New(store Store, cfg Config) *Limiter {
	cfg = cfg.normalized()
	return &Limiter{
		store:      store,
		window:     cfg.Window,
		maxHits:    cfg.MaxHits,
		onStoreErr: cfg.OnStoreErr,
	}
}

// Check reports whether key is still under the limit WITHOUT incrementing.
// It returns allowed=true when the current count is below maxHits.
func (l *Limiter) Check(ctx context.Context, key string) (allowed bool, err error) {
	n, err := l.store.Get(ctx, key)
	if err != nil {
		l.alert(err)
		return false, ErrStoreUnavailable
	}
	return n < l.maxHits, nil
}

// Incr increments the failure counter for key and returns whether the limit
// is now exceeded. allowed=false means the caller should reject the request.
func (l *Limiter) Incr(ctx context.Context, key string) (allowed bool, err error) {
	n, err := l.store.Incr(ctx, key, l.window)
	if err != nil {
		l.alert(err)
		return false, ErrStoreUnavailable
	}
	return n <= l.maxHits, nil
}

// Reset clears the counter for key (called after a successful authentication).
func (l *Limiter) Reset(ctx context.Context, key string) error {
	if err := l.store.Del(ctx, key); err != nil {
		l.alert(err)
		return ErrStoreUnavailable
	}
	return nil
}

func (l *Limiter) alert(err error) {
	if l.onStoreErr != nil {
		l.onStoreErr(err)
	}
}
