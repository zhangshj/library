// Package factory constructs translation drivers from provider-specific configuration.
package factory

import (
    "fmt"
    "time"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/aliyun"
    "github.com/zhangshj/library/pkg/translate/baidu"
    "github.com/zhangshj/library/pkg/translate/tencent"
)

// Provider identifies a translation provider.
type Provider string

const (
    ProviderAliyun  Provider = "aliyun"
    ProviderBaidu   Provider = "baidu"
    ProviderTencent Provider = "tencent"
)

// AliyunConfig contains Alibaba Cloud credentials.
type AliyunConfig struct {
    AccessKeyID     string
    AccessKeySecret string
}

// BaiduConfig contains Baidu credentials and mode-specific options.
type BaiduConfig struct {
    AppID     string
    APIKey    string
    SecretKey string
    Mode      string
    LLMAuth   string
    TermIDs   string
    Reference string
}

// TencentConfig contains TokenHub credentials and model options.
type TencentConfig struct {
    APIKey        string
    Model         string
    Models        []string
    BaseURL       string
    Separator     string
    Domain        string
    ModelObserver func(string)
}

// Config selects a provider and carries its provider-specific settings.
type Config struct {
    Provider Provider
    Timeout  time.Duration
    Region   string

    Aliyun  *AliyunConfig
    Baidu   *BaiduConfig
    Tencent *TencentConfig
}

// New constructs a translation driver from provider-specific configuration.
func New(cfg Config) (translate.Translator, error) {
    if cfg.Provider == "" {
        return nil, fmt.Errorf("translate: provider is required")
    }
    switch cfg.Provider {
    case ProviderAliyun:
        if cfg.Aliyun == nil {
            return nil, fmt.Errorf("translate: aliyun config is required")
        }
        providerCfg := aliyun.DefaultConfig()
        return aliyun.New(providerCfg,
            aliyun.WithAccessKey(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret),
            aliyun.WithTimeout(cfg.Timeout),
            aliyun.WithRegion(cfg.Region),
        )
    case ProviderBaidu:
        if cfg.Baidu == nil {
            return nil, fmt.Errorf("translate: baidu config is required")
        }
        providerCfg := baidu.DefaultConfig()
        return baidu.New(providerCfg,
            baidu.WithAppID(cfg.Baidu.AppID),
            baidu.WithAPIKey(cfg.Baidu.APIKey),
            baidu.WithSecretKey(cfg.Baidu.SecretKey),
            baidu.WithMode(cfg.Baidu.Mode),
            baidu.WithLLMAuth(cfg.Baidu.LLMAuth),
            baidu.WithTermIDs(cfg.Baidu.TermIDs),
            baidu.WithReference(cfg.Baidu.Reference),
            baidu.WithTimeout(cfg.Timeout),
            baidu.WithRegion(cfg.Region),
        )
    case ProviderTencent:
        if cfg.Tencent == nil {
            return nil, fmt.Errorf("translate: tencent config is required")
        }
        providerCfg := tencent.DefaultConfig()
        return tencent.New(providerCfg,
            tencent.WithAPIKey(cfg.Tencent.APIKey),
            tencent.WithModel(cfg.Tencent.Model),
            tencent.WithModels(cfg.Tencent.Models...),
            tencent.WithBaseURL(cfg.Tencent.BaseURL),
            tencent.WithSeparator(cfg.Tencent.Separator),
            tencent.WithDomain(cfg.Tencent.Domain),
            tencent.WithModelObserver(cfg.Tencent.ModelObserver),
            tencent.WithTimeout(cfg.Timeout),
            tencent.WithRegion(cfg.Region),
        )
    default:
        return nil, fmt.Errorf("translate: unsupported provider %q", cfg.Provider)
    }
}
