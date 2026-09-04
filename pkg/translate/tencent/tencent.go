package tencent

import (
    "context"
    "encoding/xml"
    "errors"
    "fmt"
    "math/rand"
    "net/http"
    "strings"
    "time"

    "github.com/sashabaranov/go-openai"

    "github.com/zhangshj/library/pkg/translate"
)

const (
    DefaultBatchSize = 5
    DefaultModel     = "hy-mt2-plus"
    DefaultBaseURL   = "https://tokenhub.tencentmaas.com/v1"
    DefaultSeparator = "<SEP>"
    ModelAuto        = "auto"
)

// langName maps ISO 639-1 codes to Chinese language names used by TokenHub.
var langName = map[string]string{
    "zh": "中文",
    "en": "英语",
    "ja": "日语",
    "ko": "韩语",
    "fr": "法语",
    "de": "德语",
    "es": "西班牙语",
    "ru": "俄语",
    "pt": "葡萄牙语",
    "it": "意大利语",
    "vi": "越南语",
    "th": "泰语",
    "ar": "阿拉伯语",
    "hi": "印地语",
    "id": "印尼语",
    "ms": "马来语",
    "tr": "土耳其语",
    "pl": "波兰语",
    "nl": "荷兰语",
    "sv": "瑞典语",
    "da": "丹麦语",
    "fi": "芬兰语",
    "el": "希腊语",
    "cs": "捷克语",
    "ro": "罗马尼亚语",
    "hu": "匈牙利语",
    "uk": "乌克兰语",
    "bg": "保加利亚语",
    "ca": "加泰罗尼亚语",
    "no": "挪威语",
    "he": "希伯来语",
    "fa": "波斯语",
}

// Config holds TokenHub specific configuration.
type Config struct {
    translate.Config
    APIKey        string
    Model         string
    Models        []string
    BaseURL       string
    Separator     string
    Domain        string
    ModelObserver func(string)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
    return &Config{
        Config:    *translate.DefaultConfig(),
        Model:     DefaultModel,
        BaseURL:   DefaultBaseURL,
        Separator: DefaultSeparator,
    }
}

// Option defines a functional option for TokenHub client configuration.
type Option func(*Config)

// WithAPIKey sets the TokenHub API key for authentication.
func WithAPIKey(apiKey string) Option {
    return func(c *Config) {
        c.APIKey = apiKey
    }
}

// WithModel sets the translation model name.
func WithModel(model string) Option {
    return func(c *Config) {
        if model != "" {
            c.Model = model
        }
    }
}

// WithModels sets the candidate models used when ModelAuto is selected.
func WithModels(models ...string) Option {
    return func(c *Config) {
        c.Models = append([]string(nil), models...)
    }
}

// WithModelObserver registers a callback invoked with the model used for each request.
func WithModelObserver(observer func(string)) Option {
    return func(c *Config) {
        c.ModelObserver = observer
    }
}

// WithBaseURL sets the TokenHub API base URL.
func WithBaseURL(baseURL string) Option {
    return func(c *Config) {
        if baseURL != "" {
            c.BaseURL = baseURL
        }
    }
}

// WithSeparator sets the separator for batch translation.
func WithSeparator(separator string) Option {
    return func(c *Config) {
        if separator != "" {
            c.Separator = separator
        }
    }
}

// WithDomain sets the professional domain for translation (e.g. "智能驾驶", "机器人", "生物医疗").
// When empty, no domain-specific instruction is added to the prompt.
func WithDomain(domain string) Option {
    return func(c *Config) {
        c.Domain = domain
    }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
    return func(c *Config) {
        translate.WithTimeout(timeout)(&c.Config)
    }
}

// WithRegion sets the target cloud region (kept for compatibility, not used by TokenHub).
func WithRegion(region string) Option {
    return func(c *Config) {
        translate.WithRegion(region)(&c.Config)
    }
}

