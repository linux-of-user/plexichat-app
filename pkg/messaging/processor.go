package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/database"
	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/security"
)

// MessageProcessor handles message processing and formatting
type MessageProcessor struct {
	db       *database.Database
	logger   *logging.Logger
	handlers map[MessageType]MessageHandler
	filters  []MessageFilter
	mu       sync.RWMutex
}

// MessageType represents different types of messages
type MessageType string

const (
	MessageTypeText         MessageType = "text"
	MessageTypeImage        MessageType = "image"
	MessageTypeFile         MessageType = "file"
	MessageTypeCode         MessageType = "code"
	MessageTypeMarkdown     MessageType = "markdown"
	MessageTypeEmoji        MessageType = "emoji"
	MessageTypeMention      MessageType = "mention"
	MessageTypeCommand      MessageType = "command"
	MessageTypeReaction     MessageType = "reaction"
	MessageTypeThread       MessageType = "thread"
	MessageTypeEdit         MessageType = "edit"
	MessageTypeDelete       MessageType = "delete"
	MessageTypeSystem       MessageType = "system"
	MessageTypeNotification MessageType = "notification"
)

// ProcessedMessage represents a processed message with metadata
type ProcessedMessage struct {
	ID          int64                  `json:"id"`
	ChannelID   string                 `json:"channel_id"`
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Content     string                 `json:"content"`
	Type        MessageType            `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	EditedAt    *time.Time             `json:"edited_at,omitempty"`
	Mentions    []string               `json:"mentions,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	Reactions   []Reaction             `json:"reactions,omitempty"`
	Thread      *ThreadInfo            `json:"thread,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Formatted   string                 `json:"formatted"`
	Preview     *LinkPreview           `json:"preview,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

// Reaction represents a message reaction
type Reaction struct {
	Emoji   string    `json:"emoji"`
	Users   []string  `json:"users"`
	Count   int       `json:"count"`
	UserID  string    `json:"user_id,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

// ThreadInfo represents thread information
type ThreadInfo struct {
	ParentID     int64     `json:"parent_id"`
	ReplyCount   int       `json:"reply_count"`
	LastReplyAt  time.Time `json:"last_reply_at"`
	Participants []string  `json:"participants"`
}

// LinkPreview represents a link preview
type LinkPreview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
}

// MessageHandler defines the interface for message handlers
type MessageHandler interface {
	Handle(ctx context.Context, msg *ProcessedMessage) error
	CanHandle(msgType MessageType) bool
}

// MessageFilter defines the interface for message filters
type MessageFilter interface {
	Filter(ctx context.Context, msg *ProcessedMessage) (*ProcessedMessage, error)
	Priority() int
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(db *database.Database) *MessageProcessor {
	processor := &MessageProcessor{
		db:       db,
		logger:   logging.NewLogger(logging.INFO, nil, true),
		handlers: make(map[MessageType]MessageHandler),
		filters:  make([]MessageFilter, 0),
	}

	// Register default handlers
	processor.RegisterHandler(MessageTypeText, &TextMessageHandler{})
	processor.RegisterHandler(MessageTypeMarkdown, &MarkdownMessageHandler{})
	processor.RegisterHandler(MessageTypeCode, &CodeMessageHandler{})
	processor.RegisterHandler(MessageTypeMention, &MentionMessageHandler{})
	processor.RegisterHandler(MessageTypeCommand, &CommandMessageHandler{})

	// Register default filters
	processor.RegisterFilter(&SecurityFilter{})
	processor.RegisterFilter(&MentionFilter{})
	processor.RegisterFilter(&LinkPreviewFilter{})
	processor.RegisterFilter(&EmojiFilter{})

	return processor
}

// RegisterHandler registers a message handler
func (mp *MessageProcessor) RegisterHandler(msgType MessageType, handler MessageHandler) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.handlers[msgType] = handler
}

// RegisterFilter registers a message filter
func (mp *MessageProcessor) RegisterFilter(filter MessageFilter) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.filters = append(mp.filters, filter)

	// Sort filters by priority
	for i := 0; i < len(mp.filters)-1; i++ {
		for j := i + 1; j < len(mp.filters); j++ {
			if mp.filters[i].Priority() > mp.filters[j].Priority() {
				mp.filters[i], mp.filters[j] = mp.filters[j], mp.filters[i]
			}
		}
	}
}

// ProcessMessage processes a raw message and returns a processed message
func (mp *MessageProcessor) ProcessMessage(ctx context.Context, rawMsg *database.Message) (*ProcessedMessage, error) {
	// Convert to processed message
	processed := &ProcessedMessage{
		ID:        rawMsg.ID,
		ChannelID: rawMsg.ChannelID,
		UserID:    rawMsg.UserID,
		Username:  rawMsg.Username,
		Content:   rawMsg.Content,
		Type:      MessageType(rawMsg.MessageType),
		Timestamp: rawMsg.Timestamp,
		EditedAt:  rawMsg.EditedAt,
		Metadata:  make(map[string]interface{}),
		Formatted: rawMsg.Content,
	}

	// Parse metadata
	if rawMsg.Metadata != "" {
		if err := json.Unmarshal([]byte(rawMsg.Metadata), &processed.Metadata); err != nil {
			mp.logger.Error("Failed to parse message metadata: %v", err)
		}
	}

	// Parse attachments
	if rawMsg.Attachments != "" {
		if err := json.Unmarshal([]byte(rawMsg.Attachments), &processed.Attachments); err != nil {
			mp.logger.Error("Failed to parse message attachments: %v", err)
		}
	}

	// Apply filters
	mp.mu.RLock()
	filters := make([]MessageFilter, len(mp.filters))
	copy(filters, mp.filters)
	mp.mu.RUnlock()

	for _, filter := range filters {
		var err error
		processed, err = filter.Filter(ctx, processed)
		if err != nil {
			return nil, fmt.Errorf("filter error: %w", err)
		}
		if processed == nil {
			return nil, fmt.Errorf("message filtered out")
		}
	}

	// Apply handler
	mp.mu.RLock()
	handler, exists := mp.handlers[processed.Type]
	mp.mu.RUnlock()

	if exists && handler.CanHandle(processed.Type) {
		if err := handler.Handle(ctx, processed); err != nil {
			mp.logger.Error("Handler error for message type %s: %v", processed.Type, err)
		}
	}

	return processed, nil
}

// TextMessageHandler handles plain text messages
type TextMessageHandler struct{}

func (h *TextMessageHandler) Handle(ctx context.Context, msg *ProcessedMessage) error {
	// Basic text formatting
	msg.Formatted = strings.TrimSpace(msg.Content)
	return nil
}

func (h *TextMessageHandler) CanHandle(msgType MessageType) bool {
	return msgType == MessageTypeText
}

// MarkdownMessageHandler handles markdown messages
type MarkdownMessageHandler struct{}

func (h *MarkdownMessageHandler) Handle(ctx context.Context, msg *ProcessedMessage) error {
	// Convert markdown to HTML (simplified)
	formatted := msg.Content

	// Bold text
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*`)
	formatted = boldRegex.ReplaceAllString(formatted, `<strong>$1</strong>`)

	// Italic text
	italicRegex := regexp.MustCompile(`\*(.*?)\*`)
	formatted = italicRegex.ReplaceAllString(formatted, `<em>$1</em>`)

	// Code blocks
	codeRegex := regexp.MustCompile("```([\\s\\S]*?)```")
	formatted = codeRegex.ReplaceAllString(formatted, `<pre><code>$1</code></pre>`)

	// Inline code
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	formatted = inlineCodeRegex.ReplaceAllString(formatted, `<code>$1</code>`)

	msg.Formatted = formatted
	return nil
}

