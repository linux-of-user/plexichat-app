# API Documentation

This document describes how to use the PlexiChat client API programmatically.

## Client Package

The `pkg/client` package provides a robust HTTP client for interacting with the PlexiChat API.

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "plexichat-client/pkg/client"
)

func main() {
    // Create client
    c := client.NewClient("http://localhost:8000")
    
    // Configure authentication
    c.SetAPIKey("your-api-key")
    // OR
    c.SetToken("your-jwt-token")
    
    // Make requests
    ctx := context.Background()
    resp, err := c.Get(ctx, "/api/channels")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    
    // Parse response
    var channels []map[string]interface{}
    err = c.ParseResponse(resp, &channels)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d channels\n", len(channels))
}
```

### Client Configuration

#### Basic Configuration

```go
client := client.NewClient("http://localhost:8000")

// Set authentication
client.SetAPIKey("your-api-key")
client.SetToken("your-jwt-token")

// Configure timeouts
client.SetTimeout(30 * time.Second)

// Enable debug mode
client.SetDebug(true)

// Configure basic retry
client.SetRetryConfig(5, 2*time.Second)
```

#### Advanced Retry Configuration

```go
// Configure exponential backoff
retryConfig := client.RetryConfig{
    MaxRetries:    5,
    Delay:         500 * time.Millisecond,
    BackoffFactor: 2.0,  // Double delay each retry
    MaxDelay:      30 * time.Second,
}
client.SetAdvancedRetryConfig(retryConfig)
```

### HTTP Methods

#### GET Requests

```go
// Simple GET
resp, err := client.Get(ctx, "/api/channels")

// GET with query parameters
resp, err := client.Get(ctx, "/api/messages?channel=1&limit=50")
```

#### POST Requests

```go
// POST with JSON body
data := map[string]interface{}{
    "content": "Hello, world!",
    "channel": 1,
}
resp, err := client.Post(ctx, "/api/messages", data)

// POST with struct
type Message struct {
    Content string `json:"content"`
    Channel int    `json:"channel"`
}
msg := Message{Content: "Hello!", Channel: 1}
resp, err := client.Post(ctx, "/api/messages", msg)
```

#### PUT and DELETE Requests

```go
// PUT request
updateData := map[string]string{"name": "New Channel Name"}
resp, err := client.Put(ctx, "/api/channels/1", updateData)

// DELETE request
resp, err := client.Delete(ctx, "/api/channels/1")
```

### Response Handling

#### Parse JSON Response

```go
// Parse into map
var result map[string]interface{}
err = client.ParseResponse(resp, &result)

// Parse into struct
type Channel struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
var channel Channel
err = client.ParseResponse(resp, &channel)

// Parse into slice
var channels []Channel
err = client.ParseResponse(resp, &channels)
```

#### Handle Errors

```go
resp, err := client.Get(ctx, "/api/channels")
if err != nil {
    // Network or client error
    log.Printf("Request failed: %v", err)
    return
}

if resp.StatusCode >= 400 {
    // HTTP error
    var errorResp map[string]interface{}
    client.ParseResponse(resp, &errorResp)
    log.Printf("API error: %v", errorResp)
    return
}
```

### File Uploads

```go
// Upload file
file, err := os.Open("document.pdf")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.UploadFile(ctx, "/api/files", "file", "document.pdf", file)
if err != nil {
    log.Fatal(err)
}
```

### Health Checks

```go
// Check server health
healthy, err := client.Health(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
} else if healthy {
    log.Println("Server is healthy")
} else {
    log.Println("Server is unhealthy")
}
```

## WebSocket Package

The `pkg/websocket` package provides real-time communication capabilities.

### Hub Usage

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "plexichat-client/pkg/websocket"
)

func main() {
    // Create WebSocket hub
    hub := websocket.NewHub()
    
    // Start hub
    ctx := context.Background()
    go hub.Run(ctx)
    
    // Handle WebSocket connections
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        // Extract user info from request (authentication)
        userID := getUserID(r)
        username := getUsername(r)
        
        hub.HandleWebSocket(w, r, userID, username)
    })
    
    // Start server
    log.Println("WebSocket server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Message Types

```go
// Create different message types
chatMsg := websocket.Message{
    Type:      websocket.MessageTypeChat,
    Data:      "Hello, everyone!",
    UserID:    "user123",
    ChannelID: "general",
}

notificationMsg := websocket.Message{
    Type: websocket.MessageTypeNotification,
    Data: map[string]string{
        "title": "New User Joined",
        "body":  "Alice has joined the channel",
    },
}

typingMsg := websocket.Message{
    Type:      websocket.MessageTypeTyping,
    Data:      map[string]bool{"typing": true},
    UserID:    "user123",
    ChannelID: "general",
}
```

### Channel Management

```go
// Join channel
err := hub.JoinChannel("client123", "general")
if err != nil {
    log.Printf("Failed to join channel: %v", err)
}

// Leave channel
err = hub.LeaveChannel("client123", "general")
if err != nil {
    log.Printf("Failed to leave channel: %v", err)
}

// Send message to channel
hub.SendToChannel("general", chatMsg)

// Send message to specific user
hub.SendToUser("user123", notificationMsg)
```

### Hub Statistics

```go
// Get hub statistics
stats := hub.GetStats()
fmt.Printf("Connected clients: %v\n", stats["total_clients"])
fmt.Printf("Active channels: %v\n", stats["total_channels"])

