package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/transcoder"
    "github.com/zhangshj/library/pkg/transcoder/factory"
)

func main() {
    input := os.Getenv("TRANSCODER_INPUT")
    output := os.Getenv("TRANSCODER_OUTPUT")
    if input == "" || output == "" {
        log.Fatal("set TRANSCODER_INPUT and TRANSCODER_OUTPUT")
    }

    t, err := factory.New(transcoder.Config{
        Provider:        transcoder.ProviderLocal,
        FFmpeg:          "ffmpeg",
        HardwareBackend: os.Getenv("TRANSCODER_HW"),
        HardwareDevice:  os.Getenv("TRANSCODER_HW_DEVICE"),
    })
    if err != nil {
        log.Fatal(err)
    }

    taskID, err := t.SubmitTask(context.Background(), transcoder.TranscodeRequest{
        SrcObject: input,
        DstObject: output,
        Template: transcoder.TranscodeTemplate{
            Name:         "h264-aac",
            Container:    "mp4",
            VideoCodec:   "libx264",
            AudioCodec:   "aac",
            AudioBitrate: 128,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    result, err := t.QueryTask(context.Background(), taskID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("task=%s state=%s output=%s\n", result.TaskID, result.State, result.OutputURL)
}
