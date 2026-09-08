// Package transcoder provides a unified interface for media transcoding.
package transcoder

import "context"

// TranscodeTemplate describes a provider-neutral transcoding profile.
type TranscodeTemplate struct {
	Name            string
	Container       string
	VideoCodec      string
	Width           int
	Height          int
	ShortSide       int
	Bitrate         int
	Fps             int
	AudioCodec      string
	AudioBitrate    int
	AudioSamplerate int
	AudioChannels   int
}

// TranscodeRequest describes a source, destination, and profile for a task.
type TranscodeRequest struct {
	SrcBucket string
	SrcObject string
	SrcRegion string

	DstBucket string
	DstObject string
	DstRegion string

	Template    TranscodeTemplate
	CallbackURL string
}

// TaskState is the normalized state of a transcoding task.
type TaskState string

const (
	TaskWaiting TaskState = "Waiting"
	TaskRunning TaskState = "Running"
	TaskSuccess TaskState = "Success"
	TaskFailed  TaskState = "Failed"
)

// TranscodeResult is the normalized result of a transcoding task.
type TranscodeResult struct {
	TaskID    string
	State     TaskState
	OutputURL string
	ErrMsg    string
	Raw       interface{}
}

// Transcoder is the common contract implemented by local and cloud drivers.
type Transcoder interface {
	// CreatePreset creates or verifies a provider-side template.
	CreatePreset(ctx context.Context, tmpl TranscodeTemplate) error
	// SubmitTask submits a transcoding task and returns its provider task ID.
	SubmitTask(ctx context.Context, req TranscodeRequest) (string, error)
	// QueryTask returns the normalized state of a submitted task.
	QueryTask(ctx context.Context, taskID string) (*TranscodeResult, error)
	// Name returns the driver name.
	Name() string
}
