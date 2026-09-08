// Package local implements transcoding with a local ffmpeg binary.
package local

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"

    "github.com/zhangshj/library/pkg/transcoder"
)

// CommandRunner executes an external command. It is injectable for tests.
type CommandRunner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
    return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Config configures the local ffmpeg driver.
type Config struct {
    FFmpegPath      string
    WorkDir         string
    HardwareBackend string
    HardwareDevice  string
}

// DefaultConfig returns a local configuration that uses ffmpeg from PATH.
func DefaultConfig() Config { return Config{FFmpegPath: "ffmpeg"} }

// Option customizes a local transcoder.
type Option func(*Translator)

// WithRunner replaces the command runner, primarily for tests.
func WithRunner(runner CommandRunner) Option {
    return func(t *Translator) {
        if runner != nil {
            t.runner = runner
        }
    }
}

// Translator is a synchronous local ffmpeg transcoder.
type Translator struct {
    ffmpegPath string
    workDir    string
    hardware   hardwareConfig
    runner     CommandRunner
    sequence   uint64
    mu         sync.RWMutex
    templates  map[string]transcoder.TranscodeTemplate
    tasks      map[string]*transcoder.TranscodeResult
}

// New creates a local ffmpeg transcoder.
func New(cfg Config, opts ...Option) (*Translator, error) {
    if cfg.FFmpegPath == "" {
        cfg.FFmpegPath = "ffmpeg"
    }
    hardware, err := normalizeHardwareConfig(cfg.HardwareBackend, cfg.HardwareDevice)
    if err != nil {
        return nil, err
    }
    t := &Translator{
        ffmpegPath: cfg.FFmpegPath,
        workDir:    cfg.WorkDir,
        runner:     execRunner{},
        hardware:   hardware,
        templates:  make(map[string]transcoder.TranscodeTemplate),
        tasks:      make(map[string]*transcoder.TranscodeResult),
    }
    for _, opt := range opts {
        opt(t)
    }
    return t, nil
}

// Name returns local.
func (t *Translator) Name() string { return "local" }

// CreatePreset stores a local profile for validation and reuse.
func (t *Translator) CreatePreset(_ context.Context, tmpl transcoder.TranscodeTemplate) error {
    if strings.TrimSpace(tmpl.Name) == "" {
        return fmt.Errorf("local: template name is required")
    }
    t.mu.Lock()
    t.templates[tmpl.Name] = tmpl
    t.mu.Unlock()
    return nil
}

// SubmitTask runs ffmpeg synchronously and records the completed task.
func (t *Translator) SubmitTask(ctx context.Context, req transcoder.TranscodeRequest) (string, error) {
    if req.SrcObject == "" || req.DstObject == "" {
        return "", fmt.Errorf("local: source and destination objects are required")
    }
    if err := t.CreatePreset(ctx, req.Template); err != nil {
        return "", err
    }
    source := t.resolvePath(req.SrcBucket, req.SrcObject)
    destination := t.resolvePath(req.DstBucket, req.DstObject)
    if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
        return "", fmt.Errorf("local: create destination directory: %w", err)
    }
    args := buildFFmpegArgsWithHardware(source, destination, req.Template, t.hardware)
    output, err := t.runner.Run(ctx, t.ffmpegPath, args...)
    id := "local-" + strconv.FormatUint(atomic.AddUint64(&t.sequence, 1), 10)
    result := &transcoder.TranscodeResult{TaskID: id, Raw: string(output)}
    if err != nil {
        result.State = transcoder.TaskFailed
        result.ErrMsg = strings.TrimSpace(string(output))
    } else {
        result.State = transcoder.TaskSuccess
        result.OutputURL = fileURL(destination)
    }
    t.mu.Lock()
    t.tasks[id] = result
    t.mu.Unlock()
    if err != nil {
        return id, fmt.Errorf("local: ffmpeg failed: %w", err)
    }
    return id, nil
}

// QueryTask returns the result recorded by the synchronous local task.
func (t *Translator) QueryTask(_ context.Context, taskID string) (*transcoder.TranscodeResult, error) {
    t.mu.RLock()
    result, ok := t.tasks[taskID]
    t.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("local: task %q not found", taskID)
    }
    copy := *result
    return &copy, nil
}

func localPath(bucket, object string) string {
    if bucket == "" {
        return object
    }
    return filepath.Join(bucket, object)
}

func (t *Translator) resolvePath(bucket, object string) string {
    path := localPath(bucket, object)
    if t.workDir != "" && !filepath.IsAbs(path) {
        return filepath.Join(t.workDir, path)
    }
    return path
}

