package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"os/exec"
	"runtime"

	"plexichat-client/pkg/logging"
)

// DesktopProvider provides desktop notifications
type DesktopProvider struct {
	enabled bool
	logger  *logging.Logger
}

func NewDesktopProvider() *DesktopProvider {
	return &DesktopProvider{
		enabled: true,
		logger:  logging.NewLogger(logging.INFO, nil, true),
	}
}

func (dp *DesktopProvider) Send(ctx context.Context, notification *Notification) error {
	if !dp.enabled {
		return fmt.Errorf("desktop provider disabled")
	}

	switch runtime.GOOS {
	case "windows":
		return dp.sendWindows(notification)
	case "darwin":
		return dp.sendMacOS(notification)
	case "linux":
		return dp.sendLinux(notification)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (dp *DesktopProvider) sendWindows(notification *Notification) error {
	// Use PowerShell to show Windows toast notification
	script := fmt.Sprintf(`
		Add-Type -AssemblyName System.Windows.Forms
		$notification = New-Object System.Windows.Forms.NotifyIcon
		$notification.Icon = [System.Drawing.SystemIcons]::Information
		$notification.BalloonTipTitle = "%s"
		$notification.BalloonTipText = "%s"
		$notification.Visible = $true
		$notification.ShowBalloonTip(5000)
	`, notification.Title, notification.Message)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

func (dp *DesktopProvider) sendMacOS(notification *Notification) error {
	// Use osascript to show macOS notification
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, 
		notification.Message, notification.Title)
	
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func (dp *DesktopProvider) sendLinux(notification *Notification) error {
	// Use notify-send for Linux notifications
	cmd := exec.Command("notify-send", notification.Title, notification.Message)
	return cmd.Run()
}

func (dp *DesktopProvider) GetType() string {
	return "desktop"
}

func (dp *DesktopProvider) IsEnabled() bool {
	return dp.enabled
}

func (dp *DesktopProvider) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		dp.enabled = enabled
	}
	return nil
}

// SoundProvider provides sound notifications
type SoundProvider struct {
	enabled bool
	logger  *logging.Logger
}

func NewSoundProvider() *SoundProvider {
	return &SoundProvider{
		enabled: true,
		logger:  logging.NewLogger(logging.INFO, nil, true),
	}
}

func (sp *SoundProvider) Send(ctx context.Context, notification *Notification) error {
	if !sp.enabled {
		return fmt.Errorf("sound provider disabled")
	}

	soundFile := notification.Sound
	if soundFile == "" {
		soundFile = "notification.wav"
	}

	switch runtime.GOOS {
	case "windows":
		return sp.playSoundWindows(soundFile)
	case "darwin":
		return sp.playSoundMacOS(soundFile)
	case "linux":
		return sp.playSoundLinux(soundFile)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (sp *SoundProvider) playSoundWindows(soundFile string) error {
	// Use PowerShell to play sound on Windows
	script := fmt.Sprintf(`
		Add-Type -AssemblyName System.Media
		$player = New-Object System.Media.SoundPlayer
		$player.SoundLocation = "%s"
		$player.Play()
	`, soundFile)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

func (sp *SoundProvider) playSoundMacOS(soundFile string) error {
	// Use afplay to play sound on macOS
	cmd := exec.Command("afplay", soundFile)
	return cmd.Run()
}

func (sp *SoundProvider) playSoundLinux(soundFile string) error {
	// Try different audio players on Linux
	players := []string{"aplay", "paplay", "play"}
	
	for _, player := range players {
		cmd := exec.Command(player, soundFile)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("no audio player found")
}

func (sp *SoundProvider) GetType() string {
	return "sound"
}

func (sp *SoundProvider) IsEnabled() bool {
	return sp.enabled
}

func (sp *SoundProvider) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		sp.enabled = enabled
	}
	return nil
}

// EmailProvider provides email notifications
type EmailProvider struct {
	enabled      bool
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromAddress  string
	tlsEnabled   bool
	logger       *logging.Logger
}

func NewEmailProvider() *EmailProvider {
	return &EmailProvider{
		enabled: false,
		logger:  logging.NewLogger(logging.INFO, nil, true),
	}
}

func (ep *EmailProvider) Send(ctx context.Context, notification *Notification) error {
	if !ep.enabled {
		return fmt.Errorf("email provider disabled")
	}

	if ep.smtpHost == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	// Create email message
	subject := notification.Title
	body := notification.Message
	
	message := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)

	// Setup authentication
	auth := smtp.PlainAuth("", ep.smtpUsername, ep.smtpPassword, ep.smtpHost)

	// Send email
	addr := fmt.Sprintf("%s:%d", ep.smtpHost, ep.smtpPort)
	to := []string{"user@example.com"} // This would come from configuration
	
	err := smtp.SendMail(addr, auth, ep.fromAddress, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	ep.logger.Info("Email notification sent: %s", notification.Title)
	return nil
}

func (ep *EmailProvider) GetType() string {
	return "email"
}

func (ep *EmailProvider) IsEnabled() bool {
	return ep.enabled
}

func (ep *EmailProvider) Configure(config map[string]interface{}) error {
	if host, ok := config["smtp_host"].(string); ok {
		ep.smtpHost = host
	}
	if port, ok := config["smtp_port"].(int); ok {
		ep.smtpPort = port
	}
	if username, ok := config["smtp_username"].(string); ok {
		ep.smtpUsername = username
	}
	if password, ok := config["smtp_password"].(string); ok {
		ep.smtpPassword = password
	}
	if from, ok := config["from_address"].(string); ok {
		ep.fromAddress = from
	}
	if tls, ok := config["tls_enabled"].(bool); ok {
		ep.tlsEnabled = tls
	}
	
	// Enable if basic configuration is present
	ep.enabled = ep.smtpHost != "" && ep.fromAddress != ""
	
	return nil
}

// PushProvider provides push notifications
type PushProvider struct {
	enabled    bool
	serviceURL string
	apiKey     string
	deviceID   string
	headers    map[string]string
	logger     *logging.Logger
}

func NewPushProvider() *PushProvider {
	return &PushProvider{
		enabled: false,
		headers: make(map[string]string),
		logger:  logging.NewLogger(logging.INFO, nil, true),
	}
}

func (pp *PushProvider) Send(ctx context.Context, notification *Notification) error {
	if !pp.enabled {
		return fmt.Errorf("push provider disabled")
	}

	if pp.serviceURL == "" {
		return fmt.Errorf("push service URL not configured")
	}

	// This would implement actual push notification sending
	// For now, just log the notification
	pp.logger.Info("Push notification would be sent: %s - %s", 
		notification.Title, notification.Message)
	
	return nil
}

func (pp *PushProvider) GetType() string {
	return "push"
}

func (pp *PushProvider) IsEnabled() bool {
	return pp.enabled
}

func (pp *PushProvider) Configure(config map[string]interface{}) error {
	if url, ok := config["service_url"].(string); ok {
		pp.serviceURL = url
	}
	if key, ok := config["api_key"].(string); ok {
		pp.apiKey = key
	}
	if device, ok := config["device_id"].(string); ok {
		pp.deviceID = device
	}
	if headers, ok := config["headers"].(map[string]string); ok {
		pp.headers = headers
	}
	
	// Enable if basic configuration is present
	pp.enabled = pp.serviceURL != "" && pp.apiKey != ""
	
	return nil
}
