package mock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zhangshj/library/pkg/translate"
)

func TestNew_Defaults(t *testing.T) {
	tr := New(DefaultConfig())
	if tr == nil {
		t.Fatal("expected non-nil Translator")
	}
	result, err := tr.Translate(context.Background(), "hello", "en", "zh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[mock-translated]" {
		t.Fatalf("unexpected result: %s", result)
	}
}

func TestNew_WithResponse(t *testing.T) {
	tr := New(DefaultConfig(), WithResponse("hello world"))
	result, err := tr.Translate(context.Background(), "test", "en", "zh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("unexpected result: %s", result)
	}
}

func TestNew_WithError(t *testing.T) {
	expected := errors.New("injected error")
	tr := New(DefaultConfig(), WithError(expected))
	_, err := tr.Translate(context.Background(), "test", "en", "zh")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected injected error, got: %v", err)
	}
}

func TestNew_WithDelay(t *testing.T) {
	tr := New(DefaultConfig(), WithDelay(50*time.Millisecond))
	start := time.Now()
	_, err := tr.Translate(context.Background(), "test", "en", "zh")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected delay of at least 50ms, got %v", elapsed)
	}
}

func TestTranslate_EmptyText(t *testing.T) {
	tr := New(DefaultConfig())
	_, err := tr.Translate(context.Background(), "", "en", "zh")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestTranslate_CanceledContext(t *testing.T) {
	tr := New(DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tr.Translate(ctx, "test", "en", "zh")
	if err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestSetResponse_Race(t *testing.T) {
	tr := New(DefaultConfig())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Translate(context.Background(), "test", "en", "zh")
		}()
	}
	wg.Wait()
}

func TestTranslateBatch_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		texts     []string
		expectErr bool
	}{
		{
			name:      "single text",
			opts:      []Option{WithResponse("hello world")},
			texts:     []string{"hello"},
			expectErr: false,
		},
		{
			name:      "multiple texts",
			opts:      []Option{WithResponse("translated")},
			texts:     []string{"hello", "world"},
			expectErr: false,
		},
		{
			name:      "empty text in batch",
			opts:      []Option{WithResponse("translated")},
			texts:     []string{"", "hello"},
			expectErr: true,
		},
		{
			name:      "injected error",
			opts:      []Option{WithError(errors.New("batch error"))},
			texts:     []string{"hello"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := New(DefaultConfig(), tt.opts...)
			results, err := tr.TranslateBatch(context.Background(), tt.texts, "en", "zh")
			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.expectErr && len(results) != len(tt.texts) {
				t.Fatalf("expected %d results, got %d", len(tt.texts), len(results))
			}
		})
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ translate.Translator = (*Translator)(nil)
}

func TestSetDelay(t *testing.T) {
	tr := New(DefaultConfig())
	tr.SetDelay(10 * time.Millisecond)
	start := time.Now()
	_, err := tr.Translate(context.Background(), "test", "en", "zh")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 10*time.Millisecond {
		t.Fatalf("expected delay of at least 10ms, got %v", elapsed)
	}
}

func TestSetError(t *testing.T) {
	tr := New(DefaultConfig())
	expected := errors.New("runtime error")
	tr.SetError(expected)
	_, err := tr.Translate(context.Background(), "test", "en", "zh")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected runtime error, got: %v", err)
	}
}
