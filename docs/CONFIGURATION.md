# Configuration Guide

This guide covers all configuration options for the PlexiChat client.

## Configuration File

The client uses a YAML configuration file located at:
- **Linux/macOS**: `~/.plexichat-client.yaml`
- **Windows**: `%USERPROFILE%\.plexichat-client.yaml`

### Initialize Configuration

Create a default configuration file:

```bash
plexichat-cli config init
```

This creates a configuration file with sensible defaults.

## Configuration Options

### Server Settings

```yaml
# PlexiChat server URL
url: "http://localhost:8000"

# Request timeout
timeout: "30s"

# Maximum retry attempts
retries: 3

# Maximum concurrent requests
concurrent_requests: 10
```

### Authentication

```yaml
# API key for authentication (optional)
api_key: ""

# JWT token for authentication (optional)
token: ""

# Refresh token (optional)
refresh_token: ""
```

### Chat Settings

```yaml
chat:
  # Default room/channel ID to join
  default_room: 1
  
  # Number of messages to load from history
  message_history_limit: 50
  
  # Automatically reconnect on disconnect
  auto_reconnect: true
  
  # WebSocket ping interval
  ping_interval: "30s"
```

### Security Settings

```yaml
security:
  # Timeout for security tests
  test_timeout: "60s"
  
  # Timeout for security scans
  scan_timeout: "300s"
  
  # Maximum concurrent security tests
  max_concurrent_tests: 5
  
  # Report format for security results
  report_format: "json"
```

### Benchmark Settings

```yaml
benchmark:
  # Default duration for benchmark tests
  default_duration: "30s"
  
  # Default number of concurrent connections
  default_concurrent: 10
  
  # Target response time
  response_time_target: "1ms"
  
  # Number of microsecond samples
  microsecond_samples: 1000
```

### Logging Settings

```yaml
logging:
  # Log level: debug, info, warn, error, fatal
  level: "info"
  
  # Log format: text, json
  format: "text"
```

### UI Settings

```yaml
# Enable colored output
color: true

# Output format for CLI commands
format: "table"

# Enable verbose output
verbose: false
```

### Feature Flags

```yaml
features:
  # Enable experimental commands
  experimental_commands: false
  
  # Enable beta features
  beta_features: false
  
  # Enable advanced security features
  advanced_security: true
  
  # Enable performance monitoring
  performance_monitoring: true
```

## Environment Variables

You can override any configuration option using environment variables. The format is `PLEXICHAT_<SECTION>_<OPTION>` in uppercase.

### Common Environment Variables

```bash
# Server URL
export PLEXICHAT_URL="https://chat.example.com"

# Authentication
export PLEXICHAT_API_KEY="your-api-key"
export PLEXICHAT_TOKEN="your-jwt-token"

# Logging
export PLEXICHAT_LOGGING_LEVEL="debug"
export PLEXICHAT_VERBOSE="true"

# Chat settings
export PLEXICHAT_CHAT_DEFAULT_ROOM="5"
export PLEXICHAT_CHAT_AUTO_RECONNECT="false"
```

### Nested Configuration

For nested configuration options, use underscores:

```bash
# chat.default_room
export PLEXICHAT_CHAT_DEFAULT_ROOM="1"

# security.test_timeout
export PLEXICHAT_SECURITY_TEST_TIMEOUT="120s"

# logging.level
export PLEXICHAT_LOGGING_LEVEL="debug"
```

## Command-Line Flags

Most configuration options can be overridden with command-line flags:

### Global Flags

```bash
# Server URL
--url "http://localhost:8000"

# Enable debug mode
--debug

# Set log level
--log-level debug

# Enable verbose output
--verbose

# Disable colored output
--no-color

# Set output format
--format json

# Set timeout
--timeout 60s

# Set retry count
--retries 5
```

### Examples

