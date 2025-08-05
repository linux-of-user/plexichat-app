package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"plexichat-client/pkg/logging"
)

// EncryptionManager handles message encryption and decryption
type EncryptionManager struct {
	logger *logging.Logger
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager() *EncryptionManager {
	return &EncryptionManager{
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// GenerateKey generates a key from a password using simple key derivation
func (em *EncryptionManager) GenerateKey(password string, salt []byte) []byte {
	// Simple key derivation - in production use proper PBKDF2
	combined := append([]byte(password), salt...)
	hash := sha256.Sum256(combined)
	return hash[:]
}

// GenerateSalt generates a random salt
func (em *EncryptionManager) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// EncryptMessage encrypts a message using AES-GCM
func (em *EncryptionManager) EncryptMessage(message, password string) (string, error) {
	// Generate salt
	salt, err := em.GenerateSalt()
	if err != nil {
		return "", err
	}

	// Generate key from password
	key := em.GenerateKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt message
	ciphertext := gcm.Seal(nonce, nonce, []byte(message), nil)

	// Combine salt and ciphertext
	result := append(salt, ciphertext...)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(result)

	em.logger.Debug("Message encrypted successfully")
	return encoded, nil
}

// DecryptMessage decrypts a message using AES-GCM
func (em *EncryptionManager) DecryptMessage(encryptedMessage, password string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encryptedMessage)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < 16 {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract salt and ciphertext
	salt := data[:16]
	ciphertext := data[16:]

	// Generate key from password
	key := em.GenerateKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and encrypted data
	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt message
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	em.logger.Debug("Message decrypted successfully")
	return string(plaintext), nil
}

// SecureStorage handles secure storage of sensitive data
type SecureStorage struct {
	encryptionManager *EncryptionManager
	logger            *logging.Logger
}

// NewSecureStorage creates a new secure storage instance
func NewSecureStorage() *SecureStorage {
	return &SecureStorage{
		encryptionManager: NewEncryptionManager(),
		logger:            logging.NewLogger(logging.INFO, nil, true),
	}
}

// StoreCredentials securely stores user credentials
func (ss *SecureStorage) StoreCredentials(username, password, masterPassword string) (string, error) {
	// Convert to JSON-like string
	credStr := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)

	// Encrypt credentials
	encrypted, err := ss.encryptionManager.EncryptMessage(credStr, masterPassword)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	ss.logger.Info("Credentials stored securely")
	return encrypted, nil
}

// RetrieveCredentials retrieves and decrypts stored credentials
func (ss *SecureStorage) RetrieveCredentials(encryptedCredentials, masterPassword string) (username, password string, err error) {
	// Decrypt credentials
	credStr, err := ss.encryptionManager.DecryptMessage(encryptedCredentials, masterPassword)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Parse credentials (simple parsing for this example)
	// In a real implementation, you'd use proper JSON parsing
	if len(credStr) > 20 {
		// Extract username and password from JSON-like string
		// This is a simplified parser - use proper JSON in production
		start := len(`{"username":"`)
		end := credStr[start:]
		usernameEnd := start
		for i, char := range end {
			if char == '"' {
				usernameEnd = start + i
				break
			}
		}
		username = credStr[start:usernameEnd]

		passwordStart := usernameEnd + len(`","password":"`)
		passwordEnd := len(credStr) - 2 // Remove "}
		password = credStr[passwordStart:passwordEnd]
	}

	ss.logger.Info("Credentials retrieved successfully")
	return username, password, nil
}

// SecurityValidator provides additional security validation
type SecurityValidator struct {
	logger *logging.Logger
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// ValidatePasswordStrength validates password strength
func (sv *SecurityValidator) ValidatePasswordStrength(password string) (bool, []string) {
	var issues []string

	if len(password) < 8 {
		issues = append(issues, "Password must be at least 8 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 32 && char <= 126: // Printable ASCII
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
				hasSpecial = true
			}
		}
	}

	if !hasUpper {
		issues = append(issues, "Password must contain at least one uppercase letter")
	}
	if !hasLower {
		issues = append(issues, "Password must contain at least one lowercase letter")
	}
	if !hasDigit {
		issues = append(issues, "Password must contain at least one digit")
	}
	if !hasSpecial {
		issues = append(issues, "Password must contain at least one special character")
	}

	return len(issues) == 0, issues
}

// ValidateServerCertificate validates server SSL certificate
func (sv *SecurityValidator) ValidateServerCertificate(serverURL string) error {
	// This would implement certificate validation
	// For now, just log the validation attempt
	sv.logger.Info("Validating server certificate for: %s", serverURL)

	// In a real implementation, this would:
	// 1. Connect to the server
	// 2. Retrieve the certificate
	// 3. Validate the certificate chain
	// 4. Check for revocation
	// 5. Verify the hostname matches

	return nil
}

// SanitizeInput sanitizes user input to prevent injection attacks
func (sv *SecurityValidator) SanitizeInput(input string) string {
	// Remove potentially dangerous characters
	sanitized := input

	// Remove null bytes
	for i := 0; i < len(sanitized); i++ {
		if sanitized[i] == 0 {
			sanitized = sanitized[:i] + sanitized[i+1:]
			i--
		}
	}

	// Remove control characters except newline and tab
	result := make([]rune, 0, len(sanitized))
	for _, char := range sanitized {
		if char >= 32 || char == '\n' || char == '\t' {
			result = append(result, char)
		}
	}

	return string(result)
}

// GenerateSecureToken generates a cryptographically secure random token
func (sv *SecurityValidator) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashPassword creates a secure hash of a password
func (sv *SecurityValidator) HashPassword(password string) (string, error) {
	salt, err := sv.GenerateSecureToken(16)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(password + salt))
	return base64.StdEncoding.EncodeToString(hash[:]) + ":" + salt, nil
}

// VerifyPassword verifies a password against its hash
func (sv *SecurityValidator) VerifyPassword(password, hashedPassword string) bool {
	parts := make([]string, 0, 2)
	colonIndex := -1
	for i, char := range hashedPassword {
		if char == ':' {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		return false
	}

	parts = append(parts, hashedPassword[:colonIndex])
	parts = append(parts, hashedPassword[colonIndex+1:])

	if len(parts) != 2 {
		return false
	}

	hash := sha256.Sum256([]byte(password + parts[1]))
	expectedHash := base64.StdEncoding.EncodeToString(hash[:])

	return expectedHash == parts[0]
}