// Get online users
users := hub.GetOnlineUsers()
fmt.Printf("Online users: %v\n", users)

// Get users in specific channel
channelUsers := hub.GetChannelUsers("general")
fmt.Printf("Users in #general: %v\n", channelUsers)
```

## Security Package

The `pkg/security` package provides input validation and security features.

### Validation

```go
package main

import (
    "fmt"
    "plexichat-client/pkg/security"
)

func main() {
    validator := security.NewValidator()
    
    // Validate email
    validator.ValidateEmail("email", "user@example.com")
    
    // Validate username
    validator.ValidateUsername("username", "john_doe")
    
    // Validate password
    validator.ValidatePassword("password", "SecurePass123!")
    
    // Validate channel name
    validator.ValidateChannelName("channel", "general-chat")
    
    // Validate message content
    validator.ValidateMessageContent("message", "Hello, world!")
    
    // Check for errors
    if validator.HasErrors() {
        errors := validator.GetErrors()
        for field, fieldErrors := range errors {
            for _, err := range fieldErrors {
                fmt.Printf("%s: %s (%s)\n", field, err.Message, err.Code)
            }
        }
    } else {
        fmt.Println("All validations passed!")
    }
}
```

### File Upload Validation

```go
// Validate file upload
validator.ValidateFileUpload("file", "document.pdf", 1024*1024) // 1MB file

// Check validation results
if validator.HasErrors() {
    errors := validator.GetErrors()
    if fileErrors, ok := errors["file"]; ok {
        for _, err := range fileErrors {
            fmt.Printf("File validation error: %s\n", err.Message)
        }
    }
}
```

## Logging Package

The `pkg/logging` package provides ASCII-only logging with colorization.

### Basic Logging

```go
package main

import "plexichat-client/pkg/logging"

func main() {
    // Use global logger
    logging.Debug("Debug message")
    logging.Info("Info message")
    logging.Warn("Warning message")
    logging.Error("Error message")
    // logging.Fatal("Fatal message") // This exits the program
}
```

### Custom Logger

```go
import (
    "os"
    "plexichat-client/pkg/logging"
)

func main() {
    // Create custom logger
    logger := logging.NewLogger(
        logging.INFO,    // Log level
        os.Stdout,       // Output writer
        true,            // Enable colors
    )
    
    // Configure logger
    logger.SetLevel(logging.DEBUG)
    logger.SetColorized(false)
    logger.SetPrefix("MYAPP")
    logger.SetTimeFormat("15:04:05")
    
    // Use logger
    logger.Info("Application started")
    logger.Debug("Debug information")
    logger.Error("Something went wrong")
}
```

### Log Levels

```go
// Set global log level
logging.SetGlobalLevel(logging.DEBUG)

// Parse log level from string
level := logging.ParseLogLevel("debug")
logging.SetGlobalLevel(level)

// Available levels
levels := []logging.LogLevel{
    logging.DEBUG,
    logging.INFO,
    logging.WARN,
    logging.ERROR,
    logging.FATAL,
}
```

## Error Handling

### Client Errors

```go
resp, err := client.Get(ctx, "/api/channels")
if err != nil {
    // Check error type
    switch {
    case strings.Contains(err.Error(), "connection refused"):
        log.Println("Server is not running")
    case strings.Contains(err.Error(), "timeout"):
        log.Println("Request timed out")
    case strings.Contains(err.Error(), "unauthorized"):
        log.Println("Authentication failed")
    default:
        log.Printf("Unknown error: %v", err)
    }
    return
}
```

### Validation Errors

```go
validator := security.NewValidator()
validator.ValidateEmail("email", "invalid-email")

if validator.HasErrors() {
    errors := validator.GetErrors()
    for field, fieldErrors := range errors {
        for _, err := range fieldErrors {
            switch err.Code {
            case "INVALID_EMAIL":
                fmt.Println("Please enter a valid email address")
            case "EMAIL_TOO_LONG":
                fmt.Println("Email address is too long")
            default:
                fmt.Printf("Validation error: %s\n", err.Message)
            }
        }
    }
}
```

## Best Practices

### Context Usage

Always use context for cancellation and timeouts:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use context in requests
resp, err := client.Get(ctx, "/api/channels")
```

### Resource Management

Always close response bodies:

```go
resp, err := client.Get(ctx, "/api/channels")
if err != nil {
    return err
}
defer resp.Body.Close() // Important!

// Process response...
```

### Error Handling

Handle errors appropriately:

```go
resp, err := client.Post(ctx, "/api/messages", data)
if err != nil {
    // Log error and return
    logging.Error("Failed to send message: %v", err)
    return err
}

if resp.StatusCode >= 400 {
    // Handle HTTP errors
    var apiError map[string]interface{}
    client.ParseResponse(resp, &apiError)
    return fmt.Errorf("API error: %v", apiError)
}
```

### Logging

Use appropriate log levels:

```go
logging.Debug("Detailed debugging info")     // Development only
logging.Info("General information")          // Normal operation
logging.Warn("Something unexpected")         // Potential issues
logging.Error("Error occurred")              // Errors that don't stop execution
logging.Fatal("Critical error")              // Errors that stop execution
```
