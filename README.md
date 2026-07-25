# Go-Kit Library / Go 内部组件库

<p align="center">
  <strong>A unified, production-grade Go component library for machine translation across multiple cloud providers.</strong>
</p>

---

# Go 内部组件库

<p align="center">
  <strong>一套高内聚、低耦合的 Go 组件库，统一封装多云厂商机器翻译能力，开箱即用。</strong>
</p>

---

## Table of Contents / 目录

- [Features / 特性](#features--特性)
- [Supported Providers / 支持的平台](#supported-providers--支持的平台)
- [Quick Start / 快速开始](#quick-start--快速开始)
- [Project Structure / 项目结构](#project-structure--项目结构)
- [Documentation / 文档](#documentation--文档)
- [Development / 开发](#development--开发)
- [License / 许可证](#license--许可证)

---

## Features / 特性

- **Unified Interface / 统一接口**  
  All providers implement the same `Translator` interface (`Translate` / `TranslateBatch`), so business code never depends on a specific cloud SDK.  
  所有平台均实现相同的 `Translator` 接口（`Translate` / `TranslateBatch`），业务代码不依赖具体云厂商 SDK。

- **Credential Isolation / 凭据隔离**  
  Sensitive credentials are injected via Functional Options at runtime. No hardcoded secrets or default values in Config structs.  
  敏感凭据通过 Functional Options 在运行时注入，Config 结构体中不提供默认值或硬编码。

- **Single & Batch Translation / 单条与批量翻译**  
  Every driver supports both single-text translation and batch translation.  
  所有驱动均支持单条翻译和批量翻译。

- **Driver Isolation / 驱动隔离**  
  Each provider lives in its own sub-package with independent Config, error types, and tests. No cross-dependencies between drivers.  
  每个平台独立子包，拥有独立的 Config、错误类型和测试，驱动间无跨依赖。

- **Production Ready / 生产就绪**  
  Table-Driven unit tests with ≥50% coverage, error wrapping with context, GoDoc comments, and Conventional Commits.  
  单元测试覆盖率 ≥50%，错误包装携带上下文，GoDoc 注释完整，Commit 遵循 Conventional Commits。

---

## Supported Providers / 支持的平台

| Provider / 平台 | Mode / 模式 | Auth / 鉴权 | Single / 单条 | Batch / 批量 | Package / 包 |
| --- | --- | --- | --- | --- | --- |
| **Alibaba Cloud** / 阿里云 | General | AccessKey | ✅ | ✅ (native batch API) | `pkg/translate/aliyun` |
| **Baidu Translate** / 百度翻译 | General | AppID + SecretKey (MD5 sign) | ✅ | ✅ (`\n` joined single API call) | `pkg/translate/baidu` |
| **Baidu Translate** / 百度翻译 | Large Model (LLM) | Bearer Token / MD5 sign | ✅ | ✅ (`\n` joined single API call) | `pkg/translate/baidu` |
| **Tencent TokenHub** / 腾讯云 | Large Model (LLM) | API Key | ✅ | ✅ (`<SEP>` joined single prompt) | `pkg/translate/tencent` |

---

## Quick Start / 快速开始

```bash
go get github.com/zhangshj/library/pkg/translate/aliyun
go get github.com/zhangshj/library/pkg/translate/baidu
go get github.com/zhangshj/library/pkg/translate/tencent
```

### Alibaba Cloud / 阿里云

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/aliyun"
)

func main() {
    ctx := context.Background()
    tr, err := aliyun.New(
        translate.DefaultConfig(),
        aliyun.WithAccessKey("your-access-key-id", "your-access-key-secret"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Single text / 单条翻译
    result, err := tr.Translate(ctx, "Hello World", "en", "zh")
    if err != nil {
        log.Println("translate failed:", err)
    } else {
        fmt.Println(result)
    }

    // Batch translation / 批量翻译
    results, err := tr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("batch translate failed:", err)
    } else {
        fmt.Println(results)
    }
}
```

### Baidu Translate / 百度翻译

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/baidu"
)

func main() {
    ctx := context.Background()
    tr, err := baidu.New(
        translate.DefaultConfig(),
        baidu.WithAppID("your-appid"),
        baidu.WithAPIKey("your-api-key"),
        baidu.WithSecretKey("your-secret-key"),
        baidu.WithMode("llm"),               // "general" or "llm"
        baidu.WithLLMAuth("bearer"),          // "bearer" or "sign"
    )
    if err != nil {
        log.Fatal(err)
    }

    // Single text / 单条翻译
    result, err := tr.Translate(ctx, "Hello World", "en", "zh")
    if err != nil {
        log.Println("translate failed:", err)
    } else {
        fmt.Println(result)
    }

    // Batch translation / 批量翻译
    results, err := tr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("batch translate failed:", err)
    } else {
        fmt.Println(results)
    }
}
```

### Tencent TokenHub / 腾讯云

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/tencent"
)

func main() {
    ctx := context.Background()
    tr, err := tencent.New(
        translate.DefaultConfig(),
        tencent.WithAPIKey(os.Getenv("TOKENHUB_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Single text / 单条翻译
    result, err := tr.Translate(ctx, "Hello World", "en", "zh")
    if err != nil {
        log.Println("translate failed:", err)
    } else {
        fmt.Println(result)
    }

    // Batch translation / 批量翻译
    results, err := tr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("batch translate failed:", err)
    } else {
        fmt.Println(results)
    }
}
```

---

## Project Structure / 项目结构

```
.
├── AGENTS.md                          # Development rules / 开发规范
├── LICENSE                            # Apache 2.0 / Apache 2.0 许可证
├── go.mod                             # Go module definition
├── pkg/
│   └── translate/
│       ├── interface.go               # Translator interface / 翻译接口定义
│       ├── config.go                  # Shared top-level config / 共享配置
│       ├── aliyun/                    # Alibaba Cloud driver / 阿里云驱动
│       │   ├── aliyun.go
│       │   └── aliyun_test.go
│       ├── baidu/                     # Baidu Translate driver / 百度翻译驱动
│       │   ├── baidu.go
│       │   └── baidu_test.go
│       ├── tencent/                   # Tencent TokenHub driver / 腾讯云驱动
│       │   ├── tencent.go
│       │   └── tencent_test.go
│       └── mock/                      # In-memory mock driver / 内存 Mock 驱动
│           ├── mock.go
│           └── mock_test.go
├── examples/
│   └── translate/
│       ├── aliyun/main.go             # Alibaba Cloud example / 阿里云示例
│       ├── baidu/main.go              # Baidu Translate example / 百度示例
│       └── tencent/main.go            # Tencent TokenHub example / 腾讯示例
└── docs/
    └── translate/
        ├── getting_started.md         # Quick start guide / 快速开始
        ├── configuration.md           # Config reference / 配置参考
        └── architecture.md            # Design rationale / 设计思路
```

---

## Documentation / 文档

- **Getting Started / 快速开始**: `docs/translate/getting_started.md`
- **Configuration / 配置说明**: `docs/translate/configuration.md`
- **Architecture / 架构设计**: `docs/translate/architecture.md`

---

## Development / 开发

### Prerequisites / 前置要求

- Go 1.20+
- Git

### Build & Test / 构建与测试

```bash
go build ./...
go test ./...
```

### Code Standards / 代码规范

This project follows strict modular and security rules defined in [`AGENTS.md`](./AGENTS.md):

- **Modular / 模块化**: Each driver in its own sub-package; no cross-dependencies.  
  每个驱动独立子包，禁止跨组件依赖。
- **Security / 安全**: No hardcoded secrets; credentials injected at runtime.  
  无硬编码密钥；凭据运行时注入。
- **Documentation Sync / 文档同步**: Technical changes must be reflected in docs immediately.  
  技术方案变更后，文档必须同步更新。
- **Testing / 测试**: Table-Driven tests with ≥50% coverage.  
  单元测试覆盖率 ≥50%。

---

## License / 许可证

Licensed under the [Apache License 2.0](./LICENSE).

本仓库采用 [Apache License 2.0](./LICENSE) 开源协议。
