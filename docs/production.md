# 生产部署指南

> 本指南帮助你在生产环境中安全高效地部署 `track17` Go SDK。

---

## 推荐配置

```go
import (
    "log/slog"
    "os"
    "time"
    "github.com/imokyou/track17"
)

func newProductionClient() (*track17.Client, error) {
    // 使用 JSON 结构化日志，便于 ELK/Loki 等日志系统收集
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    return track17.New(
        os.Getenv("TRACK17_API_KEY"),

        // 1. 超时：建议 15~30s（17Track API 有时较慢）
        track17.WithTimeout(20*time.Second),

        // 2. 自动重试（指数退避 + Jitter，仅重试 5xx / 429）
        track17.WithRetry(3, time.Second),

        // 3. 熔断器：5 次连续失败后开路，30s 后探测恢复
        track17.WithCircuitBreaker(5, 30*time.Second),

        // 4. 速率限制：根据你的 17Track 套餐设置
        //   - 免费版: 3 req/s
        //   - 标准版: 10 req/s
        //   - 企业版: 50 req/s
        track17.WithRateLimit(3),

        // 5. 结构化日志（生产环境不建议开启 debug）
        track17.WithSlogLogger(logger),
    )
}
```

---

## 环境变量配置

| 变量名 | 说明 | 示例值 |
|--------|------|-------|
| `TRACK17_API_KEY` | 17Track API 密钥（**必填**） | `sk-xxxxxxxxxxxxxxxx` |

> [!CAUTION]
> **绝对不要将 API Key 硬编码到代码中**，始终通过环境变量或密钥管理系统（如 Vault、AWS Secrets Manager）注入。

---

## 熔断器参数调优

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| `maxFailures` | 3~10 | 低流量服务建议 3，高流量服务建议 10 |
| `resetTimeout` | 15s~60s | API 常见故障恢复时间约 10~30s，建议 30s |

```go
// 高流量生产环境（>100 QPS）
track17.WithCircuitBreaker(10, 30*time.Second)

// 低流量环境或对延迟敏感
track17.WithCircuitBreaker(3, 15*time.Second)
```

---

## Webhook 生产部署

```go
handler := track17.WebhookHandler(apiKey, func(event track17.WebhookEvent) {
    // 建议异步处理，避免阻塞 HTTP 响应
    go processEvent(event)
})

// 使用 TLS（生产必须）
srv := &http.Server{
    Addr:         ":443",
    Handler:      mux,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}
```

> [!IMPORTANT]
> 防重放保护说明：SDK 内置时间戳校验，超过 **±5 分钟** 的 Webhook 事件会被自动拒绝。请确保服务器时钟与 NTP 同步。

---

## 可观测性

### 错误监控

```go
resp, err := client.Tracking.Register(ctx, items)
if err != nil {
    if apiErr, ok := track17.IsAPIError(err); ok {
        // 上报到监控系统（Prometheus、Datadog 等）
        metrics.Counter("track17.api_error", map[string]string{
            "code": strconv.Itoa(apiErr.Code),
        }).Inc()
    }
    if _, ok := err.(*track17.ErrCircuitOpen); ok {
        // 熔断器开路告警
        alerts.Send("track17 circuit breaker is OPEN")
    }
}
```

### 关键指标

| 指标 | 告警阈值 | 说明 |
|------|---------|------|
| API 成功率 | < 95% | 触发 P2 告警 |
| 平均响应时间 | > 5s | 触发 P3 告警 |
| 余额剩余 | < 1000 | 提前充值告警 |
| 熔断器状态 | Open | 立即 P1 告警 |

---

## 健康检查

```go
func healthCheck(client *track17.Client) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _, err := client.Query.GetQuota(ctx)
    return err
}
```

---

## 故障排查清单

| 错误码 | 原因 | 解决方案 |
|--------|------|---------|
| `-18010003` | API Key 无效 | 检查 `TRACK17_API_KEY` 环境变量 |
| `-18010004` | IP 不在白名单 | 登录 17Track 控制台添加白名单 IP |
| `-18010005` | 频率超限 | 降低 `WithRateLimit()` 值或升级套餐 |
| `-18010006` | 余额不足 | 立即充值，同时设置余额告警 |
| `ErrCircuitOpen` | 上游持续故障 | 等待熔断器自动恢复 (`resetTimeout`) |
