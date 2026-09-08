# go-kit 组件库文档导航

本库提供高内聚、低耦合的内部基础设施组件，每个组件独立配置、独立文档、独立示例。

## 组件清单

| 组件 | 包路径 | 文档 | 示例 |
| --- | --- | --- | --- |
| **translate** | `pkg/translate` | [docs/translate/](docs/translate/) | [examples/translate/](examples/translate/) |
| **ratelimit** | `pkg/ratelimit` | [docs/ratelimit/](docs/ratelimit/) | [examples/ratelimit/](examples/ratelimit/) |
| **transcoder** | `pkg/transcoder` | [docs/transcoder/](docs/transcoder/) | [examples/transcoder/](examples/transcoder/) |

### translate

机器翻译统一封装，支持多云驱动。

- 接口定义：`pkg/translate/interface.go`
- 配置：`pkg/translate/config.go`
- 工厂：`pkg/translate/factory`
- 阿里云驱动：`pkg/translate/aliyun`
- 腾讯云驱动：`pkg/translate/tencent`
- 快速上手：[docs/translate/getting_started.md](docs/translate/getting_started.md)
- 配置说明：[docs/translate/configuration.md](docs/translate/configuration.md)
- 架构设计：[docs/translate/architecture.md](docs/translate/architecture.md)

### ratelimit

框架无关的固定窗口限流组件，支持内存 / Redis 双后端，用于登录类接口防爆破。

- 接口定义：`pkg/ratelimit/interface.go`
- 配置：`pkg/ratelimit/config.go`
- 内存后端：`pkg/ratelimit/memory_store.go`
- Redis 后端（Lua 原子计数）：`pkg/ratelimit/redis_store.go`
- 快速上手：[docs/ratelimit/getting_started.md](docs/ratelimit/getting_started.md)
- 配置说明：[docs/ratelimit/configuration.md](docs/ratelimit/configuration.md)
- 架构设计：[docs/ratelimit/architecture.md](docs/ratelimit/architecture.md)

### transcoder

统一媒体转码封装，支持本地 ffmpeg、阿里云 MTS 和腾讯云 CI。

- 接口定义：`pkg/transcoder/interface.go`
- 配置：`pkg/transcoder/config.go`
- 阿里云驱动：`pkg/transcoder/aliyun`
- 本地驱动：`pkg/transcoder/local`
- 腾讯云驱动：`pkg/transcoder/tencent`
- 工厂：`pkg/transcoder/factory`
- 快速上手：[docs/transcoder/getting_started.md](docs/transcoder/getting_started.md)
- 配置说明：[docs/transcoder/configuration.md](docs/transcoder/configuration.md)
- 架构设计：[docs/transcoder/architecture.md](docs/transcoder/architecture.md)
