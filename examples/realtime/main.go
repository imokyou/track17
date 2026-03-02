// Package main demonstrates real-time tracking queries.
//
// Real-time queries fetch tracking info directly from carriers
// without requiring prior registration. Great for one-off lookups.
//
// Usage:
//
//	export TRACK17_API_KEY="your-api-key"
//	go run ./examples/realtime/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/imokyou/track17"
)

func main() {
	apiKey := os.Getenv("TRACK17_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ 请设置环境变量: export TRACK17_API_KEY=你的密钥")
	}

	client := track17.New(apiKey,
		track17.WithTimeout(30*time.Second), // 实时查询可能比较慢
		track17.WithRetry(2, time.Second),
	)
	defer client.Close()

	ctx := context.Background()

	// ============================================
	// 实时查询 — 标准模式 (消耗 1 额度)
	// ============================================
	fmt.Println("⚡ 实时查询 — 标准模式")
	fmt.Println("   (不需要提前注册单号，直接查！)")
	fmt.Println()

	resp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
		{
			Number:      "RR123456789CN",
			CarrierCode: 3011,                         // 中国邮政
			Mode:        track17.RealTimeModeStandard, // 标准模式
			Lang:        "cn",                         // 中文结果
		},
	})
	if err != nil {
		if apiErr, ok := track17.IsAPIError(err); ok {
			fmt.Printf("API 错误: [%d] %s\n", apiErr.Code, apiErr.Message)
		} else {
			log.Fatalf("查询失败: %v", err)
		}
		return
	}

	// 处理成功结果
	for _, item := range resp.Accepted {
		printTrackInfo(item.Number, item.Carrier, item.Track)
	}

	// 处理失败结果
	for _, item := range resp.Rejected {
		fmt.Printf("❌ %s — 错误: %s\n", item.Number, item.Error.Message)
	}

	// ============================================
	// 批量实时查询 (最多 40 个)
	// ============================================
	fmt.Println("\n⚡ 批量实时查询")

	batchResp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
		{Number: "RR111111111CN", CarrierCode: 3011, Mode: track17.RealTimeModeStandard, Lang: "cn"},
		{Number: "EE222222222US", CarrierCode: 21051, Mode: track17.RealTimeModeStandard, Lang: "en"},
	})
	if err != nil {
		log.Fatalf("批量查询失败: %v", err)
	}

	for _, item := range batchResp.Accepted {
		printTrackInfo(item.Number, item.Carrier, item.Track)
	}
}

// printTrackInfo 格式化打印轨迹信息
func printTrackInfo(number string, carrier int, track *track17.TrackDetail) {
	fmt.Printf("📦 单号: %s (运输商: %d)\n", number, carrier)

	if track == nil {
		fmt.Println("   暂无轨迹信息")
		return
	}

	fmt.Printf("   状态: %s\n", statusText(track.LatestStatus))
	if track.LatestEvent != "" {
		fmt.Printf("   最新: %s\n", track.LatestEvent)
	}
	if track.OriginCountry != "" {
		fmt.Printf("   路线: %s → %s\n", track.OriginCountry, track.DestCountry)
	}
	if track.TransitDays > 0 {
		fmt.Printf("   耗时: %d 天\n", track.TransitDays)
	}

	// 里程碑
	if track.Milestone != nil {
		fmt.Println("   📌 关键节点:")
		if track.Milestone.PickedUp != "" {
			fmt.Printf("      揽收: %s\n", track.Milestone.PickedUp)
		}
		if track.Milestone.DepartOrigin != "" {
			fmt.Printf("      离开始发地: %s\n", track.Milestone.DepartOrigin)
		}
		if track.Milestone.ArriveDest != "" {
			fmt.Printf("      到达目的地: %s\n", track.Milestone.ArriveDest)
		}
		if track.Milestone.Delivered != "" {
			fmt.Printf("      签收: %s\n", track.Milestone.Delivered)
		}
	}

	// 轨迹事件
	if len(track.Events) > 0 {
		fmt.Println("   📍 轨迹详情:")
		for _, event := range track.Events {
			loc := event.Location
			if loc == "" {
				loc = "--"
			}
			fmt.Printf("      [%s] %s | %s\n", event.Time, loc, event.Description)
		}
	}
	fmt.Println()
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
