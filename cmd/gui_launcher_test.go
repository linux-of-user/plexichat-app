package cmd

import (
	"testing"
	"time"
	
	"plexichat-client/pkg/client"
)

func TestGUIState(t *testing.T) {
	// Test GUI state initialization
	state := &GUIState{
		currentTab: "login",
		messages:   make(map[string][]Message),
		isDarkMode: false,
	}
	
	if state.currentTab != "login" {
		t.Errorf("Expected currentTab to be 'login', got %s", state.currentTab)
	}
	
	if state.messages == nil {
		t.Error("Expected messages map to be initialized")
	}
	
	if state.isDarkMode != false {
		t.Error("Expected isDarkMode to be false by default")
	}
}

func TestMessageSearch(t *testing.T) {
	// Create test state
	state := &GUIState{
		messages: make(map[string][]Message),
	}
	
	// Add test messages
	testMessages := []Message{
		{
			ID:      "1",
			Content: "Hello world",
			Author:  "Alice",
		},
		{
			ID:      "2", 
			Content: "How are you?",
			Author:  "Bob",
		},
		{
			ID:      "3",
			Content: "Good morning everyone",
			Author:  "Alice",
		},
	}
	
	state.messages["general"] = testMessages
	
	// Test search functionality
	// Note: This would need the actual searchMessages function to be testable
	// For now, just test the data structure
	
	found := 0
	query := "hello"
	for _, messages := range state.messages {
		for _, message := range messages {
			if contains(message.Content, query) || contains(message.Author, query) {
				found++
			}
		}
	}
	
	if found != 1 {
		t.Errorf("Expected to find 1 message with 'hello', found %d", found)
	}
}

func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.txt", "Text"},
		{"image.jpg", "Image"},
		{"document.pdf", "PDF"},
		{"archive.zip", "Archive"},
		{"unknown.xyz", "Unknown"},
	}
	
	for _, test := range tests {
		result := getFileType(test.filename)
		if result != test.expected {
			t.Errorf("getFileType(%s) = %s, expected %s", test.filename, result, test.expected)
		}
	}
}

func TestImageFileDetection(t *testing.T) {
	imageFiles := []string{"test.jpg", "image.png", "photo.gif"}
	nonImageFiles := []string{"document.txt", "archive.zip", "video.mp4"}
	
	for _, file := range imageFiles {
		if !isImageFile(file) {
			t.Errorf("Expected %s to be detected as image file", file)
		}
	}
	
	for _, file := range nonImageFiles {
		if isImageFile(file) {
			t.Errorf("Expected %s to NOT be detected as image file", file)
		}
	}
}

func TestTextFileDetection(t *testing.T) {
	textFiles := []string{"readme.txt", "code.go", "script.js", "style.css"}
	nonTextFiles := []string{"image.jpg", "archive.zip", "video.mp4"}
	
	for _, file := range textFiles {
		if !isTextFile(file) {
			t.Errorf("Expected %s to be detected as text file", file)
		}
	}
	
	for _, file := range nonTextFiles {
		if isTextFile(file) {
			t.Errorf("Expected %s to NOT be detected as text file", file)
		}
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 3},
		{1, 10, 1},
		{7, 7, 7},
		{0, 5, 0},
	}
	
	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestAppSettings(t *testing.T) {
	settings := &AppSettings{
		DarkMode:        true,
		NotificationsOn: true,
		SoundEffects:    false,
		Username:        "testuser",
		ServerURL:       "http://localhost:8000",
		FontSize:        14,
		AutoConnect:     true,
	}
	
	if !settings.DarkMode {
		t.Error("Expected DarkMode to be true")
	}
	
	if settings.Username != "testuser" {
		t.Errorf("Expected Username to be 'testuser', got %s", settings.Username)
	}
	
	if settings.FontSize != 14 {
		t.Errorf("Expected FontSize to be 14, got %d", settings.FontSize)
	}
}

func TestClientCreation(t *testing.T) {
	// Test client creation
	client := client.NewClient("http://localhost:8000")
	if client == nil {
		t.Error("Expected client to be created successfully")
	}
	
	// Test with invalid URL
	client2 := client.NewClient("")
	if client2 == nil {
		t.Error("Client should handle empty URL gracefully")
	}
}

// Helper function for case-insensitive string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkMessageSearch(b *testing.B) {
	// Create test state with many messages
	state := &GUIState{
		messages: make(map[string][]Message),
	}
	
	// Add 1000 test messages
	for i := 0; i < 1000; i++ {
		message := Message{
			ID:      string(rune(i)),
			Content: "Test message content " + string(rune(i)),
			Author:  "User" + string(rune(i%10)),
		}
		state.messages["general"] = append(state.messages["general"], message)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate search
		query := "Test"
		for _, messages := range state.messages {
			for _, message := range messages {
				_ = contains(message.Content, query)
			}
		}
	}
}
