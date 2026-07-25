// Package baidu implements the Translator interface using Baidu Translation APIs.
// It supports both General Text Translation (通用文本翻译) and Large Model Text Translation (大模型文本翻译).
package baidu

import (
    "bytes"
    "context"
    "crypto/md5"
    "encoding/json"
    "fmt"
    "io"
    "math/rand"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/zhangshj/library/pkg/translate"
)

const (
    // ModeGeneral uses the legacy General Text Translation API (通用文本翻译).
    ModeGeneral = "general"
    // ModeLLM uses the Large Model Text Translation API (大模型文本翻译).
    ModeLLM = "llm"
    // ModeAuto randomly chooses between General and LLM for each request.
    ModeAuto = "auto"

    // DefaultGeneralBaseURL is the default endpoint for General Text Translation.
    DefaultGeneralBaseURL = "https://fanyi-api.baidu.com/api/trans/vip/translate"
    // DefaultLLMBaseURL is the default endpoint for Large Model Text Translation.
    DefaultLLMBaseURL = "https://fanyi-api.baidu.com/ait/api/aiTextTranslate"

    // LLMAuthBearer uses Bearer Token authentication for LLM mode.
    LLMAuthBearer = "bearer"
    // LLMAuthSign uses MD5 sign authentication for LLM mode.
    LLMAuthSign = "sign"

    // DefaultTimeout is the default HTTP timeout.
    DefaultTimeout = 10 * time.Second
)

var (
    generalBaseURL = DefaultGeneralBaseURL
    llmBaseURL     = DefaultLLMBaseURL
)

// SetGeneralBaseURL sets the base URL for General Text Translation (for testing).
func SetGeneralBaseURL(u string) {
    generalBaseURL = u
}

// SetLLMBaseURL sets the base URL for Large Model Text Translation (for testing).
func SetLLMBaseURL(u string) {
    llmBaseURL = u
}

// ResetURLs resets all URLs to their default values (for testing).
func ResetURLs() {
    generalBaseURL = DefaultGeneralBaseURL
    llmBaseURL = DefaultLLMBaseURL
}

// langCode maps ISO 639-1 codes to Baidu language codes.
var langCode = map[string]string{
    "zh": "zh",
    "en": "en",
    "ja": "jp",
    "ko": "kor",
    "fr": "fra",
    "de": "de",
    "ru": "ru",
    "es": "spa",
    "pt": "pt",
    "it": "it",
    "vi": "vie",
    "th": "th",
    "ar": "ara",
    "hi": "hi",
}

// Config holds Baidu Translation specific configuration.
// Sensitive credentials MUST be injected via Functional Options at runtime.
type Config struct {
    translate.Config
    AppID     string
    APIKey    string
    SecretKey string
    Mode      string
    LLMAuth   string
    TermIDs   string
    Reference string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
    return &Config{
        Config:  *translate.DefaultConfig(),
        Mode:    ModeGeneral,
        LLMAuth: LLMAuthBearer,
    }
}

// Option defines a functional option for Baidu client configuration.
type Option func(*Config)

// WithAppID sets the Baidu APPID (used in request body for both General and LLM modes).
func WithAppID(appID string) Option {
    return func(c *Config) {
        c.AppID = appID
    }
}

// WithAPIKey sets the Baidu API Key (used as Bearer token for LLM mode).
func WithAPIKey(apiKey string) Option {
    return func(c *Config) {
        c.APIKey = apiKey
    }
}

// WithSecretKey sets the Baidu Secret Key for authentication.
func WithSecretKey(secretKey string) Option {
    return func(c *Config) {
        c.SecretKey = secretKey
    }
}

// WithMode sets the translation mode: ModeGeneral or ModeLLM.
func WithMode(mode string) Option {
    return func(c *Config) {
        if mode != "" {
            c.Mode = mode
        }
    }
}

// WithLLMAuth sets the authentication method for LLM mode: LLMAuthBearer or LLMAuthSign.
func WithLLMAuth(auth string) Option {
    return func(c *Config) {
        if auth != "" {
            c.LLMAuth = auth
        }
    }
}

// WithTermIDs sets the term base IDs for LLM translation (optional).
func WithTermIDs(termIDs string) Option {
    return func(c *Config) {
        c.TermIDs = termIDs
    }
}

