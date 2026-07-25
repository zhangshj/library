package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/aliyun"
)

func main() {
    ctx := context.Background()

    cfg := translate.DefaultConfig()
    cfg.Region = "cn-beijing"

    client, err := aliyun.New(
        aliyun.DefaultConfig(),
        aliyun.WithAccessKey(
            os.Getenv("ALIYUN_ACCESS_KEY_ID"),
            os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
        ),
    )
    if err != nil {
        log.Fatalf("init aliyun translator failed: %v", err)
    }

    text := "Hello, this is a test from Go."
    translated, err := client.Translate(ctx, text, "en", "zh")
    if err != nil {
        log.Printf("aliyun translate error: %v", err)
        return
    }

    fmt.Printf("Source:  %s\n", text)
    fmt.Printf("Result:  %s\n", translated)
}
