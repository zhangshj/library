# RateLimit — Architecture

## 设计目标

为登录类接口（如 `CheckLogin` / `Login`）提供**防爆破**能力，同时满足：

1. **框架无关**：不依赖 Fiber / Gin，可在任何 HTTP 层或纯逻辑层复用。
2. **可测试**：核心算法只依赖 `Store` 接口，用内存 fake 即可覆盖全部分支。
3. **多副本安全**：生产用 Redis 后端，计数原子，避免多 Pod 丢计数。

## 分层

```
pkg/ratelimit/
  interface.go     Store 接口 + Limiter 算法（Check/Incr/Reset/Set）
  config.go        Config 结构体 + 默认值常量（含 BanDuration）
  memory_store.go  Store 的内存实现（测试 / 降级）
  redis_store.go   Store 的 Redis 实现（原子 INCR+EXPIRE，Set 走原生 SET EX）
```

- **算法层**（`Limiter`）只认 `Store` 接口，不关心背后是 Redis 还是内存。
- **实现层**（`MemoryStore` / `RedisStore`）各自提供 `Store`，业务侧按需选择。
- 业务侧只需一行接线：`ratelimit.NewRedisStore(client)`，无需重写底层逻辑——
  这正是把它放进 library 而非散落在各业务方的意义。

## 为什么是固定窗口

登录爆破是"慢速、多账号"威胁，固定窗口（计数 + TTL）已足够：

- 实现简单、可测试、Redis 侧原子操作保证并发安全。
- 窗口边界的 2x 突发对登录场景可接受（爆破者不会卡点）。

若未来需要更平滑的限流（如令牌桶），在 `Limiter` 外新增一种算法实现 `Store`
即可，接口不变。

## 为什么 fail-open

`Check` / `Incr` 在 Store 出错时返回 `ErrStoreUnavailable`。调用方**默认放行**
（fail-open）：

- 登录不可用是 P0 事故，被爆破是 P1。
- 通过 `OnStoreErr` 钩子把故障告警出来，而不是让 Redis 抖动升级成全站登录故障。

> 注意：fail-open 意味着 Redis 挂掉期间限流失效。这是刻意的产品权衡，不是 bug。

## 只对失败计数（两段式）

限流不能是纯前置中间件——它执行时还不知道本次请求会不会失败。因此采用两段式：

```
请求 → Check(key)   判断可否继续（只读）
     → 业务（密码校验）
     → 失败? Incr(key)   仅失败消耗配额
     → 成功? Reset(key)  清零，体验更好
```

这避免了对正常成功请求无谓消耗配额，也避免"成功请求洗白爆破配额"的漏洞
（成功只清 username 维度，不清 IP 维度，防止攻击者用一次成功抹掉 IP 黑名单）。

## 封禁与计数器解耦

`Incr` 超出 `MaxHits` 时，会在 `ban:<key>` 上写入一个独立 TTL 的 key：

- 计数器 `key` 的 TTL 是 `Window`（自然过期）。
- 封禁 `ban:<key>` 的 TTL 是 `BanDuration`（与 `Window` 无关）。

这保证了：窗口到期后计数器清零，但封禁仍然生效，直到 `BanDuration` 结束。

`Store.Set` 可直接写入任意带 TTL 的 key，用于主动封禁（如管理员操作、异常检测触发），
不依赖计数器逻辑。

## 并发原子性

`RedisStore.Incr` 用 Lua 脚本在 Redis 服务端原子执行 `INCR` + 条件 `EXPIRE`
（仅首次创建时设 TTL），避免 `GET → +1 → SET` 在并发下丢更新。多副本 K8s
部署下计数准确，不会超发。

`RedisStore.Set` 直接使用 Redis 原生 `SET key value EX ttl` 命令，本身已是原子操作，
无需额外 Lua 脚本。
