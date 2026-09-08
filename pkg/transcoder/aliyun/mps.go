// Package aliyun implements transcoding with Alibaba Cloud MTS.
package aliyun

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "sync"

    "github.com/aliyun/alibaba-cloud-sdk-go/services/mts"

    "github.com/zhangshj/library/pkg/transcoder"
)

// Config contains Alibaba Cloud MTS credentials and region.
type Config struct {
    AccessKeyID     string
    AccessKeySecret string
    Region          string
    RoleARN         string
}

// Transcoder is the Alibaba Cloud MTS transcoder.
type Transcoder struct {
    client    *mts.Client
    region    string
    roleARN   string
    mu        sync.RWMutex
    templates map[string]string
}

// New creates an Alibaba Cloud MTS transcoder.
func New(cfg Config) (*Transcoder, error) {
    if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
        return nil, fmt.Errorf("aliyun: access key pair is required")
    }
    if cfg.Region == "" {
        return nil, fmt.Errorf("aliyun: region is required")
    }
    var client *mts.Client
    var err error
    if cfg.RoleARN != "" {
        client, err = mts.NewClientWithRamRoleArn(cfg.Region, cfg.AccessKeyID, cfg.AccessKeySecret, cfg.RoleARN, "library-transcoder")
    } else {
        client, err = mts.NewClientWithAccessKey(cfg.Region, cfg.AccessKeyID, cfg.AccessKeySecret)
    }
    if err != nil {
        return nil, fmt.Errorf("aliyun: create MTS client: %w", err)
    }
    return &Transcoder{client: client, region: cfg.Region, roleARN: cfg.RoleARN, templates: make(map[string]string)}, nil
}

// Name returns aliyun.
func (t *Transcoder) Name() string { return "aliyun" }

// CreatePreset creates an MTS template and remembers its provider template ID.
func (t *Transcoder) CreatePreset(ctx context.Context, tmpl transcoder.TranscodeTemplate) error {
    if err := contextErr(ctx); err != nil {
        return err
    }
    if strings.TrimSpace(tmpl.Name) == "" {
        return fmt.Errorf("aliyun: template name is required")
    }
    t.mu.RLock()
    _, ok := t.templates[tmpl.Name]
    t.mu.RUnlock()
    if ok {
        return nil
    }

    req := mts.CreateAddTemplateRequest()
    req.Name = tmpl.Name
    req.Container = jsonString(map[string]string{"Format": tmpl.Container})
    req.Video = jsonString(videoConfig(tmpl))
    req.Audio = jsonString(audioConfig(tmpl))
    resp, err := t.client.AddTemplate(req)
    if err != nil {
        return fmt.Errorf("aliyun: add MTS template: %w", err)
    }
    if !resp.IsSuccess() {
        return fmt.Errorf("aliyun: add MTS template failed")
    }
    id := resp.Template.TemplateId
    if id == "" {
        id = resp.Template.Id
    }
    if id == "" {
        return fmt.Errorf("aliyun: add MTS template returned empty template id")
    }
    t.mu.Lock()
    t.templates[tmpl.Name] = id
    t.mu.Unlock()
    return nil
}

// SubmitTask submits an OSS-to-OSS MTS job.
func (t *Transcoder) SubmitTask(ctx context.Context, req transcoder.TranscodeRequest) (string, error) {
    if err := contextErr(ctx); err != nil {
        return "", err
    }
    if req.SrcBucket == "" || req.SrcObject == "" || req.DstBucket == "" || req.DstObject == "" {
        return "", fmt.Errorf("aliyun: source and destination bucket/object are required")
    }
    if err := t.CreatePreset(ctx, req.Template); err != nil {
        return "", err
    }
    t.mu.RLock()
    templateID := t.templates[req.Template.Name]
    t.mu.RUnlock()
    input, err := json.Marshal(map[string]string{
        "Location": ossLocation(firstNonEmpty(req.SrcRegion, t.region)),
        "Bucket":   req.SrcBucket,
        "Object":   req.SrcObject,
    })
    if err != nil {
        return "", fmt.Errorf("aliyun: marshal MTS input: %w", err)
    }
    outputs, err := json.Marshal([]map[string]string{{"OutputObject": req.DstObject, "TemplateId": templateID}})
    if err != nil {
        return "", fmt.Errorf("aliyun: marshal MTS output: %w", err)
    }
    rpc := mts.CreateSubmitJobsRequest()
    rpc.Input = string(input)
    rpc.Outputs = string(outputs)
    rpc.OutputBucket = "oss://" + req.DstBucket
    resp, err := t.client.SubmitJobs(rpc)
    if err != nil {
        return "", fmt.Errorf("aliyun: submit MTS job: %w", err)
    }
    if len(resp.JobResultList.JobResult) == 0 {
        return "", fmt.Errorf("aliyun: submit MTS job returned no job")
    }
    item := resp.JobResultList.JobResult[0]
    if !item.Success || item.Job.JobId == "" {
        return "", fmt.Errorf("aliyun: submit MTS job failed: code=%s message=%s", item.Code, item.Message)
    }
    return item.Job.JobId, nil
}