func (h *MarkdownMessageHandler) CanHandle(msgType MessageType) bool {
	return msgType == MessageTypeMarkdown
}

// CodeMessageHandler handles code messages
type CodeMessageHandler struct{}

func (h *CodeMessageHandler) Handle(ctx context.Context, msg *ProcessedMessage) error {
	// Wrap in code block
	msg.Formatted = fmt.Sprintf("<pre><code>%s</code></pre>", msg.Content)
	return nil
}

func (h *CodeMessageHandler) CanHandle(msgType MessageType) bool {
	return msgType == MessageTypeCode
}

// MentionMessageHandler handles mention messages
type MentionMessageHandler struct{}

func (h *MentionMessageHandler) Handle(ctx context.Context, msg *ProcessedMessage) error {
	// Extract mentions from content
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(msg.Content, -1)

	mentions := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	msg.Mentions = mentions

	// Format mentions
	formatted := mentionRegex.ReplaceAllString(msg.Content, `<span class="mention">@$1</span>`)
	msg.Formatted = formatted

	return nil
}

func (h *MentionMessageHandler) CanHandle(msgType MessageType) bool {
	return msgType == MessageTypeMention
}

// CommandMessageHandler handles command messages
type CommandMessageHandler struct{}

func (h *CommandMessageHandler) Handle(ctx context.Context, msg *ProcessedMessage) error {
	// Parse command
	if strings.HasPrefix(msg.Content, "/") {
		parts := strings.Fields(msg.Content)
		if len(parts) > 0 {
			command := parts[0][1:] // Remove the /
			args := parts[1:]

			msg.Metadata["command"] = command
			msg.Metadata["args"] = args

			msg.Formatted = fmt.Sprintf(`<span class="command">/%s</span>`, command)
			if len(args) > 0 {
				msg.Formatted += fmt.Sprintf(` <span class="command-args">%s</span>`, strings.Join(args, " "))
			}
		}
	}

	return nil
}

