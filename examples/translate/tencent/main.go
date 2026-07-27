package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/translate/tencent"
)

func main() {
    ctx := context.Background()

    client, err := tencent.New(
        tencent.DefaultConfig(),
        tencent.WithAPIKey(os.Getenv("TOKENHUB_API_KEY")),
        tencent.WithDomain("数据标注，智能驾驶，机器人"),
    )
    if err != nil {
        log.Fatalf("init tencent translator failed: %v", err)
    }

    text := "是"
    translated, err := client.Translate(ctx, text, "zh", "en")
    if err != nil {
        log.Printf("tencent translate error: %v", err)
        return
    }

    texts := []string{"是", "12345", "你好"}
    translatedBatch, err := client.TranslateBatch(ctx, texts, "zh", "en")
    if err != nil {
        log.Printf("tencent batch translate error: %v", err)
        return
    }
    fmt.Printf("Source:  %s\n", text)
    fmt.Printf("Result:  %s\n", translated)
    fmt.Printf("Batch Result:  %v\n", translatedBatch)
}
