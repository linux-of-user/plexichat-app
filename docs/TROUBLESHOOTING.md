# Troubleshooting Guide

This guide helps you diagnose and fix common issues with the PlexiChat client.

## Common Issues

### Connection Issues

#### "Connection Refused" Error

**Symptoms:**
```
Error: dial tcp [::1]:8000: connectex: No connection could be made because the target machine actively refused it.
```

**Causes:**
- PlexiChat server is not running
- Wrong server URL in configuration
- Firewall blocking connection
- Network connectivity issues

**Solutions:**

1. **Check server status:**
   ```bash
   # Test if server is reachable
   curl http://localhost:8000/health
   
   # Or use the client health check
   plexichat-cli health
   ```

2. **Verify configuration:**
   ```bash
   plexichat-cli config get url
   ```

3. **Update server URL:**
   ```bash
   plexichat-cli config set url "http://your-server:8000"
   ```

4. **Check firewall settings:**
   - Ensure port 8000 (or your server port) is not blocked
   - Add exception for PlexiChat client if needed

#### "Timeout" Errors

**Symptoms:**
```
Error: context deadline exceeded
```

**Solutions:**

1. **Increase timeout:**
   ```bash
   plexichat-cli config set timeout "60s"
   ```

2. **Check network latency:**
   ```bash
   ping your-server-hostname
   ```

3. **Use command-line override:**
   ```bash
   plexichat-cli --timeout 30s chat
   ```

### Authentication Issues

#### "Unauthorized" Error

**Symptoms:**
```
Error: API error (status 401): Unauthorized
```

**Solutions:**

1. **Check API key/token:**
   ```bash
   plexichat-cli config get api_key
   plexichat-cli config get token
   ```

2. **Set authentication:**
   ```bash
   # Using API key
   plexichat-cli config set api_key "your-api-key"
   
   # Using JWT token
   plexichat-cli config set token "your-jwt-token"
   ```

3. **Login again:**
   ```bash
   plexichat-cli auth login
   ```

#### "Forbidden" Error

**Symptoms:**
```
Error: API error (status 403): Forbidden
```

**Causes:**
- Insufficient permissions
- Expired token
- Account suspended

**Solutions:**

1. **Check account status with administrator**
2. **Refresh authentication:**
   ```bash
   plexichat-cli auth refresh
   ```

### Configuration Issues

#### "Configuration file not found"

**Symptoms:**
```
Error: no configuration file found
```

**Solutions:**

1. **Initialize configuration:**
   ```bash
   plexichat-cli config init
   ```

2. **Check file location:**
   ```bash
   # Linux/macOS
   ls -la ~/.plexichat-client.yaml
   
   # Windows
   dir %USERPROFILE%\.plexichat-client.yaml
   ```

#### "Invalid YAML syntax"

**Symptoms:**
```
Error: invalid YAML syntax: yaml: line 5: found character that cannot start any token
```

**Solutions:**

1. **Validate configuration:**
   ```bash
   plexichat-cli config validate
   ```

2. **Check YAML syntax:**
   - Ensure proper indentation (spaces, not tabs)
   - Check for missing quotes around strings with special characters
   - Verify colons have spaces after them

3. **Reset to defaults:**
   ```bash
   plexichat-cli config init --force
   ```

### GUI Issues

#### CGO Errors

**Symptoms:**
```
Error: CGO_ENABLED=0 but cgo required
```

**Solutions:**

1. **Enable CGO:**
   ```bash
   export CGO_ENABLED=1
   go build -o plexichat-gui plexichat-gui.go
   ```

