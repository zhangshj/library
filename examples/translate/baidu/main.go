package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/translate/baidu"
)

func main() {
    ctx := context.Background()
    cfg := baidu.DefaultConfig()
    cfg.Mode = "auto"
    //cfg.APIKey = os.Getenv("BAIDU_APIKEY")
    cfg.LLMAuth = "sign"
    client, err := baidu.New(
        cfg,
        baidu.WithAppID(os.Getenv("BAIDU_APPID")),
        baidu.WithAPIKey(os.Getenv("BAIDU_APIKEY")),
        baidu.WithSecretKey(os.Getenv("BAIDU_SECRET_KEY")),
    )
    if err != nil {
        log.Fatalf("init baidu translator failed: %v", err)
    }

    text := "Hello, this is a test from Go."
    translated, err := client.Translate(ctx, text, "en", "zh")
    if err != nil {
        log.Printf("baidu translate error: %v", err)
        return
    }

    texts := []string{"Hello, this is a test from Go.", "How are you?"}
    translatedBatch, err := client.TranslateBatch(ctx, texts, "en", "zh")
    if err != nil {
        log.Printf("baidu batch translate error: %v", err)
        return
    }

    fmt.Printf("Batch Result:  %v\n", translatedBatch)
    fmt.Printf("Source:  %s\n", text)
    fmt.Printf("Result:  %s\n", translated)
}
