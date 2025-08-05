package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

type HealthResponse struct {
	Status               string   `json:"status"`
	Message              string   `json:"message,omitempty"`
	Features             []string `json:"features,omitempty"`
	WebsocketConnections int      `json:"websocket_connections,omitempty"`
}

type WSMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	fmt.Println("🧪 PlexiChat End-to-End Testing Suite")
	fmt.Println("====================================================")
	fmt.Println("Testing complete client-server integration...")
	fmt.Println()

	var results []TestResult
	baseURL := "http://localhost:8000"

	// Test 1: Server Health Check
	results = append(results, testServerHealth(baseURL))

	// Test 2: API Endpoints
	results = append(results, testAPIEndpoints(baseURL))

	// Test 3: WebSocket Connection
	results = append(results, testWebSocketConnection(baseURL))

	// Test 4: WebSocket Messaging
	results = append(results, testWebSocketMessaging(baseURL))

	// Test 5: Multiple Client Simulation
	results = append(results, testMultipleClients(baseURL))

	// Print Results
	fmt.Println("\n📊 Test Results Summary")
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

	fmt.Printf("\n📈 Overall Results: %d/%d tests passed (%.1f%%)\n",
		passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("\n🎉 ALL TESTS PASSED!")
		fmt.Println("🚀 PlexiChat system is fully functional!")
		fmt.Println("✨ Ready for production use!")
	} else {
		fmt.Printf("\n⚠️  %d tests failed. System needs attention.\n", total-passed)
	}
}

func testServerHealth(baseURL string) TestResult {
	fmt.Println("🔍 Testing server health...")

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return TestResult{"Server Health", false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return TestResult{"Server Health", false, fmt.Sprintf("Status code: %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestResult{"Server Health", false, fmt.Sprintf("Read error: %v", err)}
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return TestResult{"Server Health", false, fmt.Sprintf("JSON parse error: %v", err)}
	}

	if health.Status != "ok" {
		return TestResult{"Server Health", false, fmt.Sprintf("Status not OK: %s", health.Status)}
	}

	return TestResult{"Server Health", true, "Server is healthy and responding"}
}

func testAPIEndpoints(baseURL string) TestResult {
	fmt.Println("🔍 Testing API endpoints...")

	endpoints := []string{
		"/api/v1/health",
		"/api/v1/test/ping",
		"/api/v1/stats",
	}

	for _, endpoint := range endpoints {
		resp, err := http.Get(baseURL + endpoint)
		if err != nil {
			return TestResult{"API Endpoints", false, fmt.Sprintf("Failed to reach %s: %v", endpoint, err)}
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return TestResult{"API Endpoints", false, fmt.Sprintf("%s returned status %d", endpoint, resp.StatusCode)}
		}
	}

	return TestResult{"API Endpoints", true, "All API endpoints responding correctly"}
}

func testWebSocketConnection(baseURL string) TestResult {
	fmt.Println("🔍 Testing WebSocket connection...")

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/test_user_1"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return TestResult{"WebSocket Connection", false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()

	// Wait for welcome message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		return TestResult{"WebSocket Connection", false, fmt.Sprintf("No welcome message: %v", err)}
	}

	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		return TestResult{"WebSocket Connection", false, fmt.Sprintf("Invalid welcome message: %v", err)}
	}

	if wsMsg.Type != "welcome" {
		return TestResult{"WebSocket Connection", false, fmt.Sprintf("Expected welcome, got: %s", wsMsg.Type)}
	}

	return TestResult{"WebSocket Connection", true, "WebSocket connection established and welcome received"}
}

func testWebSocketMessaging(baseURL string) TestResult {
	fmt.Println("🔍 Testing WebSocket messaging...")

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/test_user_2"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return TestResult{"WebSocket Messaging", false, fmt.Sprintf("Connection failed: %v", err)}
	}
	defer conn.Close()

	// Skip welcome message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()

	// Send test message
	testMsg := map[string]interface{}{
		"type":    "test",
		"content": "Hello from E2E test",
	}

	msgBytes, _ := json.Marshal(testMsg)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		return TestResult{"WebSocket Messaging", false, fmt.Sprintf("Send failed: %v", err)}
	}

	// Wait for echo
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		return TestResult{"WebSocket Messaging", false, fmt.Sprintf("No response: %v", err)}
	}

	var wsMsg WSMessage
	if err := json.Unmarshal(response, &wsMsg); err != nil {
		return TestResult{"WebSocket Messaging", false, fmt.Sprintf("Invalid response: %v", err)}
	}

	if wsMsg.Type != "message_echo" {
		return TestResult{"WebSocket Messaging", false, fmt.Sprintf("Expected echo, got: %s", wsMsg.Type)}
	}

	return TestResult{"WebSocket Messaging", true, "WebSocket messaging working correctly"}
}

func testMultipleClients(baseURL string) TestResult {
	fmt.Println("🔍 Testing multiple client connections...")

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/api/v1/realtime/ws/"

	// Connect multiple clients
	var connections []*websocket.Conn
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("user_%d", i), nil)
		if err != nil {
			// Close existing connections
			for _, c := range connections {
				c.Close()
			}
			return TestResult{"Multiple Clients", false, fmt.Sprintf("Client %d connection failed: %v", i, err)}
		}
		connections = append(connections, conn)

		// Skip welcome message
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		conn.ReadMessage()
	}

	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}

	// Check server stats
	resp, err := http.Get(baseURL + "/api/v1/stats")
	if err != nil {
		return TestResult{"Multiple Clients", false, fmt.Sprintf("Stats check failed: %v", err)}
	}
	defer resp.Body.Close()

	return TestResult{"Multiple Clients", true, "Multiple client connections handled successfully"}
}
