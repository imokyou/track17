// Package main demonstrates the complete track17 SDK functionality.
//
// This example covers all major API operations:
//   - Account quota check
//   - Tracking registration
//   - Track info queries
//   - Track list with pagination
//   - Carrier changes
//   - Info updates
//   - Stop / Restart / Delete tracking
//   - Real-time queries
//   - Manual push
//
// Usage:
//
//	export TRACK17_API_KEY="your-api-key"
//	go run ./examples/basic/
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

	// 创建客户端 — 生产级配置
	client, err := track17.New(apiKey,
		track17.WithTimeout(10*time.Second),
		track17.WithRetry(3, time.Second),
		track17.WithDebug(true), // 开启调试日志，生产环境建议关闭
		track17.WithCircuitBreaker(5, 30*time.Second),
	)
	if err != nil {
		log.Fatalf("初始化客户端失败: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// ==============================
	// 1. 查询账户余额
	// ==============================
	section("1. 查询账户余额")
	quota, err := client.Query.GetQuota(ctx)
	if err != nil {
		log.Fatalf("查询余额失败: %v", err)
	}
	fmt.Printf("总额度: %d | 已使用: %d | 剩余: %d\n",
		quota.Total, quota.Used, quota.Remaining)

	// ==============================
	// 2. 注册物流单号
	// ==============================
	section("2. 注册物流单号")
	regResp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
		{
			Number:      "RR123456789CN",
			CarrierCode: 3011,
			Lang:        "cn",
			Tag:         "测试订单",
			Remark:      "SDK示例",
		},
	})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	for _, item := range regResp.Accepted {
		fmt.Printf("✅ 注册成功: %s (运输商: %d)\n", item.Number, item.Carrier)
	}
	for _, item := range regResp.Rejected {
		fmt.Printf("❌ 注册失败: %s — [%d] %s\n",
			item.Number, item.Error.Code, item.Error.Message)
	}

	// ==============================
	// 3. 查询物流轨迹
	// ==============================
	section("3. 查询物流轨迹")
	infoResp, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		log.Fatalf("查询轨迹失败: %v", err)
	}
	for _, info := range infoResp.Accepted {
		fmt.Printf("📦 %s\n", info.Number)
		if info.Track != nil {
			fmt.Printf("   状态: %d (子状态: %d)\n",
				info.Track.LatestStatus, info.Track.LatestSubStatus)
			fmt.Printf("   最新事件: %s\n", info.Track.LatestEvent)
			for _, event := range info.Track.Events {
				fmt.Printf("   📍 [%s] %s — %s\n",
					event.Time, event.Location, event.Description)
			}
		} else {
			fmt.Println("   暂无轨迹信息")
		}
	}

	// ==============================
	// 4. 获取已注册单号列表（分页）
	// ==============================
	section("4. 获取已注册单号列表")
	listResp, err := client.Query.GetTrackList(ctx, track17.GetTrackListRequest{
		TrackingStatus: track17.StatusInTransit,
		PageNo:         1,
	})
	if err != nil {
		log.Fatalf("查询列表失败: %v", err)
	}
	for _, info := range listResp.Accepted {
		fmt.Printf("   %s (运输商: %d)\n", info.Number, info.Carrier)
	}
	fmt.Printf("是否有下一页: %v\n", listResp.HasNext)

	// ==============================
	// 5. 实时查询（不需要先注册）
	// ==============================
	section("5. 实时查询")
	rtResp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
		{
			Number:      "RR123456789CN",
			CarrierCode: 3011,
			Mode:        track17.RealTimeModeStandard,
			Lang:        "cn",
		},
	})
	if err != nil {
		log.Fatalf("实时查询失败: %v", err)
	}
	for _, item := range rtResp.Accepted {
		fmt.Printf("📦 %s → 状态: %d\n", item.Number, item.Track.LatestStatus)
	}

	// ==============================
	// 6. 手动推送
	// ==============================
	section("6. 手动触发推送")
	pushResp, err := client.Push.Push(ctx, []track17.PushRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		log.Fatalf("推送失败: %v", err)
	}
	for _, item := range pushResp.Accepted {
		fmt.Printf("✅ 已触发推送: %s\n", item.Number)
	}

	section("完成")
	fmt.Println("🎉 所有 API 演示完毕！")
}

func section(title string) {
	fmt.Printf("\n══════ %s ══════\n", title)
}
