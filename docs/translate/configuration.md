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

## Tencent Cloud Config (`pkg/translate/tencent/config.go`)

| Name | Type | Required | Default | Security |
| --- | --- | --- | --- | --- |
| `SecretID` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `SecretKey` | `string` | Yes (via option) | — | **Sensitive — do not log** |
| `Timeout` | `time.Duration` | No | `5s` | Safe to log |
| `Region` | `string` | No | `cn-hangzhou` | Safe to log |

### Functional Options

- `tencent.WithSecretKey(id, key string)` — injects credentials at runtime.
- `tencent.WithTimeout(d time.Duration)` — overrides HTTP timeout.
- `tencent.WithRegion(region string)` — overrides target cloud region.

## Security Recommendations

1. Load secrets from environment variables or a secrets manager, never from source code.
2. Do not commit Config structs containing raw credentials to version control.
3. The `Config` structs themselves are non-sensitive and safe to serialize; secrets are injected via closures.
4. Rotate credentials regularly and use fine-grained IAM policies where supported.
