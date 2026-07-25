# Translate Component

Unified machine translation interface backed by Alibaba Cloud and Tencent Cloud.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zhangshj/library/pkg/translate"
    "github.com/zhangshj/library/pkg/translate/aliyun"
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

    // Alibaba Cloud - batch translation (single API call, avoids rate limits)
    aliResults, err := aliTr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("aliyun batch translate failed:", err)
    } else {
        fmt.Println("aliyun batch:", aliResults)
    }

    // Tencent Cloud - single text translation
    tenTr, err := tencent.New(
        translate.DefaultConfig(),
        tencent.WithSecretKey("your-secret-id", "your-secret-key"),
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

    // Tencent Cloud - batch translation (concurrent with bounded parallelism)
    tenResults, err := tenTr.TranslateBatch(ctx, []string{"Hello", "World"}, "en", "zh")
    if err != nil {
        log.Println("tencent batch translate failed:", err)
    } else {
        fmt.Println("tencent batch:", tenResults)
    }
}
```

See [configuration.md](configuration.md) for all options and [architecture.md](architecture.md) for design rationale.
