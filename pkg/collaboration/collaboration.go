package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// CollaborationType represents different types of collaboration
type CollaborationType string

const (
	CollabScreenShare    CollaborationType = "screen_share"
	CollabVoiceCall      CollaborationType = "voice_call"
	CollabVideoCall      CollaborationType = "video_call"
	CollabDocumentEdit   CollaborationType = "document_edit"
	CollabWhiteboard     CollaborationType = "whiteboard"
	CollabCodeEdit       CollaborationType = "code_edit"
	CollabPresentation   CollaborationType = "presentation"
	CollabRemoteControl  CollaborationType = "remote_control"
)

// CollaborationStatus represents collaboration session status
type CollaborationStatus string

const (
	StatusInvited    CollaborationStatus = "invited"
	StatusConnecting CollaborationStatus = "connecting"
	StatusActive     CollaborationStatus = "active"
	StatusPaused     CollaborationStatus = "paused"
	StatusEnded      CollaborationStatus = "ended"
	StatusFailed     CollaborationStatus = "failed"
)

// ParticipantRole represents participant roles
type ParticipantRole string

const (
	RoleHost      ParticipantRole = "host"
	RoleModerator ParticipantRole = "moderator"
	RolePresenter ParticipantRole = "presenter"
	RoleViewer    ParticipantRole = "viewer"
	RoleEditor    ParticipantRole = "editor"
)

// CollaborationSession represents a collaboration session
type CollaborationSession struct {
	ID           string                 `json:"id"`
	Type         CollaborationType      `json:"type"`
	Status       CollaborationStatus    `json:"status"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	HostID       string                 `json:"host_id"`
	Participants []*Participant         `json:"participants"`
	Settings     *SessionSettings       `json:"settings"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	EndedAt      *time.Time             `json:"ended_at,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Metadata     map[string]interface{} `json:"metadata"`
	Recording    *RecordingInfo         `json:"recording,omitempty"`
}

// Participant represents a session participant
type Participant struct {
	UserID      string          `json:"user_id"`
	Username    string          `json:"username"`
	Role        ParticipantRole `json:"role"`
	Status      string          `json:"status"` // connected, disconnected, muted, etc.
	JoinedAt    time.Time       `json:"joined_at"`
	LeftAt      *time.Time      `json:"left_at,omitempty"`
	Permissions *Permissions    `json:"permissions"`
	Device      *DeviceInfo     `json:"device,omitempty"`
}

// SessionSettings represents session configuration
type SessionSettings struct {
	MaxParticipants    int           `json:"max_participants"`
	RequireApproval    bool          `json:"require_approval"`
	AllowRecording     bool          `json:"allow_recording"`
	AllowScreenShare   bool          `json:"allow_screen_share"`
	AllowFileSharing   bool          `json:"allow_file_sharing"`
	AllowChat          bool          `json:"allow_chat"`
	AutoRecord         bool          `json:"auto_record"`
	RecordingQuality   string        `json:"recording_quality"`
	SessionTimeout     time.Duration `json:"session_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	Password           string        `json:"password,omitempty"`
	WaitingRoom        bool          `json:"waiting_room"`
	MuteOnJoin         bool          `json:"mute_on_join"`
	VideoOnJoin        bool          `json:"video_on_join"`
}

// Permissions represents participant permissions
type Permissions struct {
	CanShare       bool `json:"can_share"`
	CanRecord      bool `json:"can_record"`
	CanMute        bool `json:"can_mute"`
	CanKick        bool `json:"can_kick"`
	CanInvite      bool `json:"can_invite"`
	CanEdit        bool `json:"can_edit"`
	CanPresent     bool `json:"can_present"`
	CanControl     bool `json:"can_control"`
	CanChat        bool `json:"can_chat"`
	CanAnnotate    bool `json:"can_annotate"`
}

