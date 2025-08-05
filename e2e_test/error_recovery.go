package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type ErrorTestResult struct {
	Name    string
	Passed  bool
	Message string
}

func main() {
	fmt.Println("🛡️  PlexiChat Error Handling & Recovery Tests")
	fmt.Println("====================================================")
	fmt.Println("Testing system resilience and error recovery...")
	fmt.Println()

	var results []ErrorTestResult
	baseURL := "http://localhost:8000"

	// Test 1: Invalid Endpoints
	results = append(results, testInvalidEndpoints(baseURL))

	// Test 2: WebSocket Error Handling
	results = append(results, testWebSocketErrors(baseURL))

	// Test 3: Malformed JSON
	results = append(results, testMalformedJSON(baseURL))

	// Test 4: Connection Timeout Handling
	results = append(results, testConnectionTimeout())

	// Test 5: Server Overload Simulation
	results = append(results, testServerOverload(baseURL))

	// Print Results
	fmt.Println("\n📊 Error Handling Test Results")
	fmt.Println("====================================================")
	
	passed := 0
	total := len(results)
	
	for _, result := range results {
		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
			passed++
		}
		fmt.Printf("%s %s: %s\n", status, result.Name, result.Message)
	}

	fmt.Printf("\n📈 Error Handling Results: %d/%d tests passed (%.1f%%)\n", 
		passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("\n🛡️  ERROR HANDLING TESTS PASSED!")
		fmt.Println("🔒 System is resilient and handles errors gracefully!")
	} else {
		fmt.Printf("\n⚠️  %d error handling tests failed.\n", total-passed)
	}
}

func testInvalidEndpoints(baseURL string) ErrorTestResult {
	fmt.Println("🔍 Testing invalid endpoint handling...")
	
	invalidEndpoints := []string{
		"/nonexistent",
		"/api/v1/invalid",
		"/api/v2/health",
		"/admin/secret",
	}

	for _, endpoint := range invalidEndpoints {
		resp, err := http.Get(baseURL + endpoint)
		if err != nil {
			return ErrorTestResult{"Invalid Endpoints", false, fmt.Sprintf("Network error on %s: %v", endpoint, err)}
		}
		resp.Body.Close()

		// Should return 404 or similar error status
		if resp.StatusCode == 200 {
			return ErrorTestResult{"Invalid Endpoints", false, fmt.Sprintf("Invalid endpoint %s returned 200", endpoint)}
		}
	}

	return ErrorTestResult{"Invalid Endpoints", true, "Invalid endpoints properly return error status codes"}
}

func testWebSocketErrors(baseURL string) ErrorTestResult {
	fmt.Println("🔍 Testing WebSocket error handling...")
	
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/error_test_user"
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return ErrorTestResult{"WebSocket Errors", false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()

	// Skip welcome message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()

	// Send invalid JSON
	invalidJSON := "{ invalid json }"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(invalidJSON)); err != nil {
		return ErrorTestResult{"WebSocket Errors", false, fmt.Sprintf("Failed to send invalid JSON: %v", err)}
	}

	// Wait for error response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		return ErrorTestResult{"WebSocket Errors", false, fmt.Sprintf("No error response received: %v", err)}
	}

	var wsMsg map[string]interface{}
	if err := json.Unmarshal(response, &wsMsg); err != nil {
		return ErrorTestResult{"WebSocket Errors", false, fmt.Sprintf("Invalid error response: %v", err)}
	}

	msgType, exists := wsMsg["type"]
	if !exists || msgType != "error" {
		return ErrorTestResult{"WebSocket Errors", false, "Server did not send error message for invalid JSON"}
	}

	return ErrorTestResult{"WebSocket Errors", true, "WebSocket properly handles and reports errors"}
}

func testMalformedJSON(baseURL string) ErrorTestResult {
	fmt.Println("🔍 Testing malformed JSON handling...")
	
	// Test with various malformed JSON payloads
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/json_test_user"
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return ErrorTestResult{"Malformed JSON", false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()

	// Skip welcome message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()

	malformedJSONs := []string{
		"{ incomplete",
		"{ \"key\": }",
		"{ \"key\": \"value\" extra }",
		"not json at all",
		"",
	}

	errorCount := 0
	for _, badJSON := range malformedJSONs {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(badJSON)); err != nil {
			continue
		}

		// Check for error response
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, response, err := conn.ReadMessage()
		if err != nil {
			continue
		}

		var wsMsg map[string]interface{}
		if err := json.Unmarshal(response, &wsMsg); err != nil {
			continue
		}

		if msgType, exists := wsMsg["type"]; exists && msgType == "error" {
			errorCount++
		}
	}

	if errorCount == 0 {
		return ErrorTestResult{"Malformed JSON", false, "Server did not handle any malformed JSON properly"}
	}

	return ErrorTestResult{"Malformed JSON", true, fmt.Sprintf("Server handled %d/%d malformed JSON cases", errorCount, len(malformedJSONs))}
}

func testConnectionTimeout() ErrorTestResult {
	fmt.Println("🔍 Testing connection timeout handling...")
	
	// Try to connect to a non-existent server
	client := &http.Client{Timeout: 2 * time.Second}
	
	start := time.Now()
	_, err := client.Get("http://localhost:9999/nonexistent")
	duration := time.Since(start)
	
	if err == nil {
		return ErrorTestResult{"Connection Timeout", false, "Expected timeout error but got success"}
	}

	// Should timeout within reasonable time (2-3 seconds)
	if duration > 5*time.Second {
		return ErrorTestResult{"Connection Timeout", false, fmt.Sprintf("Timeout took too long: %v", duration)}
	}

	return ErrorTestResult{"Connection Timeout", true, fmt.Sprintf("Connection timeout handled properly in %v", duration)}
}

func testServerOverload(baseURL string) ErrorTestResult {
	fmt.Println("🔍 Testing server overload handling...")
	
	// Create multiple concurrent connections to test server limits
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/"
	
	var connections []*websocket.Conn
	maxConnections := 10
	successfulConnections := 0

	for i := 0; i < maxConnections; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("load_test_%d", i), nil)
		if err != nil {
			break
		}
		connections = append(connections, conn)
		successfulConnections++
		
		// Small delay to avoid overwhelming the server
		time.Sleep(10 * time.Millisecond)
	}

	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}

	if successfulConnections == 0 {
		return ErrorTestResult{"Server Overload", false, "Could not establish any connections"}
	}

	// Test rapid requests
	client := &http.Client{Timeout: 5 * time.Second}
	rapidRequests := 20
	successfulRequests := 0

	for i := 0; i < rapidRequests; i++ {
		resp, err := client.Get(baseURL + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				successfulRequests++
			}
		}
	}

	if successfulRequests < rapidRequests/2 {
		return ErrorTestResult{"Server Overload", false, fmt.Sprintf("Only %d/%d rapid requests succeeded", successfulRequests, rapidRequests)}
	}

	return ErrorTestResult{"Server Overload", true, fmt.Sprintf("Server handled %d connections and %d/%d rapid requests", successfulConnections, successfulRequests, rapidRequests)}
}
