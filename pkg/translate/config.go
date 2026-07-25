// Package translate provides top-level configuration for the translation component.
package translate

import "time"

// DefaultTimeout is the default HTTP timeout for translation requests.
const DefaultTimeout = 5 * time.Second

// DefaultRegion is the default cloud region when not explicitly configured.
const DefaultRegion = "cn-hangzhou"

// Config holds non-sensitive, shareable configuration for translation drivers.
// Sensitive credentials MUST be injected via Functional Options at runtime.
type Config struct {
	Timeout time.Duration
	Region  string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Timeout: DefaultTimeout,
		Region:  DefaultRegion,
	}
}

// WithTimeout sets the HTTP timeout for all translation requests.
func WithTimeout(timeout time.Duration) func(*Config) {
	return func(c *Config) {
		if timeout > 0 {
			c.Timeout = timeout
		}
	}
}

// WithRegion sets the target cloud region for translation requests.
func WithRegion(region string) func(*Config) {
	return func(c *Config) {
		if region != "" {
			c.Region = region
		}
	}
}
