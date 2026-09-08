package factory

import (
	"testing"

	"github.com/zhangshj/library/pkg/transcoder"
)

func TestNewLocal(t *testing.T) {
	tr, err := New(transcoder.Config{Provider: transcoder.ProviderLocal, FFmpeg: "ffmpeg"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if tr.Name() != "local" {
		t.Fatalf("Name() = %q", tr.Name())
	}
}

func TestNewRejectsMissingCloudSettings(t *testing.T) {
	for _, provider := range []transcoder.Provider{transcoder.ProviderAliyun, transcoder.ProviderTencent} {
		t.Run(string(provider), func(t *testing.T) {
			if _, err := New(transcoder.Config{Provider: provider}); err == nil {
				t.Fatal("expected missing cloud settings error")
			}
		})
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	if _, err := New(transcoder.Config{}); err == nil {
		t.Fatal("expected provider validation error")
	}
	if _, err := New(transcoder.Config{Provider: transcoder.Provider("unknown")}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
