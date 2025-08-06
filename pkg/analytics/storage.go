package analytics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// AnalyticsStorage handles persistent storage of analytics data
type AnalyticsStorage struct {
	baseDir   string
	logger    *logging.Logger
	mu        sync.RWMutex
	fileCache map[string]*os.File
	buffers   map[string]*bufio.Writer
}

// NewAnalyticsStorage creates a new analytics storage instance
func NewAnalyticsStorage(baseDir string) *AnalyticsStorage {
	storage := &AnalyticsStorage{
		baseDir:   baseDir,
		logger:    logging.NewLogger(logging.INFO, nil, true),
		fileCache: make(map[string]*os.File),
		buffers:   make(map[string]*bufio.Writer),
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		storage.logger.Error("Failed to create analytics directory: %v", err)
	}

	return storage
}

// StoreEvents stores analytics events to disk
func (as *AnalyticsStorage) StoreEvents(events []*AnalyticsEvent) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Group events by date for efficient storage
	eventsByDate := make(map[string][]*AnalyticsEvent)
	for _, event := range events {
		dateKey := event.Timestamp.Format("2006-01-02")
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	// Store events for each date
	for dateKey, dateEvents := range eventsByDate {
		if err := as.storeEventsForDate(dateKey, dateEvents); err != nil {
			return fmt.Errorf("failed to store events for %s: %w", dateKey, err)
		}
	}

	return nil
}

// StoreMetrics stores performance metrics to disk
func (as *AnalyticsStorage) StoreMetrics(metrics []*Metric) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Group metrics by date
	metricsByDate := make(map[string][]*Metric)
	for _, metric := range metrics {
		dateKey := metric.Timestamp.Format("2006-01-02")
		metricsByDate[dateKey] = append(metricsByDate[dateKey], metric)
	}

	// Store metrics for each date
	for dateKey, dateMetrics := range metricsByDate {
		if err := as.storeMetricsForDate(dateKey, dateMetrics); err != nil {
			return fmt.Errorf("failed to store metrics for %s: %w", dateKey, err)
		}
	}

	return nil
}

// StoreSession stores session data to disk
func (as *AnalyticsStorage) StoreSession(session *SessionData) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	dateKey := session.StartTime.Format("2006-01-02")
	return as.storeSessionForDate(dateKey, session)
}

