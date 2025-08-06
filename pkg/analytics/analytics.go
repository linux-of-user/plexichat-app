package analytics

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/security"
)

// EventType represents different types of analytics events
type EventType string

const (
	EventUserAction      EventType = "user_action"
	EventSystemEvent     EventType = "system_event"
	EventPerformance     EventType = "performance"
	EventError           EventType = "error"
	EventNavigation      EventType = "navigation"
	EventFeatureUsage    EventType = "feature_usage"
	EventSessionStart    EventType = "session_start"
	EventSessionEnd      EventType = "session_end"
	EventMessageSent     EventType = "message_sent"
	EventMessageReceived EventType = "message_received"
	EventFileUpload      EventType = "file_upload"
	EventFileDownload    EventType = "file_download"
	EventSearch          EventType = "search"
	EventConfigChange    EventType = "config_change"
	EventPluginAction    EventType = "plugin_action"
)

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	Category   string                 `json:"category"`
	Action     string                 `json:"action"`
	Label      string                 `json:"label,omitempty"`
	Value      float64                `json:"value,omitempty"`
	Properties map[string]interface{} `json:"properties"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Context    EventContext           `json:"context"`
}

// EventContext provides context about the event
type EventContext struct {
	AppVersion   string `json:"app_version"`
	Platform     string `json:"platform"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	ScreenWidth  int    `json:"screen_width,omitempty"`
	ScreenHeight int    `json:"screen_height,omitempty"`
	WindowWidth  int    `json:"window_width,omitempty"`
	WindowHeight int    `json:"window_height,omitempty"`
	Theme        string `json:"theme,omitempty"`
	PluginsCount int    `json:"plugins_count,omitempty"`
}

// MetricType represents different types of metrics
type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
	MetricTimer     MetricType = "timer"
	MetricSet       MetricType = "set"
)

// Metric represents a performance metric
type Metric struct {
	Name       string                 `json:"name"`
	Type       MetricType             `json:"type"`
	Value      float64                `json:"value"`
	Unit       string                 `json:"unit"`
	Tags       map[string]string      `json:"tags"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
}

// PerformanceData represents performance measurements
type PerformanceData struct {
	CPUUsage       float64       `json:"cpu_usage"`
	MemoryUsage    int64         `json:"memory_usage"`
	MemoryTotal    int64         `json:"memory_total"`
	GoroutineCount int           `json:"goroutine_count"`
	ResponseTime   time.Duration `json:"response_time"`
	Throughput     float64       `json:"throughput"`
	ErrorRate      float64       `json:"error_rate"`
	Uptime         time.Duration `json:"uptime"`
	NetworkIn      int64         `json:"network_in"`
	NetworkOut     int64         `json:"network_out"`
	DiskUsage      int64         `json:"disk_usage"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
}

