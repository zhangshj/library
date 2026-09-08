package factory

import (
	"testing"

	"github.com/zhangshj/library/pkg/translate"
)

func TestNewRejectsMissingProviderConfig(t *testing.T) {
	tests := []Provider{ProviderAliyun, ProviderBaidu, ProviderTencent}
	for _, provider := range tests {
		t.Run(string(provider), func(t *testing.T) {
			if _, err := New(Config{Provider: provider}); err == nil {
				t.Fatal("expected provider config error")
			}
		})
	}
}

func TestNewRejectsUnknownProvider(t *testing.T) {
	if _, err := New(Config{Provider: Provider("unknown")}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestNewValidProviders(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "aliyun",
			cfg:  Config{Provider: ProviderAliyun, Aliyun: &AliyunConfig{AccessKeyID: "id", AccessKeySecret: "secret"}},
			want: "aliyun",
		},
		{
			name: "baidu",
			cfg:  Config{Provider: ProviderBaidu, Baidu: &BaiduConfig{AppID: "app", APIKey: "key", SecretKey: "secret"}},
			want: "baidu",
		},
		{
			name: "tencent",
			cfg:  Config{Provider: ProviderTencent, Tencent: &TencentConfig{APIKey: "key"}},
			want: "tencent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := New(tt.cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if tr == nil {
				t.Fatal("New() returned nil translator")
			}
			var _ translate.Translator = tr
		})
	}
}
