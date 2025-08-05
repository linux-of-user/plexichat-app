package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	fmt.Println("🔌 Testing PlexiChat WebSocket Connection")
	fmt.Println("====================================================")

	// Connect to WebSocket
	fmt.Println("🔗 Connecting to WebSocket...")
	url := "ws://localhost:8000/api/v1/realtime/ws/test_user"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Printf("❌ Failed to connect to WebSocket: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("✅ WebSocket connected successfully")

	// Set up message handling
	done := make(chan struct{})
	messageCount := 0

	// Start reading messages
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("❌ WebSocket error: %v\n", err)
				}
				return
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				fmt.Printf("❌ Error parsing message: %v\n", err)
				continue
			}

			messageCount++
			fmt.Printf("📨 Received message %d: Type=%s\n", messageCount, wsMsg.Type)

			switch wsMsg.Type {
			case "welcome":
				fmt.Printf("   🎉 Welcome message: %v\n", wsMsg.Data["message"])
			case "message_echo":
				fmt.Printf("   🔄 Echo received: %v\n", wsMsg.Data["status"])
			case "broadcast":
				fmt.Printf("   📢 Broadcast from: %v\n", wsMsg.Data["from_user"])
			default:
				fmt.Printf("   📋 Data: %+v\n", wsMsg.Data)
			}
		}
	}()

	// Wait for welcome message
	fmt.Println("⏳ Waiting for welcome message...")
	time.Sleep(1 * time.Second)

	// Send test messages
	fmt.Println("\n📤 Sending test messages...")

	testMessages := []map[string]interface{}{
		{
			"type":      "test_message",
			"content":   "Hello from Go client!",
			"timestamp": time.Now().Unix(),
		},
		{
			"type": "ping",
			"data": "test_data",
		},
		{
			"type":    "user_message",
			"content": "This is a test message from the Go WebSocket client",
			"user":    "test_user",
		},
	}

	for i, msg := range testMessages {
		fmt.Printf("📤 Sending message %d...\n", i+1)

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("❌ Error marshaling message: %v\n", err)
			continue
		}

		err = conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			fmt.Printf("❌ Error sending message: %v\n", err)
			continue
		}

		fmt.Printf("✅ Message %d sent successfully\n", i+1)
		time.Sleep(500 * time.Millisecond) // Small delay between messages
	}

	// Wait for responses
	fmt.Println("\n⏳ Waiting for responses...")
	time.Sleep(2 * time.Second)

	// Send close message
	fmt.Println("\n🔚 Closing connection...")
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		fmt.Printf("❌ Error sending close message: %v\n", err)
	}

	// Wait for close or timeout
	select {
	case <-done:
		fmt.Println("✅ WebSocket closed gracefully")
	case <-time.After(1 * time.Second):
		fmt.Println("⏰ Timeout waiting for close")
	}

	fmt.Printf("\n📊 Test Summary:\n")
	fmt.Printf("   📨 Total messages received: %d\n", messageCount)
	fmt.Printf("   📤 Total messages sent: %d\n", len(testMessages))

	if messageCount > 0 {
		fmt.Println("\n🎉 WebSocket test completed successfully!")
		fmt.Println("🔌 Real-time communication is working")
	} else {
		fmt.Println("\n❌ WebSocket test failed - no messages received")
	}
}