// SessionData represents user session information
type SessionData struct {
	SessionID     string        `json:"session_id"`
	UserID        string        `json:"user_id"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	EventCount    int           `json:"event_count"`
	MessagesSent  int           `json:"messages_sent"`
	FilesUploaded int           `json:"files_uploaded"`
	SearchCount   int           `json:"search_count"`
	ErrorCount    int           `json:"error_count"`
	Features      []string      `json:"features_used"`
	LastActivity  time.Time     `json:"last_activity"`
}

// AnalyticsConfig represents analytics configuration
type AnalyticsConfig struct {
	Enabled             bool          `json:"enabled"`
	SamplingRate        float64       `json:"sampling_rate"`
	BatchSize           int           `json:"batch_size"`
	FlushInterval       time.Duration `json:"flush_interval"`
	RetentionDays       int           `json:"retention_days"`
	AnonymizeData       bool          `json:"anonymize_data"`
	CollectPerformance  bool          `json:"collect_performance"`
	CollectErrors       bool          `json:"collect_errors"`
	CollectFeatureUsage bool          `json:"collect_feature_usage"`
	ExportFormat        string        `json:"export_format"`
	StorageDir          string        `json:"storage_dir"`
}

// Analytics manages analytics collection and reporting
type Analytics struct {
	config         *AnalyticsConfig
	events         chan *AnalyticsEvent
	metrics        chan *Metric
	sessions       map[string]*SessionData
	currentSession *SessionData
	storage        *AnalyticsStorage
	logger         *logging.Logger
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	startTime      time.Time
}

// NewAnalytics creates a new analytics instance
func NewAnalytics(config *AnalyticsConfig) *Analytics {
	if config == nil {
		config = &AnalyticsConfig{
			Enabled:             true,
			SamplingRate:        1.0,
			BatchSize:           100,
			FlushInterval:       30 * time.Second,
			RetentionDays:       30,
			AnonymizeData:       true,
			CollectPerformance:  true,
			CollectErrors:       true,
			CollectFeatureUsage: true,
			ExportFormat:        "json",
			StorageDir:          "analytics",
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	analytics := &Analytics{
		config:    config,
		events:    make(chan *AnalyticsEvent, 1000),
		metrics:   make(chan *Metric, 1000),
		sessions:  make(map[string]*SessionData),
		storage:   NewAnalyticsStorage(config.StorageDir),
		logger:    logging.NewLogger(logging.INFO, nil, true),
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	// Start background processing
	if config.Enabled {
		go analytics.processEvents()
		go analytics.collectSystemMetrics()
	}

	return analytics
}

// TrackEvent tracks an analytics event
func (a *Analytics) TrackEvent(eventType EventType, category, action string, properties map[string]interface{}) {
	if !a.config.Enabled {
		return
	}

	// Security validation
	category = security.SanitizeInput(category)
	action = security.SanitizeInput(action)

	// Validate properties for security
	if properties != nil {
		if err := security.ValidateRequestBody(properties); err != nil {
			a.logger.Error("Analytics event properties validation failed: %v", err)
			return
		}
	}

	// Apply sampling
	if a.config.SamplingRate < 1.0 {
		// Simple sampling implementation
		if time.Now().UnixNano()%100 >= int64(a.config.SamplingRate*100) {
			return
		}
	}

	event := &AnalyticsEvent{
		ID:         generateEventID(),
		Type:       eventType,
		Category:   category,
		Action:     action,
		Properties: properties,
		SessionID:  a.getCurrentSessionID(),
		Timestamp:  time.Now(),
		Context:    a.getEventContext(),
	}

	select {
	case a.events <- event:
	default:
		a.logger.Error("Analytics event queue full, dropping event")
	}
}

// TrackPerformance tracks a performance metric
func (a *Analytics) TrackPerformance(name string, value float64, unit string, tags map[string]string) {
	if !a.config.Enabled || !a.config.CollectPerformance {
		return
	}

	metric := &Metric{
		Name:      name,
		Type:      MetricGauge,
		Value:     value,
		Unit:      unit,
		Tags:      tags,
		Timestamp: time.Now(),
	}

	select {
	case a.metrics <- metric:
	default:
		a.logger.Error("Analytics metrics queue full, dropping metric")
	}
}

// TrackTimer tracks execution time
func (a *Analytics) TrackTimer(name string, duration time.Duration, tags map[string]string) {
	if !a.config.Enabled || !a.config.CollectPerformance {
		return
	}

	metric := &Metric{
		Name:      name,
		Type:      MetricTimer,
		Value:     float64(duration.Milliseconds()),
		Unit:      "ms",
		Tags:      tags,
		Timestamp: time.Now(),
	}

	select {
	case a.metrics <- metric:
	default:
		a.logger.Error("Analytics metrics queue full, dropping metric")
	}
}

// StartSession starts a new analytics session
func (a *Analytics) StartSession(userID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	sessionID := generateSessionID()
	session := &SessionData{
		SessionID:    sessionID,
		UserID:       userID,
		StartTime:    time.Now(),
		Features:     make([]string, 0),
		LastActivity: time.Now(),
	}

	a.sessions[sessionID] = session
	a.currentSession = session

	// Track session start event
	a.TrackEvent(EventSessionStart, "session", "start", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
	})

	a.logger.Info("Started analytics session: %s", sessionID)
	return sessionID
}

// EndSession ends the current analytics session
func (a *Analytics) EndSession() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentSession == nil {
		return
	}

	session := a.currentSession
	session.EndTime = time.Now()
	session.Duration = session.EndTime.Sub(session.StartTime)

	// Track session end event
	a.TrackEvent(EventSessionEnd, "session", "end", map[string]interface{}{
		"session_id": session.SessionID,
		"duration":   session.Duration.Seconds(),
		"events":     session.EventCount,
	})

	// Store session data
	a.storage.StoreSession(session)

	a.logger.Info("Ended analytics session: %s (duration: %v)", session.SessionID, session.Duration)
	a.currentSession = nil
}

// GetSessionStats returns session statistics
func (a *Analytics) GetSessionStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.currentSession == nil {
		return map[string]interface{}{
			"active": false,
		}
	}

	session := a.currentSession
	return map[string]interface{}{
		"active":         true,
		"session_id":     session.SessionID,
		"duration":       time.Since(session.StartTime).Seconds(),
		"events":         session.EventCount,
		"messages_sent":  session.MessagesSent,
		"files_uploaded": session.FilesUploaded,
		"search_count":   session.SearchCount,
		"error_count":    session.ErrorCount,
		"features_used":  len(session.Features),
	}
}

// GetAnalyticsReport generates an analytics report
func (a *Analytics) GetAnalyticsReport(startDate, endDate time.Time) (*AnalyticsReport, error) {
	events, err := a.storage.GetEvents(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	metrics, err := a.storage.GetMetrics(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	sessions, err := a.storage.GetSessions(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	report := &AnalyticsReport{
		StartDate:    startDate,
		EndDate:      endDate,
		GeneratedAt:  time.Now(),
		EventCount:   len(events),
		MetricCount:  len(metrics),
		SessionCount: len(sessions),
		Summary:      a.generateSummary(events, metrics, sessions),
		TopEvents:    a.getTopEvents(events),
		TopFeatures:  a.getTopFeatures(events),
		Performance:  a.getPerformanceSummary(metrics),
		UserActivity: a.getUserActivity(sessions),
	}

	return report, nil
}

// processEvents processes analytics events in background
func (a *Analytics) processEvents() {
	ticker := time.NewTicker(a.config.FlushInterval)
	defer ticker.Stop()

	var batch []*AnalyticsEvent
	var metricBatch []*Metric

	for {
		select {
		case event := <-a.events:
			batch = append(batch, event)
			a.updateSessionStats(event)

			if len(batch) >= a.config.BatchSize {
				a.flushEvents(batch)
				batch = nil
			}

		case metric := <-a.metrics:
			metricBatch = append(metricBatch, metric)

			if len(metricBatch) >= a.config.BatchSize {
				a.flushMetrics(metricBatch)
				metricBatch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				a.flushEvents(batch)
				batch = nil
			}
			if len(metricBatch) > 0 {
				a.flushMetrics(metricBatch)
				metricBatch = nil
			}

		case <-a.ctx.Done():
			// Flush remaining events before shutdown
			if len(batch) > 0 {
				a.flushEvents(batch)
			}
			if len(metricBatch) > 0 {
				a.flushMetrics(metricBatch)
			}
			return
		}
	}
}

// flushEvents stores events to persistent storage
func (a *Analytics) flushEvents(events []*AnalyticsEvent) {
	if err := a.storage.StoreEvents(events); err != nil {
		a.logger.Error("Failed to store analytics events: %v", err)
	} else {
		a.logger.Debug("Stored %d analytics events", len(events))
	}
}

// flushMetrics stores metrics to persistent storage
func (a *Analytics) flushMetrics(metrics []*Metric) {
	if err := a.storage.StoreMetrics(metrics); err != nil {
		a.logger.Error("Failed to store analytics metrics: %v", err)
	} else {
		a.logger.Debug("Stored %d analytics metrics", len(metrics))
	}
}

// updateSessionStats updates session statistics
func (a *Analytics) updateSessionStats(event *AnalyticsEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentSession == nil {
		return
	}

	session := a.currentSession
	session.EventCount++
	session.LastActivity = time.Now()

	// Update specific counters based on event type
	switch event.Type {
	case EventMessageSent:
		session.MessagesSent++
	case EventFileUpload:
		session.FilesUploaded++
	case EventSearch:
		session.SearchCount++
	case EventError:
		session.ErrorCount++
	}

	// Track feature usage
	if event.Category != "" {
		found := false
		for _, feature := range session.Features {
			if feature == event.Category {
				found = true
				break
			}
		}
		if !found {
			session.Features = append(session.Features, event.Category)
		}
	}
}

// getCurrentSessionID returns the current session ID
func (a *Analytics) getCurrentSessionID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.currentSession != nil {
		return a.currentSession.SessionID
	}
	return ""
}

// getEventContext returns event context information
func (a *Analytics) getEventContext() EventContext {
	return EventContext{
		AppVersion:   "1.0.0",
		Platform:     "desktop",
		OS:           "windows", // This would be detected at runtime
		Architecture: "amd64",   // This would be detected at runtime
		Language:     "en",
		Timezone:     time.Now().Location().String(),
	}
}

// collectSystemMetrics collects system performance metrics
func (a *Analytics) collectSystemMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Collect system metrics
			// This would use actual system monitoring libraries
			a.TrackPerformance("memory_usage", 1024*1024*100, "bytes", map[string]string{"type": "heap"})
			a.TrackPerformance("cpu_usage", 15.5, "percent", map[string]string{"type": "process"})
			a.TrackPerformance("goroutines", 25, "count", map[string]string{"type": "runtime"})

		case <-a.ctx.Done():
			return
		}
	}
}

// Helper functions for report generation
func (a *Analytics) generateSummary(events []*AnalyticsEvent, metrics []*Metric, sessions []*SessionData) map[string]interface{} {
	return map[string]interface{}{
		"total_events":   len(events),
		"total_metrics":  len(metrics),
		"total_sessions": len(sessions),
		"avg_session_duration": func() float64 {
			if len(sessions) == 0 {
				return 0
			}
			total := time.Duration(0)
			for _, session := range sessions {
				total += session.Duration
			}
			return total.Seconds() / float64(len(sessions))
		}(),
	}
}

func (a *Analytics) getTopEvents(events []*AnalyticsEvent) []map[string]interface{} {
	eventCounts := make(map[string]int)
	for _, event := range events {
		key := fmt.Sprintf("%s:%s", event.Category, event.Action)
		eventCounts[key]++
	}

	type eventCount struct {
		Event string
		Count int
	}

	var sorted []eventCount
	for event, count := range eventCounts {
		sorted = append(sorted, eventCount{Event: event, Count: count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	var result []map[string]interface{}
	for i, ec := range sorted {
		if i >= 10 { // Top 10
			break
		}
		result = append(result, map[string]interface{}{
			"event": ec.Event,
			"count": ec.Count,
		})
	}

	return result
}

func (a *Analytics) getTopFeatures(events []*AnalyticsEvent) []map[string]interface{} {
	featureCounts := make(map[string]int)
	for _, event := range events {
		if event.Category != "" {
			featureCounts[event.Category]++
		}
	}

	type featureCount struct {
		Feature string
		Count   int
	}

	var sorted []featureCount
	for feature, count := range featureCounts {
		sorted = append(sorted, featureCount{Feature: feature, Count: count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	var result []map[string]interface{}
	for i, fc := range sorted {
		if i >= 10 { // Top 10
			break
		}
		result = append(result, map[string]interface{}{
			"feature": fc.Feature,
			"count":   fc.Count,
		})
	}

	return result
}

func (a *Analytics) getPerformanceSummary(metrics []*Metric) map[string]interface{} {
	if len(metrics) == 0 {
		return map[string]interface{}{}
	}

	// Calculate averages for different metric types
	metricSums := make(map[string]float64)
	metricCounts := make(map[string]int)

	for _, metric := range metrics {
		metricSums[metric.Name] += metric.Value
		metricCounts[metric.Name]++
	}

	averages := make(map[string]float64)
	for name, sum := range metricSums {
		averages[name] = sum / float64(metricCounts[name])
	}

	return map[string]interface{}{
		"averages": averages,
		"counts":   metricCounts,
	}
}

func (a *Analytics) getUserActivity(sessions []*SessionData) map[string]interface{} {
	if len(sessions) == 0 {
		return map[string]interface{}{}
	}

	totalDuration := time.Duration(0)
	totalEvents := 0
	totalMessages := 0

	for _, session := range sessions {
		totalDuration += session.Duration
		totalEvents += session.EventCount
		totalMessages += session.MessagesSent
	}

	return map[string]interface{}{
		"avg_session_duration":     totalDuration.Seconds() / float64(len(sessions)),
		"avg_events_per_session":   float64(totalEvents) / float64(len(sessions)),
		"avg_messages_per_session": float64(totalMessages) / float64(len(sessions)),
		"total_sessions":           len(sessions),
	}
}

// AnalyticsReport represents a comprehensive analytics report
type AnalyticsReport struct {
	StartDate    time.Time                `json:"start_date"`
	EndDate      time.Time                `json:"end_date"`
	GeneratedAt  time.Time                `json:"generated_at"`
	EventCount   int                      `json:"event_count"`
	MetricCount  int                      `json:"metric_count"`
	SessionCount int                      `json:"session_count"`
	Summary      map[string]interface{}   `json:"summary"`
	TopEvents    []map[string]interface{} `json:"top_events"`
	TopFeatures  []map[string]interface{} `json:"top_features"`
	Performance  map[string]interface{}   `json:"performance"`
	UserActivity map[string]interface{}   `json:"user_activity"`
}

// Helper functions
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

// Shutdown gracefully shuts down analytics
func (a *Analytics) Shutdown() {
	a.logger.Info("Shutting down analytics...")
	a.cancel()

	// End current session if active
	if a.currentSession != nil {
		a.EndSession()
	}

	a.logger.Info("Analytics shutdown complete")
}
