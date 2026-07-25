# Translate Component

Unified machine translation interface backed by Alibaba Cloud, Baidu Translate, and Tencent Cloud TokenHub.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/aliyun"
    "github.com/zhangshj/library/pkg/translate/baidu"
    "github.com/zhangshj/library/pkg/translate/tencent"
)

func main() {
    ctx := context.Background()

    // Alibaba Cloud - single text translation
    aliTr, err := aliyun.New(
        translate.DefaultConfig(),
        aliyun.WithAccessKey("your-access-key-id", "your-access-key-secret"),
    )
    if err != nil {
        log.Fatal(err)
    }
    aliResult, err := aliTr.Translate(ctx, "Hello World", "en", "zh")
    if err != nil {
        log.Println("aliyun translate failed:", err)
    } else {
        fmt.Println("aliyun:", aliResult)
    }

    // Baidu Translate - General mode, batch translation with newline-joined single API call
    baiduTr, err := baidu.New(
        translate.DefaultConfig(),
        baidu.WithAppID("your-appid"),
        baidu.WithAPIKey("your-api-key"),
        baidu.WithSecretKey("your-secret-key"),
    )
    if err != nil {
        log.Fatal(err)
    }
    baiduResults, err := baiduTr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("baidu batch translate failed:", err)
    } else {
        fmt.Println("baidu batch:", baiduResults)
    }

    // Tencent Cloud TokenHub - single text translation
    tenTr, err := tencent.New(
        translate.DefaultConfig(),
        tencent.WithAPIKey(os.Getenv("TOKENHUB_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }
    tenResult, err := tenTr.Translate(ctx, "Hello World", "en", "zh")
    if err != nil {
        log.Println("tencent translate failed:", err)
    } else {
        fmt.Println("tencent:", tenResult)
    }
}
```

See [configuration.md](configuration.md) for all options and [architecture.md](architecture.md) for design rationale.
