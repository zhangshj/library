# Transcoder Configuration

| 配置项 | 类型 | 必填 | 默认值 | 安全建议 |
| --- | --- | --- | --- | --- |
| `Provider` | `transcoder.Provider` | 是 | 无 | 只使用 `local` 或 `aliyun` |
| `Region` | `string` | 阿里云必填 | 无 | 使用目标 OSS/MTS 地域 |
| `AccessKey` | `string` | 阿里云必填 | 无 | 从环境变量或密钥管理系统注入 |
| `SecretKey` | `string` | 阿里云必填 | 无 | 不要写入代码、日志或示例 |
| `RoleARN` | `string` | 否 | 空 | 优先使用 RAM 角色限制权限 |
| `FFmpeg` | `string` | 本地可选 | `ffmpeg` | 使用受信任的可执行文件路径 |
| `WorkDir` | `string` | 否 | 当前目录 | 限制本地输入输出目录权限 |
| `BucketURL` | `string` | 腾讯云必填 | 无 | 使用对应 COS 桶域名，不要从用户输入直接拼接 |
| `HardwareBackend` | `string` | 否 | 空（CPU） | 本地可选 `videotoolbox`、`cuda`、`nvenc`、`vaapi`、`qsv`、`amf` |
| `HardwareDevice` | `string` | 否 | 空 | NVIDIA 可填 GPU 序号，VAAPI 可填 `/dev/dri/renderD128` |
| `MaxConcurrent` | `int` | 否 | `1` | 按 GPU 编码会话能力设置，避免多个任务争抢硬件资源 |
| `QueueSize` | `int` | 否 | `100` | 限制等待队列长度，防止任务无限堆积 |

阿里云转码请求中的 `SrcBucket`、`DstBucket` 是 OSS bucket 名，`SrcObject`、`DstObject` 是对象键。腾讯云通过 `BucketURL` 指定 COS 桶，输出使用请求中的目标 bucket/object。`local` 驱动中 bucket/object 被解释为路径前缀和相对文件路径；通常只填写 `SrcObject` 和 `DstObject`。

本地 GPU 示例：macOS 使用 `HardwareBackend: "videotoolbox"`；NVIDIA 使用 `HardwareBackend: "cuda"` 或 `"nvenc"`；Linux Intel 可使用 `"qsv"`，Linux VAAPI 可使用 `"vaapi"`。FFmpeg 必须是带对应硬件编码器的构建，驱动不会自动安装或模拟 GPU。

本地转码是异步队列模型：`SubmitTask` 只入队并返回任务 ID，`QueryTask` 可查询 `Waiting`、`Running`、`Success` 或 `Failed`。`MaxConcurrent` 控制同时运行的 FFmpeg 数量，建议根据 GPU 数量和编码器会话上限配置。

