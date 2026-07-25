package aliyun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/zhangshj/library/pkg/translate"
)

func TestNew_DefaultConfig(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg)
    if err == nil {
        t.Fatal("expected error for missing credentials")
    }
    if tr != nil {
        t.Fatal("expected nil Translator for invalid config")
    }
}

func TestNew_MissingAccessKey(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg, WithAccessKey("", "secret"))
    if err == nil {
        t.Fatal("expected error for missing access key id")
    }
    if tr != nil {
        t.Fatal("expected nil Translator for invalid config")
    }
}

func TestNew_MissingSecretKey(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg, WithAccessKey("id", ""))
    if err == nil {
        t.Fatal("expected error for missing access key secret")
    }
    if tr != nil {
        t.Fatal("expected nil Translator for invalid config")
    }
}

func TestNew_ValidConfig(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg, WithAccessKey("test-id", "test-secret"))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if tr == nil {
        t.Fatal("expected non-nil Translator")
    }
}

func TestFunctionalOptions(t *testing.T) {
    cfg := DefaultConfig()
    if cfg.Timeout != translate.DefaultTimeout {
        t.Fatalf("expected default timeout %v, got %v", translate.DefaultTimeout, cfg.Timeout)
    }
    if cfg.Region != translate.DefaultRegion {
        t.Fatalf("expected default region %s, got %s", translate.DefaultRegion, cfg.Region)
    }

    WithTimeout(10 * time.Second)(cfg)
    if cfg.Timeout != 10*time.Second {
        t.Fatalf("expected timeout 10s, got %v", cfg.Timeout)
    }

    WithRegion("ap-southeast-1")(cfg)
    if cfg.Region != "ap-southeast-1" {
        t.Fatalf("expected region ap-southeast-1, got %s", cfg.Region)
    }
}

func TestTranslate_EmptyText(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("id", "secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.Translate(context.Background(), "", "en", "zh")
    if err == nil {
        t.Fatal("expected error for empty text")
    }
}

func TestTranslate_ContextCanceled(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("id", "secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    _, err = tr.Translate(ctx, "hello", "en", "zh")
    if err == nil {
        t.Fatal("expected context canceled error")
    }
}

func TestTranslateBatch_EmptySlice(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("id", "secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.TranslateBatch(context.Background(), []string{}, "en", "zh")
    if err == nil {
        t.Fatal("expected error for empty slice")
    }
}

func TestTranslateBatch_EmptyText(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("id", "secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.TranslateBatch(context.Background(), []string{""}, "en", "zh")
    if err == nil {
        t.Fatal("expected error for empty text")
    }
}

func TestTranslateBatch_ContextCanceled(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("id", "secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    _, err = tr.TranslateBatch(ctx, []string{"hello"}, "en", "zh")
    if err == nil {
        t.Fatal("expected context canceled error")
    }
}

func TestTranslateBatch_TableDriven(t *testing.T) {
    tests := []struct {
        name      string
        setup     func() (*Translator, error)
        texts     []string
        expectErr bool
    }{
        {
            name: "missing credentials",
            setup: func() (*Translator, error) {
                return New(nil)
            },
            expectErr: true,
        },
        {
            name: "empty text in batch",
            setup: func() (*Translator, error) {
                return New(DefaultConfig(), WithAccessKey("id", "secret"))
            },
            texts:     []string{"", "hello"},
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tr, err := tt.setup()
            if err != nil {
                if !tt.expectErr {
                    t.Fatalf("unexpected setup error: %v", err)
                }
                return
            }
            _, err = tr.TranslateBatch(context.Background(), tt.texts, "en", "zh")
            if tt.expectErr && err == nil {
                t.Fatal("expected error but got nil")
            }
            if !tt.expectErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
        })
    }
}

func TestTranslateBatch_APIError(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAccessKey("invalid-key-id", "invalid-secret"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.TranslateBatch(context.Background(), []string{"hello"}, "en", "zh")
    if err == nil {
        t.Fatal("expected error from API call")
    }
}

func TestTranslateBatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"Code":      200,
			"Message":   "success",
			"RequestId": "test-request-id",
			"TranslatedList": []map[string]interface{}{
				{"Id": "1", "Translated": "你好"},
				{"Id": "2", "Translated": "世界"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	accessKeyID := os.Getenv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	if accessKeyID == "" {
		accessKeyID = "test-id"
	}
	if accessKeySecret == "" {
		accessKeySecret = "test-secret"
	}

	tr, err := New(DefaultConfig(), WithAccessKey(accessKeyID, accessKeySecret))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.client.SetTransport(&http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	})

	results, err := tr.TranslateBatch(context.Background(), []string{"hello", "world"}, "en", "zh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0] != "你好" {
		t.Fatalf("expected 你好, got %s", results[0])
	}
	if results[1] != "世界" {
		t.Fatalf("expected 世界, got %s", results[1])
	}
}

func TestErrorWrapping(t *testing.T) {
    tests := []struct {
        name      string
        setup     func() (*Translator, error)
        text      string
        src       string
        dst       string
        expectErr bool
    }{
        {
            name: "missing credentials",
            setup: func() (*Translator, error) {
                return New(nil)
            },
            expectErr: true,
        },
        {
            name: "empty text",
            setup: func() (*Translator, error) {
                return New(DefaultConfig(), WithAccessKey("id", "secret"))
            },
            text:      "",
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tr, err := tt.setup()
            if err != nil {
                if !tt.expectErr {
                    t.Fatalf("unexpected setup error: %v", err)
                }
                return
            }
            _, err = tr.Translate(context.Background(), tt.text, tt.src, tt.dst)
            if tt.expectErr && err == nil {
                t.Fatal("expected error but got nil")
            }
            if !tt.expectErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
        })
    }
}