// GetEvents retrieves events within a date range
func (as *AnalyticsStorage) GetEvents(startDate, endDate time.Time) ([]*AnalyticsEvent, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var allEvents []*AnalyticsEvent

	// Iterate through each day in the range
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		events, err := as.getEventsForDate(dateKey)
		if err != nil {
			as.logger.Warn("Failed to get events for %s: %v", dateKey, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// Filter events by exact time range
	var filteredEvents []*AnalyticsEvent
	for _, event := range allEvents {
		if (event.Timestamp.After(startDate) || event.Timestamp.Equal(startDate)) &&
			(event.Timestamp.Before(endDate) || event.Timestamp.Equal(endDate)) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents, nil
}

// GetMetrics retrieves metrics within a date range
func (as *AnalyticsStorage) GetMetrics(startDate, endDate time.Time) ([]*Metric, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var allMetrics []*Metric

	// Iterate through each day in the range
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		metrics, err := as.getMetricsForDate(dateKey)
		if err != nil {
			as.logger.Warn("Failed to get metrics for %s: %v", dateKey, err)
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}

	// Filter metrics by exact time range
	var filteredMetrics []*Metric
	for _, metric := range allMetrics {
		if (metric.Timestamp.After(startDate) || metric.Timestamp.Equal(startDate)) &&
			(metric.Timestamp.Before(endDate) || metric.Timestamp.Equal(endDate)) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	return filteredMetrics, nil
}

// GetSessions retrieves sessions within a date range
func (as *AnalyticsStorage) GetSessions(startDate, endDate time.Time) ([]*SessionData, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var allSessions []*SessionData

	// Iterate through each day in the range
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		sessions, err := as.getSessionsForDate(dateKey)
		if err != nil {
			as.logger.Warn("Failed to get sessions for %s: %v", dateKey, err)
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	// Filter sessions by exact time range
	var filteredSessions []*SessionData
	for _, session := range allSessions {
		if (session.StartTime.After(startDate) || session.StartTime.Equal(startDate)) &&
			(session.StartTime.Before(endDate) || session.StartTime.Equal(endDate)) {
			filteredSessions = append(filteredSessions, session)
		}
	}

	return filteredSessions, nil
}

// CleanupOldData removes data older than the specified retention period
func (as *AnalyticsStorage) CleanupOldData(retentionDays int) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Get all subdirectories (dates)
	entries, err := os.ReadDir(as.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read analytics directory: %w", err)
	}

	var removedCount int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse directory name as date
		dirDate, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue // Skip non-date directories
		}

		// Remove if older than cutoff
		if dirDate.Before(cutoffDate) {
			dirPath := filepath.Join(as.baseDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				as.logger.Error("Failed to remove old analytics data %s: %v", dirPath, err)
			} else {
				removedCount++
				as.logger.Debug("Removed old analytics data: %s", entry.Name())
			}
		}
	}

	if removedCount > 0 {
		as.logger.Info("Cleaned up %d days of old analytics data", removedCount)
	}

	return nil
}

// GetStorageStats returns storage statistics
func (as *AnalyticsStorage) GetStorageStats() (map[string]interface{}, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	stats := map[string]interface{}{
		"base_dir": as.baseDir,
	}

	// Calculate total size
	var totalSize int64
	var fileCount int
	var dirCount int

	err := filepath.Walk(as.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirCount++
		} else {
			fileCount++
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return stats, fmt.Errorf("failed to calculate storage stats: %w", err)
	}

	stats["total_size"] = totalSize
	stats["file_count"] = fileCount
	stats["dir_count"] = dirCount
	stats["size_mb"] = float64(totalSize) / (1024 * 1024)

	return stats, nil
}

// storeEventsForDate stores events for a specific date
func (as *AnalyticsStorage) storeEventsForDate(dateKey string, events []*AnalyticsEvent) error {
	filePath := filepath.Join(as.baseDir, dateKey, "events.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	// Write events as JSON lines
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode event: %w", err)
		}
	}

	return nil
}

// storeMetricsForDate stores metrics for a specific date
func (as *AnalyticsStorage) storeMetricsForDate(dateKey string, metrics []*Metric) error {
	filePath := filepath.Join(as.baseDir, dateKey, "metrics.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	// Write metrics as JSON lines
	encoder := json.NewEncoder(file)
	for _, metric := range metrics {
		if err := encoder.Encode(metric); err != nil {
			return fmt.Errorf("failed to encode metric: %w", err)
		}
	}

	return nil
}

// storeSessionForDate stores a session for a specific date
func (as *AnalyticsStorage) storeSessionForDate(dateKey string, session *SessionData) error {
	filePath := filepath.Join(as.baseDir, dateKey, "sessions.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open sessions file: %w", err)
	}
	defer file.Close()

	// Write session as JSON line
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(session); err != nil {
		return fmt.Errorf("failed to encode session: %w", err)
	}

	return nil
}

// getEventsForDate retrieves events for a specific date
func (as *AnalyticsStorage) getEventsForDate(dateKey string) ([]*AnalyticsEvent, error) {
	filePath := filepath.Join(as.baseDir, dateKey, "events.jsonl")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*AnalyticsEvent{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	var events []*AnalyticsEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var event AnalyticsEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			as.logger.Warn("Failed to decode event: %v", err)
			continue
		}
		events = append(events, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan events file: %w", err)
	}

	return events, nil
}

// getMetricsForDate retrieves metrics for a specific date
func (as *AnalyticsStorage) getMetricsForDate(dateKey string) ([]*Metric, error) {
	filePath := filepath.Join(as.baseDir, dateKey, "metrics.jsonl")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*Metric{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	var metrics []*Metric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var metric Metric
		if err := json.Unmarshal(scanner.Bytes(), &metric); err != nil {
			as.logger.Warn("Failed to decode metric: %v", err)
			continue
		}
		metrics = append(metrics, &metric)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan metrics file: %w", err)
	}

	return metrics, nil
}

// getSessionsForDate retrieves sessions for a specific date
func (as *AnalyticsStorage) getSessionsForDate(dateKey string) ([]*SessionData, error) {
	filePath := filepath.Join(as.baseDir, dateKey, "sessions.jsonl")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*SessionData{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sessions file: %w", err)
	}
	defer file.Close()

	var sessions []*SessionData
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var session SessionData
		if err := json.Unmarshal(scanner.Bytes(), &session); err != nil {
			as.logger.Warn("Failed to decode session: %v", err)
			continue
		}
		sessions = append(sessions, &session)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan sessions file: %w", err)
	}

	return sessions, nil
}