// Translator is the TokenHub machine translation implementation.
type Translator struct {
    client        *openai.Client
    model         string
    models        []string
    separator     string
    domain        string
    modelObserver func(string)
}

// New creates a new TokenHub Translator.
// The API key MUST be provided via WithAPIKey or in Config; otherwise New returns an error.
func New(cfg *Config, opts ...Option) (*Translator, error) {
    if cfg == nil {
        cfg = DefaultConfig()
    }
    for _, opt := range opts {
        opt(cfg)
    }

    if cfg.APIKey == "" {
        return nil, fmt.Errorf("tencent: api key is required: %w", ErrInvalidConfig)
    }

    clientConfig := openai.DefaultConfig(cfg.APIKey)
    clientConfig.BaseURL = cfg.BaseURL
    client := openai.NewClientWithConfig(clientConfig)

    model := cfg.Model
    if model == "" {
        model = DefaultModel
    }

    separator := cfg.Separator
    if separator == "" {
        separator = DefaultSeparator
    }

    models := append([]string(nil), cfg.Models...)
    if len(models) == 0 {
        models = []string{DefaultModel}
    }

    return &Translator{client: client, model: model, models: models, separator: separator, domain: cfg.Domain, modelObserver: cfg.ModelObserver}, nil
}

// resolveLangName converts an ISO 639-1 code to the Chinese language name expected by TokenHub.
// If the code is not in the map, it returns the original code.
func resolveLangName(code string) string {
    if name, ok := langName[code]; ok {
        return name
    }
    return code
}

// Translate sends a single translation request to TokenHub using the basic translation prompt.
func (t *Translator) Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    if text == "" {
        return "", fmt.Errorf("tencent: source text is empty: %w", ErrInvalidInput)
    }

    prompt := t.singlePrompt(text, sourceLang, targetLang)
    resp, err := t.createChatCompletion(ctx, prompt)
    if err != nil {
        return "", fmt.Errorf("tencent: translate request failed: %w", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("tencent: translate request returned empty response")
    }

    return parseSingleResponse(resp.Choices[0].Message.Content), nil
}

// TranslateBatch sends a batch translation request to TokenHub using the separator-based translation prompt.
// It joins texts with the configured separator, sends a single API call, and splits the result.
func (t *Translator) TranslateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    if len(texts) == 0 {
        return nil, fmt.Errorf("tencent: source texts are empty: %w", ErrInvalidInput)
    }

    for i, text := range texts {
        if text == "" {
            return nil, fmt.Errorf("tencent: source text %d is empty: %w", i, ErrInvalidInput)
        }
        if strings.Contains(text, t.separator) {
            return nil, fmt.Errorf("tencent: source text %d contains separator %q", i, t.separator)
        }
    }

    prompt := t.batchPrompt(texts, sourceLang, targetLang)
    resp, err := t.createChatCompletion(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("tencent: batch translate request failed: %w", err)
    }

    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("tencent: batch translate request returned empty response")
    }

    parts, err := t.parseBatchResponse(resp.Choices[0].Message.Content, len(texts))
    if err != nil {
        return nil, err
    }

    return parts, nil
}

func (t *Translator) singlePrompt(text, sourceLang, targetLang string) string {
    domainHint := ""
    if t.domain != "" {
        domainHint = fmt.Sprintf("使用%s领域的专业术语。", t.domain)
    }
    return fmt.Sprintf("你是翻译引擎。目标语言：%s。源语言：%s。%s仅输出<translation>标签中的内容，不要解释、改写或输出标签之外的内容。待翻译文本严格位于<source>标签内：<source>%s</source>。输出格式必须是<translation>译文</translation>。", resolveLangName(targetLang), sourceLang, domainHint, text)
}

type singleResponse struct {
    Translation string `xml:",chardata"`
}

