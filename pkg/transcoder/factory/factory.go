// Package factory constructs transcoder drivers from a shared configuration.
package factory

import (
    "fmt"

    "github.com/zhangshj/library/pkg/transcoder"
    "github.com/zhangshj/library/pkg/transcoder/aliyun"
    "github.com/zhangshj/library/pkg/transcoder/local"
    "github.com/zhangshj/library/pkg/transcoder/tencent"
)

// New creates a transcoder for the configured provider.
func New(cfg transcoder.Config) (transcoder.Transcoder, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    switch cfg.Provider {
    case transcoder.ProviderAliyun:
        return aliyun.New(aliyun.Config{
            AccessKeyID:     cfg.AccessKey,
            AccessKeySecret: cfg.SecretKey,
            Region:          cfg.Region,
            RoleARN:         cfg.RoleARN,
        })
    case transcoder.ProviderLocal:
        return local.New(local.Config{
            FFmpegPath:      cfg.FFmpeg,
            WorkDir:         cfg.WorkDir,
            HardwareBackend: cfg.HardwareBackend,
            HardwareDevice:  cfg.HardwareDevice,
        })
    case transcoder.ProviderTencent:
        return tencent.New(tencent.Config{BucketURL: cfg.BucketURL, SecretID: cfg.AccessKey, SecretKey: cfg.SecretKey})
    default:
        return nil, fmt.Errorf("transcoder: unsupported provider %q", cfg.Provider)
    }
}
