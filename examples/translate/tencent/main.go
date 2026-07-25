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
    )
    if err != nil {
        log.Fatalf("init tencent translator failed: %v", err)
    }

    text := "Hello, this is a test from Go."
    translated, err := client.Translate(ctx, text, "en", "zh")
    if err != nil {
        log.Printf("tencent translate error: %v", err)
        return
    }

    texts := []string{"Hello, this is a test from Go.", "How are you?"}
    translatedBatch, err := client.TranslateBatch(ctx, texts, "en", "zh")
    if err != nil {
        log.Printf("tencent batch translate error: %v", err)
        return
    }
    fmt.Printf("Source:  %s\n", text)
    fmt.Printf("Result:  %s\n", translated)
    fmt.Printf("Batch Result:  %v\n", translatedBatch)
}