func fileURL(path string) string {
    absolute, err := filepath.Abs(path)
    if err != nil {
        return "file://" + filepath.ToSlash(path)
    }
    return "file://" + filepath.ToSlash(absolute)
}

func buildFFmpegArgs(source, destination string, tmpl transcoder.TranscodeTemplate) []string {
    return buildFFmpegArgsWithHardware(source, destination, tmpl, hardwareConfig{})
}

type hardwareConfig struct {
    backend string
    device  string
}

func normalizeHardwareConfig(backend, device string) (hardwareConfig, error) {
    backend = strings.ToLower(strings.TrimSpace(backend))
    switch backend {
    case "", "none", "videotoolbox", "cuda", "nvenc", "vaapi", "qsv", "amf":
    default:
        return hardwareConfig{}, fmt.Errorf("local: unsupported hardware backend %q", backend)
    }
    return hardwareConfig{backend: backend, device: device}, nil
}

func buildFFmpegArgsWithHardware(source, destination string, tmpl transcoder.TranscodeTemplate, hardware hardwareConfig) []string {
    args := []string{"-y"}
    args = appendHardwareInputArgs(args, hardware)
    args = append(args, "-i", source)
    if tmpl.VideoCodec != "" {
        args = append(args, "-c:v", hardwareVideoCodec(tmpl.VideoCodec, hardware.backend))
        if (hardware.backend == "cuda" || hardware.backend == "nvenc") && hardware.device != "" {
            args = append(args, "-gpu", hardware.device)
        }
    }
    if tmpl.Bitrate > 0 {
        args = append(args, "-b:v", fmt.Sprintf("%dk", tmpl.Bitrate))
    }
    if tmpl.Fps > 0 {
        args = append(args, "-r", strconv.Itoa(tmpl.Fps))
    }
    if tmpl.ShortSide > 0 {
        args = append(args, "-vf", scaleFilter(fmt.Sprintf("%d:-2", tmpl.ShortSide), hardware.backend))
    } else if tmpl.Width > 0 || tmpl.Height > 0 {
        width, height := strconv.Itoa(tmpl.Width), strconv.Itoa(tmpl.Height)
        if tmpl.Width == 0 {
            width = "-1"
        }
        if tmpl.Height == 0 {
            height = "-1"
        }
        args = append(args, "-vf", scaleFilter(width+":"+height, hardware.backend))
    }
    if tmpl.AudioCodec != "" {
        args = append(args, "-c:a", tmpl.AudioCodec)
    }
    if tmpl.AudioBitrate > 0 {
        args = append(args, "-b:a", fmt.Sprintf("%dk", tmpl.AudioBitrate))
    }
    if tmpl.AudioSamplerate > 0 {
        args = append(args, "-ar", strconv.Itoa(tmpl.AudioSamplerate))
    }
    if tmpl.AudioChannels > 0 {
        args = append(args, "-ac", strconv.Itoa(tmpl.AudioChannels))
    }
    return append(args, destination)
}

func appendHardwareInputArgs(args []string, hardware hardwareConfig) []string {
    switch hardware.backend {
    case "videotoolbox":
        return append(args, "-hwaccel", "videotoolbox")
    case "cuda", "nvenc":
        args = append(args, "-hwaccel", "cuda", "-hwaccel_output_format", "cuda")
        return args
    case "vaapi":
        if hardware.device != "" {
            args = append(args, "-vaapi_device", hardware.device)
        }
        return append(args, "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi")
    case "qsv":
        return append(args, "-hwaccel", "qsv", "-hwaccel_output_format", "qsv")
    default:
        return args
    }
}

func hardwareVideoCodec(codec, backend string) string {
    if codec == "" {
        return codec
    }
    if backend == "" {
        return codec
    }
    if codec == "copy" {
        return codec
    }
    codec = strings.TrimPrefix(codec, "lib")
    switch codec {
    case "x264":
        codec = "h264"
    case "x265":
        codec = "hevc"
    }
    if strings.Contains(codec, "_") {
        return codec
    }
    switch backend {
    case "videotoolbox":
        return codec + "_videotoolbox"
    case "cuda", "nvenc":
        return codec + "_nvenc"
    case "vaapi":
        return codec + "_vaapi"
    case "qsv":
        return codec + "_qsv"
    case "amf":
        return codec + "_amf"
    default:
        return codec
    }
}

func scaleFilter(size, backend string) string {
    switch backend {
    case "cuda", "nvenc":
        return "scale_cuda=" + size
    case "vaapi":
        return "scale_vaapi=" + size
    default:
        return "scale=" + size
    }
}
