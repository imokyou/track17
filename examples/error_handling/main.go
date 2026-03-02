// Package main demonstrates comprehensive error handling patterns.
//
// This example shows how to properly handle all types of errors
// when using the track17 SDK in a production environment.
//
// Usage:
//
//	export TRACK17_API_KEY="your-api-key"
//	go run ./examples/error_handling/
package main

import (
	"context"
	"errors"
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

	// 生产级配置：自动重试 + 超时
	client := track17.New(apiKey,
		track17.WithTimeout(10*time.Second),
		track17.WithRetry(3, time.Second),
	)
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== 示例 1: API 错误处理 ===")
	demoAPIError(ctx, client)

	fmt.Println("\n=== 示例 2: 批量操作部分失败 ===")
	demoBatchPartialFailure(ctx, client)

	fmt.Println("\n=== 示例 3: 超时控制 ===")
	demoTimeout(ctx, client)

	fmt.Println("\n=== 示例 4: errors.As 用法 ===")
	demoErrorsAs(ctx, client)
}

// demoAPIError 演示 API 错误分类处理
func demoAPIError(ctx context.Context, client *track17.Client) {
	resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
		{Number: "TEST123"},
	})
	if err != nil {
		// 判断是否为 17Track API 错误
		if apiErr, ok := track17.IsAPIError(err); ok {
			switch {
			case apiErr.IsRateLimited():
				fmt.Println("⏱️  请求频率超限 — SDK 已内置限流，正常情况不会触发")
				fmt.Println("   建议: 检查是否有多个客户端同时使用同一 API Key")

			case apiErr.IsInsufficientQuota():
				fmt.Println("💰 余额不足")
				fmt.Println("   建议: 登录 17Track 控制台充值")

			case apiErr.IsInvalidAPIKey():
				fmt.Println("🔑 API Key 无效")
				fmt.Println("   建议: 检查环境变量是否正确设置")

			case apiErr.IsIPNotAllowed():
				fmt.Println("🌐 IP 不在白名单")
				fmt.Println("   建议: 在 17Track 控制台添加当前服务器 IP")

			default:
				fmt.Printf("❓ 其他 API 错误: [%d] %s\n", apiErr.Code, apiErr.Message)
			}
		} else {
			// 网络错误、DNS 解析失败、连接超时等
			fmt.Printf("🌐 网络错误: %v\n", err)
			fmt.Println("   建议: 检查网络连接和 DNS 配置")
		}
		return
	}

	fmt.Printf("注册成功: %d 个, 失败: %d 个\n",
		len(resp.Accepted), len(resp.Rejected))
}

// demoBatchPartialFailure 演示批量操作中的部分失败处理
func demoBatchPartialFailure(ctx context.Context, client *track17.Client) {
	resp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
		{Number: "PKG001", Tag: "新单号"},
		{Number: "PKG002", Tag: "可能已注册"},
		{Number: "PKG003", Tag: "新单号"},
	})
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}

	// ✅ 成功的单号
	fmt.Printf("成功: %d 个\n", len(resp.Accepted))
	for _, item := range resp.Accepted {
		fmt.Printf("  ✅ %s (运输商: %d)\n", item.Number, item.Carrier)
	}

	// ❌ 失败的单号 — 生产环境中必须处理！
	if len(resp.Rejected) > 0 {
		fmt.Printf("失败: %d 个\n", len(resp.Rejected))
		for _, item := range resp.Rejected {
			fmt.Printf("  ❌ %s — 错误码: %d, 原因: %s\n",
				item.Number, item.Error.Code, item.Error.Message)

			// 根据错误码做不同处理
			switch item.Error.Code {
			case track17.ErrAlreadyRegistered:
				fmt.Printf("     → 已注册，跳过，可直接查询\n")
			case track17.ErrInvalidNumber:
				fmt.Printf("     → 单号格式无效，请检查\n")
			case track17.ErrNumberTooShort:
				fmt.Printf("     → 单号太短，最少 5 个字符\n")
			default:
				fmt.Printf("     → 需要人工处理\n")
			}
		}
	}
}

// demoTimeout 演示请求超时处理
func demoTimeout(ctx context.Context, client *track17.Client) {
	// 设置一个非常短的超时来演示
	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// 故意等一下让超时生效
	time.Sleep(2 * time.Millisecond)

	_, err := client.Query.GetQuota(shortCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("⏰ 请求超时！")
			fmt.Println("   建议: 增加超时时间或检查网络")
		} else if errors.Is(err, context.Canceled) {
			fmt.Println("🚫 请求被取消")
		} else {
			fmt.Printf("其他错误: %v\n", err)
		}
		return
	}

	fmt.Println("查询成功（在超时前完成）")
}

// demoErrorsAs 演示 Go 标准 errors.As 用法
func demoErrorsAs(ctx context.Context, client *track17.Client) {
	_, err := client.Query.GetQuota(ctx)
	if err != nil {
		// Go 1.13+ 标准做法
		var apiErr *track17.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("使用 errors.As 捕获到 API 错误:\n")
			fmt.Printf("  错误码:    %d\n", apiErr.Code)
			fmt.Printf("  HTTP 状态: %d\n", apiErr.StatusCode)
			fmt.Printf("  消息:      %s\n", apiErr.Message)
		} else {
			fmt.Printf("非 API 错误: %v\n", err)
		}
		return
	}

	fmt.Println("查询成功，没有错误发生")
}
