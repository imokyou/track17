package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/imokyou/track17"
)

func main() {
	apiKey := os.Getenv("TRACK17_API_KEY")
	if apiKey == "" {
		log.Fatal("TRACK17_API_KEY environment variable is required")
	}

	client := track17.New(apiKey,
		track17.WithRetry(3, 0),
		track17.WithDebug(true),
	)

	ctx := context.Background()

	// ==============================
	// 1. Check quota
	// ==============================
	fmt.Println("=== Checking Quota ===")
	quota, err := client.Query.GetQuota(ctx)
	if err != nil {
		log.Fatalf("GetQuota failed: %v", err)
	}
	fmt.Printf("Total: %d, Used: %d, Remaining: %d\n",
		quota.Total, quota.Used, quota.Remaining)

	// ==============================
	// 2. Register tracking numbers
	// ==============================
	fmt.Println("\n=== Registering Tracking Numbers ===")
	regResp, err := client.Tracking.Register(ctx, []track17.RegisterRequest{
		{
			Number:      "RR123456789CN",
			CarrierCode: 3011,
			Lang:        "en",
			Tag:         "test-order",
		},
	})
	if err != nil {
		log.Fatalf("Register failed: %v", err)
	}
	for _, item := range regResp.Accepted {
		fmt.Printf("✓ Registered: %s (carrier: %d)\n", item.Number, item.Carrier)
	}
	for _, item := range regResp.Rejected {
		fmt.Printf("✗ Rejected: %s (error: %d - %s)\n",
			item.Number, item.Error.Code, item.Error.Message)
	}

	// ==============================
	// 3. Get tracking info
	// ==============================
	fmt.Println("\n=== Getting Tracking Info ===")
	infoResp, err := client.Query.GetTrackInfo(ctx, []track17.GetTrackInfoRequest{
		{Number: "RR123456789CN", CarrierCode: 3011},
	})
	if err != nil {
		log.Fatalf("GetTrackInfo failed: %v", err)
	}
	for _, info := range infoResp.Accepted {
		fmt.Printf("Number: %s\n", info.Number)
		if info.Track != nil {
			fmt.Printf("  Status: %d (sub: %d)\n",
				info.Track.LatestStatus, info.Track.LatestSubStatus)
			fmt.Printf("  Latest: %s\n", info.Track.LatestEvent)
			for _, event := range info.Track.Events {
				fmt.Printf("  [%s] %s - %s\n",
					event.Time, event.Location, event.Description)
			}
		}
	}

	// ==============================
	// 4. Get tracking list
	// ==============================
	fmt.Println("\n=== Listing Tracked Numbers ===")
	listResp, err := client.Query.GetTrackList(ctx, track17.GetTrackListRequest{
		TrackingStatus: track17.StatusInTransit,
		PageNo:         1,
	})
	if err != nil {
		log.Fatalf("GetTrackList failed: %v", err)
	}
	for _, info := range listResp.Accepted {
		fmt.Printf("  %s (carrier: %d)\n", info.Number, info.Carrier)
	}
	fmt.Printf("Has more pages: %v\n", listResp.HasNext)

	// ==============================
	// 5. Real-time query
	// ==============================
	fmt.Println("\n=== Real-time Query ===")
	rtResp, err := client.RealTime.GetRealTimeTrackInfo(ctx, []track17.RealTimeRequest{
		{
			Number:      "RR123456789CN",
			CarrierCode: 3011,
			Mode:        track17.RealTimeModeStandard,
			Lang:        "en",
		},
	})
	if err != nil {
		log.Fatalf("GetRealTimeTrackInfo failed: %v", err)
	}
	for _, item := range rtResp.Accepted {
		fmt.Printf("Real-time: %s → Status: %d\n",
			item.Number, item.Track.LatestStatus)
	}

	fmt.Println("\nDone!")
}
