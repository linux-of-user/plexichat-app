package history

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"plexichat-client/pkg/cache"
	"plexichat-client/pkg/client"
	"plexichat-client/pkg/logging"
)

// MessageFilter represents filters for message search
type MessageFilter struct {
	Query       string    `json:"query"`        // Text search query
	UserID      string    `json:"user_id"`      // Filter by specific user
	StartDate   time.Time `json:"start_date"`   // Messages after this date
	EndDate     time.Time `json:"end_date"`     // Messages before this date
	MessageType string    `json:"message_type"` // Filter by message type
	Limit       int       `json:"limit"`        // Maximum results
}

// MessageSearchResult represents a search result
type MessageSearchResult struct {
	Message    *client.Message `json:"message"`
	Relevance  float64         `json:"relevance"`  // Search relevance score
	Context    []string        `json:"context"`    // Surrounding message context
	Highlights []string        `json:"highlights"` // Highlighted search terms
}

// ConversationSummary provides summary of a conversation
type ConversationSummary struct {
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	MessageCount   int       `json:"message_count"`
	LastMessage    string    `json:"last_message"`
	LastMessageAt  time.Time `json:"last_message_at"`
	FirstMessageAt time.Time `json:"first_message_at"`
	UnreadCount    int       `json:"unread_count"`
}

