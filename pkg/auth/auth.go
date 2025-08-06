package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"plexichat-client/pkg/database"
	"plexichat-client/pkg/logging"

	"golang.org/x/crypto/bcrypt"
)

// AuthManager manages authentication and authorization
type AuthManager struct {
	db          *database.Database
	logger      *logging.Logger
	sessions    map[string]*AuthSession
	tokenSecret []byte
	mu          sync.RWMutex
	config      *AuthConfig
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	TokenExpiry         time.Duration `json:"token_expiry"`
	RefreshTokenExpiry  time.Duration `json:"refresh_token_expiry"`
	MaxLoginAttempts    int           `json:"max_login_attempts"`
	LockoutDuration     time.Duration `json:"lockout_duration"`
	PasswordMinLength   int           `json:"password_min_length"`
	RequireSpecialChars bool          `json:"require_special_chars"`
	RequireNumbers      bool          `json:"require_numbers"`
	RequireUppercase    bool          `json:"require_uppercase"`
	EnableTwoFactor     bool          `json:"enable_two_factor"`
	SessionTimeout      time.Duration `json:"session_timeout"`
}

// AuthSession represents an authenticated session
type AuthSession struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	ExpiresAt    time.Time              `json:"expires_at"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TwoFactor  string `json:"two_factor,omitempty"`
	RememberMe bool   `json:"remember_me"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success      bool      `json:"success"`
	Token        string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
	Error        string    `json:"error,omitempty"`
	RequiresTFA  bool      `json:"requires_tfa,omitempty"`
}

