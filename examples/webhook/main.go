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
	if apiKey == "" {
		log.Fatal("TRACK17_API_KEY environment variable is required")
	}

	// Create a webhook handler that processes tracking events
	handler := track17.WebhookHandler(apiKey, func(event track17.WebhookEvent) {
		switch event.Event {
		case track17.EventTrackingUpdated:
			fmt.Printf("📦 Tracking Updated: %s\n", event.Data.Number)
			if event.Data.Track != nil {
				fmt.Printf("   Status: %d\n", event.Data.Track.LatestStatus)
				fmt.Printf("   Latest: %s\n", event.Data.Track.LatestEvent)

				// Check for delivery
				if event.Data.Track.LatestStatus == track17.StatusDelivered {
					fmt.Printf("   ✅ Package delivered!\n")
				}
			}

		case track17.EventTrackingStopped:
			fmt.Printf("🛑 Tracking Stopped: %s\n", event.Data.Number)
		}
	})

	// Start HTTP server
	addr := ":8080"
	http.Handle("/webhook/17track", handler)
	fmt.Printf("🚀 Webhook server listening on %s\n", addr)
	fmt.Println("Configure your 17Track webhook URL to: http://your-domain:8080/webhook/17track")
	log.Fatal(http.ListenAndServe(addr, nil))
}
