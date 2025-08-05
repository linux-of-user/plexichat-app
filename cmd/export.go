package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plexichat-client/pkg/cache"
	"plexichat-client/pkg/client"
	"plexichat-client/pkg/history"
	"plexichat-client/pkg/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	exportFormat   string
	exportOutput   string
	exportUserID   string
	exportAll      bool
	exportConfig   bool
	exportMessages bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export chat history and configuration",
	Long: `Export chat history, configuration, and other data from PlexiChat.

This command allows you to export:
- Chat conversations with specific users
- All chat history
- Application configuration
- User data and settings

Examples:
  # Export conversation with a specific user
  plexichat-cli export --user-id 123 --format json --output conversation.json

  # Export all conversations
  plexichat-cli export --all --format text --output all_chats.txt

  # Export configuration
  plexichat-cli export --config --output config_backup.yaml

  # Export everything
  plexichat-cli export --all --config --output backup.json`,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format (json, text, yaml)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringVar(&exportUserID, "user-id", "", "Export conversation with specific user")
	exportCmd.Flags().BoolVar(&exportAll, "all", false, "Export all conversations")
	exportCmd.Flags().BoolVar(&exportConfig, "config", false, "Export configuration")
	exportCmd.Flags().BoolVar(&exportMessages, "messages", true, "Include messages in export")
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Validate flags
	if !exportAll && exportUserID == "" && !exportConfig {
		return fmt.Errorf("must specify --all, --user-id, or --config")
	}

	if exportFormat != "json" && exportFormat != "text" && exportFormat != "yaml" {
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	// Initialize client
	apiClient := client.NewClient(viper.GetString("url"))
	if token := viper.GetString("token"); token != "" {
		apiClient.SetToken(token)
	}

	// Create cached client
	cachedClient := cache.NewCachedClient(apiClient, nil)

	// Create history manager
	historyManager := history.NewHistoryManager(cachedClient)

	var exportData map[string]interface{}
	var err error

	if exportConfig {
		exportData, err = exportConfiguration()
		if err != nil {
			return fmt.Errorf("failed to export configuration: %w", err)
		}
	} else if exportAll {
		exportData, err = exportAllConversations(ctx, historyManager, cachedClient)
		if err != nil {
			return fmt.Errorf("failed to export all conversations: %w", err)
		}
	} else if exportUserID != "" {
		exportData, err = exportUserConversation(ctx, historyManager, cachedClient, exportUserID)
		if err != nil {
			return fmt.Errorf("failed to export conversation: %w", err)
		}
	}

	// Format and output data
	return outputExportData(exportData, exportFormat, exportOutput)
}

func exportConfiguration() (map[string]interface{}, error) {
	logging.Info("Exporting configuration...")

	config := map[string]interface{}{
		"export_info": map[string]interface{}{
			"exported_at": time.Now(),
			"version":     "1.0.0",
			"type":        "configuration",
		},
		"settings": viper.AllSettings(),
	}

	return config, nil
}

func exportAllConversations(ctx context.Context, historyManager *history.HistoryManager, cachedClient *cache.CachedClient) (map[string]interface{}, error) {
	logging.Info("Exporting all conversations...")

	// Get recent conversations
	conversations, err := historyManager.GetRecentConversations(ctx, 100) // Get up to 100 conversations
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	exportData := map[string]interface{}{
		"export_info": map[string]interface{}{
			"exported_at":        time.Now(),
			"version":            "1.0.0",
			"type":               "all_conversations",
			"conversation_count": len(conversations),
		},
		"conversations": make([]map[string]interface{}, 0, len(conversations)),
	}

	for _, conv := range conversations {
		convData, err := exportSingleConversation(ctx, historyManager, cachedClient, conv.UserID, conv.Username)
		if err != nil {
			logging.Error("Failed to export conversation with %s: %v", conv.Username, err)
			continue
		}

		exportData["conversations"] = append(exportData["conversations"].([]map[string]interface{}), convData)
	}

	return exportData, nil
}

func exportUserConversation(ctx context.Context, historyManager *history.HistoryManager, cachedClient *cache.CachedClient, userID string) (map[string]interface{}, error) {
	logging.Info("Exporting conversation with user: %s", userID)

	return exportSingleConversation(ctx, historyManager, cachedClient, userID, "")
}

func exportSingleConversation(ctx context.Context, historyManager *history.HistoryManager, cachedClient *cache.CachedClient, userID, username string) (map[string]interface{}, error) {
	// Get message statistics
	stats, err := historyManager.GetMessageStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message stats: %w", err)
	}

	// Get all messages
	var allMessages []client.Message
	page := 1
	limit := 100

	for {
		resp, err := cachedClient.GetMessages(userID, limit, page)
		if err != nil {
			return nil, fmt.Errorf("failed to get messages: %w", err)
		}

		if len(resp.Messages) == 0 {
			break
		}

		allMessages = append(allMessages, resp.Messages...)

		if !resp.HasNext {
			break
		}
		page++
	}

	// Reverse to get chronological order
	for i := len(allMessages)/2 - 1; i >= 0; i-- {
		opp := len(allMessages) - 1 - i
		allMessages[i], allMessages[opp] = allMessages[opp], allMessages[i]
	}

	convData := map[string]interface{}{
		"user_id":       userID,
		"username":      username,
		"exported_at":   time.Now(),
		"message_count": len(allMessages),
		"statistics":    stats,
	}

	if exportMessages {
		convData["messages"] = allMessages
	}

	return convData, nil
}

