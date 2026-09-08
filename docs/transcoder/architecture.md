# Transcoder Architecture

转码模块将任务描述、模板和状态统一定义在 `pkg/transcoder`，具体 provider 只负责把这些结构映射到执行引擎。

- `pkg/transcoder/local` 使用有界异步队列和可注入的命令执行器调用 ffmpeg；worker 数量由 `MaxConcurrent` 控制，任务状态保存在进程内，适合单机或开发环境。
- `pkg/transcoder/aliyun` 使用阿里云 MTS SDK，模板通过 `AddTemplate` 创建，任务通过 `SubmitJobs` 提交，状态通过 `QueryJobList` 查询。
- `pkg/transcoder/tencent` 使用腾讯云 COS CI SDK，任务通过 `CreateJob` 提交，状态通过 `DescribeMultiMediaJob` 查询。
- `pkg/transcoder/factory` 负责构造 provider，避免根包反向依赖驱动子包造成 import cycle。

云端任务和本地任务都通过统一的任务 ID 查询状态。阿里云和腾讯云任务由云端异步执行；本地任务由后台 worker 从有界队列取出执行。阿里云凭证只在驱动配置中使用，业务层只依赖统一接口。
