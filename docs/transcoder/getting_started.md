# Transcoder Getting Started

统一转码接口通过工厂选择本地 ffmpeg 或阿里云 MTS 实现。

```go
cfg := transcoder.Config{Provider: transcoder.ProviderLocal, FFmpeg: "ffmpeg"}
t, err := factory.New(cfg)
if err != nil { log.Fatal(err) }
taskID, err := t.SubmitTask(ctx, transcoder.TranscodeRequest{
    SrcObject: "input.mp4", DstObject: "output.mp4",
    Template: transcoder.TranscodeTemplate{Name: "mp4", VideoCodec: "libx264", AudioCodec: "aac"},
})
```

本地驱动同步执行 ffmpeg，提交成功后可通过 `QueryTask` 获取结果。阿里云驱动使用 OSS bucket/object 和 MTS 模板。
