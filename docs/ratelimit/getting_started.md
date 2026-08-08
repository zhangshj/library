# RateLimit Component

Framework-agnostic, fixed-window rate limiter with pluggable storage. The core
algorithm depends only on the `Store` interface, so it is testable with an
in-memory fake and production-ready with the bundled Redis backend
(atomic `INCR` + `EXPIRE` via Lua, `Set` via native `SET ... EX`).

Designed for login/CheckLogin brute-force protection: **count only failures**,
**fail-open on store errors** (login availability > brute-force protection).

## Quick Start (in-memory, no Redis)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zhangshj/library/pkg/ratelimit"
)

func main() {
	ctx := context.Background()

	// In-memory store: tests, or fallback when no Redis is configured.
	store := ratelimit.NewMemoryStore(nil)
	// 3 failures allowed per 5-minute window; after exceeding, the key is
	// banned for 10 minutes (BanDuration). On store error, call the alert hook.
	limiter := ratelimit.New(store, ratelimit.Config{
		Window:      5 * time.Minute,
		MaxHits:     3,
		BanDuration: 10 * time.Minute,
		OnStoreErr: func(err error) {
			log.Printf("ratelimit store error (fail-open): %v", err)
		},
	})

	key := "login:user:alice"

	// Before a login attempt: is this key still allowed to try?
	allowed, err := limiter.Check(ctx, key)
	if err == ratelimit.ErrStoreUnavailable {
		// fail-open: let the request through, but alert (handled above).
		allowed = true
	}
	if !allowed {
		fmt.Println("too many attempts, slow down")
		return
	}

	// ... run password check ...
	passwordOK := false

	if !passwordOK {
		// Only failures consume the quota.
		allowed, err = limiter.Incr(ctx, key)
		if err == ratelimit.ErrStoreUnavailable {
			// fail-open on incr error
			log.Printf("incr failed (fail-open): %v", err)
			allowed = true
		}
		if !allowed {
			fmt.Println("limit exceeded, now banned")
		} else {
			fmt.Println("bad password, attempt counted")
		}
		return
	}

	// Success: clear the failure counter for this user.
	if err := limiter.Reset(ctx, key); err != nil {
		log.Printf("reset failed: %v", err)
	}
	fmt.Println("login ok")
}
```

## Quick Start (Redis, production)

```go
import (
	"github.com/redis/go-redis/v9"
	"github.com/zhangshj/library/pkg/ratelimit"
)

// client can be any redis.UniversalClient, e.g. the one returned by
// gofiber's storage.Conn(), or your own *redis.Client.
var client redis.UniversalClient

store := ratelimit.NewRedisStore(client)
limiter := ratelimit.New(store, ratelimit.Config{
	Window:      5 * time.Minute,
	MaxHits:     3,
	BanDuration: 10 * time.Minute,
})
// same Check/Incr/Reset API as above
```

## Direct Ban via Store.Set

`Store.Set` writes a key with a TTL directly, independent of the counter.
Use it for admin-initiated bans or anomaly detection:

```go
// Ban a user for 2 hours without touching the failure counter.
if err := store.Set(ctx, "login:user:alice", 2*time.Hour); err != nil {
	log.Printf("set ban failed: %v", err)
}
```

See [configuration.md](configuration.md) for all options and
[architecture.md](architecture.md) for the design rationale (why fixed-window,
why fail-open, how ban duration decouples from window).
