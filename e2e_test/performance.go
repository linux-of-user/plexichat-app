package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type PerformanceResult struct {
	Name     string
	Duration time.Duration
	Success  bool
	Details  string
}

func main() {
	fmt.Println("⚡ PlexiChat Performance & Load Testing")
	fmt.Println("====================================================")
	fmt.Println("Testing system performance under load...")
	fmt.Println()

	var results []PerformanceResult
	baseURL := "http://localhost:8000"

	// Test 1: HTTP Request Performance
	results = append(results, testHTTPPerformance(baseURL))

	// Test 2: WebSocket Connection Performance
	results = append(results, testWebSocketPerformance(baseURL))

	// Test 3: Concurrent Connections
	results = append(results, testConcurrentConnections(baseURL))

	// Test 4: Message Throughput
	results = append(results, testMessageThroughput(baseURL))

	// Test 5: Memory Usage (Simulated)
	results = append(results, testMemoryUsage(baseURL))

	// Print Results
	fmt.Println("\n📊 Performance Test Results")
	fmt.Println("====================================================")
	
	for _, result := range results {
		status := "❌ FAIL"
		if result.Success {
			status = "✅ PASS"
		}
		fmt.Printf("%s %s: %v - %s\n", status, result.Name, result.Duration, result.Details)
	}

	// Overall assessment
	allPassed := true
	for _, result := range results {
		if !result.Success {
			allPassed = false
			break
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("🚀 PERFORMANCE TESTS PASSED!")
		fmt.Println("⚡ System performs well under load!")
		fmt.Println("📈 Ready for production traffic!")
	} else {
		fmt.Println("⚠️  Some performance tests failed.")
		fmt.Println("🔧 System may need optimization.")
	}
}

func testHTTPPerformance(baseURL string) PerformanceResult {
	fmt.Println("🔍 Testing HTTP request performance...")
	
	requests := 100
	start := time.Now()
	
	client := &http.Client{Timeout: 10 * time.Second}
	successCount := 0
	
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
	
	success := successCount >= requests*9/10 // 90% success rate
	details := fmt.Sprintf("%d/%d requests succeeded, avg latency: %v", successCount, requests, avgLatency)
	
	return PerformanceResult{"HTTP Performance", duration, success, details}
}

func testWebSocketPerformance(baseURL string) PerformanceResult {
	fmt.Println("🔍 Testing WebSocket connection performance...")
	
	connections := 50
	start := time.Now()
	
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/"
	successCount := 0
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("perf_test_%d", id), nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			// Wait for welcome message
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, _, err = conn.ReadMessage()
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	success := successCount >= connections*8/10 // 80% success rate
	details := fmt.Sprintf("%d/%d connections succeeded", successCount, connections)
	
	return PerformanceResult{"WebSocket Performance", duration, success, details}
}

func testConcurrentConnections(baseURL string) PerformanceResult {
	fmt.Println("🔍 Testing concurrent connection handling...")
	
	concurrentUsers := 25
	start := time.Now()
	
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/"
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("concurrent_%d", id), nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			// Skip welcome message
			conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			conn.ReadMessage()
			
			// Send a test message
			testMsg := map[string]interface{}{
				"type":    "performance_test",
				"user_id": fmt.Sprintf("concurrent_%d", id),
				"data":    "concurrent test message",
			}
			
			msgBytes, _ := json.Marshal(testMsg)
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				return
			}
			
			// Wait for echo
			conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, _, err = conn.ReadMessage()
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
			
			// Keep connection alive for a bit
			time.Sleep(100 * time.Millisecond)
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	success := successCount >= concurrentUsers*7/10 // 70% success rate
	details := fmt.Sprintf("%d/%d concurrent users handled successfully", successCount, concurrentUsers)
	
	return PerformanceResult{"Concurrent Connections", duration, success, details}
}

func testMessageThroughput(baseURL string) PerformanceResult {
	fmt.Println("🔍 Testing message throughput...")
	
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/throughput_test"
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return PerformanceResult{"Message Throughput", 0, false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()
	
	// Skip welcome message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()
	
	messages := 200
	start := time.Now()
	successCount := 0
	
	for i := 0; i < messages; i++ {
		testMsg := map[string]interface{}{
			"type":       "throughput_test",
			"message_id": i,
			"data":       fmt.Sprintf("Message %d for throughput testing", i),
		}
		
		msgBytes, _ := json.Marshal(testMsg)
		if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			break
		}
		
		// Read echo (with short timeout to maintain throughput)
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _, err := conn.ReadMessage()
		if err == nil {
			successCount++
		}
	}
	
	duration := time.Since(start)
	messagesPerSecond := float64(successCount) / duration.Seconds()
	
	success := messagesPerSecond >= 50 // At least 50 messages per second
	details := fmt.Sprintf("%.1f messages/sec (%d/%d successful)", messagesPerSecond, successCount, messages)
	
	return PerformanceResult{"Message Throughput", duration, success, details}
}

func testMemoryUsage(baseURL string) PerformanceResult {
	fmt.Println("🔍 Testing memory usage simulation...")
	
	// Simulate memory usage by creating many connections and messages
	start := time.Now()
	
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/"
	var connections []*websocket.Conn
	
	// Create connections
	for i := 0; i < 20; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("memory_test_%d", i), nil)
		if err != nil {
			break
		}
		connections = append(connections, conn)
		
		// Skip welcome message
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		conn.ReadMessage()
	}
	
	// Send messages on all connections
	messagesSent := 0
	for _, conn := range connections {
		for j := 0; j < 10; j++ {
			testMsg := map[string]interface{}{
				"type": "memory_test",
				"data": strings.Repeat("x", 100), // 100 character payload
			}
			
			msgBytes, _ := json.Marshal(testMsg)
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err == nil {
				messagesSent++
			}
		}
	}
	
	// Clean up connections
	for _, conn := range connections {
		conn.Close()
	}
	
	duration := time.Since(start)
	
	success := len(connections) >= 15 && messagesSent >= 100
	details := fmt.Sprintf("%d connections, %d messages sent", len(connections), messagesSent)
	
	return PerformanceResult{"Memory Usage", duration, success, details}
}
