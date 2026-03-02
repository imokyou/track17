# 🚀 Track17 Go SDK 保姆级使用教程

> 从零开始，手把手教你用 Go 对接 17Track 物流查询 API

## 📋 目录

1. [准备工作](#-1-准备工作)
2. [安装 SDK](#-2-安装-sdk)
3. [第一个程序：查询余额](#-3-第一个程序查询余额)
4. [注册物流单号](#-4-注册物流单号)
5. [查询物流轨迹](#-5-查询物流轨迹)
6. [批量操作与分页](#-6-批量操作与分页)
7. [实时查询](#-7-实时查询)
8. [错误处理最佳实践](#-8-错误处理最佳实践)
9. [Webhook 接收推送](#-9-webhook-接收推送)
10. [生产环境配置](#-10-生产环境配置)
11. [完整示例汇总](#-11-完整示例汇总)

---

## 🔧 1. 准备工作

### 1.1 获取 17Track API Key

1. 访问 [17Track 官方 API 平台](https://api.17track.net)
2. 注册账号并登录
3. 在控制台创建应用，获取 **API Key**
4. 将 IP 添加到白名单（如果有此要求）

### 1.2 环境要求

- **Go 1.18+**（推荐 1.21+）
- 一个可用的 17Track API Key

### 1.3 项目初始化

```bash
# 创建项目目录
mkdir my-tracker && cd my-tracker

# 初始化 Go 模块
go mod init my-tracker

# 安装 SDK
go get github.com/imokyou/track17
```

---

## 📦 2. 安装 SDK

```bash
go get github.com/imokyou/track17
```

SDK 零外部依赖，只使用 Go 标准库，所以安装非常快。

---

## 🎯 3. 第一个程序：查询余额

让我们从最简单的例子开始 —— 查询你的 API 账户余额：

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/imokyou/track17"
)

func main() {
    // 第 1 步：从环境变量读取 API Key（不要硬编码！）
    apiKey := os.Getenv("TRACK17_API_KEY")
    if apiKey == "" {
        log.Fatal("请设置环境变量: export TRACK17_API_KEY=你的密钥")
    }

    // 第 2 步：创建客户端
    client := track17.New(apiKey)
    defer client.Close() // 用完记得关闭

    // 第 3 步：查询余额
    ctx := context.Background()
    quota, err := client.Query.GetQuota(ctx)
    if err != nil {
        log.Fatalf("查询余额失败: %v", err)
    }

    // 第 4 步：打印结果
    fmt.Printf("总额度: %d\n", quota.Total)
    fmt.Printf("已使用: %d\n", quota.Used)
    fmt.Printf("剩余:   %d\n", quota.Remaining)
}
```

运行：

```bash
export TRACK17_API_KEY="你的API密钥"
go run main.go
```

输出示例：

```
总额度: 10000
已使用: 1234
剩余:   8766
```

> **💡 提示**: 永远不要在代码中硬编码 API Key，使用环境变量或配置文件管理。

---

## 📝 4. 注册物流单号

要查询物流信息，首先需要把单号注册到 17Track 系统中：

```go
// 注册单个单号
resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
    {
        Number:      "RR123456789CN",  // 物流单号（必填）
        CarrierCode: 3011,             // 运输商代码（可选，不填会自动识别）
        Lang:        "cn",             // 翻译语言（可选）
        Tag:         "订单-001",        // 自定义标签（可选，方便管理）
    },
})
if err != nil {
    log.Fatalf("注册失败: %v", err)
}

// 检查成功的
for _, item := range resp.Accepted {
    fmt.Printf("✅ 注册成功: %s (运输商: %d)\n", item.Number, item.Carrier)
}

// 检查失败的（比如单号已注册）
for _, item := range resp.Rejected {
    fmt.Printf("❌ 注册失败: %s (错误: %s)\n", item.Number, item.Error.Message)
}
```

### 批量注册（每次最多 40 个）

```go
items := []track17.RegisterRequest{
    {Number: "RR111111111CN", CarrierCode: 3011, Tag: "批次A"},
    {Number: "EE222222222US"},                              // 自动识别运输商
    {Number: "JD0012345678", CarrierCode: 190011, Lang: "en"},
}

resp, err := client.Tracking.Register(ctx, items)
// ... 处理 resp.Accepted 和 resp.Rejected
```

### 常见运输商代码

| 运输商 | 代码 |
|---|---|
| 中国邮政 | 3011 |
| 圆通速递 | 190012 |
| 中通快递 | 190008 |
| 韵达快递 | 190001 |
| 顺丰速运 | 100003 |
| UPS | 21051 |
| FedEx | 100003 |
| DHL | 7041 |

> **💡 提示**: 运输商代码可以在 [17Track 官方文档](https://api.17track.net) 查询完整列表。

---

## 📡 5. 查询物流轨迹

注册完成后，就可以查询物流轨迹了：

```go
resp, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})
if err != nil {
    log.Fatalf("查询失败: %v", err)
}

for _, info := range resp.Accepted {
    fmt.Printf("📦 单号: %s\n", info.Number)
    fmt.Printf("   运输商: %d\n", info.Carrier)

    if info.Track == nil {
        fmt.Println("   暂无轨迹信息")
        continue
    }

    // 打印状态
    fmt.Printf("   最新状态: %s\n", statusText(info.Track.LatestStatus))
    fmt.Printf("   最新事件: %s\n", info.Track.LatestEvent)

    // 打印运输时长
    if info.Track.TransitDays > 0 {
        fmt.Printf("   运输天数: %d 天\n", info.Track.TransitDays)
    }

    // 打印事件历史
    fmt.Println("   --- 轨迹详情 ---")
    for _, event := range info.Track.Events {
        fmt.Printf("   [%s] %s - %s\n",
            event.Time, event.Location, event.Description)
    }
}

// 辅助函数：状态码转中文
func statusText(status int) string {
    switch status {
    case track17.StatusNotFound:
        return "未查询到"
    case track17.StatusInTransit:
        return "运输中 🚚"
    case track17.StatusPickedUp:
        return "已揽收 📬"
    case track17.StatusDelivered:
        return "已签收 ✅"
    case track17.StatusExpired:
        return "已过期 ⏰"
    case track17.StatusUndeliverable:
        return "无法投递 ⚠️"
    case track17.StatusAlert:
        return "异常 🚨"
    default:
        return fmt.Sprintf("未知(%d)", status)
    }
}
```

---

## 📄 6. 批量操作与分页

### 6.1 获取已注册单号列表（分页查询）

```go
page := 1
for {
    resp, err := client.Query.GetTrackList(ctx, track17.GetTrackListRequest{
        TrackingStatus: track17.StatusInTransit, // 只看运输中的
        PageNo:         page,
    })
    if err != nil {
        log.Fatalf("查询列表失败: %v", err)
    }

    fmt.Printf("--- 第 %d 页 ---\n", page)
    for _, info := range resp.Accepted {
        fmt.Printf("  %s (运输商: %d)\n", info.Number, info.Carrier)
    }

    if !resp.HasNext {
        break // 没有下一页了
    }
    page++
}
```

### 6.2 修改运输商

单号识别错了运输商？可以修改（每个单号最多改 5 次）：

```go
resp, err := client.Tracking.ChangeCarrier(ctx, []track17.ChangeCarrierRequest{
    {
        Number:     "RR123456789CN",
        CarrierOld: 3011,  // 旧运输商
        CarrierNew: 3012,  // 新运输商
    },
})
```

### 6.3 修改附加信息

可以修改标签、备注、订单号等：

```go
newTag := "VIP客户"
newRemark := "加急处理"
resp, err := client.Tracking.ChangeInfo(ctx, []track17.ChangeInfoRequest{
    {
        Number:      "RR123456789CN",
        CarrierCode: 3011,
        Tag:         &newTag,    // 使用指针，允许区分"不修改"和"清空"
        Remark:      &newRemark,
    },
})
```

### 6.4 停止 / 重启 / 删除跟踪

```go
// 停止跟踪
client.Tracking.StopTrack(ctx, []track17.StopTrackRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})

// 重启跟踪（每个单号只能重启 1 次）
client.Tracking.ReTrack(ctx, []track17.ReTrackRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})

// 永久删除（数据不可恢复！）
client.Tracking.DeleteTrack(ctx, []track17.DeleteTrackRequest{
    {Number: "RR123456789CN", CarrierCode: 3011},
})
```

---

## ⚡ 7. 实时查询

实时查询不需要先注册单号，直接查！适合一次性查询场景：

```go
resp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
    {
        Number:      "RR123456789CN",
        CarrierCode: 3011,
        Mode:        track17.RealTimeModeStandard, // 标准模式，消耗 1 额度
        Lang:        "cn",                          // 中文结果
    },
})
if err != nil {
    log.Fatalf("实时查询失败: %v", err)
}

for _, item := range resp.Accepted {
    fmt.Printf("📦 %s → 状态: %s\n", item.Number, statusText(item.Track.LatestStatus))
    for _, event := range item.Track.Events {
        fmt.Printf("  [%s] %s\n", event.Time, event.Description)
    }
}
```

> **💡 两种模式**:
> - `RealTimeModeStandard` — 标准模式，消耗 **1** 额度，适合大部分场景
> - `RealTimeModeInstant` — 即时模式，消耗 **10** 额度，响应更快

---

## 🛡️ 8. 错误处理最佳实践

### 8.1 基本错误处理

```go
resp, err := client.Tracking.Register(ctx, items)
if err != nil {
    // 判断是否为 API 错误（而不是网络错误）
    if apiErr, ok := track17.IsAPIError(err); ok {
        switch {
        case apiErr.IsRateLimited():
            fmt.Println("⏱️ 请求太频繁，请稍后重试")
        case apiErr.IsInsufficientQuota():
            fmt.Println("💰 余额不足，请充值")
        case apiErr.IsInvalidAPIKey():
            fmt.Println("🔑 API Key 无效，请检查配置")
        default:
            fmt.Printf("API 错误 %d: %s\n", apiErr.Code, apiErr.Message)
        }
    } else {
        // 网络错误、超时等
        fmt.Printf("网络错误: %v\n", err)
    }
    return
}

// 批量操作还要检查部分失败
for _, rejected := range resp.Rejected {
    fmt.Printf("单号 %s 处理失败: [%d] %s\n",
        rejected.Number, rejected.Error.Code, rejected.Error.Message)
}
```

### 8.2 使用 errors.As（Go 1.13+）

```go
import "errors"

var apiErr *track17.APIError
if errors.As(err, &apiErr) {
    // apiErr 可用
    fmt.Printf("API 错误码: %d\n", apiErr.Code)
}
```

### 8.3 常见错误码速查

| 错误码 | 常量 | 含义 | 处理建议 |
|---|---|---|---|
| -18010003 | `ErrInvalidAPIKey` | API Key 无效 | 检查密钥 |
| -18010004 | `ErrIPNotAllowed` | IP 不在白名单 | 添加服务器 IP |
| -18010005 | `ErrRateLimited` | 请求频率超限 | SDK 内置限流，一般不会触发 |
| -18010006 | `ErrInsufficientQuota` | 余额不足 | 充值 |
| -18019901 | `ErrAlreadyRegistered` | 单号已注册 | 直接查询即可 |
| -18019902 | `ErrNotRegistered` | 单号未注册 | 先注册 |
| -18019906 | `ErrChangeCarrierLimit` | 修改运输商次数上限 | 最多 5 次 |
| -18019907 | `ErrRetrackLimit` | 重启跟踪次数上限 | 最多 1 次 |

---

## 🔔 9. Webhook 接收推送

Webhook 让你被动接收物流状态变更通知，无需轮询。

### 9.1 方式一：使用内置 Handler（推荐）

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/imokyou/track17"
)

func main() {
    apiKey := os.Getenv("TRACK17_API_KEY")

    handler := track17.WebhookHandler(apiKey, func(event track17.WebhookEvent) {
        switch event.Event {
        case track17.EventTrackingUpdated:
            fmt.Printf("📦 物流更新: %s\n", event.Data.Number)
            if event.Data.Track != nil {
                fmt.Printf("   状态: %d\n", event.Data.Track.LatestStatus)

                // 签收通知
                if event.Data.Track.LatestStatus == track17.StatusDelivered {
                    fmt.Printf("   ✅ 包裹已签收！\n")
                    // 这里可以发送通知给用户...
                }
            }

        case track17.EventTrackingStopped:
            fmt.Printf("🛑 跟踪已停止: %s\n", event.Data.Number)
        }
    })

    http.Handle("/webhook/17track", handler)
    fmt.Println("🚀 Webhook 服务器启动在 :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 9.2 方式二：手动解析

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    event, err := track17.ParseWebhook(r, apiKey)
    if err != nil {
        http.Error(w, "签名验证失败", http.StatusBadRequest)
        return
    }

    // 处理事件...
    fmt.Printf("收到事件: %s\n", event.Event)

    w.WriteHeader(http.StatusOK) // 必须返回 200，否则 17Track 会重试
}
```

### 9.3 方式三：仅验证签名

```go
payload := []byte(`{"event":"TRACKING_UPDATED","data":{...}}`)
signature := r.Header.Get("sign")

if track17.VerifySignature(payload, signature, apiKey) {
    fmt.Println("签名有效 ✅")
} else {
    fmt.Println("签名无效 ❌")
}
```

> **💡 部署提示**: 在 17Track 控制台配置你的 Webhook URL，例如 `https://你的域名/webhook/17track`

---

## ⚙️ 10. 生产环境配置

### 10.1 推荐配置

```go
client := track17.New(apiKey,
    // 超时设置：避免慢请求卡住
    track17.WithTimeout(10 * time.Second),

    // 自动重试：5xx 和 429 错误自动重试 3 次
    track17.WithRetry(3, time.Second),

    // 开发环境打开调试
    // track17.WithDebug(true),
)
defer client.Close()
```

### 10.2 使用自定义 Logger

```go
import "log"

logger := log.New(os.Stderr, "[17TRACK] ", log.LstdFlags|log.Lshortfile)
client := track17.New(apiKey,
    track17.WithDebug(true),
    track17.WithLogger(logger),
)
```

### 10.3 使用自定义 HTTP Client（代理 / mTLS）

```go
import "net/http"

httpClient := &http.Client{
    Timeout: 15 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        // 配置代理
        // Proxy: http.ProxyURL(proxyURL),
    },
}

client := track17.New(apiKey,
    track17.WithHTTPClient(httpClient),
)
```

### 10.4 并发安全

`Client` 是并发安全的，可以在多个 goroutine 中共享：

```go
client := track17.New(apiKey)

// 安全！多个 goroutine 共享同一个 client
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        quota, _ := client.Query.GetQuota(ctx)
        fmt.Printf("goroutine %d: remaining = %d\n", i, quota.Remaining)
    }(i)
}
wg.Wait()
```

### 10.5 Context 超时控制

```go
// 单个请求设置 5 秒超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Query.GetTrackInfo(ctx, items)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("请求超时！")
    }
}
```

---

## 📁 11. 完整示例汇总

SDK 内置了多个可直接运行的示例程序：

```
examples/
├── quickstart/      # 快速开始 — 5 分钟上手
├── basic/           # 完整功能演示 — 所有 API 端点
├── error_handling/  # 错误处理 — 生产级异常处理
├── realtime/        # 实时查询 — 不注册直接查
└── webhook/         # Webhook 服务器 — 接收推送通知
```

运行任意示例：

```bash
export TRACK17_API_KEY="你的密钥"
go run ./examples/quickstart/
```

---

## ❓ FAQ

### Q: SDK 有外部依赖吗？
没有，SDK 只使用 Go 标准库，不会给你的项目引入额外依赖。

### Q: 限流怎么处理？
SDK 内置了限流器（默认 3 次/秒），内部自动等待，你无需关心。

### Q: 遇到 5xx 错误怎么办？
配置 `WithRetry(3, time.Second)` 后，SDK 会自动使用指数退避策略重试。

### Q: `CarrierCode` 不知道填什么？
可以设为 0 或不填，17Track 会自动识别运输商。也可以设置 `AutoDetect: true`：
```go
{Number: "xxx", AutoDetect: true}
```

### Q: 单号注册后多久能查到轨迹？
通常几分钟到几小时不等，取决于运输商和17Track的数据刷新频率。急需查询可以用实时查询 API。
