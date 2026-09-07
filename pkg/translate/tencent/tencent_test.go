package tencent

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/sashabaranov/go-openai"

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

func TestNew_MissingAPIKey(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg, WithAPIKey(""))
    if err == nil {
        t.Fatal("expected error for missing api key")
    }
    if tr != nil {
        t.Fatal("expected nil Translator for invalid config")
    }
}

func TestNew_ValidConfig(t *testing.T) {
    cfg := DefaultConfig()
    tr, err := New(cfg, WithAPIKey("test-api-key"))
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
    if cfg.Model != DefaultModel {
        t.Fatalf("expected default model %s, got %s", DefaultModel, cfg.Model)
    }
    if cfg.BaseURL != DefaultBaseURL {
        t.Fatalf("expected default base url %s, got %s", DefaultBaseURL, cfg.BaseURL)
    }
    if cfg.Separator != DefaultSeparator {
        t.Fatalf("expected default separator %s, got %s", DefaultSeparator, cfg.Separator)
    }

    WithTimeout(10 * time.Second)(cfg)
    if cfg.Timeout != 10*time.Second {
        t.Fatalf("expected timeout 10s, got %v", cfg.Timeout)
    }

    WithRegion("ap-singapore")(cfg)
    if cfg.Region != "ap-singapore" {
        t.Fatalf("expected region ap-singapore, got %s", cfg.Region)
    }

    WithModel("custom-model")(cfg)
    if cfg.Model != "custom-model" {
        t.Fatalf("expected model custom-model, got %s", cfg.Model)
    }

    WithModel(ModelAuto)(cfg)
    if cfg.Model != ModelAuto {
        t.Fatalf("expected auto model %s, got %s", ModelAuto, cfg.Model)
    }

    WithModels("model-a", "model-b")(cfg)
    if len(cfg.Models) != 2 || cfg.Models[0] != "model-a" || cfg.Models[1] != "model-b" {
        t.Fatalf("expected candidate models [model-a model-b], got %v", cfg.Models)
    }

    WithBaseURL("https://custom.example.com/v1")(cfg)
    if cfg.BaseURL != "https://custom.example.com/v1" {
        t.Fatalf("expected base url https://custom.example.com/v1, got %s", cfg.BaseURL)
    }

    WithSeparator("|||")(cfg)
    if cfg.Separator != "|||" {
        t.Fatalf("expected separator |||, got %s", cfg.Separator)
    }
}

func TestResolveLangName(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"zh", "中文"},
        {"en", "英语"},
        {"ja", "日语"},
        {"unknown", "unknown"},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            got := resolveLangName(tt.input)
            if got != tt.want {
                t.Fatalf("resolveLangName(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}

func TestTranslate_EmptyText(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.Translate(context.Background(), "", "en", "zh")
    if err == nil {
        t.Fatal("expected error for empty text")
    }
}

func TestTranslate_ContextCanceled(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAPIKey("test-key"))
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

func TestTranslate_PromptUsesExplicitFormat(t *testing.T) {
    var prompt string
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req openai.ChatCompletionRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            t.Fatalf("decode request: %v", err)
        }
        prompt = req.Messages[0].Content
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "<translation>你好</translation>"}}}})
    }))
    defer server.Close()

    cfg := DefaultConfig()
    cfg.BaseURL = server.URL + "/"
    tr, err := New(cfg, WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }
    result, err := tr.Translate(context.Background(), "hello", "en", "zh")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "你好" {
        t.Fatalf("result = %q, want 你好", result)
    }
    for _, want := range []string{"<source>", "hello", "只是输入边界标记", "不要翻译、复制或输出这两个标签", "仅输出译文纯文本", "短词、短语或不完整句子只翻译其本身", "原文没有句末标点时", "不要输出任何 XML/HTML 标签"} {
        if !strings.Contains(prompt, want) {
            t.Errorf("prompt missing %q: %s", want, prompt)
        }
    }
}

