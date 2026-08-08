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

	// Set unconditionally stores key with the given ttl (overwriting any
	// existing value/TTL). Used to start a ban independent of the counter.
	Set(ctx context.Context, key string, ttl time.Duration) error

	// Get returns the current counter value for key (0 if missing).
	Get(ctx context.Context, key string) (int64, error)

	// Del removes key. Used to reset a counter after a successful auth.
	Del(ctx context.Context, key string) error
}

// banKey prefixes the counter key to produce the dedicated ban key.
// A counter that exceeds maxHits triggers a separate ban entry with its own
// TTL (BanDuration), decoupled from the counting window.
func banKey(key string) string { return "ban:" + key }

// Limiter enforces a fixed-window limit for a logical bucket identified by key.
type Limiter struct {
	store       Store
	window      time.Duration
	banDuration time.Duration
	maxHits     int64
	onStoreErr  func(error) // optional hook for alerting on store failures
}

// New constructs a Limiter. Zero-valued Config fields fall back to defaults
// (see DefaultWindow / DefaultMaxHits).
func New(store Store, cfg Config) *Limiter {
	cfg = cfg.normalized()
	return &Limiter{
		store:       store,
		window:      cfg.Window,
		banDuration: cfg.BanDuration,
		maxHits:     cfg.MaxHits,
		onStoreErr:  cfg.OnStoreErr,
	}
}

// Check reports whether key is still allowed WITHOUT incrementing.
// A key is rejected when either:
//   - it is currently banned (a ban:<key> entry exists, set when the limit was
//     exceeded, with TTL BanDuration and independent of the counting window), or
//   - its current failure count has reached maxHits.
func (l *Limiter) Check(ctx context.Context, key string) (allowed bool, err error) {
	banned, berr := l.store.Get(ctx, banKey(key))
	if berr != nil {
		l.alert(berr)
		return false, ErrStoreUnavailable
	}
	if banned > 0 {
		return false, nil
	}
	n, err := l.store.Get(ctx, key)
	if err != nil {
		l.alert(err)
		return false, ErrStoreUnavailable
	}
	return n < l.maxHits, nil
}

// Incr increments the failure counter for key and returns whether the caller
// is still allowed. allowed=false means the request should be rejected.
//
// Counting uses Window as the counter TTL (fixed-window semantics, so the
// window length still matters). When the counter reaches maxHits (i.e. the
// failing request has exhausted the quota), a separate ban entry (ban:<key>)
// is set with TTL BanDuration, starting the ban clock at that moment — NOT at
// the first failure. The counter key is left in place (it naturally expires via
// Window) while the ban entry governs rejection until BanDuration elapses.
//
// Note: Check rejects once n >= maxHits, so the request that brings the count
// to maxHits is the one that triggers the ban here (>=, not >). This closes the
// gap where a caller would be rate-limited but never actually banned.
func (l *Limiter) Incr(ctx context.Context, key string) (allowed bool, err error) {
	n, err := l.store.Incr(ctx, key, l.window)
	if err != nil {
		l.alert(err)
		return false, ErrStoreUnavailable
	}
	if n >= l.maxHits {
		if berr := l.store.Set(ctx, banKey(key), l.banDuration); berr != nil {
			l.alert(berr)
			return false, ErrStoreUnavailable
		}
		return false, nil
	}
	return true, nil
}

// Reset clears the counter AND any active ban for key (called after a
// successful authentication, so a legitimate user is never held by a stale ban).
func (l *Limiter) Reset(ctx context.Context, key string) error {
	if err := l.store.Del(ctx, key); err != nil {
		l.alert(err)
		return ErrStoreUnavailable
	}
	if err := l.store.Del(ctx, banKey(key)); err != nil {
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
