# Translate Component Configuration

All sensitive credentials MUST be injected via Functional Options at runtime.
No default values or hardcoded secrets are provided in Config structs.

## Top-Level Config (`pkg/translate/config.go`)

| Name | Type | Required | Default | Security |
| --- | --- | --- | --- | --- |
| `Timeout` | `time.Duration` | No | `5s` | Safe to log |
| `Region` | `string` | No | `cn-hangzhou` | Safe to log |

## Alibaba Cloud Config (`pkg/translate/aliyun/config.go`)

| Name | Type | Required | Default | Security |
| --- | --- | --- | --- | --- |
| `AccessKeyID` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `AccessKeySecret` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `Timeout` | `time.Duration` | No | `5s` | Safe to log |
| `Region` | `string` | No | `cn-hangzhou` | Safe to log |

### Functional Options

- `aliyun.WithAccessKey(id, secret string)` — injects credentials at runtime.
- `aliyun.WithTimeout(d time.Duration)` — overrides HTTP timeout.
- `aliyun.WithRegion(region string)` — overrides target cloud region.

## Baidu Translate Config (`pkg/translate/baidu/config.go`)

| Name | Type | Required | Default | Security |
| --- | --- | --- | --- | --- |
| `AppID` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `APIKey` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `SecretKey` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `Mode` | `string` | No | `general` | Safe to log |
|  |  |  |  | Valid values: `general`, `llm`, `auto`. |
| `LLMAuth` | `string` | No | `bearer` | Safe to log |
| `TermIDs` | `string` | No | — | Safe to log |
| `Reference` | `string` | No | — | Safe to log |
| `Timeout` | `time.Duration` | No | `10s` | Safe to log |
| `Region` | `string` | No | `cn-hangzhou` | Safe to log |

### Functional Options

- `baidu.WithAppID(appID string)` — injects APPID at runtime (request body `appid`).
- `baidu.WithAPIKey(apiKey string)` — injects API Key at runtime (used as Bearer token for LLM mode).
- `baidu.WithSecretKey(secretKey string)` — injects Secret Key at runtime (used for sign generation).
- `baidu.WithMode(mode string)` — sets translation mode: `general` (通用文本翻译), `llm` (大模型文本翻译), or `auto` (randomly choose between the two).
- `baidu.WithLLMAuth(auth string)` — sets LLM auth method: `bearer` (Bearer Token) or `sign` (MD5 sign).
- `baidu.WithTermIDs(ids string)` — optional term base IDs for LLM translation.
- `baidu.WithReference(ref string)` — optional custom translation instruction for LLM translation.
- `baidu.WithTimeout(d time.Duration)` — overrides HTTP timeout.
- `baidu.WithRegion(region string)` — kept for compatibility, not used by Baidu APIs.

## Tencent Cloud Config (`pkg/translate/tencent/config.go`)

| Name | Type | Required | Default | Security |
| --- | --- | --- | --- | --- |
| `APIKey` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `Model` | `string` | No | `hy-mt2-plus` | Safe to log; use `auto` to select from `Models` |
| `Models` | `[]string` | No | `[hy-mt2-plus]` | Safe to log; candidate models for `auto` |
| `BaseURL` | `string` | No | `https://tokenhub.tencentmaas.com/v1` | Safe to log |
| `Separator` | `string` | No | `<SEP>` | Safe to log; legacy batch response fallback |
| `ModelObserver` | `func(string)` | No | `nil` | Do not log credentials from the callback |
| `Timeout` | `time.Duration` | No | `5s` | Safe to log |
| `Region` | `string` | No | `cn-hangzhou` | Safe to log |

### Functional Options

- `tencent.WithAPIKey(apiKey string)` — injects TokenHub API Key at runtime.
- `tencent.WithModel(model string)` — overrides translation model name.
- `tencent.WithModels(models ...string)` — sets candidate models used by `ModelAuto`.
- `tencent.WithModelObserver(func(model string))` — observes the actual model before each request, including retries.
- `tencent.WithBaseURL(url string)` — overrides TokenHub API base URL.
- `tencent.WithSeparator(sep string)` — overrides separator for batch translation.
- `tencent.WithTimeout(d time.Duration)` — overrides HTTP timeout.
- `tencent.WithRegion(region string)` — kept for compatibility, not used by TokenHub.

Set `Model` to `tencent.ModelAuto` to enable automatic selection. A non-2xx response is retried once. In auto mode with multiple candidates, the retry switches to the next candidate model; otherwise the same model is retried.

## Security Recommendations

1. Load secrets from environment variables or a secrets manager, never from source code.
2. Do not commit Config structs containing raw credentials to version control.
3. The `Config` structs themselves are non-sensitive and safe to serialize; secrets are injected via closures.
4. Rotate credentials regularly and use fine-grained IAM policies where supported.