// ExportData exports analytics data in various formats
func (as *AnalyticsStorage) ExportData(startDate, endDate time.Time, format string) ([]byte, error) {
	events, err := as.GetEvents(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	metrics, err := as.GetMetrics(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	sessions, err := as.GetSessions(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	exportData := map[string]interface{}{
		"export_info": map[string]interface{}{
			"start_date":    startDate,
			"end_date":      endDate,
			"exported_at":   time.Now(),
			"format":        format,
			"event_count":   len(events),
			"metric_count":  len(metrics),
			"session_count": len(sessions),
		},
		"events":   events,
		"metrics":  metrics,
		"sessions": sessions,
	}

	switch format {
	case "json":
		return json.MarshalIndent(exportData, "", "  ")
	case "csv":
		return as.exportAsCSV(events, metrics, sessions)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportAsCSV exports data as CSV format
func (as *AnalyticsStorage) exportAsCSV(events []*AnalyticsEvent, metrics []*Metric, sessions []*SessionData) ([]byte, error) {
	var csv strings.Builder

	// Export events
	csv.WriteString("Events\n")
	csv.WriteString("ID,Type,Category,Action,Timestamp,SessionID\n")
	for _, event := range events {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			event.ID, event.Type, event.Category, event.Action,
			event.Timestamp.Format(time.RFC3339), event.SessionID))
	}

	csv.WriteString("\nMetrics\n")
	csv.WriteString("Name,Type,Value,Unit,Timestamp\n")
	for _, metric := range metrics {
		csv.WriteString(fmt.Sprintf("%s,%s,%.2f,%s,%s\n",
			metric.Name, metric.Type, metric.Value, metric.Unit,
			metric.Timestamp.Format(time.RFC3339)))
	}

	csv.WriteString("\nSessions\n")
	csv.WriteString("SessionID,UserID,StartTime,Duration,EventCount\n")
	for _, session := range sessions {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%d\n",
			session.SessionID, session.UserID,
			session.StartTime.Format(time.RFC3339),
			session.Duration.Seconds(), session.EventCount))
	}

	return []byte(csv.String()), nil
}

// GetAvailableDates returns all dates with analytics data
func (as *AnalyticsStorage) GetAvailableDates() ([]string, error) {
	entries, err := os.ReadDir(as.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read analytics directory: %w", err)
	}

	var dates []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Validate date format
			if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
				dates = append(dates, entry.Name())
			}
		}
	}

	// Sort dates
	sort.Strings(dates)
	return dates, nil
}

// Close closes any open files and cleans up resources
func (as *AnalyticsStorage) Close() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Close all buffered writers
	for _, buffer := range as.buffers {
		buffer.Flush()
	}

	// Close all open files
	for _, file := range as.fileCache {
		file.Close()
	}

	// Clear caches
	as.fileCache = make(map[string]*os.File)
	as.buffers = make(map[string]*bufio.Writer)

	as.logger.Info("Analytics storage closed")
	return nil
}
