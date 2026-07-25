// Package mock provides a non-networked, in-memory implementation of Translator for testing and demos.
package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhangshj/library/pkg/translate"
)

// DefaultBatchSize is the default number of concurrent translation requests.
const DefaultBatchSize = 5

// Config holds mock driver configuration.
type Config struct {
	translate.Config
	Delay     time.Duration
	BatchSize int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Config: *translate.DefaultConfig(),
	}
}

// Translator is an in-memory mock that always succeeds with a deterministic response.
type Translator struct {
	mu        sync.RWMutex
	response  string
	err       error
	delay     time.Duration
	batchSize int
}

// Option defines a functional option for the mock Translator.
type Option func(*Translator)

// WithResponse sets the mock translation result.
func WithResponse(text string) Option {
	return func(t *Translator) {
		t.response = text
	}
}

// WithError sets a fixed error to return from Translate.
func WithError(err error) Option {
	return func(t *Translator) {
		t.err = err
	}
}

// WithDelay introduces a simulated network delay on each Translate call.
func WithDelay(d time.Duration) Option {
	return func(t *Translator) {
		t.delay = d
	}
}

// WithBatchSize sets the maximum number of concurrent translation requests.
func WithBatchSize(size int) Option {
	return func(t *Translator) {
		if size > 0 {
			t.batchSize = size
		}
	}
}

// New creates a new mock Translator.
func New(_ *Config, opts ...Option) *Translator {
	t := &Translator{response: "[mock-translated]", batchSize: DefaultBatchSize}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Translate returns the mock response or injected error.
func (t *Translator) Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("mock: context error: %w", err)
		}
	}
	if text == "" {
		return "", fmt.Errorf("mock: source text is empty: %w", translate.ErrInvalidInput)
	}

	t.mu.RLock()
	delay := t.delay
	mockErr := t.err
	mockResp := t.response
	t.mu.RUnlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	if mockErr != nil {
		return "", mockErr
	}
	return mockResp, nil
}

// TranslateBatch translates multiple texts concurrently with bounded parallelism.
// Results preserve input order. The first error encountered cancels the context
// for in-flight requests and returns immediately.
func (t *Translator) TranslateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	results := make([]string, len(texts))
	errCh := make(chan error, 1)
	sem := make(chan struct{}, t.batchSize)
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			sem <- struct{}{}
			defer func() {
				<-sem
				wg.Done()
			}()

			select {
			case <-ctx.Done():
				select {
				case errCh <- fmt.Errorf("mock: batch item %d: %w", idx, ctx.Err()):
				default:
				}
				return
			default:
			}

			translated, err := t.Translate(ctx, txt, sourceLang, targetLang)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("mock: batch item %d: %w", idx, err):
				default:
				}
				return
			}
			results[idx] = translated
		}(i, text)
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return nil, err
	}

	return results, nil
}

// SetResponse updates the mock response at runtime.
func (t *Translator) SetResponse(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.response = text
}

// SetError updates the mock error at runtime.
func (t *Translator) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = err
}

// SetDelay updates the simulated network delay at runtime.
func (t *Translator) SetDelay(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.delay = d
}

// Ensure mock satisfies the Translator interface at compile time.
var _ translate.Translator = (*Translator)(nil)
