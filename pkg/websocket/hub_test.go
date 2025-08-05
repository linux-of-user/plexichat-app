package websocket

import (
	"context"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	
	if hub.clients == nil {
		t.Error("Hub clients map not initialized")
	}
	
	if hub.channels == nil {
		t.Error("Hub channels map not initialized")
	}
	
	if hub.register == nil {
		t.Error("Hub register channel not initialized")
	}
	
	if hub.unregister == nil {
		t.Error("Hub unregister channel not initialized")
	}
	
	if hub.broadcast == nil {
		t.Error("Hub broadcast channel not initialized")
	}
}

func TestHub_GetStats(t *testing.T) {
	hub := NewHub()
	
	stats := hub.GetStats()
	
	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}
	
	if totalClients, ok := stats["total_clients"]; !ok || totalClients != 0 {
		t.Errorf("Expected total_clients to be 0, got %v", totalClients)
	}
	
	if totalChannels, ok := stats["total_channels"]; !ok || totalChannels != 0 {
		t.Errorf("Expected total_channels to be 0, got %v", totalChannels)
	}
	
	if _, ok := stats["timestamp"]; !ok {
		t.Error("Expected timestamp in stats")
	}
}

func TestHub_GetOnlineUsers(t *testing.T) {
	hub := NewHub()
	
	users := hub.GetOnlineUsers()
	if len(users) != 0 {
		t.Errorf("Expected 0 online users, got %d", len(users))
	}
}

func TestHub_GetChannelUsers(t *testing.T) {
	hub := NewHub()
	
	users := hub.GetChannelUsers("nonexistent")
	if len(users) != 0 {
		t.Errorf("Expected 0 users in nonexistent channel, got %d", len(users))
	}
}

func TestMessage_Types(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MessageTypeChat, "chat"},
		{MessageTypePresence, "presence"},
		{MessageTypeNotification, "notification"},
		{MessageTypeTyping, "typing"},
		{MessageTypeJoin, "join"},
		{MessageTypeLeave, "leave"},
		{MessageTypeError, "error"},
		{MessageTypePing, "ping"},
		{MessageTypePong, "pong"},
	}
	
	for _, test := range tests {
		if string(test.msgType) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.msgType))
		}
	}
}

func TestHub_SendToChannel(t *testing.T) {
	hub := NewHub()
	
	// Start hub in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go hub.Run(ctx)
	
	// Give hub time to start
	time.Sleep(10 * time.Millisecond)
	
	// Send message to channel
	message := Message{
		Type: MessageTypeChat,
		Data: "test message",
	}
	
	// This should not panic
	hub.SendToChannel("test-channel", message)
	
	// Give time for message to be processed
	time.Sleep(10 * time.Millisecond)
}

func TestHub_SendToUser(t *testing.T) {
	hub := NewHub()
	
	// Send message to non-existent user (should not panic)
	message := Message{
		Type: MessageTypeNotification,
		Data: "test notification",
	}
	
	hub.SendToUser("nonexistent-user", message)
}

func TestClient_Channels(t *testing.T) {
	client := &Client{
		ID:       "test-client",
		UserID:   "test-user",
		Username: "testuser",
		Channels: make(map[string]bool),
	}
	
	if len(client.Channels) != 0 {
		t.Errorf("Expected 0 channels, got %d", len(client.Channels))
	}
	
	// Add channel
	client.Channels["test-channel"] = true
	
	if len(client.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(client.Channels))
	}
	
	if !client.Channels["test-channel"] {
		t.Error("Expected client to be in test-channel")
	}
}

func TestHub_JoinLeaveChannel_WithoutClient(t *testing.T) {
	hub := NewHub()
	
	// Try to join channel with non-existent client
	err := hub.JoinChannel("nonexistent", "test-channel")
	if err == nil {
		t.Error("Expected error when joining channel with non-existent client")
	}
	
	// Try to leave channel with non-existent client
	err = hub.LeaveChannel("nonexistent", "test-channel")
	if err == nil {
		t.Error("Expected error when leaving channel with non-existent client")
	}
}

func TestMessage_Creation(t *testing.T) {
	now := time.Now()
	
	message := Message{
		Type:      MessageTypeChat,
		Data:      "Hello, world!",
		Timestamp: now,
		UserID:    "user123",
		ChannelID: "channel456",
		MessageID: "msg789",
	}
	
	if message.Type != MessageTypeChat {
		t.Errorf("Expected type %s, got %s", MessageTypeChat, message.Type)
	}
	
	if message.Data != "Hello, world!" {
		t.Errorf("Expected data 'Hello, world!', got %v", message.Data)
	}
	
	if message.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got %s", message.UserID)
	}
	
	if message.ChannelID != "channel456" {
		t.Errorf("Expected ChannelID 'channel456', got %s", message.ChannelID)
	}
	
	if message.MessageID != "msg789" {
		t.Errorf("Expected MessageID 'msg789', got %s", message.MessageID)
	}
}
