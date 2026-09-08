package aliyun

import (
	"context"
	"testing"

	"github.com/zhangshj/library/pkg/transcoder"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{name: "missing access key", cfg: Config{Region: "cn-hangzhou", AccessKeySecret: "secret"}, want: true},
		{name: "missing secret", cfg: Config{Region: "cn-hangzhou", AccessKeyID: "id"}, want: true},
		{name: "missing region", cfg: Config{AccessKeyID: "id", AccessKeySecret: "secret"}, want: true},
		{name: "valid", cfg: Config{Region: "cn-hangzhou", AccessKeyID: "id", AccessKeySecret: "secret"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if (err != nil) != tt.want {
				t.Fatalf("New() error = %v, want error = %v", err, tt.want)
			}
		})
	}
}

func TestConfigMapping(t *testing.T) {
	tmpl := transcoder.TranscodeTemplate{
		VideoCodec: "H.264", Bitrate: 1000, Fps: 25, Width: 1280, Height: 720,
		AudioCodec: "AAC", AudioBitrate: 128, AudioSamplerate: 44100, AudioChannels: 2,
	}
	video := videoConfig(tmpl)
	if video["Bitrate"] != "1000000" || video["Width"] != "1280" || video["Height"] != "720" {
		t.Fatalf("video config = %#v", video)
	}
	audio := audioConfig(tmpl)
	if audio["Bitrate"] != "128000" || audio["Channels"] != "2" {
		t.Fatalf("audio config = %#v", audio)
	}
	if ossLocation("cn-shanghai") != "oss-cn-shanghai" {
		t.Fatalf("unexpected OSS location")
	}
	if got := ossURL("bucket", "out.mp4", "oss-cn-hangzhou"); got != "https://bucket.oss-cn-hangzhou.aliyuncs.com/out.mp4" {
		t.Fatalf("ossURL() = %q", got)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := contextErr(ctx); err == nil {
		t.Fatal("expected context cancellation")
	}
}
