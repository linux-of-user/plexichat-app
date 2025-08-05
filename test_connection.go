package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

func main() {
	fmt.Println("🚀 Testing PlexiChat Client-Server Connection")
	fmt.Println("====================================================")

	// Test basic health endpoint
	fmt.Println("📡 Testing basic health endpoint...")
	if testEndpoint("http://localhost:8000/health") {
		fmt.Println("✅ Basic health endpoint: OK")
	} else {
		fmt.Println("❌ Basic health endpoint: FAILED")
		return
	}

	// Test API health endpoint
	fmt.Println("📡 Testing API health endpoint...")
	if testEndpoint("http://localhost:8000/api/v1/health") {
		fmt.Println("✅ API health endpoint: OK")
	} else {
		fmt.Println("❌ API health endpoint: FAILED")
		return
	}

	fmt.Println("\n🎉 All connection tests passed!")
	fmt.Println("🔗 Client can successfully communicate with server")
}

func testEndpoint(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return false
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return false
	}

	if health.Status != "ok" {
		fmt.Printf("❌ Server status not OK: %s\n", health.Status)
		return false
	}

	fmt.Printf("   📊 Response: %s\n", string(body))
	return true
}
