package ratelimit

import "time"

// Default values for the fixed-window limiter. These are sensible starting
// points for login/CheckLogin brute-force protection; tune per deployment.
const (
	// DefaultWindow is the fixed-window length if Config.Window is zero.
	DefaultWindow = time.Minute
	// DefaultMaxHits is the total allowed hits per window if Config.MaxHits is zero.
	DefaultMaxHits int64 = 5
)

// Config holds the construction parameters for a Limiter.
// It performs no network access and stores no secrets; all fields are safe
// to construct in-process.
type Config struct {
	// Window is the fixed-window length. Defaults to DefaultWindow when zero.
	Window time.Duration
	// MaxHits is the total allowed hits per window. Defaults to DefaultMaxHits when zero.
	MaxHits int64
	// OnStoreErr, if set, is invoked whenever the underlying Store returns an
	// error. Use it to emit metrics/alerts. It must not block the request path.
	OnStoreErr func(error)
}

// normalized returns a Config with zero-valued fields replaced by defaults.
func (c Config) normalized() Config {
	if c.Window <= 0 {
		c.Window = DefaultWindow
	}
	if c.MaxHits <= 0 {
		c.MaxHits = DefaultMaxHits
	}
	return c
}