// WithReference sets the translation instruction for LLM translation (optional).
func WithReference(reference string) Option {
    return func(c *Config) {
        c.Reference = reference
    }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
    return func(c *Config) {
        translate.WithTimeout(timeout)(&c.Config)
    }
}

// WithRegion sets the target cloud region (kept for compatibility, not used by Baidu APIs).
func WithRegion(region string) Option {
    return func(c *Config) {
        translate.WithRegion(region)(&c.Config)
    }
}

// Translator is the Baidu machine translation implementation.
type Translator struct {
    httpClient *http.Client
    appID      string
    apiKey     string
    secretKey  string
    mode       string
    llmAuth    string
    termIDs    string
    reference  string
}

// New creates a new Baidu Translator.
// The APPID, API key, and Secret Key MUST be provided via WithAppID, WithAPIKey, and WithSecretKey; otherwise New returns an error.
func New(cfg *Config, opts ...Option) (*Translator, error) {
    if cfg == nil {
        cfg = DefaultConfig()
    }
    for _, opt := range opts {
        opt(cfg)
    }

    if cfg.AppID == "" || cfg.APIKey == "" || cfg.SecretKey == "" {
        return nil, fmt.Errorf("baidu: app id, api key and secret key are required: %w", ErrInvalidConfig)
    }

    mode := cfg.Mode
    if mode == "" {
        mode = ModeGeneral
    }
    if mode != ModeGeneral && mode != ModeLLM && mode != ModeAuto {
        return nil, fmt.Errorf("baidu: invalid mode %q: %w", mode, ErrInvalidConfig)
    }

    llmAuth := cfg.LLMAuth
    if llmAuth == "" {
        llmAuth = LLMAuthBearer
    }
    if llmAuth != LLMAuthBearer && llmAuth != LLMAuthSign {
        return nil, fmt.Errorf("baidu: invalid llm auth %q: %w", llmAuth, ErrInvalidConfig)
    }

    timeout := cfg.Timeout
    if timeout <= 0 {
        timeout = DefaultTimeout
    }

    return &Translator{
        httpClient: &http.Client{Timeout: timeout},
        appID:      cfg.AppID,
        apiKey:     cfg.APIKey,
        secretKey:  cfg.SecretKey,
        mode:       mode,
        llmAuth:    llmAuth,
        termIDs:    cfg.TermIDs,
        reference:  cfg.Reference,
    }, nil
}

// resolveLangCode converts an ISO 639-1 code to Baidu language code.
// If the code is not in the map, it returns the original code.
func resolveLangCode(code string) string {
    if c, ok := langCode[code]; ok {
        return c
    }
    return code
}

// Translate sends a single translation request to Baidu.
func (t *Translator) Translate(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    if text == "" {
        return "", fmt.Errorf("baidu: source text is empty: %w", ErrInvalidInput)
    }

    if t.mode == ModeGeneral {
        return t.translateGeneral(ctx, text, sourceLang, targetLang)
    }
    if t.mode == ModeLLM {
        return t.translateLLM(ctx, text, sourceLang, targetLang)
    }
    return t.translateAuto(ctx, text, sourceLang, targetLang)
}

