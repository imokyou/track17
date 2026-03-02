// Package main demonstrates a webhook server for receiving 17Track push notifications.
//
// This server listens for tracking updates and processes them.
// Configure your 17Track webhook URL to point to this server.
//
// Usage:
//
//	export TRACK17_API_KEY="your-api-key"
//	go run ./examples/webhook/
//
// Then configure your 17Track webhook URL to:
//
//	http://your-server:8080/webhook/17track
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/imokyou/track17"
)

func main() {
	apiKey := os.Getenv("TRACK17_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ 请设置环境变量: export TRACK17_API_KEY=你的密钥")
	}

	// ============================================
	// 方式一：使用内置 WebhookHandler（推荐）
	// ============================================
	handler := track17.WebhookHandler(apiKey, func(event track17.WebhookEvent) {
		now := time.Now().Format("15:04:05")

		switch event.Event {
		case track17.EventTrackingUpdated:
			fmt.Printf("[%s] 📦 物流更新: %s\n", now, event.Data.Number)
			if event.Data.Track != nil {
				fmt.Printf("         运输商: %d\n", event.Data.Carrier)
				fmt.Printf("         状态:   %s\n", statusText(event.Data.Track.LatestStatus))
				fmt.Printf("         最新:   %s\n", event.Data.Track.LatestEvent)

				// 签收通知 — 在这里触发业务逻辑
				if event.Data.Track.LatestStatus == track17.StatusDelivered {
					fmt.Printf("         🎉 包裹已签收！\n")
					onDelivered(event.Data)
				}

				// 异常警报
				if event.Data.Track.LatestStatus == track17.StatusAlert {
					fmt.Printf("         🚨 物流异常！\n")
					onAlert(event.Data)
				}
			}

		case track17.EventTrackingStopped:
			fmt.Printf("[%s] 🛑 跟踪停止: %s\n", now, event.Data.Number)

		default:
			fmt.Printf("[%s] ❓ 未知事件: %s\n", now, event.Event)
		}
	})

	// ============================================
	// 方式二：手动解析（更灵活）
	// ============================================
	manualHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event, err := track17.ParseWebhook(r, apiKey)
		if err != nil {
			log.Printf("Webhook 验证失败: %v", err)
			http.Error(w, "invalid", http.StatusBadRequest)
			return
		}

		// 记录原始事件（调试用）
		data, _ := json.MarshalIndent(event, "", "  ")
		log.Printf("收到 Webhook:\n%s", string(data))

		w.WriteHeader(http.StatusOK)
	})

	// ============================================
	// 启动服务器
	// ============================================
	mux := http.NewServeMux()
	mux.Handle("/webhook/17track", handler)             // 推荐方式
	mux.Handle("/webhook/17track/debug", manualHandler) // 调试 endpoint

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := ":8080"
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║     17Track Webhook Server               ║")
	fmt.Println("╠══════════════════════════════════════════╣")
	fmt.Printf("║  监听地址: %s                          ║\n", addr)
	fmt.Println("║  Webhook:  /webhook/17track              ║")
	fmt.Println("║  调试:     /webhook/17track/debug        ║")
	fmt.Println("║  健康检查: /health                       ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📋 请在 17Track 控制台配置 Webhook URL:")
	fmt.Println("   https://你的域名/webhook/17track")
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, mux))
}

// onDelivered 处理包裹签收事件
// 在这里触发你的业务逻辑，比如：
//   - 发送通知邮件给买家
//   - 更新订单状态为"已签收"
//   - 触发确认收货倒计时
func onDelivered(data *track17.WebhookData) {
	fmt.Printf("         → 触发签收处理: 单号=%s, 标签=%s\n",
		data.Number, data.Tag)
}

// onAlert 处理物流异常事件
// 在这里触发告警，比如：
//   - 发送告警通知给客服
//   - 记录异常日志
func onAlert(data *track17.WebhookData) {
	fmt.Printf("         → 触发异常告警: 单号=%s\n", data.Number)
}

// statusText 将状态码转为可读文本
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
