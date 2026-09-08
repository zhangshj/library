package local

import (
    "context"
    "strings"
    "sync/atomic"
    "testing"
    "time"

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
    defer tr.Close()
    id, err := tr.SubmitTask(context.Background(), transcoder.TranscodeRequest{
        SrcObject: "input.mp4", DstObject: "out/result.mp4",
        Template: transcoder.TranscodeTemplate{Name: "test", VideoCodec: "copy"},
    })
    if err != nil {
        t.Fatalf("SubmitTask() error = %v", err)
    }
    if id != "local-1" {
        t.Fatalf("id = %q", id)
    }
    waitForState(t, tr, id, transcoder.TaskSuccess)
    if runner.name != "ffmpeg" {
        t.Fatalf("runner name = %q", runner.name)
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
    defer tr.Close()
    id, err := tr.SubmitTask(context.Background(), transcoder.TranscodeRequest{
        SrcObject: "input.mp4", DstObject: "out.mp4",
        Template: transcoder.TranscodeTemplate{Name: "test"},
    })
    if err != nil || id != "local-1" {
        t.Fatalf("id/error = %q/%v", id, err)
    }
    result := waitForState(t, tr, id, transcoder.TaskFailed)
    _, queryErr := tr.QueryTask(context.Background(), id)
    if queryErr != nil || result.State != transcoder.TaskFailed {
        t.Fatalf("result/error = %+v/%v", result, queryErr)
    }
}

type concurrencyRunner struct {
    started chan struct{}
    release chan struct{}
    active  int32
    max     int32
}

func (r *concurrencyRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
    active := atomic.AddInt32(&r.active, 1)
    for {
        max := atomic.LoadInt32(&r.max)
        if active <= max || atomic.CompareAndSwapInt32(&r.max, max, active) {
            break
        }
    }
    r.started <- struct{}{}
    <-r.release
    atomic.AddInt32(&r.active, -1)
    return nil, nil
}

func TestMaxConcurrentLimitsWorkers(t *testing.T) {
    runner := &concurrencyRunner{started: make(chan struct{}, 2), release: make(chan struct{}, 2)}
    tr, err := New(Config{MaxConcurrent: 1, QueueSize: 2}, WithRunner(runner))
    if err != nil {
        t.Fatalf("New() error = %v", err)
    }
    defer tr.Close()
    request := transcoder.TranscodeRequest{
        SrcObject: "input.mp4", DstObject: "out.mp4",
        Template: transcoder.TranscodeTemplate{Name: "test", VideoCodec: "copy"},
    }
    first, err := tr.SubmitTask(context.Background(), request)
    if err != nil {
        t.Fatalf("first SubmitTask() error = %v", err)
    }
    second, err := tr.SubmitTask(context.Background(), request)
    if err != nil {
        t.Fatalf("second SubmitTask() error = %v", err)
    }
    select {
    case <-runner.started:
    case <-time.After(time.Second):
        t.Fatal("first task did not start")
    }
    if result, err := tr.QueryTask(context.Background(), second); err != nil || result.State != transcoder.TaskWaiting {
        t.Fatalf("second task state/error = %s/%v, want Waiting/nil", result.State, err)
    }
    runner.release <- struct{}{}
    select {
    case <-runner.started:
    case <-time.After(time.Second):
        t.Fatal("second task did not start")
    }
    runner.release <- struct{}{}
    waitForState(t, tr, first, transcoder.TaskSuccess)
    waitForState(t, tr, second, transcoder.TaskSuccess)
    if got := atomic.LoadInt32(&runner.max); got != 1 {
        t.Fatalf("max concurrent runners = %d, want 1", got)
    }
}

func waitForState(t *testing.T, tr *Transcoder, taskID string, want transcoder.TaskState) *transcoder.TranscodeResult {
    t.Helper()
    deadline := time.Now().Add(time.Second)
    for time.Now().Before(deadline) {
        result, err := tr.QueryTask(context.Background(), taskID)
        if err != nil {
            t.Fatalf("QueryTask() error = %v", err)
        }
        if result.State == want {
            return result
        }
        time.Sleep(time.Millisecond)
    }
    result, _ := tr.QueryTask(context.Background(), taskID)
    t.Fatalf("task %q state = %s, want %s", taskID, result.State, want)
    return nil
}
