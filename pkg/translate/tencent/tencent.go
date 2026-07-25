package tencent

import (
	"context"
	"fmt"
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
	APIKey     string
	Model      string
	BaseURL    string
	Separator  string
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
	client    *openai.Client
	model     string
	separator string
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

	return &Translator{client: client, model: model, separator: separator}, nil
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

	targetLangName := resolveLangName(targetLang)
	prompt := fmt.Sprintf("将以下文本翻译为 %s，注意只需要输出翻译后的结果，不要额外解释：%s", targetLangName, text)

	resp, err := t.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: t.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("tencent: translate request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("tencent: translate request returned empty response")
	}

	return resp.Choices[0].Message.Content, nil
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

	targetLangName := resolveLangName(targetLang)
	sourceText := strings.Join(texts, t.separator)
	prompt := fmt.Sprintf("请将以下文本准确翻译为 %s。你必须在译文中保留等量的分隔符，绝对不可遗漏、转义或翻译该符号，并注意分隔符的位置。%s", targetLangName, sourceText)

	resp, err := t.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: t.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("tencent: batch translate request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("tencent: batch translate request returned empty response")
	}

	translated := resp.Choices[0].Message.Content
	parts := strings.Split(translated, t.separator)

	if len(parts) != len(texts) {
		return nil, fmt.Errorf("tencent: batch translate returned %d parts, expected %d", len(parts), len(texts))
	}

	return parts, nil
}

// ErrInvalidConfig indicates the client configuration is invalid.
var ErrInvalidConfig = fmt.Errorf("invalid config")

// ErrInvalidInput indicates the input parameters are invalid.
var ErrInvalidInput = fmt.Errorf("invalid input")