func (t *Translator) translateGeneral(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    salt := fmt.Sprintf("%d", time.Now().UnixNano())
    sign := fmt.Sprintf("%x", md5.Sum([]byte(t.appID+text+salt+t.secretKey)))

    params := url.Values{}
    params.Set("q", text)
    params.Set("from", sourceLang)
    params.Set("to", targetLang)
    params.Set("appid", t.appID)
    params.Set("salt", salt)
    params.Set("sign", sign)

    req, err := http.NewRequestWithContext(ctx, "POST", generalBaseURL, strings.NewReader(params.Encode()))
    if err != nil {
        return "", fmt.Errorf("baidu: failed to create general translate request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("baidu: general translate request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("baidu: failed to read general translate response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode string `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return "", fmt.Errorf("baidu: failed to parse general translate response: %w", err)
    }

    if result.ErrorCode != "" {
        return "", fmt.Errorf("baidu: general translate error: code=%s, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) == 0 {
        return "", fmt.Errorf("baidu: general translate returned empty result")
    }

    return result.TransResult[0].Dst, nil
}

func (t *Translator) translateLLM(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    if t.llmAuth == LLMAuthSign {
        return t.translateLLMSign(ctx, text, sourceLang, targetLang)
    }
    return t.translateLLMBearer(ctx, text, sourceLang, targetLang)
}

func (t *Translator) translateAuto(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    if rand.Intn(2) == 0 {
        return t.translateGeneral(ctx, text, sourceLang, targetLang)
    }
    return t.translateLLM(ctx, text, sourceLang, targetLang)
}

func (t *Translator) translateLLMBearer(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    bodyMap := map[string]interface{}{
        "appid": t.appID,
        "from":  sourceLang,
        "to":    targetLang,
        "q":     text,
    }
    if t.termIDs != "" {
        bodyMap["termIds"] = t.termIDs
    }
    if t.reference != "" {
        bodyMap["reference"] = t.reference
    }

    jsonBody, err := json.Marshal(bodyMap)
    if err != nil {
        return "", fmt.Errorf("baidu: failed to marshal llm translate request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", llmBaseURL, bytes.NewReader(jsonBody))
    if err != nil {
        return "", fmt.Errorf("baidu: failed to create llm translate request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json;charset=utf-8")
    req.Header.Set("Authorization", "Bearer "+t.apiKey)

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("baidu: llm translate request failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("baidu: failed to read llm translate response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode int    `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(respBody, &result); err != nil {
        return "", fmt.Errorf("baidu: failed to parse llm translate response: %w", err)
    }

    if result.ErrorCode != 0 {
        return "", fmt.Errorf("baidu: llm translate error: code=%d, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) == 0 {
        return "", fmt.Errorf("baidu: llm translate returned empty result")
    }

    return result.TransResult[0].Dst, nil
}

func (t *Translator) translateLLMSign(ctx context.Context, text string, sourceLang, targetLang string) (string, error) {
    salt := fmt.Sprintf("%d", time.Now().UnixNano())
    sign := fmt.Sprintf("%x", md5.Sum([]byte(t.appID+text+salt+t.secretKey)))

    params := url.Values{}
    params.Set("q", text)
    params.Set("from", sourceLang)
    params.Set("to", targetLang)
    params.Set("appid", t.appID)
    params.Set("salt", salt)
    params.Set("sign", sign)
    if t.termIDs != "" {
        params.Set("termIds", t.termIDs)
    }
    if t.reference != "" {
        params.Set("reference", t.reference)
    }

    reqURL := fmt.Sprintf("%s?%s", llmBaseURL, params.Encode())
    req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(text))
    if err != nil {
        return "", fmt.Errorf("baidu: failed to create llm sign translate request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("baidu: llm sign translate request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("baidu: failed to read llm sign translate response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode int    `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return "", fmt.Errorf("baidu: failed to parse llm sign translate response: %w", err)
    }

    if result.ErrorCode != 0 {
        return "", fmt.Errorf("baidu: llm sign translate error: code=%d, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) == 0 {
        return "", fmt.Errorf("baidu: llm sign translate returned empty result")
    }

    return result.TransResult[0].Dst, nil
}

// TranslateBatch sends a batch translation request to Baidu.
// For General mode, all texts are sent in a single API call joined by newlines.
// For LLM mode, all texts are sent in a single API call joined by newlines.
// For Auto mode, all texts are sent in a single API call joined by newlines, randomly choosing between General and LLM.
// Results preserve input order.
func (t *Translator) TranslateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    if len(texts) == 0 {
        return nil, fmt.Errorf("baidu: source texts are empty: %w", ErrInvalidInput)
    }

    for i, text := range texts {
        if text == "" {
            return nil, fmt.Errorf("baidu: source text %d is empty: %w", i, ErrInvalidInput)
        }
    }

    if t.mode == ModeGeneral {
        return t.translateGeneralBatch(ctx, texts, sourceLang, targetLang)
    }
    if t.mode == ModeLLM {
        return t.translateLLMBatch(ctx, texts, sourceLang, targetLang)
    }
    return t.translateAutoBatch(ctx, texts, sourceLang, targetLang)
}

func (t *Translator) translateLLMBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    q := strings.Join(texts, "\n")
    if t.llmAuth == LLMAuthSign {
        return t.translateLLMSignBatch(ctx, q, sourceLang, targetLang)
    }
    return t.translateLLMBearerBatch(ctx, q, sourceLang, targetLang)
}

func (t *Translator) translateLLMBearerBatch(ctx context.Context, q, sourceLang, targetLang string) ([]string, error) {
    bodyMap := map[string]interface{}{
        "appid": t.appID,
        "from":  sourceLang,
        "to":    targetLang,
        "q":     q,
    }
    if t.termIDs != "" {
        bodyMap["termIds"] = t.termIDs
    }
    if t.reference != "" {
        bodyMap["reference"] = t.reference
    }

    jsonBody, err := json.Marshal(bodyMap)
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to marshal llm translate request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", llmBaseURL, bytes.NewReader(jsonBody))
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to create llm translate request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json;charset=utf-8")
    req.Header.Set("Authorization", "Bearer "+t.apiKey)

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("baidu: llm translate request failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to read llm translate response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode int    `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("baidu: failed to parse llm translate response: %w", err)
    }

    if result.ErrorCode != 0 {
        return nil, fmt.Errorf("baidu: llm translate error: code=%d, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) == 0 {
        return nil, fmt.Errorf("baidu: llm translate returned empty result")
    }

    dsts := make([]string, len(result.TransResult))
    for i, item := range result.TransResult {
        dsts[i] = item.Dst
    }
    return dsts, nil
}

func (t *Translator) translateLLMSignBatch(ctx context.Context, q, sourceLang, targetLang string) ([]string, error) {
    salt := fmt.Sprintf("%d", time.Now().UnixNano())
    sign := fmt.Sprintf("%x", md5.Sum([]byte(t.appID+q+salt+t.secretKey)))

    params := url.Values{}
    params.Set("q", q)
    params.Set("from", sourceLang)
    params.Set("to", targetLang)
    params.Set("appid", t.appID)
    params.Set("salt", salt)
    params.Set("sign", sign)
    if t.termIDs != "" {
        params.Set("termIds", t.termIDs)
    }
    if t.reference != "" {
        params.Set("reference", t.reference)
    }

    reqURL := fmt.Sprintf("%s?%s", llmBaseURL, params.Encode())
    req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(q))
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to create llm sign translate request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("baidu: llm sign translate request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to read llm sign translate response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode int    `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("baidu: failed to parse llm sign translate response: %w", err)
    }

    if result.ErrorCode != 0 {
        return nil, fmt.Errorf("baidu: llm sign translate error: code=%d, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) == 0 {
        return nil, fmt.Errorf("baidu: llm sign translate returned empty result")
    }

    dsts := make([]string, len(result.TransResult))
    for i, item := range result.TransResult {
        dsts[i] = item.Dst
    }
    return dsts, nil
}

func (t *Translator) translateAutoBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    if rand.Intn(2) == 0 {
        return t.translateGeneralBatch(ctx, texts, sourceLang, targetLang)
    }
    return t.translateLLMBatch(ctx, texts, sourceLang, targetLang)
}

func (t *Translator) translateGeneralBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
    q := strings.Join(texts, "\n")
    salt := fmt.Sprintf("%d", time.Now().UnixNano())
    sign := fmt.Sprintf("%x", md5.Sum([]byte(t.appID+q+salt+t.secretKey)))

    params := url.Values{}
    params.Set("q", q)
    params.Set("from", sourceLang)
    params.Set("to", targetLang)
    params.Set("appid", t.appID)
    params.Set("salt", salt)
    params.Set("sign", sign)

    req, err := http.NewRequestWithContext(ctx, "POST", generalBaseURL, strings.NewReader(params.Encode()))
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to create general batch request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("baidu: general batch translate request failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("baidu: failed to read general batch response: %w", err)
    }

    var result struct {
        From        string `json:"from"`
        To          string `json:"to"`
        TransResult []struct {
            Src string `json:"src"`
            Dst string `json:"dst"`
        }   `json:"trans_result"`
        ErrorCode string `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("baidu: failed to parse general batch response: %w", err)
    }

    if result.ErrorCode != "" {
        return nil, fmt.Errorf("baidu: general batch translate error: code=%s, msg=%s", result.ErrorCode, result.ErrorMsg)
    }

    if len(result.TransResult) != len(texts) {
        return nil, fmt.Errorf("baidu: general batch translate returned %d results, expected %d", len(result.TransResult), len(texts))
    }

    results := make([]string, len(texts))
    for i, item := range result.TransResult {
        results[i] = item.Dst
    }

    return results, nil
}

// ErrInvalidConfig indicates the client configuration is invalid.
var ErrInvalidConfig = fmt.Errorf("invalid config")

// ErrInvalidInput indicates the input parameters are invalid.
var ErrInvalidInput = fmt.Errorf("invalid input")