```bash
# Connect to different server with debug logging
plexichat-cli --url https://chat.example.com --debug chat

# Send message with custom timeout
plexichat-cli --timeout 10s send "Hello, world!"

# List channels with JSON output
plexichat-cli --format json channels list

# Join channel with verbose output
plexichat-cli --verbose channels join general
```

## Configuration Management Commands

### View Configuration

```bash
# Show all configuration
plexichat-cli config show

# Show configuration including sensitive values
plexichat-cli config show --secrets

# Get specific value
plexichat-cli config get url
plexichat-cli config get chat.default_room
```

### Modify Configuration

```bash
# Set server URL
plexichat-cli config set url "https://chat.example.com"

# Set log level
plexichat-cli config set logging.level "debug"

# Enable auto-reconnect
plexichat-cli config set chat.auto_reconnect true

# Set API key
plexichat-cli config set api_key "your-api-key"
```

### Configuration File Management

```bash
# Create default configuration
plexichat-cli config init

# Force overwrite existing configuration
plexichat-cli config init --force

# Validate configuration
plexichat-cli config validate

# Edit configuration in default editor
plexichat-cli config edit

# Backup configuration
plexichat-cli config backup

# Backup to specific file
plexichat-cli config backup --output config-backup.yaml

# Restore from backup
plexichat-cli config restore config-backup.yaml
```

## Configuration Precedence

Configuration values are applied in the following order (highest to lowest precedence):

1. **Command-line flags**
2. **Environment variables**
3. **Configuration file**
4. **Default values**

This means command-line flags will override environment variables, which will override the configuration file, which will override default values.

## Advanced Configuration

### Custom Retry Configuration

```yaml
# Advanced retry configuration with exponential backoff
retry_config:
  max_retries: 5
  delay: "500ms"
  backoff_factor: 2.0
  max_delay: "30s"
```

### Proxy Settings

```yaml
# HTTP proxy configuration
proxy:
  enabled: true
  url: "http://proxy.example.com:8080"
  username: "proxy-user"
  password: "proxy-pass"
```

### TLS Configuration

```yaml
# TLS/SSL settings
tls:
  enabled: true
  verify_certificates: true
  client_cert: "/path/to/client.crt"
  client_key: "/path/to/client.key"
  ca_cert: "/path/to/ca.crt"
```

## Troubleshooting Configuration

### Common Issues

1. **Configuration file not found**
   ```bash
   plexichat-cli config init
   ```

2. **Invalid YAML syntax**
   ```bash
   plexichat-cli config validate
   ```

3. **Permission denied**
   - Check file permissions on configuration directory
   - Ensure user has write access to home directory

4. **Environment variables not working**
   - Verify variable names use correct format: `PLEXICHAT_SECTION_OPTION`
   - Check that variables are exported: `export PLEXICHAT_URL="..."`

### Debug Configuration

Enable debug mode to see which configuration values are being used:

```bash
plexichat-cli --debug config show
```

This will show:
- Which configuration file is being used
- Which environment variables are set
- Final resolved configuration values

### Configuration Validation

Validate your configuration file:

```bash
plexichat-cli config validate
```

This checks:
- YAML syntax
- Required fields
- Value formats (URLs, durations, etc.)
- Logical constraints

## Examples

### Development Configuration

```yaml
url: "http://localhost:8000"
logging:
  level: "debug"
verbose: true
chat:
  auto_reconnect: true
features:
  experimental_commands: true
  beta_features: true
```

### Production Configuration

```yaml
url: "https://chat.company.com"
timeout: "10s"
retries: 5
logging:
  level: "info"
  format: "json"
security:
  advanced_security: true
features:
  experimental_commands: false
  beta_features: false
  performance_monitoring: true
```

### High-Performance Configuration

```yaml
url: "https://chat.example.com"
concurrent_requests: 50
timeout: "5s"
retries: 3
retry_config:
  max_retries: 3
  delay: "100ms"
  backoff_factor: 1.5
  max_delay: "5s"
benchmark:
  default_concurrent: 100
  response_time_target: "500ms"
```