func parseSingleResponse(content string) string {
    trimmed := strings.TrimSpace(content)
    if strings.HasPrefix(trimmed, "<translation>") {
        var response singleResponse
        if err := xml.Unmarshal([]byte(trimmed), &response); err == nil {
            return response.Translation
        }
    }
    return content
}

func (t *Translator) batchPrompt(texts []string, sourceLang, targetLang string) string {
    lines := make([]string, len(texts))
    for i, text := range texts {
        lines[i] = fmt.Sprintf("<item index=\"%d\"><source>%s</source></item>", i+1, text)
    }
    domainHint := ""
    if t.domain != "" {
        domainHint = fmt.Sprintf("使用%s领域的专业术语。", t.domain)
    }
    return fmt.Sprintf("你是翻译引擎。目标语言：%s。源语言：%s。%s请逐项翻译以下输入。每个 item 是一条独立数据，必须按 index 原顺序返回，不能合并、遗漏或改写 index。只输出合法 XML，不要解释：输入格式为 <item index=\"N\"><source>原文</source></item>；输出格式必须为 <response><item index=\"N\"><translation>译文</translation></item>...</response>。每项必须返回一个 item，译文不能包含 XML 标签。输入：\n%s", resolveLangName(targetLang), sourceLang, domainHint, strings.Join(lines, "\n"))
}

type batchResponse struct {
    Items []batchResponseItem `xml:"item"`
}

type batchResponseItem struct {
    Index       int    `xml:"index,attr"`
    Translation string `xml:"translation"`
}

func (t *Translator) parseBatchResponse(content string, expected int) ([]string, error) {
    var response batchResponse
    xmlContent := strings.TrimSpace(content)
    if strings.HasPrefix(xmlContent, "<response>") {
        if err := xml.Unmarshal([]byte(xmlContent), &response); err == nil && len(response.Items) == expected {
            parts := make([]string, expected)
            valid := true
            for i, item := range response.Items {
                if item.Index != i+1 {
                    valid = false
                    break
                }
                parts[i] = item.Translation
            }
            if valid {
                return parts, nil
            }
        }
    }

    parts := strings.Split(content, t.separator)
    if len(parts) != expected {
        return nil, fmt.Errorf("tencent: batch translate returned %d parts, expected %d", len(parts), expected)
    }
    return parts, nil
}

func (t *Translator) createChatCompletion(ctx context.Context, prompt string) (openai.ChatCompletionResponse, error) {
    modelIndex := 0
    if t.model == ModelAuto && len(t.models) > 1 {
        modelIndex = rand.Intn(len(t.models))
    }
    model := t.requestModel(modelIndex)
    for attempt := 0; attempt < 2; attempt++ {
        if t.modelObserver != nil {
            t.modelObserver(model)
        }
        resp, err := t.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
            Model: model,
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: prompt},
            },
        })
        if err == nil {
            return resp, nil
        }
        if attempt == 1 || !hasHTTPErrorStatus(err) {
            return openai.ChatCompletionResponse{}, err
        }
        if t.model == ModelAuto && len(t.models) > 1 {
            model = t.requestModel((modelIndex + 1) % len(t.models))
        }
    }
    return openai.ChatCompletionResponse{}, fmt.Errorf("tencent: request retry exhausted")
}

func (t *Translator) requestModel(index int) string {
    if t.model != ModelAuto {
        return t.model
    }
    return t.models[index%len(t.models)]
}

func hasHTTPErrorStatus(err error) bool {
    var apiErr *openai.APIError
    if errors.As(err, &apiErr) {
        return apiErr.HTTPStatusCode != http.StatusOK
    }
    var requestErr *openai.RequestError
    if errors.As(err, &requestErr) {
        return requestErr.HTTPStatusCode != http.StatusOK
    }
    return false
}

// ErrInvalidConfig indicates the client configuration is invalid.
var ErrInvalidConfig = fmt.Errorf("invalid config")

// ErrInvalidInput indicates the input parameters are invalid.
var ErrInvalidInput = fmt.Errorf("invalid input")
