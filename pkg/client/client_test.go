package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://api.example.com"
	client := NewClient(baseURL)

	if client.BaseURL != baseURL {
		t.Errorf("Expected BaseURL %s, got %s", baseURL, client.BaseURL)
	}

	if client.UserAgent != "PlexiChat-Go-Client/1.0" {
		t.Errorf("Expected UserAgent 'PlexiChat-Go-Client/1.0', got %s", client.UserAgent)
	}

	if client.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", client.MaxRetries)
	}

	if client.RetryDelay != 1*time.Second {
		t.Errorf("Expected RetryDelay 1s, got %v", client.RetryDelay)
	}
}

func TestClient_SetAPIKey(t *testing.T) {
	client := NewClient("https://api.example.com")
	apiKey := "test-api-key"

	client.SetAPIKey(apiKey)

	if client.APIKey != apiKey {
		t.Errorf("Expected APIKey %s, got %s", apiKey, client.APIKey)
	}
}

func TestClient_SetToken(t *testing.T) {
	client := NewClient("https://api.example.com")
	token := "test-jwt-token"

	client.SetToken(token)

	if client.Token != token {
		t.Errorf("Expected Token %s, got %s", token, client.Token)
	}
}

func TestClient_SetDebug(t *testing.T) {
	client := NewClient("https://api.example.com")

	client.SetDebug(true)

	if !client.Debug {
		t.Errorf("Expected Debug true, got %v", client.Debug)
	}
}

func TestClient_SetRetryConfig(t *testing.T) {
	client := NewClient("https://api.example.com")
	maxRetries := 5
	retryDelay := 2 * time.Second

	client.SetRetryConfig(maxRetries, retryDelay)

	if client.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", maxRetries, client.MaxRetries)
	}

	if client.RetryDelay != retryDelay {
		t.Errorf("Expected RetryDelay %v, got %v", retryDelay, client.RetryDelay)
	}
}

func TestClient_SetAdvancedRetryConfig(t *testing.T) {
	client := NewClient("http://example.com")

	config := RetryConfig{
		MaxRetries:    5,
		Delay:         500 * time.Millisecond,
		BackoffFactor: 1.5,
		MaxDelay:      10 * time.Second,
	}

	client.SetAdvancedRetryConfig(config)

	if client.RetryConfig.MaxRetries != 5 {
		t.Errorf("Expected RetryConfig.MaxRetries to be 5, got %d", client.RetryConfig.MaxRetries)
	}

	if client.RetryConfig.Delay != 500*time.Millisecond {
		t.Errorf("Expected RetryConfig.Delay to be 500ms, got %v", client.RetryConfig.Delay)
	}

	if client.RetryConfig.BackoffFactor != 1.5 {
		t.Errorf("Expected RetryConfig.BackoffFactor to be 1.5, got %f", client.RetryConfig.BackoffFactor)
	}

	if client.RetryConfig.MaxDelay != 10*time.Second {
		t.Errorf("Expected RetryConfig.MaxDelay to be 10s, got %v", client.RetryConfig.MaxDelay)
	}
}

func TestClient_CalculateRetryDelay(t *testing.T) {
	client := NewClient("http://example.com")

	config := RetryConfig{
		MaxRetries:    3,
		Delay:         100 * time.Millisecond,
		BackoffFactor: 2.0,
		MaxDelay:      1 * time.Second,
	}
	client.SetAdvancedRetryConfig(config)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at MaxDelay
	}

	for _, test := range tests {
		delay := client.calculateRetryDelay(test.attempt)
		if delay != test.expected {
			t.Errorf("For attempt %d, expected delay %v, got %v", test.attempt, test.expected, delay)
		}
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.Delay != time.Second {
		t.Errorf("Expected Delay to be 1s, got %v", config.Delay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}

	if config.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay to be 30s, got %v", config.MaxDelay)
	}
}

func TestClient_SetTimeout(t *testing.T) {
	client := NewClient("https://api.example.com")
	timeout := 60 * time.Second

	client.SetTimeout(timeout)

	if client.HTTPClient.Timeout != timeout {
		t.Errorf("Expected Timeout %v, got %v", timeout, client.HTTPClient.Timeout)
	}
}

func TestClient_Request_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("User-Agent") != "PlexiChat-Go-Client/1.0" {
			t.Errorf("Expected User-Agent PlexiChat-Go-Client/1.0, got %s", r.Header.Get("User-Agent"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	resp.Body.Close()
}

func TestClient_Request_WithAPIKey(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			t.Errorf("Expected X-API-Key test-api-key, got %s", apiKey)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetAPIKey("test-api-key")
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	resp.Body.Close()
}

func TestClient_Request_WithToken(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-jwt-token" {
			t.Errorf("Expected Authorization Bearer test-jwt-token, got %s", auth)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-jwt-token")
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	resp.Body.Close()
}

func TestClient_Request_Retry(t *testing.T) {
	attempts := 0

	// Create test server that fails first two attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(3, 10*time.Millisecond) // Fast retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	resp.Body.Close()
}

func TestClient_Request_RetryExhausted(t *testing.T) {
	attempts := 0

	// Create test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(2, 10*time.Millisecond) // Fast retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	if attempts != 3 { // Initial attempt + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	resp.Body.Close()
}

func TestClient_ParseResponse_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success", "data": {"id": 123}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var result map[string]interface{}
	err = client.ParseResponse(resp, &result)

	if err != nil {
		t.Errorf("ParseResponse failed: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got %v", result["message"])
	}
}

func TestClient_ParseResponse_Error(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "validation failed", "message": "Invalid input"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var result map[string]interface{}
	err = client.ParseResponse(resp, &result)

	if err == nil {
		t.Errorf("Expected error but got none")
	}

	if err.Error() != "API error (status 400): Invalid input" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestClient_Health(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "version": "1.0.0"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetRetryConfig(0, 0) // No retries for test

	ctx := context.Background()
	health, err := client.Health(ctx)

	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", health.Status)
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", health.Version)
	}
}
