package transcoder

import (
    "fmt"
    "strings"
)

// Provider identifies a transcoding driver.
type Provider string

const (
    ProviderAliyun  Provider = "aliyun"
    ProviderLocal   Provider = "local"
    ProviderTencent Provider = "tencent"
)

// Config contains provider-neutral transcoder settings.
type Config struct {
    Provider        Provider
    Region          string
    AccessKey       string
    SecretKey       string
    RoleARN         string
    BucketURL       string
    FFmpeg          string
    WorkDir         string
    HardwareBackend string
    HardwareDevice  string
}

// Validate checks the settings shared by the factory and drivers.
func (c Config) Validate() error {
    if strings.TrimSpace(string(c.Provider)) == "" {
        return fmt.Errorf("transcoder: provider is required")
    }
    if c.Provider == ProviderAliyun || c.Provider == ProviderTencent {
        if c.Region == "" {
            return fmt.Errorf("transcoder: region is required for aliyun")
        }
        if c.AccessKey == "" || c.SecretKey == "" {
            return fmt.Errorf("transcoder: access key pair is required for cloud provider %q", c.Provider)
        }
    }
    if c.Provider == ProviderTencent && c.BucketURL == "" {
        return fmt.Errorf("transcoder: bucket URL is required for tencent")
    }
    return nil
}