// UserInfo represents user information
type UserInfo struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Avatar      string    `json:"avatar"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	LastLogin   time.Time `json:"last_login"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	IssuedAt    time.Time `json:"iat"`
	ExpiresAt   time.Time `json:"exp"`
	SessionID   string    `json:"session_id"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(db *database.Database, config *AuthConfig) (*AuthManager, error) {
	if config == nil {
		config = DefaultAuthConfig()
	}

	// Generate token secret
	tokenSecret := make([]byte, 32)
	if _, err := rand.Read(tokenSecret); err != nil {
		return nil, fmt.Errorf("failed to generate token secret: %w", err)
	}

	return &AuthManager{
		db:          db,
		logger:      logging.NewLogger(logging.INFO, nil, true),
		sessions:    make(map[string]*AuthSession),
		tokenSecret: tokenSecret,
		config:      config,
	}, nil
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		TokenExpiry:         24 * time.Hour,
		RefreshTokenExpiry:  7 * 24 * time.Hour,
		MaxLoginAttempts:    5,
		LockoutDuration:     15 * time.Minute,
		PasswordMinLength:   8,
		RequireSpecialChars: true,
		RequireNumbers:      true,
		RequireUppercase:    true,
		EnableTwoFactor:     false,
		SessionTimeout:      2 * time.Hour,
	}
}

// Login authenticates a user and creates a session
func (am *AuthManager) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Validate input
	if err := am.validateLoginRequest(req); err != nil {
		return &LoginResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check for account lockout
	if locked, err := am.isAccountLocked(ctx, req.Username); err != nil {
		return nil, fmt.Errorf("failed to check account lockout: %w", err)
	} else if locked {
		return &LoginResponse{
			Success: false,
			Error:   "Account is temporarily locked due to too many failed login attempts",
		}, nil
	}

	// Get user from database
	user, err := am.db.GetUserByUsername(ctx, req.Username)
	if err != nil {
		am.recordFailedLogin(ctx, req.Username, req.IPAddress)
		return &LoginResponse{
			Success: false,
			Error:   "Invalid username or password",
		}, nil
	}

	// Verify password
	if err := am.verifyPassword(user.ID, req.Password); err != nil {
		am.recordFailedLogin(ctx, req.Username, req.IPAddress)
		return &LoginResponse{
			Success: false,
			Error:   "Invalid username or password",
		}, nil
	}

	// Check two-factor authentication if enabled
	if am.config.EnableTwoFactor {
		if req.TwoFactor == "" {
			return &LoginResponse{
				Success:     false,
				RequiresTFA: true,
				Error:       "Two-factor authentication required",
			}, nil
		}

		if !am.verifyTwoFactor(user.ID, req.TwoFactor) {
			am.recordFailedLogin(ctx, req.Username, req.IPAddress)
			return &LoginResponse{
				Success: false,
				Error:   "Invalid two-factor authentication code",
			}, nil
		}
	}

	// Create session
	session, err := am.createSession(ctx, user, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate tokens
	token, err := am.generateToken(session)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := am.generateRefreshToken(session)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Clear failed login attempts
	am.clearFailedLogins(ctx, req.Username)

	// Update last login
	am.db.UpdateUserLastSeen(ctx, user.ID)

	return &LoginResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    session.ExpiresAt,
		User: &UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Avatar:      user.Avatar,
			LastLogin:   time.Now(),
		},
	}, nil
}

// Logout invalidates a session
func (am *AuthManager) Logout(ctx context.Context, sessionID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Remove from memory
	delete(am.sessions, sessionID)

	// Remove from database
	if err := am.db.DeleteSession(ctx, sessionID); err != nil {
		am.logger.Error("Failed to delete session from database: %v", err)
	}

	am.logger.Info("User logged out, session: %s", sessionID)
	return nil
}

// ValidateToken validates a JWT token and returns the session
func (am *AuthManager) ValidateToken(ctx context.Context, token string) (*AuthSession, error) {
	// Parse and validate token
	claims, err := am.parseToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	// Get session
	am.mu.RLock()
	session, exists := am.sessions[claims.SessionID]
	am.mu.RUnlock()

	if !exists {
		// Try to load from database
		dbSession, err := am.db.GetSession(ctx, claims.SessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %w", err)
		}

		session = &AuthSession{
			ID:           dbSession.ID,
			UserID:       dbSession.UserID,
			CreatedAt:    dbSession.CreatedAt,
			LastActivity: dbSession.LastUsed,
			ExpiresAt:    dbSession.ExpiresAt,
			IPAddress:    dbSession.IPAddress,
			UserAgent:    dbSession.UserAgent,
		}

		// Cache session
		am.mu.Lock()
		am.sessions[session.ID] = session
		am.mu.Unlock()
	}

	// Check session expiration
	if time.Now().After(session.ExpiresAt) {
		am.Logout(ctx, session.ID)
		return nil, fmt.Errorf("session expired")
	}

	// Update last activity
	session.LastActivity = time.Now()
	am.db.UpdateSessionLastUsed(ctx, session.ID)

	return session, nil
}

// RefreshToken refreshes an access token using a refresh token
func (am *AuthManager) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Validate refresh token
	claims, err := am.parseRefreshToken(refreshToken)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Error:   "Invalid refresh token",
		}, nil
	}

	// Get session
	session, err := am.ValidateToken(ctx, claims.SessionID)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Error:   "Invalid session",
		}, nil
	}

	// Generate new access token
	newToken, err := am.generateToken(session)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	return &LoginResponse{
		Success:   true,
		Token:     newToken,
		ExpiresAt: time.Now().Add(am.config.TokenExpiry),
	}, nil
}

// Helper methods

func (am *AuthManager) validateLoginRequest(req *LoginRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.Password) < am.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", am.config.PasswordMinLength)
	}
	return nil
}

func (am *AuthManager) isAccountLocked(ctx context.Context, username string) (bool, error) {
	// Implementation would check failed login attempts
	return false, nil
}

func (am *AuthManager) recordFailedLogin(ctx context.Context, username, ipAddress string) {
	am.logger.Warn("Failed login attempt for user %s from %s", username, ipAddress)
	// Implementation would record failed attempt in database
}

func (am *AuthManager) clearFailedLogins(ctx context.Context, username string) {
	// Implementation would clear failed login attempts
}

func (am *AuthManager) verifyPassword(userID, password string) error {
	// This would get the hashed password from database and verify
	// For now, we'll use a simple check
	return bcrypt.CompareHashAndPassword([]byte("$2a$10$example"), []byte(password))
}

func (am *AuthManager) verifyTwoFactor(userID, code string) bool {
	// Implementation would verify TOTP code
	return code == "123456" // Placeholder
}

func (am *AuthManager) createSession(ctx context.Context, user *database.User, req *LoginRequest) (*AuthSession, error) {
	sessionID := am.generateSessionID()

	session := &AuthSession{
		ID:           sessionID,
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(am.config.SessionTimeout),
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Metadata:     make(map[string]interface{}),
	}

	// Save to database
	dbSession := &database.Session{
		ID:        session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
		LastUsed:  session.LastActivity,
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
	}

	if err := am.db.SaveSession(ctx, dbSession); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Cache session
	am.mu.Lock()
	am.sessions[sessionID] = session
	am.mu.Unlock()

	return session, nil
}

func (am *AuthManager) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (am *AuthManager) generateToken(session *AuthSession) (string, error) {
	claims := &TokenClaims{
		UserID:      session.UserID,
		Username:    session.Username,
		Email:       session.Email,
		Roles:       session.Roles,
		Permissions: session.Permissions,
		IssuedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(am.config.TokenExpiry),
		SessionID:   session.ID,
	}

	// Simple token generation (in production, use proper JWT)
	data, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

func (am *AuthManager) generateRefreshToken(session *AuthSession) (string, error) {
	// Simple refresh token generation
	data := map[string]interface{}{
		"session_id": session.ID,
		"expires_at": time.Now().Add(am.config.RefreshTokenExpiry),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(jsonData), nil
}

func (am *AuthManager) parseToken(token string) (*TokenClaims, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	var claims TokenClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("invalid token data: %w", err)
	}

	return &claims, nil
}

func (am *AuthManager) parseRefreshToken(token string) (*TokenClaims, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token format: %w", err)
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("invalid refresh token data: %w", err)
	}

	sessionID, ok := tokenData["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid session ID in refresh token")
	}

	return &TokenClaims{SessionID: sessionID}, nil
}
