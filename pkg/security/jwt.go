package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey       []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey, issuer string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secretKey),
		issuer:          issuer,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateTokenPair generates access and refresh tokens
func (jm *JWTManager) GenerateTokenPair(userID, username, email string, roles, permissions []string) (string, string, error) {
	sessionID, err := generateSecureRandomString(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Access token
	accessClaims := &JWTClaims{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   userID,
			Audience:  []string{"plexichat-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jm.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        sessionID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jm.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token (longer TTL, minimal claims)
	refreshClaims := &jwt.RegisteredClaims{
		Issuer:    jm.issuer,
		Subject:   userID,
		Audience:  []string{"plexichat-refresh"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(jm.refreshTokenTTL)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        sessionID,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jm.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, nil
}

// ValidateToken validates and parses a JWT token
func (jm *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Additional validation
	if claims.Issuer != jm.issuer {
		return nil, errors.New("invalid token issuer")
	}

	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

// RefreshToken validates refresh token and generates new access token
func (jm *JWTManager) RefreshToken(refreshTokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(refreshTokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.secretKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid refresh token")
	}

	// Verify audience
	if len(claims.Audience) == 0 || claims.Audience[0] != "plexichat-refresh" {
		return "", errors.New("invalid refresh token audience")
	}

	// TODO: In production, fetch user details from database using claims.Subject
	// For now, create minimal access token
	newClaims := &JWTClaims{
		UserID:    claims.Subject,
		SessionID: claims.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   claims.Subject,
			Audience:  []string{"plexichat-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jm.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        claims.ID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return accessToken.SignedString(jm.secretKey)
}

// ExtractTokenFromRequest extracts JWT token from HTTP request
func ExtractTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Bearer token format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check query parameter (less secure, for WebSocket upgrades)
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	// Check cookie (for web applications)
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// JWTAuthMiddleware validates JWT tokens
func (jm *JWTManager) JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractTokenFromRequest(r)
		if token == "" {
			http.Error(w, "Missing authentication token", http.StatusUnauthorized)
			return
		}

		claims, err := jm.ValidateToken(token)
		if err != nil {
			LogSecurityEvent("INVALID_TOKEN", GetClientIP(r), r.UserAgent(), err.Error())
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "user_claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware checks if user has required permission
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("user_claims").(*JWTClaims)
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has required permission
			hasPermission := false
			for _, perm := range claims.Permissions {
				if perm == permission || perm == "*" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				LogSecurityEvent("PERMISSION_DENIED", GetClientIP(r), r.UserAgent(),
					fmt.Sprintf("User %s lacks permission: %s", claims.Username, permission))
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("user_claims").(*JWTClaims)
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has required role
			hasRole := false
			for _, userRole := range claims.Roles {
				if userRole == role || userRole == "admin" {
					hasRole = true
					break
				}
			}

			if !hasRole {
				LogSecurityEvent("ROLE_DENIED", GetClientIP(r), r.UserAgent(),
					fmt.Sprintf("User %s lacks role: %s", claims.Username, role))
				http.Error(w, "Insufficient role", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateSecureRandomString generates a cryptographically secure random string
func generateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GetUserFromContext extracts user claims from request context
func GetUserFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value("user_claims").(*JWTClaims)
	return claims, ok
}
