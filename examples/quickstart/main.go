// Package main demonstrates the quickstart usage of the track17 SDK.
//
// This is the simplest example to get you started in 5 minutes.
//
// Usage:
//
//	export TRACK17_API_KEY="your-api-key"
//	go run ./examples/quickstart/
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/imokyou/track17"
)

func main() {
	// ============================================
	// 第 1 步：创建客户端
	// ============================================
	apiKey := os.Getenv("TRACK17_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ 请设置环境变量: export TRACK17_API_KEY=你的密钥")
	}

	client := track17.New(apiKey)
	defer client.Close()

	ctx := context.Background()

	// ============================================
	// 第 2 步：查询账户余额
	// ============================================
	fmt.Println("📊 查询账户余额...")
	quota, err := client.Query.GetQuota(ctx)
	if err != nil {
		log.Fatalf("查询余额失败: %v", err)
	}
	fmt.Printf("   总额度: %d | 已使用: %d | 剩余: %d\n\n",
		quota.Total, quota.Used, quota.Remaining)

	// ============================================
	// 第 3 步：注册一个物流单号
	// ============================================
	trackingNumber := "RR123456789CN"
	fmt.Printf("📝 注册单号: %s\n", trackingNumber)

	regResp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
		{
			Number:      trackingNumber,
			CarrierCode: 3011, // 中国邮政
			Lang:        "cn", // 中文翻译
			Tag:         "我的第一个包裹",
		},
	})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	for _, item := range regResp.Accepted {
		fmt.Printf("   ✅ 注册成功: %s (运输商: %d)\n", item.Number, item.Carrier)
	}
	for _, item := range regResp.Rejected {
		fmt.Printf("   ❌ 注册失败: %s — %s\n", item.Number, item.Error.Message)
	}

	// ============================================
	// 第 4 步：查询物流轨迹
	// ============================================
	fmt.Printf("\n📦 查询轨迹: %s\n", trackingNumber)

	infoResp, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
		{Number: trackingNumber, CarrierCode: 3011},
	})
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}

	for _, info := range infoResp.Accepted {
		if info.Track == nil {
			fmt.Println("   暂无轨迹信息（刚注册，请稍后再试）")
			continue
		}
		fmt.Printf("   状态: %s\n", statusText(info.Track.LatestStatus))
		fmt.Printf("   最新: %s\n", info.Track.LatestEvent)
		for _, event := range info.Track.Events {
			fmt.Printf("   📍 [%s] %s — %s\n",
				event.Time, event.Location, event.Description)
		}
	}

	fmt.Println("\n🎉 完成！")
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