func (h *CommandMessageHandler) CanHandle(msgType MessageType) bool {
	return msgType == MessageTypeCommand
}

// SecurityFilter filters messages for security threats
type SecurityFilter struct{}

func (f *SecurityFilter) Filter(ctx context.Context, msg *ProcessedMessage) (*ProcessedMessage, error) {
	// Check for malicious content
	if security.ContainsMaliciousContent(msg.Content) {
		return nil, fmt.Errorf("message contains malicious content")
	}

	// Sanitize content
	msg.Content = security.SanitizeInput(msg.Content)

	return msg, nil
}

func (f *SecurityFilter) Priority() int {
	return 1 // Highest priority
}

// MentionFilter extracts mentions from messages
type MentionFilter struct{}

func (f *MentionFilter) Filter(ctx context.Context, msg *ProcessedMessage) (*ProcessedMessage, error) {
	// Extract mentions
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(msg.Content, -1)

	mentions := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	if len(mentions) > 0 {
		msg.Mentions = mentions
		msg.Type = MessageTypeMention
	}

	return msg, nil
}

func (f *MentionFilter) Priority() int {
	return 5
}

// LinkPreviewFilter generates link previews
type LinkPreviewFilter struct{}

func (f *LinkPreviewFilter) Filter(ctx context.Context, msg *ProcessedMessage) (*ProcessedMessage, error) {
	// Extract URLs
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlRegex.FindAllString(msg.Content, -1)

	if len(urls) > 0 {
		// For now, just create a basic preview for the first URL
		url := urls[0]
		msg.Preview = &LinkPreview{
			URL:         url,
			Title:       "Link Preview",
			Description: "Click to open link",
			SiteName:    extractDomain(url),
		}
	}

	return msg, nil
}

func (f *LinkPreviewFilter) Priority() int {
	return 10
}

// EmojiFilter processes emoji in messages
type EmojiFilter struct{}

func (f *EmojiFilter) Filter(ctx context.Context, msg *ProcessedMessage) (*ProcessedMessage, error) {
	// Simple emoji detection (Unicode emoji)
	emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]`)

	if emojiRegex.MatchString(msg.Content) {
		msg.Metadata["has_emoji"] = true
	}

	// Custom emoji detection :emoji_name:
	customEmojiRegex := regexp.MustCompile(`:(\w+):`)
	customEmojis := customEmojiRegex.FindAllStringSubmatch(msg.Content, -1)

	if len(customEmojis) > 0 {
		emojiNames := make([]string, 0)
		for _, match := range customEmojis {
			if len(match) > 1 {
				emojiNames = append(emojiNames, match[1])
			}
		}
		msg.Metadata["custom_emojis"] = emojiNames
	}

	return msg, nil
}

func (f *EmojiFilter) Priority() int {
	return 15
}

// Helper function to extract domain from URL
func extractDomain(url string) string {
	// Simple domain extraction
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return url
}
