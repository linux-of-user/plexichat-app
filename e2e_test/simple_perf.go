package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	fmt.Println("⚡ Simple Performance Test")
	fmt.Println("====================================================")
	
	baseURL := "http://localhost:8000"
	
	// Test HTTP performance
	fmt.Println("🔍 Testing HTTP request performance...")
	
	requests := 50
	start := time.Now()
	successCount := 0
	
	client := &http.Client{Timeout: 10 * time.Second}
	
	for i := 0; i < requests; i++ {
		resp, err := client.Get(baseURL + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				successCount++
			}
		}
	}
	
	duration := time.Since(start)
	avgLatency := duration / time.Duration(requests)
	requestsPerSecond := float64(successCount) / duration.Seconds()
	
	fmt.Printf("📊 Results:\n")
	fmt.Printf("   Total requests: %d\n", requests)
	fmt.Printf("   Successful: %d\n", successCount)
	fmt.Printf("   Success rate: %.1f%%\n", float64(successCount)/float64(requests)*100)
	fmt.Printf("   Total time: %v\n", duration)
	fmt.Printf("   Average latency: %v\n", avgLatency)
	fmt.Printf("   Requests/second: %.1f\n", requestsPerSecond)
	
	if successCount >= requests*9/10 && requestsPerSecond >= 10 {
		fmt.Println("\n✅ Performance test PASSED!")
		fmt.Println("🚀 System performs well!")
	} else {
		fmt.Println("\n❌ Performance test FAILED!")
		fmt.Println("🔧 System may need optimization.")
	}
}
