# RateLimit — Configuration

All configuration lives in `pkg/ratelimit/config.go` as `ratelimit.Config`.
The component holds **no global config** and **no hardcoded secrets** (per
internal spec 2.1 / 3.2).

## `ratelimit.Config`

| 字段 | 类型 | 必填 | 默认值 | 说明 / 安全建议 |
| --- | --- | --- | --- | --- |
| `Window` | `time.Duration` | 否 | `1m` (`DefaultWindow`) | 固定窗口长度。窗口边界存在 2x 突发，登录爆破场景可接受；如需更平滑改用滑动窗口（后续扩展）。 |
| `MaxHits` | `int64` | 否 | `5` (`DefaultMaxHits`) | 每个窗口内的允许次数。**只对失败计数的场景**，建议按"正常用户误输次数"设定，过小会误伤真实用户。 |
| `OnStoreErr` | `func(error)` | 否 | `nil` | Store 返回错误时回调，用于打点/告警。**不要在此阻塞请求链路**。留空则静默 fail-open。 |

## `ratelimit.Limiter` 方法行为

| 方法 | 计入配额? | Store 出错时 | 用途 |
| --- | --- | --- | --- |
| `Check(ctx, key)` | 否（只读） | 返回 `ErrStoreUnavailable`，调用方应 fail-open | 执行业务**前**判断是否已超限 |
| `Incr(ctx, key)` | 是（失败时才调用） | 返回 `ErrStoreUnavailable`，调用方应 fail-open | 认证失败**后**计数 |
| `Reset(ctx, key)` | 否（清零） | 返回 `ErrStoreUnavailable` | 认证成功后清空该 key 计数 |

## 默认值常量（`config.go`）

| 常量 | 值 | 说明 |
| --- | --- | --- |
| `DefaultWindow` | `time.Minute` | `Window` 为 0 时采用 |
| `DefaultMaxHits` | `5` | `MaxHits` 为 0 或负数时采用 |

## Store 实现选择

| 实现 | 构造 | 适用 | 注意事项 |
| --- | --- | --- | --- |
| `MemoryStore` | `NewMemoryStore(now func() time.Time)` | 单元测试、无 Redis 降级 | 进程内计数，**多副本不共享**，仅适合单实例或降级 |
| `RedisStore` | `NewRedisStore(client redis.UniversalClient)` | 生产、多副本共享 | Lua 原子 `INCR`+`EXPIRE`，避免并发丢计数 |

## 多维度限流（如 IP + username）

组件本身只认 `key` 字符串。维度组合由调用方在 `key` 上拼接，例如：

```
ipKey      := "ratelimit:ip:"      + realIP
userKey    := "ratelimit:user:"    + username
captchaKey := "ratelimit:captcha:" + captchaKeyID
```

为每个维度创建独立的 `Limiter`（可不同 `MaxHits`），分别 `Check`/`Incr`。