// HistoryManager manages message history and search
type HistoryManager struct {
	client *cache.CachedClient
	logger *logging.Logger
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(cachedClient *cache.CachedClient) *HistoryManager {
	return &HistoryManager{
		client: cachedClient,
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// SearchMessages searches through message history
func (h *HistoryManager) SearchMessages(ctx context.Context, filter *MessageFilter) ([]*MessageSearchResult, error) {
	h.logger.Info("Searching messages with query: %s", filter.Query)

	var allResults []*MessageSearchResult

	// For now, we'll search through recent conversations
	// In a real implementation, this would use a dedicated search endpoint
	conversations, err := h.GetRecentConversations(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	for _, conv := range conversations {
		messages, err := h.searchInConversation(ctx, conv.UserID, filter)
		if err != nil {
			h.logger.Error("Failed to search in conversation %s: %v", conv.UserID, err)
			continue
		}
		allResults = append(allResults, messages...)
	}

	// Sort by relevance
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Relevance > allResults[j].Relevance
	})

	// Apply limit
	if filter.Limit > 0 && len(allResults) > filter.Limit {
		allResults = allResults[:filter.Limit]
	}

	h.logger.Info("Found %d matching messages", len(allResults))
	return allResults, nil
}

// searchInConversation searches messages in a specific conversation
func (h *HistoryManager) searchInConversation(ctx context.Context, userID string, filter *MessageFilter) ([]*MessageSearchResult, error) {
	var allMessages []client.Message
	page := 1
	limit := 50

	// Fetch all messages from conversation
	for {
		resp, err := h.client.GetMessages(userID, limit, page)
		if err != nil {
			return nil, err
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

	var results []*MessageSearchResult

	for i, msg := range allMessages {
		if h.matchesFilter(&msg, filter) {
			relevance := h.calculateRelevance(&msg, filter.Query)
			if relevance > 0 {
				result := &MessageSearchResult{
					Message:    &msg,
					Relevance:  relevance,
					Context:    h.getMessageContext(allMessages, i, 2),
					Highlights: h.getHighlights(msg.Content, filter.Query),
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// matchesFilter checks if a message matches the filter criteria
func (h *HistoryManager) matchesFilter(msg *client.Message, filter *MessageFilter) bool {
	// Check user filter
	if filter.UserID != "" && fmt.Sprintf("%d", msg.UserID) != filter.UserID {
		return false
	}

	// Check date range
	if !filter.StartDate.IsZero() && msg.Timestamp.Before(filter.StartDate) {
		return false
	}

	if !filter.EndDate.IsZero() && msg.Timestamp.After(filter.EndDate) {
		return false
	}

	// Check text query
	if filter.Query != "" {
		query := strings.ToLower(filter.Query)
		content := strings.ToLower(msg.Content)
		username := strings.ToLower(msg.Username)

		if !strings.Contains(content, query) && !strings.Contains(username, query) {
			return false
		}
	}

	return true
}

// calculateRelevance calculates search relevance score
func (h *HistoryManager) calculateRelevance(msg *client.Message, query string) float64 {
	if query == "" {
		return 1.0
	}

	query = strings.ToLower(query)
	content := strings.ToLower(msg.Content)

	var score float64

	// Exact match gets highest score
	if strings.Contains(content, query) {
		score += 1.0
	}

	// Word matches
	queryWords := strings.Fields(query)
	contentWords := strings.Fields(content)

	matchedWords := 0
	for _, qWord := range queryWords {
		for _, cWord := range contentWords {
			if strings.Contains(cWord, qWord) {
				matchedWords++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		score += float64(matchedWords) / float64(len(queryWords)) * 0.5
	}

	// Boost recent messages
	age := time.Since(msg.Timestamp)
	if age < 24*time.Hour {
		score *= 1.2
	} else if age < 7*24*time.Hour {
		score *= 1.1
	}

	return score
}

// getMessageContext returns surrounding messages for context
func (h *HistoryManager) getMessageContext(messages []client.Message, index, contextSize int) []string {
	var context []string

	start := index - contextSize
	if start < 0 {
		start = 0
	}

	end := index + contextSize + 1
	if end > len(messages) {
		end = len(messages)
	}

	for i := start; i < end; i++ {
		if i != index {
			context = append(context, fmt.Sprintf("[%s] %s: %s",
				messages[i].Timestamp.Format("15:04"),
				messages[i].Username,
				messages[i].Content))
		}
	}

	return context
}

// getHighlights extracts highlighted terms from content
func (h *HistoryManager) getHighlights(content, query string) []string {
	if query == "" {
		return nil
	}

	var highlights []string
	query = strings.ToLower(query)
	content = strings.ToLower(content)

	queryWords := strings.Fields(query)
	for _, word := range queryWords {
		if strings.Contains(content, word) {
			highlights = append(highlights, word)
		}
	}

	return highlights
}

// GetRecentConversations gets a list of recent conversations
func (h *HistoryManager) GetRecentConversations(ctx context.Context, limit int) ([]*ConversationSummary, error) {
	h.logger.Info("Getting recent conversations (limit: %d)", limit)

	// Get list of users we've chatted with
	users, err := h.client.GetUsers(ctx, limit*2, 0) // Get more users to filter
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var conversations []*ConversationSummary

	for _, user := range users.Users {
		userID := fmt.Sprintf("%d", user.ID)

		// Get recent messages with this user
		messages, err := h.client.GetMessages(userID, 1, 1) // Just get the latest message
		if err != nil {
			h.logger.Debug("No messages found with user %s: %v", user.Username, err)
			continue
		}

		if len(messages.Messages) == 0 {
			continue
		}

		lastMsg := messages.Messages[0]

		summary := &ConversationSummary{
			UserID:        userID,
			Username:      user.Username,
			MessageCount:  messages.Total,
			LastMessage:   lastMsg.Content,
			LastMessageAt: lastMsg.Timestamp,
			UnreadCount:   0, // Would need additional API to get unread count
		}

		conversations = append(conversations, summary)
	}

	// Sort by last message time
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].LastMessageAt.After(conversations[j].LastMessageAt)
	})

	// Apply limit
	if len(conversations) > limit {
		conversations = conversations[:limit]
	}

	h.logger.Info("Found %d recent conversations", len(conversations))
	return conversations, nil
}

// GetMessageStats returns statistics about message history
func (h *HistoryManager) GetMessageStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	h.logger.Info("Getting message statistics for user: %s", userID)

	var allMessages []client.Message
	page := 1
	limit := 100

	// Fetch all messages
	for {
		resp, err := h.client.GetMessages(userID, limit, page)
		if err != nil {
			return nil, err
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

	if len(allMessages) == 0 {
		return map[string]interface{}{
			"total_messages": 0,
			"date_range":     nil,
		}, nil
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"total_messages": len(allMessages),
		"first_message":  allMessages[len(allMessages)-1].Timestamp,
		"last_message":   allMessages[0].Timestamp,
	}

	// Message count by day
	dailyCounts := make(map[string]int)
	userCounts := make(map[string]int)

	for _, msg := range allMessages {
		day := msg.Timestamp.Format("2006-01-02")
		dailyCounts[day]++
		userCounts[msg.Username]++
	}

	stats["daily_counts"] = dailyCounts
	stats["user_counts"] = userCounts

	// Calculate average messages per day
	if len(dailyCounts) > 0 {
		totalDays := len(dailyCounts)
		stats["avg_messages_per_day"] = float64(len(allMessages)) / float64(totalDays)
	}

	h.logger.Info("Generated statistics for %d messages", len(allMessages))
	return stats, nil
}

// ExportConversation exports a conversation to a structured format
func (h *HistoryManager) ExportConversation(ctx context.Context, userID string, format string) ([]byte, error) {
	h.logger.Info("Exporting conversation with user: %s (format: %s)", userID, format)

	var allMessages []client.Message
	page := 1
	limit := 100

	// Fetch all messages
	for {
		resp, err := h.client.GetMessages(userID, limit, page)
		if err != nil {
			return nil, err
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

	switch format {
	case "json":
		return h.exportAsJSON(allMessages)
	case "text":
		return h.exportAsText(allMessages)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportAsJSON exports messages as JSON
func (h *HistoryManager) exportAsJSON(messages []client.Message) ([]byte, error) {
	export := map[string]interface{}{
		"exported_at":   time.Now(),
		"message_count": len(messages),
		"messages":      messages,
	}

	return json.Marshal(export)
}

// exportAsText exports messages as plain text
func (h *HistoryManager) exportAsText(messages []client.Message) ([]byte, error) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("PlexiChat Conversation Export\n"))
	text.WriteString(fmt.Sprintf("Exported: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	text.WriteString(fmt.Sprintf("Total Messages: %d\n\n", len(messages)))
	text.WriteString(strings.Repeat("=", 50) + "\n\n")

	for _, msg := range messages {
		text.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			msg.Username,
			msg.Content))
	}

	return []byte(text.String()), nil
}