// DeviceInfo represents participant device information
type DeviceInfo struct {
	Type         string `json:"type"`         // desktop, mobile, tablet
	OS           string `json:"os"`           // windows, macos, linux, ios, android
	Browser      string `json:"browser"`      // chrome, firefox, safari, edge
	Version      string `json:"version"`
	Capabilities *DeviceCapabilities `json:"capabilities"`
}

// DeviceCapabilities represents device capabilities
type DeviceCapabilities struct {
	HasCamera     bool `json:"has_camera"`
	HasMicrophone bool `json:"has_microphone"`
	HasSpeakers   bool `json:"has_speakers"`
	CanShare      bool `json:"can_share"`
	CanRecord     bool `json:"can_record"`
	MaxResolution string `json:"max_resolution"`
	Codecs        []string `json:"codecs"`
}

// RecordingInfo represents recording information
type RecordingInfo struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  time.Duration `json:"duration"`
	Size      int64     `json:"size"`
	Path      string    `json:"path"`
	Quality   string    `json:"quality"`
	Format    string    `json:"format"`
}

// CollaborationEvent represents collaboration events
type CollaborationEvent struct {
	Type        string                 `json:"type"`
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

// CollaborationManager manages collaboration sessions
type CollaborationManager struct {
	sessions    map[string]*CollaborationSession
	handlers    map[string]CollaborationHandler
	logger      *logging.Logger
	mu          sync.RWMutex
	eventChan   chan *CollaborationEvent
	ctx         context.Context
	cancel      context.CancelFunc
}

// CollaborationHandler interface for handling collaboration events
type CollaborationHandler interface {
	CanHandle(eventType string) bool
	Handle(ctx context.Context, event *CollaborationEvent) error
	GetHandlerType() string
}

// NewCollaborationManager creates a new collaboration manager
func NewCollaborationManager() *CollaborationManager {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &CollaborationManager{
		sessions:  make(map[string]*CollaborationSession),
		handlers:  make(map[string]CollaborationHandler),
		logger:    logging.NewLogger(logging.INFO, nil, true),
		eventChan: make(chan *CollaborationEvent, 1000),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start event processing
	go cm.processEvents()

	return cm
}

// CreateSession creates a new collaboration session
func (cm *CollaborationManager) CreateSession(sessionType CollaborationType, hostID, title string, settings *SessionSettings) (*CollaborationSession, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if settings == nil {
		settings = &SessionSettings{
			MaxParticipants:  10,
			RequireApproval:  false,
			AllowRecording:   true,
			AllowScreenShare: true,
			AllowFileSharing: true,
			AllowChat:        true,
			AutoRecord:       false,
			RecordingQuality: "high",
			SessionTimeout:   2 * time.Hour,
			IdleTimeout:      30 * time.Minute,
			WaitingRoom:      false,
			MuteOnJoin:       false,
			VideoOnJoin:      true,
		}
	}

	sessionID := generateSessionID()
	session := &CollaborationSession{
		ID:           sessionID,
		Type:         sessionType,
		Status:       StatusInvited,
		Title:        title,
		HostID:       hostID,
		Participants: make([]*Participant, 0),
		Settings:     settings,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Add host as first participant
	hostParticipant := &Participant{
		UserID:   hostID,
		Role:     RoleHost,
		Status:   "connected",
		JoinedAt: time.Now(),
		Permissions: &Permissions{
			CanShare:    true,
			CanRecord:   true,
			CanMute:     true,
			CanKick:     true,
			CanInvite:   true,
			CanEdit:     true,
			CanPresent:  true,
			CanControl:  true,
			CanChat:     true,
			CanAnnotate: true,
		},
	}

	session.Participants = append(session.Participants, hostParticipant)
	cm.sessions[sessionID] = session

	cm.logger.Info("Created collaboration session: %s (%s)", title, sessionID)
	return session, nil
}

// JoinSession allows a user to join a collaboration session
func (cm *CollaborationManager) JoinSession(sessionID, userID, username string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if user is already in session
	for _, participant := range session.Participants {
		if participant.UserID == userID {
			return fmt.Errorf("user already in session: %s", userID)
		}
	}

	// Check participant limit
	if len(session.Participants) >= session.Settings.MaxParticipants {
		return fmt.Errorf("session is full")
	}

	// Create participant
	participant := &Participant{
		UserID:   userID,
		Username: username,
		Role:     RoleViewer,
		Status:   "connected",
		JoinedAt: time.Now(),
		Permissions: &Permissions{
			CanShare:    session.Settings.AllowScreenShare,
			CanRecord:   session.Settings.AllowRecording,
			CanMute:     false,
			CanKick:     false,
			CanInvite:   false,
			CanEdit:     false,
			CanPresent:  false,
			CanControl:  false,
			CanChat:     session.Settings.AllowChat,
			CanAnnotate: false,
		},
	}

	session.Participants = append(session.Participants, participant)

	// Start session if not already started
	if session.Status == StatusInvited {
		session.Status = StatusActive
		now := time.Now()
		session.StartedAt = &now
	}

	// Send join event
	event := &CollaborationEvent{
		Type:      "participant_joined",
		SessionID: sessionID,
		UserID:    userID,
		Data: map[string]interface{}{
			"username": username,
			"role":     string(participant.Role),
		},
		Timestamp: time.Now(),
	}

	select {
	case cm.eventChan <- event:
	default:
		cm.logger.Error("Event queue full, dropping event")
	}

	cm.logger.Info("User %s joined session %s", username, sessionID)
	return nil
}

// LeaveSession allows a user to leave a collaboration session
func (cm *CollaborationManager) LeaveSession(sessionID, userID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Find and remove participant
	for i, participant := range session.Participants {
		if participant.UserID == userID {
			now := time.Now()
			participant.LeftAt = &now
			participant.Status = "disconnected"

			// If host leaves, end session
			if participant.Role == RoleHost {
				session.Status = StatusEnded
				session.EndedAt = &now
				if session.StartedAt != nil {
					session.Duration = now.Sub(*session.StartedAt)
				}
			}

			// Send leave event
			event := &CollaborationEvent{
				Type:      "participant_left",
				SessionID: sessionID,
				UserID:    userID,
				Data: map[string]interface{}{
					"username": participant.Username,
					"role":     string(participant.Role),
				},
				Timestamp: time.Now(),
			}

			select {
			case cm.eventChan <- event:
			default:
				cm.logger.Error("Event queue full, dropping event")
			}

			cm.logger.Info("User %s left session %s", participant.Username, sessionID)
			return nil
		}
	}

	return fmt.Errorf("user not in session: %s", userID)
}

// EndSession ends a collaboration session
func (cm *CollaborationManager) EndSession(sessionID, userID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if user has permission to end session
	canEnd := false
	for _, participant := range session.Participants {
		if participant.UserID == userID && (participant.Role == RoleHost || participant.Role == RoleModerator) {
			canEnd = true
			break
		}
	}

	if !canEnd {
		return fmt.Errorf("user does not have permission to end session")
	}

	// End session
	now := time.Now()
	session.Status = StatusEnded
	session.EndedAt = &now
	if session.StartedAt != nil {
		session.Duration = now.Sub(*session.StartedAt)
	}

	// Send session ended event
	event := &CollaborationEvent{
		Type:      "session_ended",
		SessionID: sessionID,
		UserID:    userID,
		Data: map[string]interface{}{
			"duration": session.Duration.Seconds(),
		},
		Timestamp: time.Now(),
	}

	select {
	case cm.eventChan <- event:
	default:
		cm.logger.Error("Event queue full, dropping event")
	}

	cm.logger.Info("Session ended: %s", sessionID)
	return nil
}

// GetSession retrieves a collaboration session
func (cm *CollaborationManager) GetSession(sessionID string) (*CollaborationSession, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.sessions[sessionID]
	return session, exists
}

// ListSessions lists all collaboration sessions
func (cm *CollaborationManager) ListSessions(userID string) []*CollaborationSession {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var sessions []*CollaborationSession

	for _, session := range cm.sessions {
		// Check if user is participant
		for _, participant := range session.Participants {
			if participant.UserID == userID {
				sessions = append(sessions, session)
				break
			}
		}
	}

	return sessions
}

// UpdateParticipantRole updates a participant's role
func (cm *CollaborationManager) UpdateParticipantRole(sessionID, hostID, targetUserID string, newRole ParticipantRole) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if host has permission
	isHost := false
	for _, participant := range session.Participants {
		if participant.UserID == hostID && (participant.Role == RoleHost || participant.Role == RoleModerator) {
			isHost = true
			break
		}
	}

	if !isHost {
		return fmt.Errorf("user does not have permission to change roles")
	}

	// Update target user's role
	for _, participant := range session.Participants {
		if participant.UserID == targetUserID {
			oldRole := participant.Role
			participant.Role = newRole

			// Update permissions based on role
			cm.updatePermissionsForRole(participant, newRole)

			// Send role change event
			event := &CollaborationEvent{
				Type:      "role_changed",
				SessionID: sessionID,
				UserID:    targetUserID,
				Data: map[string]interface{}{
					"old_role": string(oldRole),
					"new_role": string(newRole),
					"changed_by": hostID,
				},
				Timestamp: time.Now(),
			}

			select {
			case cm.eventChan <- event:
			default:
				cm.logger.Error("Event queue full, dropping event")
			}

			cm.logger.Info("Role changed for user %s in session %s: %s -> %s", targetUserID, sessionID, oldRole, newRole)
			return nil
		}
	}

	return fmt.Errorf("user not found in session: %s", targetUserID)
}

// StartRecording starts recording a session
func (cm *CollaborationManager) StartRecording(sessionID, userID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check permissions
	canRecord := false
	for _, participant := range session.Participants {
		if participant.UserID == userID && participant.Permissions.CanRecord {
			canRecord = true
			break
		}
	}

	if !canRecord {
		return fmt.Errorf("user does not have permission to record")
	}

	if session.Recording != nil && session.Recording.Status == "recording" {
		return fmt.Errorf("session is already being recorded")
	}

	// Start recording
	recordingID := generateRecordingID()
	session.Recording = &RecordingInfo{
		ID:        recordingID,
		Status:    "recording",
		StartedAt: time.Now(),
		Quality:   session.Settings.RecordingQuality,
		Format:    "mp4",
	}

	// Send recording started event
	event := &CollaborationEvent{
		Type:      "recording_started",
		SessionID: sessionID,
		UserID:    userID,
		Data: map[string]interface{}{
			"recording_id": recordingID,
		},
		Timestamp: time.Now(),
	}

	select {
	case cm.eventChan <- event:
	default:
		cm.logger.Error("Event queue full, dropping event")
	}

	cm.logger.Info("Recording started for session %s", sessionID)
	return nil
}

// StopRecording stops recording a session
func (cm *CollaborationManager) StopRecording(sessionID, userID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Recording == nil || session.Recording.Status != "recording" {
		return fmt.Errorf("session is not being recorded")
	}

	// Stop recording
	now := time.Now()
	session.Recording.Status = "completed"
	session.Recording.EndedAt = &now
	session.Recording.Duration = now.Sub(session.Recording.StartedAt)

	// Send recording stopped event
	event := &CollaborationEvent{
		Type:      "recording_stopped",
		SessionID: sessionID,
		UserID:    userID,
		Data: map[string]interface{}{
			"recording_id": session.Recording.ID,
			"duration":     session.Recording.Duration.Seconds(),
		},
		Timestamp: time.Now(),
	}

	select {
	case cm.eventChan <- event:
	default:
		cm.logger.Error("Event queue full, dropping event")
	}

	cm.logger.Info("Recording stopped for session %s", sessionID)
	return nil
}

// RegisterHandler registers a collaboration handler
func (cm *CollaborationManager) RegisterHandler(handler CollaborationHandler) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.handlers[handler.GetHandlerType()] = handler
	cm.logger.Info("Registered collaboration handler: %s", handler.GetHandlerType())
}

// processEvents processes collaboration events
func (cm *CollaborationManager) processEvents() {
	for {
		select {
		case event := <-cm.eventChan:
			cm.handleEvent(event)
		case <-cm.ctx.Done():
			return
		}
	}
}

// handleEvent handles a collaboration event
func (cm *CollaborationManager) handleEvent(event *CollaborationEvent) {
	cm.mu.RLock()
	handlers := make([]CollaborationHandler, 0, len(cm.handlers))
	for _, handler := range cm.handlers {
		if handler.CanHandle(event.Type) {
			handlers = append(handlers, handler)
		}
	}
	cm.mu.RUnlock()

	for _, handler := range handlers {
		go func(h CollaborationHandler) {
			ctx, cancel := context.WithTimeout(cm.ctx, 30*time.Second)
			defer cancel()

			if err := h.Handle(ctx, event); err != nil {
				cm.logger.Error("Handler %s failed to process event %s: %v", h.GetHandlerType(), event.Type, err)
			}
		}(handler)
	}
}

// updatePermissionsForRole updates permissions based on role
func (cm *CollaborationManager) updatePermissionsForRole(participant *Participant, role ParticipantRole) {
	switch role {
	case RoleHost:
		participant.Permissions = &Permissions{
			CanShare:    true,
			CanRecord:   true,
			CanMute:     true,
			CanKick:     true,
			CanInvite:   true,
			CanEdit:     true,
			CanPresent:  true,
			CanControl:  true,
			CanChat:     true,
			CanAnnotate: true,
		}
	case RoleModerator:
		participant.Permissions = &Permissions{
			CanShare:    true,
			CanRecord:   true,
			CanMute:     true,
			CanKick:     true,
			CanInvite:   true,
			CanEdit:     true,
			CanPresent:  true,
			CanControl:  false,
			CanChat:     true,
			CanAnnotate: true,
		}
	case RolePresenter:
		participant.Permissions = &Permissions{
			CanShare:    true,
			CanRecord:   false,
			CanMute:     false,
			CanKick:     false,
			CanInvite:   false,
			CanEdit:     true,
			CanPresent:  true,
			CanControl:  false,
			CanChat:     true,
			CanAnnotate: true,
		}
	case RoleEditor:
		participant.Permissions = &Permissions{
			CanShare:    false,
			CanRecord:   false,
			CanMute:     false,
			CanKick:     false,
			CanInvite:   false,
			CanEdit:     true,
			CanPresent:  false,
			CanControl:  false,
			CanChat:     true,
			CanAnnotate: true,
		}
	case RoleViewer:
		participant.Permissions = &Permissions{
			CanShare:    false,
			CanRecord:   false,
			CanMute:     false,
			CanKick:     false,
			CanInvite:   false,
			CanEdit:     false,
			CanPresent:  false,
			CanControl:  false,
			CanChat:     true,
			CanAnnotate: false,
		}
	}
}

// GetStats returns collaboration statistics
func (cm *CollaborationManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_sessions": len(cm.sessions),
	}

	// Count by status
	statusCounts := make(map[CollaborationStatus]int)
	typeCounts := make(map[CollaborationType]int)
	var totalParticipants int

	for _, session := range cm.sessions {
		statusCounts[session.Status]++
		typeCounts[session.Type]++
		totalParticipants += len(session.Participants)
	}

	stats["by_status"] = statusCounts
	stats["by_type"] = typeCounts
	stats["total_participants"] = totalParticipants

	return stats
}

// Shutdown gracefully shuts down the collaboration manager
func (cm *CollaborationManager) Shutdown() {
	cm.logger.Info("Shutting down collaboration manager...")
	cm.cancel()
	cm.logger.Info("Collaboration manager shutdown complete")
}

// Helper functions
func generateSessionID() string {
	return fmt.Sprintf("collab_%d", time.Now().UnixNano())
}

func generateRecordingID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}
