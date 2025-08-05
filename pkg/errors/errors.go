package errors

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"plexichat-client/pkg/logging"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeAuth         ErrorType = "authentication"
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypePermission   ErrorType = "permission"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeServer       ErrorType = "server"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeUnknown      ErrorType = "unknown"
)

// PlexiChatError represents a structured error with context
type PlexiChatError struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	Suggestion  string    `json:"suggestion,omitempty"`
	Retryable   bool      `json:"retryable"`
	StatusCode  int       `json:"status_code,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *PlexiChatError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Message, e.Details, e.Code)
	}
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// UserFriendlyMessage returns a user-friendly error message
func (e *PlexiChatError) UserFriendlyMessage() string {
	switch e.Type {
	case ErrorTypeNetwork:
		return "Connection problem. Please check your internet connection and try again."
	case ErrorTypeAuth:
		return "Authentication failed. Please check your credentials and try again."
	case ErrorTypeValidation:
		return fmt.Sprintf("Invalid input: %s", e.Message)
	case ErrorTypePermission:
		return "You don't have permission to perform this action."
	case ErrorTypeNotFound:
		return "The requested resource was not found."
	case ErrorTypeRateLimit:
		return "Too many requests. Please wait a moment and try again."
	case ErrorTypeServer:
		return "Server error. Please try again later."
	case ErrorTypeTimeout:
		return "Request timed out. Please try again."
	default:
		return e.Message
	}
}

// WithSuggestion adds a suggestion to the error
func (e *PlexiChatError) WithSuggestion(suggestion string) *PlexiChatError {
	e.Suggestion = suggestion
	return e
}

// WithContext adds context information to the error
func (e *PlexiChatError) WithContext(key string, value interface{}) *PlexiChatError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewError creates a new PlexiChatError
func NewError(errorType ErrorType, code, message string) *PlexiChatError {
	return &PlexiChatError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Retryable: isRetryable(errorType, code),
		Timestamp: time.Now(),
	}
}

// NewNetworkError creates a network-related error
func NewNetworkError(code, message string) *PlexiChatError {
	return NewError(ErrorTypeNetwork, code, message).
		WithSuggestion("Check your internet connection and server URL")
}

// NewAuthError creates an authentication-related error
func NewAuthError(code, message string) *PlexiChatError {
	return NewError(ErrorTypeAuth, code, message).
		WithSuggestion("Verify your credentials or login again")
}

// NewValidationError creates a validation-related error
func NewValidationError(code, message string) *PlexiChatError {
	return NewError(ErrorTypeValidation, code, message).
		WithSuggestion("Please check your input and try again")
}

// NewServerError creates a server-related error
func NewServerError(code, message string) *PlexiChatError {
	return NewError(ErrorTypeServer, code, message).
		WithSuggestion("Please try again later or contact support")
}

// FromHTTPResponse creates an error from an HTTP response
func FromHTTPResponse(resp *http.Response, body []byte) *PlexiChatError {
	var errorType ErrorType
	var code string
	var suggestion string

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		errorType = ErrorTypeAuth
		code = "UNAUTHORIZED"
		suggestion = "Please login again"
	case http.StatusForbidden:
		errorType = ErrorTypePermission
		code = "FORBIDDEN"
		suggestion = "You don't have permission for this action"
	case http.StatusNotFound:
		errorType = ErrorTypeNotFound
		code = "NOT_FOUND"
		suggestion = "The requested resource was not found"
	case http.StatusTooManyRequests:
		errorType = ErrorTypeRateLimit
		code = "RATE_LIMITED"
		suggestion = "Please wait a moment before trying again"
	case http.StatusBadRequest:
		errorType = ErrorTypeValidation
		code = "BAD_REQUEST"
		suggestion = "Please check your input"
	case http.StatusRequestTimeout:
		errorType = ErrorTypeTimeout
		code = "TIMEOUT"
		suggestion = "Request timed out, please try again"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		errorType = ErrorTypeServer
		code = "SERVER_ERROR"
		suggestion = "Server is experiencing issues, please try again later"
	default:
		errorType = ErrorTypeUnknown
		code = fmt.Sprintf("HTTP_%d", resp.StatusCode)
		suggestion = "An unexpected error occurred"
	}

	message := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if len(body) > 0 {
		message = string(body)
	}

	return &PlexiChatError{
		Type:       errorType,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Retryable:  isRetryableHTTPStatus(resp.StatusCode),
		StatusCode: resp.StatusCode,
		Timestamp:  time.Now(),
	}
}

// FromError converts a standard error to PlexiChatError
func FromError(err error) *PlexiChatError {
	if plexiErr, ok := err.(*PlexiChatError); ok {
		return plexiErr
	}

	message := err.Error()
	var errorType ErrorType
	var code string

	// Analyze error message to determine type
	lowerMsg := strings.ToLower(message)
	switch {
	case strings.Contains(lowerMsg, "connection refused"), strings.Contains(lowerMsg, "no such host"):
		errorType = ErrorTypeNetwork
		code = "CONNECTION_FAILED"
	case strings.Contains(lowerMsg, "timeout"), strings.Contains(lowerMsg, "deadline exceeded"):
		errorType = ErrorTypeTimeout
		code = "TIMEOUT"
	case strings.Contains(lowerMsg, "unauthorized"), strings.Contains(lowerMsg, "authentication"):
		errorType = ErrorTypeAuth
		code = "AUTH_FAILED"
	case strings.Contains(lowerMsg, "validation"), strings.Contains(lowerMsg, "invalid"):
		errorType = ErrorTypeValidation
		code = "VALIDATION_ERROR"
	default:
		errorType = ErrorTypeUnknown
		code = "UNKNOWN_ERROR"
	}

	return NewError(errorType, code, message)
}

// isRetryable determines if an error type/code is retryable
func isRetryable(errorType ErrorType, code string) bool {
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeServer:
		return true
	case ErrorTypeRateLimit:
		return true // With delay
	case ErrorTypeAuth, ErrorTypeValidation, ErrorTypePermission, ErrorTypeNotFound:
		return false
	default:
		return false
	}
}

// isRetryableHTTPStatus determines if an HTTP status is retryable
func isRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		 http.StatusTooManyRequests,
		 http.StatusInternalServerError,
		 http.StatusBadGateway,
		 http.StatusServiceUnavailable,
		 http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger *logging.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// Handle processes an error and provides appropriate feedback
func (h *ErrorHandler) Handle(err error, context string) *PlexiChatError {
	plexiErr := FromError(err)
	
	// Log the error with context
	h.logger.Error("[%s] %s", context, plexiErr.Error())
	
	// Add context to error
	plexiErr.WithContext("operation", context)
	
	return plexiErr
}

// HandleWithRecovery handles an error and suggests recovery actions
func (h *ErrorHandler) HandleWithRecovery(err error, context string) (*PlexiChatError, []string) {
	plexiErr := h.Handle(err, context)
	
	var recoveryActions []string
	
	switch plexiErr.Type {
	case ErrorTypeNetwork:
		recoveryActions = []string{
			"Check your internet connection",
			"Verify the server URL in configuration",
			"Try again in a few moments",
		}
	case ErrorTypeAuth:
		recoveryActions = []string{
			"Login again with correct credentials",
			"Check if your session has expired",
			"Verify your account is active",
		}
	case ErrorTypeTimeout:
		recoveryActions = []string{
			"Try again with a longer timeout",
			"Check your network connection",
			"The server might be busy, try later",
		}
	case ErrorTypeRateLimit:
		recoveryActions = []string{
			"Wait a few moments before trying again",
			"Reduce the frequency of requests",
		}
	case ErrorTypeServer:
		recoveryActions = []string{
			"Try again later",
			"Check server status",
			"Contact support if the problem persists",
		}
	default:
		recoveryActions = []string{
			"Try the operation again",
			"Check your input and try again",
		}
	}
	
	return plexiErr, recoveryActions
}

// Global error handler instance
var GlobalErrorHandler = NewErrorHandler()
