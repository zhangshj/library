// Package aliyun implements the Translator interface using Alibaba Cloud Machine Translation (alimt).
package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alimt"

	"github.com/zhangshj/library/pkg/translate"
)

// Config holds Alibaba Cloud specific configuration.
type Config struct {
    translate.Config
    AccessKeyID     string
    AccessKeySecret string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
    return &Config{
        Config: *translate.DefaultConfig(),
    }
}

// Option defines a functional option for Alibaba Cloud client configuration.
type Option func(*Config)

// WithAccessKey sets the Alibaba Cloud AccessKey ID and Secret for authentication.
func WithAccessKey(accessKeyID, accessKeySecret string) Option {
    return func(c *Config) {
        c.AccessKeyID = accessKeyID
        c.AccessKeySecret = accessKeySecret
    }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
    return func(c *Config) {
        translate.WithTimeout(timeout)(&c.Config)
    }
}

// WithRegion sets the target cloud region.
func WithRegion(region string) Option {
    return func(c *Config) {
        translate.WithRegion(region)(&c.Config)
    }
}

// Translator is the Alibaba Cloud machine translation implementation.
type Translator struct {
    client *alimt.Client
    region string
}

// New creates a new Alibaba Cloud Translator.
// The AccessKey pair MUST be provided via WithAccessKey; otherwise New returns an error.
func New(cfg *Config, opts ...Option) (*Translator, error) {
    if cfg == nil {
        cfg = DefaultConfig()
    }
    for _, opt := range opts {
        opt(cfg)
    }

    if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
        return nil, fmt.Errorf("aliyun: access key pair is required: %w", ErrInvalidConfig)
    }

    client, err := alimt.NewClientWithAccessKey(cfg.Region, cfg.AccessKeyID, cfg.AccessKeySecret)
    if err != nil {
        return nil, fmt.Errorf("aliyun: failed to create alimt client: %w", err)
    }

    return &Translator{client: client, region: cfg.Region}, nil
}

// Translate sends a translation request to Alibaba Cloud using the TranslateGeneral API.
func (t *Translator) Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    if text == "" {
        return "", fmt.Errorf("aliyun: source text is empty: %w", ErrInvalidInput)
    }

    request := alimt.CreateTranslateGeneralRequest()
    request.Method = "POST"
    request.FormatType = "text"
    request.SourceLanguage = sourceLang
    request.TargetLanguage = targetLang
    request.SourceText = text
    request.Scene = "general"

    response, err := t.client.TranslateGeneral(request)
    if err != nil {
        return "", fmt.Errorf("aliyun: translate request failed: %w", err)
    }

    if !response.IsSuccess() {
        return "", fmt.Errorf("aliyun: translate request returned non-success: code=%d, message=%s", response.Code, response.Message)
    }

    return response.Data.Translated, nil
}

// TranslateBatch sends a batch translation request to Alibaba Cloud using the GetBatchTranslate API.
// All texts are translated in a single API call, avoiding per-request rate limits.
// Results preserve input order.
func (t *Translator) TranslateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    if len(texts) == 0 {
        return nil, fmt.Errorf("aliyun: source texts are empty: %w", ErrInvalidInput)
    }

	sourceTexts := make(map[string]string, len(texts))
	for i, text := range texts {
		if text == "" {
			return nil, fmt.Errorf("aliyun: source text is empty: %w", ErrInvalidInput)
		}
		sourceTexts[fmt.Sprintf("%d", i+1)] = text
	}

	sourceTextJSON, err := json.Marshal(sourceTexts)
	if err != nil {
		return nil, fmt.Errorf("aliyun: failed to marshal source texts: %w", err)
	}

    request := alimt.CreateGetBatchTranslateRequest()
    request.Method = "POST"
    request.FormatType = "text"
    request.SourceLanguage = sourceLang
    request.TargetLanguage = targetLang
    request.SourceText = string(sourceTextJSON)
    request.Scene = "general"
    request.ApiType = "translate_standard"
    response, err := t.client.GetBatchTranslate(request)
    if err != nil {
        return nil, fmt.Errorf("aliyun: batch translate request failed: %w", err)
    }

    if !response.IsSuccess() {
        return nil, fmt.Errorf("aliyun: batch translate request returned non-success: code=%d, message=%s", response.Code, response.Message)
    }

	results := make([]string, len(texts))
	for _, item := range response.TranslatedList {
		id, ok := item["Id"].(string)
		if !ok {
			return nil, fmt.Errorf("aliyun: batch translate item missing Id field")
		}
		translated, ok := item["Translated"].(string)
		if !ok {
			return nil, fmt.Errorf("aliyun: batch translate item %s: missing Translated field", id)
		}
		idx, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("aliyun: batch translate item id %s: %w", id, err)
		}
		if idx < 1 || idx > len(texts) {
			return nil, fmt.Errorf("aliyun: batch translate item id %s out of range", id)
		}
		results[idx-1] = translated
	}

    return results, nil
}

// ErrInvalidConfig indicates the client configuration is invalid.
var ErrInvalidConfig = fmt.Errorf("invalid config")

// ErrInvalidInput indicates the input parameters are invalid.
var ErrInvalidInput = fmt.Errorf("invalid input")
