package baidu

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestNew_MissingAppID(t *testing.T) {
	cfg := DefaultConfig()
	tr, err := New(cfg, WithAppID(""), WithAPIKey("id"), WithSecretKey("secret"))
	if err == nil {
		t.Fatal("expected error for missing app id")
	}
	if tr != nil {
		t.Fatal("expected nil Translator for invalid config")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	tr, err := New(cfg, WithAppID("id"), WithAPIKey(""))
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
	if tr != nil {
		t.Fatal("expected nil Translator for invalid config")
	}
}

func TestNew_MissingSecretKey(t *testing.T) {
	cfg := DefaultConfig()
	tr, err := New(cfg, WithAppID("id"), WithAPIKey("id"), WithSecretKey(""))
	if err == nil {
		t.Fatal("expected error for missing secret key")
	}
	if tr != nil {
		t.Fatal("expected nil Translator for invalid config")
	}
}

func TestNew_InvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	_, err := New(cfg, WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"), WithMode("unknown"))
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestNew_InvalidLLMAuth(t *testing.T) {
	cfg := DefaultConfig()
	_, err := New(cfg, WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"), WithMode(ModeLLM), WithLLMAuth("unknown"))
	if err == nil {
		t.Fatal("expected error for invalid llm auth")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
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
	if cfg.Mode != ModeGeneral {
		t.Fatalf("expected default mode %s, got %s", ModeGeneral, cfg.Mode)
	}
	if cfg.LLMAuth != LLMAuthBearer {
		t.Fatalf("expected default llm auth %s, got %s", LLMAuthBearer, cfg.LLMAuth)
	}

	WithTimeout(10 * time.Second)(cfg)
	if cfg.Timeout != 10*time.Second {
		t.Fatalf("expected timeout 10s, got %v", cfg.Timeout)
	}

	WithRegion("ap-southeast-1")(cfg)
	if cfg.Region != "ap-southeast-1" {
		t.Fatalf("expected region ap-southeast-1, got %s", cfg.Region)
	}

	WithMode(ModeLLM)(cfg)
	if cfg.Mode != ModeLLM {
		t.Fatalf("expected mode %s, got %s", ModeLLM, cfg.Mode)
	}

	WithLLMAuth(LLMAuthSign)(cfg)
	if cfg.LLMAuth != LLMAuthSign {
		t.Fatalf("expected llm auth %s, got %s", LLMAuthSign, cfg.LLMAuth)
	}

	WithTermIDs("id1,id2")(cfg)
	if cfg.TermIDs != "id1,id2" {
		t.Fatalf("expected termIds id1,id2, got %s", cfg.TermIDs)
	}

	WithReference("academic style")(cfg)
	if cfg.Reference != "academic style" {
		t.Fatalf("expected reference academic style, got %s", cfg.Reference)
	}
}

func TestTranslate_EmptyText(t *testing.T) {
	tr, err := New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = tr.Translate(context.Background(), "", "en", "zh")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestTranslate_ContextCanceled(t *testing.T) {
	tr, err := New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
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
	tr, err := New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = tr.TranslateBatch(context.Background(), []string{}, "en", "zh")
	if err == nil {
		t.Fatal("expected error for empty slice")
	}
}

func TestTranslateBatch_EmptyText(t *testing.T) {
	tr, err := New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = tr.TranslateBatch(context.Background(), []string{""}, "en", "zh")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestTranslateBatch_ContextCanceled(t *testing.T) {
	tr, err := New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
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
				return New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
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

func TestTranslateBatch_Success_General(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		q := r.FormValue("q")
		parts := strings.Split(q, "\n")

		transResult := make([]map[string]string, len(parts))
		for i, part := range parts {
			if strings.Contains(part, "hello") {
				transResult[i] = map[string]string{"src": part, "dst": "你好"}
			} else if strings.Contains(part, "world") {
				transResult[i] = map[string]string{"src": part, "dst": "世界"}
			} else {
				transResult[i] = map[string]string{"src": part, "dst": part}
			}
		}

		resp := map[string]interface{}{
			"from":         "en",
			"to":           "zh",
			"trans_result": transResult,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetGeneralBaseURL(server.URL + "/api/trans/vip/translate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeGeneral
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

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

func TestTranslateBatch_Success_LLM_Bearer(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		q := body["q"].(string)
		parts := strings.Split(q, "\n")

		transResult := make([]map[string]string, len(parts))
		for i, part := range parts {
			if strings.Contains(part, "hello") {
				transResult[i] = map[string]string{"src": part, "dst": "你好"}
			} else if strings.Contains(part, "world") {
				transResult[i] = map[string]string{"src": part, "dst": "世界"}
			} else {
				transResult[i] = map[string]string{"src": part, "dst": part}
			}
		}

		resp := map[string]interface{}{
			"from":         "en",
			"to":           "zh",
			"trans_result": transResult,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLLMBaseURL(server.URL + "/ait/api/aiTextTranslate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeLLM
	cfg.LLMAuth = LLMAuthBearer
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

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

	if receivedAuth != "Bearer test-api-key" {
		t.Fatalf("expected Authorization header 'Bearer test-api-key', got %q", receivedAuth)
	}
}

func TestTranslateBatch_Success_LLM_Sign(t *testing.T) {
	var receivedQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		q := r.FormValue("q")
		parts := strings.Split(q, "\n")

		transResult := make([]map[string]string, len(parts))
		for i, part := range parts {
			if strings.Contains(part, "hello") {
				transResult[i] = map[string]string{"src": part, "dst": "你好"}
			} else if strings.Contains(part, "world") {
				transResult[i] = map[string]string{"src": part, "dst": "世界"}
			} else {
				transResult[i] = map[string]string{"src": part, "dst": part}
			}
		}

		resp := map[string]interface{}{
			"from":         "en",
			"to":           "zh",
			"trans_result": transResult,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLLMBaseURL(server.URL + "/ait/api/aiTextTranslate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeLLM
	cfg.LLMAuth = LLMAuthSign
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

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

	if receivedQuery.Get("appid") != "test-appid" {
		t.Fatalf("expected appid test-appid, got %s", receivedQuery.Get("appid"))
	}
	if receivedQuery.Get("sign") == "" {
		t.Fatal("expected sign parameter in URL")
	}
	if receivedQuery.Get("salt") == "" {
		t.Fatal("expected salt parameter in URL")
	}
}

func TestTranslateBatch_APIError(t *testing.T) {
	tr, err := New(DefaultConfig(), WithAppID("invalid-appid"), WithAPIKey("invalid-api-key"), WithSecretKey("invalid-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = tr.TranslateBatch(context.Background(), []string{"hello"}, "en", "zh")
	if err == nil {
		t.Fatal("expected error from API call")
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
				return New(DefaultConfig(), WithAppID("id"), WithAPIKey("id"), WithSecretKey("secret"))
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

func TestTranslateBatch_ResultCountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"from":         "en",
			"to":           "zh",
			"trans_result": []map[string]string{{"src": "hello", "dst": "你好"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetGeneralBaseURL(server.URL + "/api/trans/vip/translate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeGeneral
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

	_, err = tr.TranslateBatch(context.Background(), []string{"hello", "world"}, "en", "zh")
	if err == nil {
		t.Fatal("expected error for result count mismatch")
	}
}

func TestTranslateLLM_Bearer_AuthorizationHeader(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		resp := map[string]interface{}{
			"from": "en",
			"to":   "zh",
			"trans_result": []map[string]string{
				{"src": "hello", "dst": "你好"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLLMBaseURL(server.URL + "/ait/api/aiTextTranslate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeLLM
	cfg.LLMAuth = LLMAuthBearer
	tr, err := New(cfg, WithAppID("my-appid"), WithAPIKey("my-api-key"), WithSecretKey("my-secret-key"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

	_, err = tr.Translate(context.Background(), "hello", "en", "zh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer my-api-key" {
		t.Fatalf("expected Authorization header 'Bearer my-api-key', got %q", receivedAuth)
	}
}

func TestTranslateLLM_Sign_RequestBodyContainsAppID(t *testing.T) {
	var receivedQuery url.Values
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		receivedBody, _ = io.ReadAll(r.Body)
		resp := map[string]interface{}{
			"from": "en",
			"to":   "zh",
			"trans_result": []map[string]string{
				{"src": "hello", "dst": "你好"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLLMBaseURL(server.URL + "/ait/api/aiTextTranslate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeLLM
	cfg.LLMAuth = LLMAuthSign
	tr, err := New(cfg, WithAppID("my-appid"), WithAPIKey("my-api-key"), WithSecretKey("my-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

	_, err = tr.Translate(context.Background(), "hello", "en", "zh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery.Get("appid") != "my-appid" {
		t.Fatalf("expected appid 'my-appid', got %v", receivedQuery.Get("appid"))
	}
	if receivedQuery.Get("from") != "en" {
		t.Fatalf("expected from 'en', got %v", receivedQuery.Get("from"))
	}
	if receivedQuery.Get("to") != "zh" {
		t.Fatalf("expected to 'zh', got %v", receivedQuery.Get("to"))
	}
	if receivedQuery.Get("q") != "hello" {
		t.Fatalf("expected q 'hello', got %v", receivedQuery.Get("q"))
	}
	if receivedQuery.Get("sign") == "" {
		t.Fatal("expected sign parameter in URL")
	}
	if receivedQuery.Get("salt") == "" {
		t.Fatal("expected salt parameter in URL")
	}
	if string(receivedBody) != "hello" {
		t.Fatalf("expected body 'hello', got %s", string(receivedBody))
	}
}

func TestTranslateLLM_Sign_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"error_code": 54000,
			"error_msg":  "appid is empty",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetLLMBaseURL(server.URL + "/ait/api/aiTextTranslate")
	defer ResetURLs()

	cfg := DefaultConfig()
	cfg.Mode = ModeLLM
	cfg.LLMAuth = LLMAuthSign
	tr, err := New(cfg, WithAppID("test-appid"), WithAPIKey("test-api-key"), WithSecretKey("test-secret"))
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	tr.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
	}

	_, err = tr.Translate(context.Background(), "hello", "en", "zh")
	if err == nil {
		t.Fatal("expected error from API call")
	}

	if !strings.Contains(err.Error(), "code=54000") {
		t.Fatalf("expected error code 54000, got: %v", err)
	}
}