2. **Install build dependencies:**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install gcc pkg-config libgl1-mesa-dev xorg-dev
   
   # CentOS/RHEL
   sudo yum install gcc pkgconfig mesa-libGL-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel
   
   # macOS
   xcode-select --install
   ```

#### GUI Won't Start

**Symptoms:**
- GUI window doesn't appear
- Application crashes on startup

**Solutions:**

1. **Check display environment:**
   ```bash
   # Linux
   echo $DISPLAY
   
   # Ensure X11 forwarding if using SSH
   ssh -X user@host
   ```

2. **Run with debug output:**
   ```bash
   ./plexichat-gui --debug
   ```

3. **Check dependencies:**
   ```bash
   ldd plexichat-gui  # Linux
   otool -L plexichat-gui  # macOS
   ```

### Performance Issues

#### Slow Response Times

**Symptoms:**
- Commands take a long time to complete
- Messages appear delayed

**Solutions:**

1. **Enable performance monitoring:**
   ```bash
   plexichat-cli config set features.performance_monitoring true
   ```

2. **Increase concurrent requests:**
   ```bash
   plexichat-cli config set concurrent_requests 20
   ```

3. **Optimize retry settings:**
   ```bash
   plexichat-cli config set retries 2
   plexichat-cli config set timeout "10s"
   ```

#### High Memory Usage

**Solutions:**

1. **Reduce message history:**
   ```bash
   plexichat-cli config set chat.message_history_limit 25
   ```

2. **Disable unnecessary features:**
   ```bash
   plexichat-cli config set features.experimental_commands false
   plexichat-cli config set features.beta_features false
   ```

### WebSocket Issues

#### Connection Drops

**Symptoms:**
- Frequent disconnections
- Messages not received in real-time

**Solutions:**

1. **Enable auto-reconnect:**
   ```bash
   plexichat-cli config set chat.auto_reconnect true
   ```

2. **Adjust ping interval:**
   ```bash
   plexichat-cli config set chat.ping_interval "15s"
   ```

3. **Check network stability:**
   ```bash
   # Test connection stability
   ping -c 100 your-server
   ```

## Debugging

### Enable Debug Mode

**Global debug mode:**
```bash
plexichat-cli --debug <command>
```

**Configuration debug:**
```bash
plexichat-cli config set verbose true
plexichat-cli config set logging.level "debug"
```

**Environment variable:**
```bash
export PLEXICHAT_DEBUG=true
plexichat-cli chat
```

### Debug Output

Debug mode provides:
- Detailed HTTP request/response information
- WebSocket connection details
- Configuration resolution steps
- Retry attempt information
- Performance metrics

### Log Files

**Save logs to file:**
```bash
plexichat-cli --debug chat 2>&1 | tee debug.log
```

**Analyze logs:**
```bash
# Search for errors
grep -i error debug.log

# Search for specific patterns
grep "retry\|timeout\|failed" debug.log

# Show last 50 lines
tail -50 debug.log
```

## Getting Help

### Built-in Help

```bash
# General help
plexichat-cli --help

# Command-specific help
plexichat-cli chat --help
plexichat-cli config --help

# Show version
plexichat-cli version
```

### Health Checks

```bash
# Check server connectivity
plexichat-cli health

# Test authentication
plexichat-cli auth status

# Validate configuration
plexichat-cli config validate
```

### System Information

**Collect system information for bug reports:**

```bash
# Client version
plexichat-cli version

# Configuration
plexichat-cli config show

# System info
uname -a  # Linux/macOS
systeminfo  # Windows

# Go version (if building from source)
go version

# Network connectivity
curl -v http://your-server:8000/health
```

## Reporting Issues

When reporting issues, please include:

1. **Client version:** `plexichat-cli version`
2. **Operating system and version**
3. **Configuration:** `plexichat-cli config show` (remove sensitive data)
4. **Error message:** Full error output
5. **Debug logs:** Run with `--debug` flag
6. **Steps to reproduce:** Exact commands used
7. **Expected vs actual behavior**

### Example Bug Report

```
**Client Version:** PlexiChat Client v1.0.0

**OS:** Ubuntu 20.04 LTS

**Issue:** Connection timeout when sending messages

**Configuration:**
```yaml
url: "http://localhost:8000"
timeout: "30s"
retries: 3
```

**Error Message:**
```
Error: context deadline exceeded
```

**Steps to Reproduce:**
1. Run `plexichat-cli config set url "http://localhost:8000"`
2. Run `plexichat-cli chat`
3. Type message and press Enter
4. Error occurs after 30 seconds

**Debug Output:**
[Include relevant debug output here]
```

## Advanced Troubleshooting

### Network Analysis

**Use tcpdump/Wireshark to analyze network traffic:**
```bash
# Capture traffic on port 8000
sudo tcpdump -i any port 8000 -w plexichat.pcap

# Analyze with tshark
tshark -r plexichat.pcap -Y "http or websocket"
```

### Performance Profiling

**Enable Go profiling (for developers):**
```bash
# Build with profiling
go build -tags profile -o plexichat-cli plexichat-cli.go

# Run with CPU profiling
plexichat-cli -cpuprofile=cpu.prof chat

# Analyze profile
go tool pprof cpu.prof
```

### Database Issues

**If using local database:**
```bash
# Check database file permissions
ls -la ~/.plexichat/

# Reset local database
rm ~/.plexichat/local.db
plexichat-cli config init
```

## Recovery Procedures

### Reset Configuration

```bash
# Backup current config
plexichat-cli config backup

# Reset to defaults
rm ~/.plexichat-client.yaml
plexichat-cli config init
```

### Clean Installation

```bash
# Remove all client data
rm -rf ~/.plexichat/
rm ~/.plexichat-client.yaml

# Reinstall/rebuild
go build -o plexichat-cli plexichat-cli.go
go build -o plexichat-gui plexichat-gui.go

# Initialize fresh configuration
./plexichat-cli config init
```

### Emergency Contacts

If you cannot resolve the issue:

1. **Check documentation:** Review all documentation files
2. **Search issues:** Look for similar issues in the project repository
3. **Create issue:** Submit a detailed bug report
4. **Contact support:** Reach out to the development team
