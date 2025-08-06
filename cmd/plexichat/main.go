package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"plexichat-client/pkg/client"
)

const (
	version          = "b.1.1-97"
	defaultServerURL = "http://localhost:8000"
	defaultTimeout   = 30 * time.Second
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		showVersion()
	case "help", "--help", "-h":
		showHelp()
	case "health":
		checkHealth()
	case "login":
		handleLogin()
	case "users":
		listUsers()
	case "messages":
		listMessages()
	case "send":
		sendMessage()
	case "upload":
		uploadFile()
	case "test":
		runTests()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Use 'plexichat help' for available commands.")
		os.Exit(1)
	}
}

func showVersion() {
	fmt.Printf("PlexiChat CLI v%s\n", version)
	fmt.Println("A command-line client for PlexiChat server")
}

func showHelp() {
	fmt.Printf("PlexiChat CLI v%s\n\n", version)
	fmt.Println("USAGE:")
	fmt.Println("  plexichat <command> [options]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  version     Show version information")
	fmt.Println("  help        Show this help message")
	fmt.Println("  health      Check server health")
	fmt.Println("  login       Login to server")
	fmt.Println("  users       List users")
	fmt.Println("  messages    List messages")
	fmt.Println("  send        Send a message")
	fmt.Println("  upload      Upload a file")
	fmt.Println("  test        Run connectivity tests")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  PLEXICHAT_SERVER_URL    Server URL (default: http://localhost:8000)")
	fmt.Println("  PLEXICHAT_TOKEN         Authentication token")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  plexichat health")
	fmt.Println("  plexichat login admin password123")
	fmt.Println("  plexichat send \"Hello, world!\"")
	fmt.Println("  plexichat test")
}

func getServerURL() string {
	if url := os.Getenv("PLEXICHAT_SERVER_URL"); url != "" {
		return url
	}
	return defaultServerURL
}

func getToken() string {
	return os.Getenv("PLEXICHAT_TOKEN")
}

func createClient() *client.Client {
	c := client.NewClient(getServerURL())
	c.SetTimeout(defaultTimeout)

	if token := getToken(); token != "" {
		c.SetToken(token)
	}

	return c
}

func checkHealth() {
	fmt.Println("Checking server health...")

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := c.Health(ctx)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Server is healthy\n")
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   Version: %s\n", health.Version)
	fmt.Printf("   Uptime: %s\n", health.Uptime)
	if len(health.Checks) > 0 {
		fmt.Printf("   Checks: %v\n", health.Checks)
	}
}

func handleLogin() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: plexichat login <username> <password>")
		os.Exit(1)
	}

	username := os.Args[2]
	password := os.Args[3]

	fmt.Printf("Logging in as %s...\n", username)

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	loginResp, err := c.Login(ctx, username, password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Login successful\n")
	if len(loginResp.AccessToken) > 20 {
		fmt.Printf("   Token: %s...\n", loginResp.AccessToken[:20])
	} else {
		fmt.Printf("   Token: %s\n", loginResp.AccessToken)
	}
	if !loginResp.ExpiresAt.IsZero() {
		fmt.Printf("   Expires: %s\n", loginResp.ExpiresAt.Format(time.RFC3339))
	}
	if loginResp.TwoFARequired {
		fmt.Printf("   2FA Required: %v\n", loginResp.TwoFARequired)
		fmt.Printf("   Available methods: %v\n", loginResp.Methods)
	}

	fmt.Println("\nTo use this token in future commands, set:")
	fmt.Printf("export PLEXICHAT_TOKEN=%s\n", loginResp.AccessToken)
}

func listUsers() {
	fmt.Println("Listing users...")

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/users/")
	if err != nil {
		fmt.Printf("Failed to list users: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.ParseResponse(resp, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Users endpoint response:\n")
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

func listMessages() {
	fmt.Println("Listing messages...")

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/messages/")
	if err != nil {
		fmt.Printf("Failed to list messages: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := c.ParseResponse(resp, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Messages endpoint response:\n")
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

func sendMessage() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: plexichat send <message>")
		os.Exit(1)
	}

	message := strings.Join(os.Args[2:], " ")
	fmt.Printf("Sending message: %s\n", message)

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"content":    message,
		"channel_id": "general",
	}

	resp, err := c.Post(ctx, "/api/v1/messages/", payload)
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		fmt.Printf("Message sent successfully\n")
	} else {
		var result map[string]interface{}
		c.ParseResponse(resp, &result)
		fmt.Printf("⚠️  Message endpoint response (status %d):\n", resp.StatusCode)
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonData))
	}
}

func uploadFile() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: plexichat upload <file_path>")
		os.Exit(1)
	}

	filePath := os.Args[2]
	fmt.Printf("Uploading file: %s\n", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s\n", filePath)
		os.Exit(1)
	}

	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uploadResp, err := c.UploadFile(ctx, "/api/v1/files/upload", filePath)
	if err != nil {
		fmt.Printf("Failed to upload file: %v\n", err)
		os.Exit(1)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode == 200 || uploadResp.StatusCode == 201 {
		fmt.Printf("File uploaded successfully\n")
		fmt.Printf("   Status: %d\n", uploadResp.StatusCode)

		// Try to parse response
		var result map[string]interface{}
		if err := c.ParseResponse(uploadResp, &result); err == nil {
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonData))
		}
	} else {
		fmt.Printf("⚠️  Upload response (status %d)\n", uploadResp.StatusCode)
	}
}

func runTests() {
	fmt.Println("Running connectivity tests...")
	fmt.Println(strings.Repeat("=", 50))

	c := createClient()
	c.SetDebug(true)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Server Health", testHealth},
		{"API Root", testAPIRoot},
		{"Auth Endpoints", testAuthEndpoints},
		{"Users Endpoint", testUsersEndpoint},
		{"Messages Endpoint", testMessagesEndpoint},
		{"Files Endpoint", testFilesEndpoint},
	}

	passed := 0
	total := len(tests)

	for _, test := range tests {
		fmt.Printf("Testing %s... ", test.name)

		if err := test.fn(); err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("PASSED\n")
			passed++
		}
	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Results: %d/%d tests passed (%.1f%%)\n",
		passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("All tests passed!")
	} else {
		fmt.Printf("%d test(s) failed\n", total-passed)
		os.Exit(1)
	}
}

func testHealth() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Health(ctx)
	return err
}

func testAPIRoot() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testAuthEndpoints() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/auth/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testUsersEndpoint() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/users/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept any non-404 status (might be 401 for auth required)
	if resp.StatusCode == 404 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

func testMessagesEndpoint() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/messages/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept any non-404 status
	if resp.StatusCode == 404 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

func testFilesEndpoint() error {
	c := createClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "/api/v1/files/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept any non-404 status
	if resp.StatusCode == 404 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}
