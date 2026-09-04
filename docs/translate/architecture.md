# Translate Component Architecture

## Design Goals

- **Driver portability**: business code depends only on `Translator` interface, not on any cloud SDK.
- **Credential isolation**: secrets are injected at runtime via Functional Options, never stored in serializable configs.
- **Parallel evolution**: Alibaba Cloud, Baidu Translate, and Tencent Cloud drivers evolve independently without cross-dependencies.

## Structure

```
pkg/translate/
├── interface.go        # Translator interface (contract)
├── config.go           # Shared, non-sensitive defaults
├── aliyun/
│   └── aliyun.go       # Alibaba Cloud alimt implementation
├── baidu/
│   └── baidu.go        # Baidu Translate implementation (General + LLM)
├── tencent/
│   └── tencent.go      # Tencent Cloud TokenHub implementation
└── mock/
    └── mock.go         # In-memory mock for testing and demos
```

## Rationale

1. **Interface-first**: The `Translate` and `TranslateBatch` methods keep the surface area minimal. Adding new drivers or new methods does not break existing callers.

2. **Functional Options for secrets**: Following the Go functional options pattern, credentials are passed as closures. This prevents accidental serialization of secrets in Config structs and makes the API self-documenting.

3. **Driver isolation**: Each cloud driver lives in its own sub-package with its own Config and error types. There is no shared mutable state between drivers.

4. **Error wrapping**: All driver errors are wrapped with context using `fmt.Errorf("driver: %w", err)`, preserving the original error chain while adding provider identification.

5. **Mock driver**: The mock driver provides a deterministic, zero-network implementation that enables unit testing of business logic without cloud dependencies.

## Batch Translation Strategy

- **Alibaba Cloud**: Uses the native `GetBatchTranslate` API, which sends all texts in a single HTTP request. This avoids per-request rate limits and is the most efficient approach.
- **Baidu Translate (General)**: Uses `\n` to join all texts into a single `q` parameter, sending one API call. The response returns an array of `trans_result` preserving input order.
- **Baidu Translate (LLM)**: Uses `\n` to join all texts into a single `q` parameter, sending one API call for both Bearer and Sign auth modes. The response returns an array of `trans_result` preserving input order.
- **Tencent Cloud TokenHub**: Sends each batch item as an indexed XML `<item>` in one Chat Completions prompt. The expected response is indexed XML, which preserves item boundaries and input order. The parser keeps `<SEP>` splitting as a compatibility fallback for older model responses.

## Tencent Model Selection and Retry

Tencent requests use the configured `Model`. When it is `auto`, one candidate from `Models` is selected for each request. Any non-2xx response is retried once; with multiple auto candidates, the retry uses the next candidate. `WithModelObserver` exposes the actual model before every request without forcing the library to write application logs.

## Extension Points

To add a new cloud provider (e.g., Google Translate):

1. Create `pkg/translate/<provider>/<provider>.go`
2. Implement `translate.Translator`
3. Add driver-specific Config and Functional Options
4. Add docs under `docs/translate/<provider>/`
5. Add runnable example under `examples/translate/`
