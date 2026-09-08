// Package tencent implements transcoding with Tencent Cloud CI.
package tencent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/tencentyun/cos-go-sdk-v5"

	"github.com/zhangshj/library/pkg/transcoder"
)

// Config configures Tencent Cloud CI.
type Config struct {
	BucketURL string
	SecretID  string
	SecretKey string
}

// Translator is a Tencent Cloud CI transcoder.
type Translator struct {
	client    *cos.Client
	bucketURL string
}

// New creates a Tencent Cloud CI transcoder.
func New(cfg Config) (*Translator, error) {
	if cfg.BucketURL == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("tencent: bucket URL and credentials are required")
	}
	u, err := url.Parse(cfg.BucketURL)
	if err != nil {
		return nil, fmt.Errorf("tencent: parse bucket URL: %w", err)
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{SecretID: cfg.SecretID, SecretKey: cfg.SecretKey},
	})
	return &Translator{client: client, bucketURL: cfg.BucketURL}, nil
}

// Name returns tencent.
func (t *Translator) Name() string { return "tencent" }

// CreatePreset is a no-op because CI receives the profile with each job.
func (t *Translator) CreatePreset(_ context.Context, _ transcoder.TranscodeTemplate) error {
	return nil
}

// SubmitTask submits a Tencent CI media processing job.
func (t *Translator) SubmitTask(ctx context.Context, req transcoder.TranscodeRequest) (string, error) {
	if req.SrcObject == "" || req.DstBucket == "" || req.DstObject == "" {
		return "", fmt.Errorf("tencent: source object, destination bucket and object are required")
	}
	options := &cos.CreateJobsOptions{
		Tag:   "Transcode",
		Input: &cos.JobInput{Object: req.SrcObject},
		Operation: &cos.MediaProcessJobOperation{
			Output:    &cos.JobOutput{Region: req.DstRegion, Bucket: req.DstBucket, Object: req.DstObject},
			Transcode: buildTranscode(req.Template),
		},
	}
	if req.CallbackURL != "" {
		options.CallBack = req.CallbackURL
	}
	response, _, err := t.client.CI.CreateJob(ctx, options)
	if err != nil {
		return "", fmt.Errorf("tencent: create CI job: %w", err)
	}
	if response == nil || response.JobsDetail == nil || response.JobsDetail.JobId == "" {
		return "", fmt.Errorf("tencent: create CI job returned empty task id")
	}
	return response.JobsDetail.JobId, nil
}

// QueryTask queries a Tencent CI media processing job.
func (t *Translator) QueryTask(ctx context.Context, taskID string) (*transcoder.TranscodeResult, error) {
	if taskID == "" {
		return nil, fmt.Errorf("tencent: task id is required")
	}
	response, _, err := t.client.CI.DescribeMultiMediaJob(ctx, []string{taskID})
	if err != nil {
		return nil, fmt.Errorf("tencent: query CI job: %w", err)
	}
	result := &transcoder.TranscodeResult{TaskID: taskID, Raw: response}
	if response == nil || len(response.JobsDetail) == 0 {
		result.State = transcoder.TaskWaiting
		return result, nil
	}
	detail := response.JobsDetail[0]
	switch detail.State {
	case "Success":
		result.State = transcoder.TaskSuccess
		if detail.Operation != nil && detail.Operation.Output != nil {
			output := detail.Operation.Output
			result.OutputURL = fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", output.Bucket, output.Region, output.Object)
		}
	case "Fail", "Failed":
		result.State = transcoder.TaskFailed
		result.ErrMsg = detail.Message
	default:
		result.State = transcoder.TaskRunning
	}
	return result, nil
}

func buildTranscode(tmpl transcoder.TranscodeTemplate) *cos.Transcode {
	video := &cos.Video{Codec: tmpl.VideoCodec}
	if tmpl.Bitrate > 0 {
		video.Bitrate = fmt.Sprintf("%d", tmpl.Bitrate)
	}
	if tmpl.Fps > 0 {
		video.Fps = fmt.Sprintf("%d", tmpl.Fps)
	}
	if tmpl.ShortSide > 0 {
		video.Width = fmt.Sprintf("%d", tmpl.ShortSide)
		video.Height = "-2"
	} else {
		if tmpl.Width > 0 {
			video.Width = fmt.Sprintf("%d", tmpl.Width)
		}
		if tmpl.Height > 0 {
			video.Height = fmt.Sprintf("%d", tmpl.Height)
		}
	}
	return &cos.Transcode{
		Container: &cos.Container{Format: tmpl.Container},
		Video:     video,
		Audio: &cos.Audio{
			Codec: tmpl.AudioCodec, Bitrate: fmt.Sprintf("%d", tmpl.AudioBitrate),
			Samplerate: fmt.Sprintf("%d", tmpl.AudioSamplerate), Channels: fmt.Sprintf("%d", tmpl.AudioChannels),
		},
	}
}
