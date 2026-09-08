package local

import (
    "context"
    "strings"
    "testing"

    "github.com/zhangshj/library/pkg/transcoder"
)

type fakeRunner struct {
    name string
    args []string
    err  error
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
    r.name = name
    r.args = append([]string(nil), args...)
    return []byte("ffmpeg output"), r.err
}

func TestBuildFFmpegArgs(t *testing.T) {
    args := buildFFmpegArgs("in.mp4", "out.mp4", transcoder.TranscodeTemplate{
        VideoCodec: "libx264", Bitrate: 1000, Width: 1280, Height: 720,
        AudioCodec: "aac", AudioBitrate: 128, AudioSamplerate: 44100, AudioChannels: 2,
    })
    joined := strings.Join(args, " ")
    for _, want := range []string{"-i in.mp4", "-c:v libx264", "-b:v 1000k", "-vf scale=1280:720", "-c:a aac", "-b:a 128k", "-ar 44100", "-ac 2", "out.mp4"} {
        if !strings.Contains(joined, want) {
            t.Errorf("args %q missing %q", joined, want)
        }
    }
}

func TestBuildHardwareFFmpegArgs(t *testing.T) {
    tests := []struct {
        name   string
        config hardwareConfig
        codec  string
        wants  []string
    }{
        {name: "mac", config: hardwareConfig{backend: "videotoolbox"}, codec: "h264", wants: []string{"-hwaccel videotoolbox", "-c:v h264_videotoolbox"}},
        {name: "nvidia", config: hardwareConfig{backend: "cuda", device: "1"}, codec: "h264", wants: []string{"-hwaccel cuda", "-hwaccel_output_format cuda", "-c:v h264_nvenc", "-gpu 1"}},
        {name: "vaapi", config: hardwareConfig{backend: "vaapi", device: "/dev/dri/renderD128"}, codec: "hevc", wants: []string{"-vaapi_device /dev/dri/renderD128", "-c:v hevc_vaapi", "scale_vaapi=1280:720"}},
        {name: "qsv", config: hardwareConfig{backend: "qsv"}, codec: "h264", wants: []string{"-hwaccel qsv", "-c:v h264_qsv"}},
        {name: "amf", config: hardwareConfig{backend: "amf"}, codec: "h264", wants: []string{"-c:v h264_amf"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            args := buildFFmpegArgsWithHardware("in.mp4", "out.mp4", transcoder.TranscodeTemplate{
                VideoCodec: tt.codec, Width: 1280, Height: 720,
            }, tt.config)
            joined := strings.Join(args, " ")
            for _, want := range tt.wants {
                if !strings.Contains(joined, want) {
                    t.Errorf("args %q missing %q", joined, want)
                }
            }
        })
    }
}

func TestNewRejectsUnsupportedHardwareBackend(t *testing.T) {
    if _, err := New(Config{HardwareBackend: "unknown"}); err == nil {
        t.Fatal("expected unsupported hardware backend error")
    }
}

func TestSubmitAndQuery(t *testing.T) {
    runner := &fakeRunner{}
    tr, err := New(Config{FFmpegPath: "ffmpeg"}, WithRunner(runner))
    if err != nil {
        t.Fatalf("New() error = %v", err)
    }
    id, err := tr.SubmitTask(context.Background(), transcoder.TranscodeRequest{
        SrcObject: "input.mp4", DstObject: "out/result.mp4",
        Template: transcoder.TranscodeTemplate{Name: "test", VideoCodec: "copy"},
    })
    if err != nil {
        t.Fatalf("SubmitTask() error = %v", err)
    }
    if id != "local-1" || runner.name != "ffmpeg" {
        t.Fatalf("id/name = %q/%q", id, runner.name)
    }
    result, err := tr.QueryTask(context.Background(), id)
    if err != nil {
        t.Fatalf("QueryTask() error = %v", err)
    }
    if result.State != transcoder.TaskSuccess || !strings.HasPrefix(result.OutputURL, "file://") {
        t.Fatalf("result = %+v", result)
    }
}

func TestSubmitFailure(t *testing.T) {
    runner := &fakeRunner{err: context.Canceled}
    tr, _ := New(Config{}, WithRunner(runner))
    id, err := tr.SubmitTask(context.Background(), transcoder.TranscodeRequest{
        SrcObject: "input.mp4", DstObject: "out.mp4",
        Template: transcoder.TranscodeTemplate{Name: "test"},
    })
    if err == nil || id != "local-1" {
        t.Fatalf("id/error = %q/%v", id, err)
    }
    result, queryErr := tr.QueryTask(context.Background(), id)
    if queryErr != nil || result.State != transcoder.TaskFailed {
        t.Fatalf("result/error = %+v/%v", result, queryErr)
    }
}
