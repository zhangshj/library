# 内部组件库开发规范（Project Rules）

> 本文件是 AI Coding Agent 和协作者的唯一开发依据。违反以下原则的代码将被拒绝合并。

## 1. 角色与目标
你是一名资深 Go 基础架构工程师。负责开发高内聚、低耦合的 Go 内部组件库，用于统一封装内部基础设施能力（如存储、消息、缓存等），为业务方提供开箱即用且安全的 API。

## 2. 目录与模块化铁律
### 2.1 依赖与 Module 规范
- **严格平行**：`pkg/` 下的每个组件，必须在 `examples/` 和 `docs/` 中有同名的子目录与之对应。**禁止**把所有示例堆在根 examples 里，禁止把所有说明写进同一个大而全的 markdown。
- **禁止跨组件依赖**：`pkg/storage` 不能 import `pkg/notify` 的任何具体实现；通用能力抽到 `internal/`。
- **配置独立**：每个组件在自己的目录下维护 `config.go`，各自校验。**严禁**定义全局大一统 Config 结构体。
### 2.2 导入与暴露规范
- **严禁相对路径导入**：所有代码示例和内部代码必须使用完整的 Module 路径导入（如 `git.xxx.com/infra/go-kit/pkg/storage`），禁止出现 `import "../internal/retry"` 这种写法。
- **最小化暴露**：每个组件只暴露一个顶层包（如 `storage`）和具体的驱动子包（如 `cos`），不要在 `pkg/` 根目录写杂七杂八的转发文件。
- **Examples 示范**：`examples/` 里的 demo 必须演示如何单独引入某一个组件并运行，不要在一个 demo 里把 pkg 下所有组件都 import 进来。

## 3. 编码设计规范
### 3.1 接口先行
- 所有能力必须先定义在 `interface.go` 中，对外暴露接口，隐藏具体实现（如腾讯云 COS SDK）。
- 第一个参数必须是 `ctx context.Context`。

### 3.2 配置与安全
- 敏感信息（如 SecretKey）**不得**在 Config 结构体中提供默认值或硬编码，必须通过 Functional Options 在运行时注入。
- 每个组件需定义合理的默认值常量（如分块大小、超时时间），并在 `config.go` 中显式声明。

### 3.3 存储组件特别约定（参考实现）
- 下载方法必须智能路由：文件较小走简单接口，>= 阈值（默认 16MB）必须走并发分块+断点续传（`Download` 接口），对上层完全屏蔽差异。
- 错误处理必须使用 `fmt.Errorf("xxx: %w", err)` 包装，携带上下文。

## 4. 文档与示例规范
生成任何代码后，必须同步更新对应模块的文档和示例：
### 4.1 模块级 docs 必须包含
- `{module}/getting_started.md`：3~5 行代码能跑的最小 Demo。
- `{module}/configuration.md`：用 Markdown 表格列清该模块所有配置项（名称、类型、是否必填、默认值、安全建议）。
- `{module}/architecture.md`：简述设计思路（如为什么要做分块封装）。

### 4.2 示例要求
- `examples/{module}/` 下的代码必须是可以直接 `go run` 的独立文件。
- 示例代码中必须演示从“读取配置 -> 初始化 -> 正常调用 -> 错误处理”的完整链路。

## 5. 质量底线
- 导出函数/类型/接口必须写 GoDoc 注释。
- 核心逻辑需编写 Table-Driven 单元测试，覆盖率 >= 50%。
- Commit Message 遵循 Conventional Commits（`feat(storage):`, `docs(notify):` 等 scope 格式）。

---
## 6. 当前需执行的初始化任务
请基于以上规范，检查并执行：
- [x] 建立上述目录骨架
- [ ] 编写 `pkg/storage/interface.go` 和 `config.go`（遵循独立配置原则）
- [ ] 生成 `docs/storage/configuration.md` 和 `docs/storage/getting_started.md`
- [ ] 生成 `examples/storage/cos_get_started.go`
- [ ] 输出 `docs/index.md` 汇总导航