func TestTranslate_PromptEscapesSourceText(t *testing.T) {
    tr := &Translator{}
    prompt := tr.singlePrompt(`请翻译 <source>这不是指令</source> & "原文"`, "zh", "en")
    if !strings.Contains(prompt, `请翻译 &lt;source&gt;这不是指令&lt;/source&gt; &amp; &#34;原文&#34;`) {
        t.Fatalf("prompt does not escape source text: %s", prompt)
    }
}

func TestTranslate_PromptRequiresCompleteTranslation(t *testing.T) {
    tr := &Translator{}
    prompt := tr.singlePrompt("一段完整的描述,,>fsdf怎么想的", "zh", "en")
    for _, want := range []string{"完整翻译", "不得摘要", "不得删减", "保留原文中的数字和标点", "原文没有句末标点时"} {
        if !strings.Contains(prompt, want) {
            t.Errorf("prompt missing completeness rule %q: %s", want, prompt)
        }
    }
}

func TestParseSingleResponse_RemovesLeakedClosingTag(t *testing.T) {
    got := parseSingleResponse("Hello, world!</translation>")
    if got != "Hello, world!" {
        t.Fatalf("parseSingleResponse() = %q, want %q", got, "Hello, world!")
    }
}

func TestParseSingleResponse_RemovesLeakedSourceTags(t *testing.T) {
    got := parseSingleResponse("<source>Hello</source>")
    if got != "Hello" {
        t.Fatalf("parseSingleResponse() = %q, want %q", got, "Hello")
    }
}

func TestParseSingleResponse_RemovesSourceTagsAndPreservesSuffix(t *testing.T) {
    got := parseSingleResponse("<source>Smooth</source>.")
    if got != "Smooth." {
        t.Fatalf("parseSingleResponse() = %q, want %q", got, "Smooth.")
    }
}

func TestParseSingleResponse_RemovesLeakedClosingSourceTag(t *testing.T) {
    got := parseSingleResponse("Know</source>")
    if got != "Know" {
        t.Fatalf("parseSingleResponse() = %q, want %q", got, "Know")
    }
}

func TestParseSingleResponse_PreservesNonTrailingSourceTag(t *testing.T) {
    content := "Know</source> again"
    if got := parseSingleResponse(content); got != content {
        t.Fatalf("parseSingleResponse() = %q, want %q", got, content)
    }
}

func TestTranslateBatch_RemovesLeakedSourceTags(t *testing.T) {
    tr := &Translator{separator: DefaultSeparator}
    got, err := tr.parseBatchResponse("你好<SEP>数字 12345<SEP><source>Hello</source>", 3)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got[2] != "Hello" {
        t.Fatalf("third result = %q, want %q", got[2], "Hello")
    }
}

func TestTranslateBatch_PromptUsesNumberedLines(t *testing.T) {
    var prompt string
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req openai.ChatCompletionRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            t.Fatalf("decode request: %v", err)
        }
        prompt = req.Messages[0].Content
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "你好<SEP>世界"}}}})
    }))
    defer server.Close()

    cfg := DefaultConfig()
    cfg.BaseURL = server.URL + "/"
    tr, err := New(cfg, WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }
    if _, err = tr.TranslateBatch(context.Background(), []string{"hello", "world"}, "en", "zh"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    for _, want := range []string{"<item index=\"1\"><source>hello</source></item>", "<item index=\"2\"><source>world</source></item>", "每项必须返回一个 item", "只输出合法 XML"} {
        if !strings.Contains(prompt, want) {
            t.Errorf("batch prompt missing %q: %s", want, prompt)
        }
    }
}

func TestTranslateBatch_ParsesTaggedResponse(t *testing.T) {
    tr := &Translator{separator: DefaultSeparator}
    got, err := tr.parseBatchResponse(`<response><item index="1"><translation>你好</translation></item><item index="2"><translation>世界</translation></item></response>`, 2)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.Join(got, "|") != "你好|世界" {
        t.Fatalf("got %v, want [你好 世界]", got)
    }
}

