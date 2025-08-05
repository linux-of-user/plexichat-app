package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"plexichat-client/pkg/client"
)

// TestBasicClientFunctionality tests core client functionality
func TestBasicClientFunctionality(t *testing.T) {
	// Test client creation
	c := client.NewClient("http://localhost:8000")
	if c == nil {
		t.Fatal("Failed to create client")
	}

	// Test client configuration
	c.SetDebug(true)
	c.SetTimeout(30 * time.Second)

	if c.BaseURL != "http://localhost:8000" {
		t.Errorf("Expected BaseURL to be 'http://localhost:8000', got %s", c.BaseURL)
	}

	if !c.Debug {
		t.Error("Expected Debug to be true")
	}

	if c.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout to be 30s, got %v", c.HTTPClient.Timeout)
	}
}

// TestClientConfiguration tests client configuration methods
func TestClientConfiguration(t *testing.T) {
	c := client.NewClient("http://localhost:8000")

	// Test SetDebug
	c.SetDebug(true)
	if !c.Debug {
		t.Error("Expected Debug to be true")
	}

	// Test SetTimeout
	timeout := 45 * time.Second
	c.SetTimeout(timeout)
	if c.HTTPClient.Timeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, c.HTTPClient.Timeout)
	}

	// Test SetToken
	token := "test-token-123"
	c.SetToken(token)
	if c.Token != token {
		t.Errorf("Expected token to be %s, got %s", token, c.Token)
	}

	// Test SetAPIKey
	apiKey := "test-api-key-456"
	c.SetAPIKey(apiKey)
	if c.APIKey != apiKey {
		t.Errorf("Expected API key to be %s, got %s", apiKey, c.APIKey)
	}
}

// TestClientWithoutServer tests client behavior when server is not available
func TestClientWithoutServer(t *testing.T) {
	c := client.NewClient("http://localhost:9999") // Use a port that's likely not in use
	c.SetDebug(true)
	c.SetTimeout(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// This should fail since no server is running on port 9999
	_, err := c.Health(ctx)
	if err == nil {
		t.Error("Expected error when connecting to non-existent server")
	}

	t.Logf("Expected error occurred: %v", err)
}

// TestClientJSONHandling tests JSON marshaling and unmarshaling
func TestClientJSONHandling(t *testing.T) {
	c := client.NewClient("http://localhost:8000")

	testData := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
		"number":   42,
		"boolean":  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This will fail to connect, but should not fail due to JSON marshaling
	_, err := c.Post(ctx, "/test", testData)

	// We expect a connection error, not a JSON marshaling error
	if err != nil {
		errStr := err.Error()
		if len(errStr) > 0 {
			// Check that it's a connection error, not a JSON error
			connectionErrors := []string{"connection refused", "no such host", "timeout", "context deadline exceeded"}
			isConnectionError := false
			for _, connErr := range connectionErrors {
				if contains(errStr, connErr) {
					isConnectionError = true
					break
				}
			}
			if !isConnectionError {
				t.Errorf("Unexpected error type (should be connection error): %v", err)
			} else {
				t.Logf("Expected connection error: %v", err)
			}
		}
	}
}

// TestClientRetryLogic tests the retry mechanism
func TestClientRetryLogic(t *testing.T) {
	c := client.NewClient("http://invalid-host-that-does-not-exist:8000")
	c.SetDebug(true)
	c.MaxRetries = 2
	c.RetryDelay = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, err := c.Request(ctx, "GET", "/test", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected error for invalid host")
	}

	// Should have retried at least once, so elapsed time should be > retry delay
	if elapsed < c.RetryDelay {
		t.Errorf("Expected elapsed time to be at least %v, got %v", c.RetryDelay, elapsed)
	}

	t.Logf("Retry test completed in %v with error: %v", elapsed, err)
}

// TestServerConnectivity tests connectivity to a running server
func TestServerConnectivity(t *testing.T) {
	c := client.NewClient("http://localhost:8000")
	c.SetDebug(true)
	c.SetTimeout(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Test health endpoint
	health, err := c.Health(ctx)
	if err != nil {
		t.Logf("Health check failed (server may not be running): %v", err)
		return // Skip remaining tests if server is not available
	}

	t.Logf("Server health: %+v", health)

	// Test API root endpoint
	resp, err := c.Get(ctx, "/api/v1/")
	if err != nil {
		t.Logf("API root request failed: %v", err)
	} else {
		defer resp.Body.Close()
		t.Logf("API root status: %d", resp.StatusCode)
	}
}

// Helper function for case-insensitive string contains
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	// Convert to lowercase for case-insensitive comparison
	sLower := ""
	substrLower := ""

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			sLower += string(r + 32)
		} else {
			sLower += string(r)
		}
	}

	for _, r := range substr {
		if r >= 'A' && r <= 'Z' {
			substrLower += string(r + 32)
		} else {
			substrLower += string(r)
		}
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// BenchmarkClientCreation benchmarks client creation
func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := client.NewClient("http://localhost:8000")
		if c == nil {
			b.Fatal("Failed to create client")
		}
	}
}

// Main function for standalone execution
func main() {
	fmt.Println("🧪 Running PlexiChat Go Client Tests")
	fmt.Println("=====================================")

	// Run tests programmatically
	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"BasicClientFunctionality", TestBasicClientFunctionality},
		{"ClientConfiguration", TestClientConfiguration},
		{"ClientWithoutServer", TestClientWithoutServer},
		{"ClientJSONHandling", TestClientJSONHandling},
		{"ClientRetryLogic", TestClientRetryLogic},
		{"ServerConnectivity", TestServerConnectivity},
	}

	passed := 0
	total := len(tests)

	for _, test := range tests {
		fmt.Printf("🔍 Running %s... ", test.name)

		// Create a mock testing.T
		mockT := &testing.T{}

		// Run the test
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("❌ PANIC: %v\n", r)
				}
			}()

			test.fn(mockT)
		}()

		if mockT.Failed() {
			fmt.Println("❌ FAILED")
		} else {
			fmt.Println("✅ PASSED")
			passed++
		}
	}

	fmt.Println("=====================================")
	fmt.Printf("📊 Results: %d/%d tests passed (%.1f%%)\n",
		passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("🎉 All client tests passed!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Some client tests failed!")
		os.Exit(1)
	}
}
