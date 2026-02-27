# track17

[![Go Reference](https://pkg.go.dev/badge/github.com/imokyou/track17.svg)](https://pkg.go.dev/github.com/imokyou/track17)

> 🚀 17Track API v2.4 企业级 Go SDK —— 零依赖、类型安全、生产就绪

## ✨ 特性

- **零外部依赖** — 仅使用 Go 标准库
- **完整 API 覆盖** — 支持全部 11 个 17Track API 端点
- **类型安全** — 所有请求/响应均有完整的类型定义
- **内置限流** — 默认 3 req/s，避免触发 API 限制
- **自动重试** — 指数退避重试策略，支持 5xx 和 429 错误
- **WebHook 支持** — SHA256 签名验证 + 便捷 HTTP Handler
- **Context 支持** — 所有 API 方法支持 `context.Context`
- **并发安全** — 客户端可安全地被多个 goroutine 共享

## 📦 安装

```bash
go get github.com/imokyou/track17
```

## 🚀 快速开始

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/imokyou/track17"
)

func main() {
    client := track17.New("your-api-key")
    ctx := context.Background()

    // 注册单号
    resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
        {Number: "RR123456789CN", CarrierCode: 3011, Lang: "en"},
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, item := range resp.Accepted {
        fmt.Printf("✓ %s registered (carrier: %d)\n", item.Number, item.Carrier)
    }

    // 查询轨迹
    info, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
        {Number: "RR123456789CN", CarrierCode: 3011},
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, t := range info.Accepted {
        fmt.Printf("📦 %s: status=%d\n", t.Number, t.Track.LatestStatus)
    }
}
```

## 📖 API 参考

### 初始化客户端

```go
// 基础用法
client := track17.New("your-api-key")

// 高级配置
client := track17.New("your-api-key",
    track17.WithTimeout(10*time.Second),     // 请求超时
    track17.WithRetry(3, time.Second),       // 重试策略
    track17.WithDebug(true),                 // 调试日志
    track17.WithHTTPClient(customClient),    // 自定义 HTTP 客户端
)
```

### Tracking 服务 — 物流单号管理

```go
// 注册单号（每次最多 40 个）
resp, _ := client.Tracking.Register(ctx, []track17.RegisterRequest{...})

// 修改运输商（每个单号限 5 次）
resp, _ := client.Tracking.ChangeCarrier(ctx, []track17.ChangeCarrierRequest{...})

// 修改附加信息
resp, _ := client.Tracking.ChangeInfo(ctx, []track17.ChangeInfoRequest{...})

// 停止跟踪
resp, _ := client.Tracking.StopTrack(ctx, []track17.StopTrackRequest{...})

// 重启跟踪（每个单号限 1 次）
resp, _ := client.Tracking.ReTrack(ctx, []track17.ReTrackRequest{...})

// 删除单号
resp, _ := client.Tracking.DeleteTrack(ctx, []track17.DeleteTrackRequest{...})
```

### Query 服务 — 查询

```go
// 获取详情
resp, _ := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})

// 获取列表（分页，每页 40 条）
list, _ := client.Query.GetTrackList(ctx, track17.GetTrackListRequest{
    TrackingStatus: track17.StatusInTransit,
    PageNo:         1,
})

// 获取剩余额度
quota, _ := client.Query.GetQuota(ctx)
```

### Push 服务 — 手动推送

```go
resp, _ := client.Push.Push(ctx, []track17.PushRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})
```

### RealTime 服务 — 实时查询

```go
resp, _ := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
    {
        Number:      "RR123456789CN",
        CarrierCode: 3011,
        Mode:        track17.RealTimeModeStandard,  // 标准模式 (1 额度)
        // Mode:     track17.RealTimeModeInstant,   // 即时模式 (10 额度)
        Lang:        "en",
    },
})
```

### WebHook — 接收推送

```go
// 方式一：使用便捷 Handler
handler := track17.WebhookHandler("your-api-key", func(event track17.WebhookEvent) {
    switch event.Event {
    case track17.EventTrackingUpdated:
        fmt.Printf("Updated: %s\n", event.Data.Number)
    case track17.EventTrackingStopped:
        fmt.Printf("Stopped: %s\n", event.Data.Number)
    }
})
http.Handle("/webhook", handler)

// 方式二：手动解析
func myHandler(w http.ResponseWriter, r *http.Request) {
    event, err := track17.ParseWebhook(r, "your-api-key")
    if err != nil {
        http.Error(w, "invalid", 400)
        return
    }
    // 处理 event...
    w.WriteHeader(200)
}

// 方式三：仅验证签名
valid := track17.VerifySignature(payload, signature, "your-api-key")
```

## 🔍 错误处理

```go
resp, err := client.Tracking.Register(ctx, items)
if err != nil {
    // 检查是否为 API 错误
    if apiErr, ok := track17.IsAPIError(err); ok {
        if apiErr.IsRateLimited() {
            // 限流，等待后重试
        }
        if apiErr.IsInsufficientQuota() {
            // 余额不足
        }
    }
    log.Fatal(err)
}

// 检查部分失败的批量请求
for _, item := range resp.Rejected {
    fmt.Printf("Rejected: %s, Code: %d, Msg: %s\n",
        item.Number, item.Error.Code, item.Error.Message)
}
```

### 错误码常量

| 常量 | 值 | 说明 |
|---|---|---|
| `ErrInvalidAPIKey` | -18010003 | API Key 无效 |
| `ErrIPNotAllowed` | -18010004 | IP 不在白名单 |
| `ErrRateLimited` | -18010005 | 请求频率超限 |
| `ErrInsufficientQuota` | -18010006 | 余额不足 |
| `ErrAlreadyRegistered` | -18019901 | 单号已注册 |
| `ErrNotRegistered` | -18019902 | 单号未注册 |
| `ErrChangeCarrierLimit` | -18019906 | 修改运输商次数已达上限 |
| `ErrRetrackLimit` | -18019907 | 重启跟踪次数已达上限 |

## 📊 物流状态码

| 常量 | 值 | 说明 |
|---|---|---|
| `StatusNotFound` | 0 | 未查询到信息 |
| `StatusInTransit` | 10 | 运输中 |
| `StatusExpired` | 20 | 已过期 |
| `StatusPickedUp` | 30 | 已揽收 |
| `StatusUndeliverable` | 35 | 无法投递 |
| `StatusDelivered` | 40 | 已签收 |
| `StatusAlert` | 50 | 异常 |

## 📁 项目结构

```
track17/
├── track17.go        # 客户端核心 (Client, 认证, 限流, 重试)
├── option.go         # 配置选项 (Functional Options)
├── errors.go         # 错误类型与错误码
├── types.go          # 公共类型 (TrackInfo, TrackEvent, Milestone...)
├── tracking.go       # Tracking 服务 (注册/修改/停止/重启/删除)
├── query.go          # Query 服务 (查询详情/列表/额度)
├── push.go           # Push 服务 (手动推送)
├── realtime.go       # RealTime 服务 (实时查询)
├── webhook.go        # WebHook (签名验证, 事件解析, Handler)
├── track17_test.go   # 核心单元测试
├── webhook_test.go   # WebHook 单元测试
└── examples/
    ├── basic/        # 基础用法示例
    └── webhook/      # WebHook 服务器示例
```

## 📄 License

[GNU General Public License v3.0](LICENSE)
