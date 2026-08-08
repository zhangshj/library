// Command ratelimit-demo demonstrates the full lifecycle of the ratelimit
// component: build config -> initialize store + limiter -> normal calls
// (Check / Incr / Reset) -> ban behavior (exceeding limit triggers a ban
// with independent BanDuration) -> error handling (fail-open on store error).
//
// Run with: go run ./examples/ratelimit
//
// This demo uses the in-memory store so it runs with zero external
// dependencies. To use Redis in production, replace NewMemoryStore with
// ratelimit.NewRedisStore(client) where client is a redis.UniversalClient
// (e.g. the one returned by gofiber's storage.Conn()).
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

	// 1) Read config (in real code this comes from your YAML/env loader).
	//    BanDuration is how long a key stays banned AFTER the limit is hit.
	//    When zero it falls back to Window, preserving the old "window == ban"
	//    behaviour.
	cfg := ratelimit.Config{
		Window:      5 * time.Minute,
		MaxHits:     3,
		BanDuration: 10 * time.Minute,
		OnStoreErr: func(err error) {
			// 2) Error hook: alert, but never block the request path.
			log.Printf("[alert] ratelimit store error (fail-open): %v", err)
		},
	}

	// 3) Initialize store + limiter.
	//    Production: store := ratelimit.NewRedisStore(redisClient)
	store := ratelimit.NewMemoryStore(nil)
	limiter := ratelimit.New(store, cfg)

	key := "login:user:alice"

	// 4) Normal call path: simulate failures up to MaxHits.
	for attempt := 1; attempt <= 5; attempt++ {
		allowed, err := limiter.Check(ctx, key)
		if err == ratelimit.ErrStoreUnavailable {
			// fail-open: let the request through but the alert hook fired.
			allowed = true
		}
		if !allowed {
			fmt.Printf("attempt %d: BLOCKED (already banned or over limit)\n", attempt)
			continue
		}

		// Pretend the password check happens here.
		passwordOK := false
		if !passwordOK {
			allowed, err = limiter.Incr(ctx, key)
			if err == ratelimit.ErrStoreUnavailable {
				// fail-open on incr error
				log.Printf("incr failed, ignoring: %v", err)
				allowed = true
			}
			fmt.Printf("attempt %d: bad password, counted (allowed=%v)\n", attempt, allowed)
			continue
		}

		// Success: reset the failure counter for this user.
		if resetErr := limiter.Reset(ctx, key); resetErr != nil {
			log.Printf("reset failed: %v", resetErr)
		}
		fmt.Printf("attempt %d: LOGIN OK, counter reset\n", attempt)
	}

	// 5) After exceeding MaxHits the key is banned for BanDuration.
	//    Even a correct password will stay blocked until the ban expires.
	fmt.Println("--- now in ban period ---")
	allowed, _ := limiter.Check(ctx, key)
	fmt.Printf("banned, allowed=%v (expect false)\n", allowed)

	// 6) Direct Set ban (optional): the Store.Set method can be used to start
	//    a ban independent of Incr, e.g. after an admin action or anomaly
	//    detection. Here we clear any existing state first to demonstrate it.
	if resetErr := limiter.Reset(ctx, key); resetErr != nil {
		log.Printf("reset failed: %v", resetErr)
	}
	if banErr := store.Set(ctx, "login:user:alice", 2*time.Minute); banErr != nil {
		log.Printf("set ban failed: %v", banErr)
	}
	allowed, _ = limiter.Check(ctx, key)
	fmt.Printf("after direct ban set, allowed=%v (expect false)\n", allowed)
}