func outputExportData(data map[string]interface{}, format, output string) error {
	var content []byte
	var err error

	switch format {
	case "json":
		content, err = jsonMarshalIndent(data, "", "  ")
	case "yaml":
		content, err = yamlMarshal(data)
	case "text":
		content, err = formatAsText(data)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to format data: %w", err)
	}

	if output == "" {
		// Output to stdout
		fmt.Print(string(content))
	} else {
		// Ensure output directory exists
		if dir := filepath.Dir(output); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write to file
		if err := os.WriteFile(output, content, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		logging.Info("Export saved to: %s", output)
	}

	return nil
}

func formatAsText(data map[string]interface{}) ([]byte, error) {
	var text strings.Builder

	// Handle different export types
	if exportInfo, ok := data["export_info"].(map[string]interface{}); ok {
		if exportType, ok := exportInfo["type"].(string); ok {
			switch exportType {
			case "configuration":
				return formatConfigAsText(data)
			case "all_conversations":
				return formatAllConversationsAsText(data)
			default:
				return formatSingleConversationAsText(data)
			}
		}
	}

	// Fallback to generic formatting
	text.WriteString("PlexiChat Export\n")
	text.WriteString(strings.Repeat("=", 50) + "\n\n")

	for key, value := range data {
		text.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return []byte(text.String()), nil
}

func formatConfigAsText(data map[string]interface{}) ([]byte, error) {
	var text strings.Builder

	text.WriteString("PlexiChat Configuration Export\n")
	text.WriteString(strings.Repeat("=", 50) + "\n\n")

	if exportInfo, ok := data["export_info"].(map[string]interface{}); ok {
		if exportedAt, ok := exportInfo["exported_at"].(time.Time); ok {
			text.WriteString(fmt.Sprintf("Exported: %s\n\n", exportedAt.Format("2006-01-02 15:04:05")))
		}
	}

	if settings, ok := data["settings"].(map[string]interface{}); ok {
		text.WriteString("Configuration Settings:\n")
		text.WriteString(strings.Repeat("-", 25) + "\n")

		for key, value := range settings {
			text.WriteString(fmt.Sprintf("%-20s: %v\n", key, value))
		}
	}

	return []byte(text.String()), nil
}

func formatAllConversationsAsText(data map[string]interface{}) ([]byte, error) {
	var text strings.Builder

	text.WriteString("PlexiChat All Conversations Export\n")
	text.WriteString(strings.Repeat("=", 50) + "\n\n")

	if exportInfo, ok := data["export_info"].(map[string]interface{}); ok {
		if exportedAt, ok := exportInfo["exported_at"].(time.Time); ok {
			text.WriteString(fmt.Sprintf("Exported: %s\n", exportedAt.Format("2006-01-02 15:04:05")))
		}
		if count, ok := exportInfo["conversation_count"].(int); ok {
			text.WriteString(fmt.Sprintf("Conversations: %d\n\n", count))
		}
	}

	if conversations, ok := data["conversations"].([]map[string]interface{}); ok {
		for i, conv := range conversations {
			if i > 0 {
				text.WriteString("\n" + strings.Repeat("-", 50) + "\n\n")
			}

			convText, err := formatSingleConversationAsText(conv)
			if err != nil {
				return nil, err
			}
			text.Write(convText)
		}
	}

	return []byte(text.String()), nil
}

func formatSingleConversationAsText(data map[string]interface{}) ([]byte, error) {
	var text strings.Builder

	if username, ok := data["username"].(string); ok && username != "" {
		text.WriteString(fmt.Sprintf("Conversation with %s\n", username))
	} else if userID, ok := data["user_id"].(string); ok {
		text.WriteString(fmt.Sprintf("Conversation with User %s\n", userID))
	}

	if messageCount, ok := data["message_count"].(int); ok {
		text.WriteString(fmt.Sprintf("Messages: %d\n", messageCount))
	}

	if exportedAt, ok := data["exported_at"].(time.Time); ok {
		text.WriteString(fmt.Sprintf("Exported: %s\n", exportedAt.Format("2006-01-02 15:04:05")))
	}

	text.WriteString("\n")

	if messages, ok := data["messages"].([]client.Message); ok {
		for _, msg := range messages {
			text.WriteString(fmt.Sprintf("[%s] %s: %s\n",
				msg.Timestamp.Format("2006-01-02 15:04:05"),
				msg.Username,
				msg.Content))
		}
	}

	return []byte(text.String()), nil
}

// Helper functions for marshaling
func jsonMarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func yamlMarshal(v interface{}) ([]byte, error) {
	// For now, just use JSON format for YAML
	return json.MarshalIndent(v, "", "  ")
}
