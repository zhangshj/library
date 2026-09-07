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
        tencent.WithModels("hy-mt2-pro", "hy-mt2-plus", "hy-mt2-lite"),
        tencent.WithModel("auto"),
        tencent.WithModelObserver(func(model string) {
            fmt.Printf("Tencent model: %s\n", model)
        }),
    )
    if err != nil {
        log.Fatalf("init tencent translator failed: %v", err)
    }

    text := "顺利"
    translated, err := client.Translate(ctx, text, "zh", "en")
    if err != nil {
        log.Printf("tencent translate error: %v", err)
        return
    }

    texts := []string{"寻找", "拿起", "寻找", "...放到...", "手收回", "浅蓝瓶子"}
    translatedBatch, err := client.TranslateBatch(ctx, texts, "zh", "en")
    if err != nil {
        log.Printf("tencent batch translate error: %v", err)
        return
    }
    fmt.Printf("Source:  %s\n", text)
    fmt.Printf("Result:  %s\n", translated)
    for i, t := range texts {
        fmt.Printf("Source:  %s\n", t)
        fmt.Printf("Result:  %s\n", translatedBatch[i])
    }
    //fmt.Printf("Batch Result:  %+v\n", translatedBatch)
}