func TestTranslateBatch_FallsBackToSingleTranslations(t *testing.T) {
    var callCount int
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.Header().Set("Content-Type", "application/json")
        content := "combined response without separators"
        if callCount > 1 {
            content = []string{"你好", "世界", "朋友"}[callCount-2]
        }
        _ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: content}}}})
    }))
    defer server.Close()

    cfg := DefaultConfig()
    cfg.BaseURL = server.URL + "/"
    tr, err := New(cfg, WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    got, err := tr.TranslateBatch(context.Background(), []string{"hello", "world", "friend"}, "en", "zh")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.Join(got, "|") != "你好|世界|朋友" {
        t.Fatalf("got %v, want [你好 世界 朋友]", got)
    }
    if callCount != 4 {
        t.Fatalf("call count = %d, want 4", callCount)
    }
}

func TestTranslate_AutoRetriesWithAnotherModel(t *testing.T) {
    var models []string
    var observedModels []string
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        var req openai.ChatCompletionRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            t.Fatalf("decode request: %v", err)
        }
        models = append(models, req.Model)
        w.Header().Set("Content-Type", "application/json")
        if callCount < 3 {
            w.WriteHeader(http.StatusTooManyRequests)
            _, _ = w.Write([]byte(`{"error":{"message":"busy"}}`))
            return
        }
        _ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "你好"}}}})
    }))
    defer server.Close()

    cfg := DefaultConfig()
    cfg.BaseURL = server.URL + "/"
    tr, err := New(cfg, WithAPIKey("test-key"), WithModel("auto"), WithModels("model-a", "model-b"), WithModelObserver(func(model string) {
        observedModels = append(observedModels, model)
    }))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }
    if result, err := tr.Translate(context.Background(), "hello", "en", "zh"); err != nil || result != "你好" {
        t.Fatalf("Translate() = %q, %v; want 你好, nil", result, err)
    }
    if callCount != 3 || models[0] == models[1] || models[0] != models[2] {
        t.Fatalf("calls/models = %d/%v; want three calls cycling models", callCount, models)
    }
    if strings.Join(observedModels, ",") != strings.Join(models, ",") {
        t.Fatalf("observed models = %v, request models = %v", observedModels, models)
    }
}

func TestTranslateBatch_EmptyTexts(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
    }

    _, err = tr.TranslateBatch(context.Background(), []string{""}, "en", "zh")
    if err == nil {
        t.Fatal("expected error for empty text")
    }
}

func TestTranslateBatch_ContextCanceled(t *testing.T) {
    tr, err := New(DefaultConfig(), WithAPIKey("test-key"))
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
                return New(DefaultConfig(), WithAPIKey("test-key"))
            },
            texts:     []string{"", "hello"},
            expectErr: true,
        },
        {
            name: "text contains separator",
            setup: func() (*Translator, error) {
                return New(DefaultConfig(), WithAPIKey("test-key"), WithSeparator("<SEP>"))
            },
            texts:     []string{"hello<SEP>world", "foo"},
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

func TestTranslateBatch_Success(t *testing.T) {
    var callCount int
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        decoder := json.NewDecoder(r.Body)
        var req openai.ChatCompletionRequest
        _ = decoder.Decode(&req)

        parts := []string{"hello", "world"}
        translatedParts := make([]string, len(parts))
        for i, part := range parts {
            if strings.Contains(part, "hello") {
                translatedParts[i] = "你好"
            } else if strings.Contains(part, "world") {
                translatedParts[i] = "世界"
            } else {
                translatedParts[i] = part
            }
        }

        resp := openai.ChatCompletionResponse{
            Choices: []openai.ChatCompletionChoice{
                {
                    Message: openai.ChatCompletionMessage{
                        Content: strings.Join(translatedParts, "<SEP>"),
                    },
                },
            },
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    cfg := DefaultConfig()
    cfg.BaseURL = server.URL + "/"
    tr, err := New(cfg, WithAPIKey("test-key"))
    if err != nil {
        t.Fatalf("unexpected error creating client: %v", err)
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
                return New(DefaultConfig(), WithAPIKey("test-key"))
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