// QueryTask queries an MTS job and normalizes its state.
func (t *Transcoder) QueryTask(ctx context.Context, taskID string) (*transcoder.TranscodeResult, error) {
    if err := contextErr(ctx); err != nil {
        return nil, err
    }
    if taskID == "" {
        return nil, fmt.Errorf("aliyun: task id is required")
    }
    req := mts.CreateQueryJobListRequest()
    req.JobIds = taskID
    resp, err := t.client.QueryJobList(req)
    if err != nil {
        return nil, fmt.Errorf("aliyun: query MTS job: %w", err)
    }
    if len(resp.JobList.Job) == 0 {
        return &transcoder.TranscodeResult{TaskID: taskID, State: transcoder.TaskWaiting}, nil
    }
    job := resp.JobList.Job[0]
    result := &transcoder.TranscodeResult{TaskID: taskID, Raw: job}
    switch strings.ToLower(job.State) {
    case "success":
        result.State = transcoder.TaskSuccess
        result.OutputURL = ossURL(job.Output.OutputFile.Bucket, job.Output.OutputFile.Object, job.Output.OutputFile.Location)
    case "fail", "failed":
        result.State = transcoder.TaskFailed
        result.ErrMsg = job.Message
    default:
        result.State = transcoder.TaskRunning
    }
    return result, nil
}

func videoConfig(tmpl transcoder.TranscodeTemplate) map[string]string {
    cfg := map[string]string{"Codec": tmpl.VideoCodec}
    if tmpl.Bitrate > 0 {
        cfg["Bitrate"] = fmt.Sprintf("%d", tmpl.Bitrate*1000)
    }
    if tmpl.Fps > 0 {
        cfg["Fps"] = fmt.Sprintf("%d", tmpl.Fps)
    }
    if tmpl.Width > 0 {
        cfg["Width"] = fmt.Sprintf("%d", tmpl.Width)
    }
    if tmpl.Height > 0 {
        cfg["Height"] = fmt.Sprintf("%d", tmpl.Height)
    }
    return cfg
}

func audioConfig(tmpl transcoder.TranscodeTemplate) map[string]string {
    cfg := map[string]string{"Codec": tmpl.AudioCodec}
    if tmpl.AudioBitrate > 0 {
        cfg["Bitrate"] = fmt.Sprintf("%d", tmpl.AudioBitrate*1000)
    }
    if tmpl.AudioSamplerate > 0 {
        cfg["Samplerate"] = fmt.Sprintf("%d", tmpl.AudioSamplerate)
    }
    if tmpl.AudioChannels > 0 {
        cfg["Channels"] = fmt.Sprintf("%d", tmpl.AudioChannels)
    }
    return cfg
}

func jsonString(value interface{}) string {
    data, _ := json.Marshal(value)
    return string(data)
}

func contextErr(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        return nil
    }
}

func firstNonEmpty(values ...string) string {
    for _, value := range values {
        if value != "" {
            return value
        }
    }
    return "cn-hangzhou"
}

func ossLocation(region string) string { return "oss-" + region }

func ossURL(bucket, object, location string) string {
    if bucket == "" || object == "" {
        return ""
    }
    region := strings.TrimPrefix(location, "oss-")
    return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", bucket, region, object)
}
