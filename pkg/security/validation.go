package security

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s", len(e), e[0].Message)
}

// Validator provides input validation functions
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// Clear clears all validation errors
func (v *Validator) Clear() {
	v.errors = v.errors[:0]
}

// ValidateRequired checks if field is not empty
func (v *Validator) ValidateRequired(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "This field is required", "REQUIRED")
	}
}

// ValidateEmail validates email format
func (v *Validator) ValidateEmail(field, email string) {
	if email == "" {
		return // Use ValidateRequired separately if needed
	}

	// Basic format check
	_, err := mail.ParseAddress(email)
	if err != nil {
		v.AddError(field, "Invalid email format", "INVALID_EMAIL")
		return
	}

	// Additional checks
	if len(email) > 254 {
		v.AddError(field, "Email address too long", "EMAIL_TOO_LONG")
	}

	// Check for dangerous characters
	if containsDangerousChars(email) {
		v.AddError(field, "Email contains invalid characters", "INVALID_CHARACTERS")
	}
}

// ValidateUsername validates username format
func (v *Validator) ValidateUsername(field, username string) {
	if username == "" {
		return
	}

	// Length check
	if len(username) < 3 {
		v.AddError(field, "Username must be at least 3 characters", "USERNAME_TOO_SHORT")
	}
	if len(username) > 30 {
		v.AddError(field, "Username must be less than 30 characters", "USERNAME_TOO_LONG")
	}

	// Character validation
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !usernameRegex.MatchString(username) {
		v.AddError(field, "Username can only contain letters, numbers, underscores, and hyphens", "INVALID_USERNAME")
	}

	// Reserved usernames
	reserved := []string{"admin", "root", "system", "api", "www", "mail", "ftp", "support", "help", "info"}
	for _, r := range reserved {
		if strings.ToLower(username) == r {
			v.AddError(field, "This username is reserved", "RESERVED_USERNAME")
			break
		}
	}
}

// ValidatePassword validates password strength
func (v *Validator) ValidatePassword(field, password string) {
	if password == "" {
		return
	}

	// Length check
	if len(password) < 8 {
		v.AddError(field, "Password must be at least 8 characters", "PASSWORD_TOO_SHORT")
	}
	if len(password) > 128 {
		v.AddError(field, "Password must be less than 128 characters", "PASSWORD_TOO_LONG")
	}

	// Strength requirements
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		v.AddError(field, "Password must contain at least one uppercase letter", "PASSWORD_NO_UPPER")
	}
	if !hasLower {
		v.AddError(field, "Password must contain at least one lowercase letter", "PASSWORD_NO_LOWER")
	}
	if !hasDigit {
		v.AddError(field, "Password must contain at least one digit", "PASSWORD_NO_DIGIT")
	}
	if !hasSpecial {
		v.AddError(field, "Password must contain at least one special character", "PASSWORD_NO_SPECIAL")
	}

	// Common password check
	commonPasswords := []string{
		"password", "123456", "123456789", "qwerty", "abc123", "password123",
		"admin", "letmein", "welcome", "monkey", "dragon", "master",
	}
	for _, common := range commonPasswords {
		if strings.ToLower(password) == common {
			v.AddError(field, "This password is too common", "COMMON_PASSWORD")
			break
		}
	}
}

// ValidateLength validates string length
func (v *Validator) ValidateLength(field, value string, min, max int) {
	length := len(value)
	if length < min {
		v.AddError(field, fmt.Sprintf("Must be at least %d characters", min), "TOO_SHORT")
	}
	if length > max {
		v.AddError(field, fmt.Sprintf("Must be less than %d characters", max), "TOO_LONG")
	}
}

// ValidateAlphanumeric validates alphanumeric characters only
func (v *Validator) ValidateAlphanumeric(field, value string) {
	if value == "" {
		return
	}

	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumericRegex.MatchString(value) {
		v.AddError(field, "Must contain only letters and numbers", "INVALID_ALPHANUMERIC")
	}
}

// ValidateNoHTML validates that input contains no HTML tags
func (v *Validator) ValidateNoHTML(field, value string) {
	if value == "" {
		return
	}

	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	if htmlRegex.MatchString(value) {
		v.AddError(field, "HTML tags are not allowed", "HTML_NOT_ALLOWED")
	}
}

// ValidateNoSQL validates that input contains no SQL injection patterns
func (v *Validator) ValidateNoSQL(field, value string) {
	if value == "" {
		return
	}

	// Common SQL injection patterns
	sqlPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "script", "javascript", "vbscript",
	}

	lowerValue := strings.ToLower(value)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerValue, pattern) {
			v.AddError(field, "Input contains potentially dangerous content", "DANGEROUS_CONTENT")
			break
		}
	}
}

// ValidateChannelName validates chat channel name
func (v *Validator) ValidateChannelName(field, name string) {
	if name == "" {
		return
	}

	// Length check
	v.ValidateLength(field, name, 1, 50)

	// Format check (letters, numbers, hyphens, underscores)
	channelRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !channelRegex.MatchString(name) {
		v.AddError(field, "Channel name can only contain letters, numbers, hyphens, and underscores", "INVALID_CHANNEL_NAME")
	}

	// Must start with letter
	if !unicode.IsLetter(rune(name[0])) {
		v.AddError(field, "Channel name must start with a letter", "INVALID_CHANNEL_START")
	}
}

// ValidateMessageContent validates chat message content
func (v *Validator) ValidateMessageContent(field, content string) {
	if content == "" {
		return
	}

	// Length check
	v.ValidateLength(field, content, 1, 4000)

	// Check for dangerous content
	v.ValidateNoHTML(field, content)
	
	// Allow some basic formatting but prevent XSS
	if containsXSSPatterns(content) {
		v.AddError(field, "Message contains potentially dangerous content", "XSS_DETECTED")
	}
}

// ValidateFileUpload validates file upload parameters
func (v *Validator) ValidateFileUpload(field, filename string, size int64, allowedTypes []string) {
	if filename == "" {
		v.AddError(field, "Filename is required", "FILENAME_REQUIRED")
		return
	}

	// File size check (10MB default)
	maxSize := int64(10 << 20)
	if size > maxSize {
		v.AddError(field, fmt.Sprintf("File size must be less than %d MB", maxSize/(1<<20)), "FILE_TOO_LARGE")
	}

	// File extension check
	if len(allowedTypes) > 0 {
		ext := strings.ToLower(getFileExtension(filename))
		allowed := false
		for _, allowedType := range allowedTypes {
			if ext == strings.ToLower(allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			v.AddError(field, "File type not allowed", "INVALID_FILE_TYPE")
		}
	}

	// Dangerous filename check
	if containsDangerousFilename(filename) {
		v.AddError(field, "Filename contains dangerous characters", "DANGEROUS_FILENAME")
	}
}

// Helper functions

func containsDangerousChars(input string) bool {
	dangerous := []string{"\x00", "\r", "\n", "<", ">", "\"", "'", "&", "script", "javascript"}
	lowerInput := strings.ToLower(input)
	for _, char := range dangerous {
		if strings.Contains(lowerInput, char) {
			return true
		}
	}
	return false
}

func containsXSSPatterns(input string) bool {
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "vbscript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
	}
	lowerInput := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func containsDangerousFilename(filename string) bool {
	dangerous := []string{"..", "/", "\\", "\x00", "<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerous {
		if strings.Contains(filename, char) {
			return true
		}
	}
	return false
}

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + parts[len(parts)-1]
}
