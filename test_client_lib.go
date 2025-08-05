package main

import (
	"context"
	"fmt"
	"time"

	"plexichat-client/pkg/client"
)

func main() {
	fmt.Println("🧪 Testing PlexiChat Client Library")
	fmt.Println("====================================================")

	// Create client
	fmt.Println("🔧 Creating PlexiChat client...")
	c := client.NewClient("http://localhost:8000")
	if c == nil {
		fmt.Println("❌ Failed to create client")
		return
	}
	fmt.Println("✅ Client created successfully")

	// Configure client
	c.SetDebug(true)
	c.SetTimeout(30 * time.Second)
	fmt.Println("✅ Client configured")

	// Test health check
	fmt.Println("\n📡 Testing health check...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := c.Health(ctx)
	if err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Health check passed: %+v\n", health)

	// Test configuration
	fmt.Println("\n⚙️  Testing client configuration...")
	if c.BaseURL != "http://localhost:8000" {
		fmt.Printf("❌ Expected BaseURL to be 'http://localhost:8000', got %s\n", c.BaseURL)
		return
	}
	fmt.Println("✅ BaseURL configured correctly")

	if !c.Debug {
		fmt.Println("❌ Expected Debug to be true")
		return
	}
	fmt.Println("✅ Debug mode enabled")

	if c.HTTPClient.Timeout != 30*time.Second {
		fmt.Printf("❌ Expected timeout to be 30s, got %v\n", c.HTTPClient.Timeout)
		return
	}
	fmt.Println("✅ Timeout configured correctly")

	fmt.Println("\n🎉 All client library tests passed!")
	fmt.Println("📚 PlexiChat client library is working correctly")
}
