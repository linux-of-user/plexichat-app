package security

import (
	"testing"
)

func TestValidator_ValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
		errCode string
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "invalid email - no @",
			email:   "userexample.com",
			wantErr: true,
			errCode: "INVALID_EMAIL",
		},
		{
			name:    "invalid email - no domain",
			email:   "user@",
			wantErr: true,
			errCode: "INVALID_EMAIL",
		},
		{
			name:    "invalid email - no user",
			email:   "@example.com",
			wantErr: true,
			errCode: "INVALID_EMAIL",
		},
		{
			name:    "email too long",
			email:   "verylongusernamethatexceedsthelimitverylongusernamethatexceedsthelimitverylongusernamethatexceedsthelimitverylongusernamethatexceedsthelimitverylongusernamethatexceedsthelimitverylongusernamethatexceedsthelimit@example.com",
			wantErr: true,
			errCode: "EMAIL_TOO_LONG",
		},
		{
			name:    "email with dangerous characters",
			email:   "user<script>@example.com",
			wantErr: true,
			errCode: "INVALID_CHARACTERS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateEmail("email", tt.email)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidateEmail() expected error but got none")
				} else {
					errors := v.Errors()
					found := false
					for _, err := range errors {
						if err.Code == tt.errCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateEmail() expected error code %s but got %v", tt.errCode, errors)
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidateEmail() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}

func TestValidator_ValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
		errCode  string
	}{
		{
			name:     "valid username",
			username: "john_doe",
			wantErr:  false,
		},
		{
			name:     "valid username with numbers",
			username: "user123",
			wantErr:  false,
		},
		{
			name:     "valid username with hyphens",
			username: "user-name",
			wantErr:  false,
		},
		{
			name:     "username too short",
			username: "ab",
			wantErr:  true,
			errCode:  "USERNAME_TOO_SHORT",
		},
		{
			name:     "username too long",
			username: "verylongusernamethatexceedsthelimit",
			wantErr:  true,
			errCode:  "USERNAME_TOO_LONG",
		},
		{
			name:     "username with invalid characters",
			username: "user@name",
			wantErr:  true,
			errCode:  "INVALID_USERNAME",
		},
		{
			name:     "reserved username",
			username: "admin",
			wantErr:  true,
			errCode:  "RESERVED_USERNAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateUsername("username", tt.username)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidateUsername() expected error but got none")
				} else {
					errors := v.Errors()
					found := false
					for _, err := range errors {
						if err.Code == tt.errCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateUsername() expected error code %s but got %v", tt.errCode, errors)
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidateUsername() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}

func TestValidator_ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errCodes []string
	}{
		{
			name:     "valid strong password",
			password: "MyStr0ng!Pass",
			wantErr:  false,
		},
		{
			name:     "password too short",
			password: "Abc1!",
			wantErr:  true,
			errCodes: []string{"PASSWORD_TOO_SHORT"},
		},
		{
			name:     "password missing uppercase",
			password: "mystr0ng!pass",
			wantErr:  true,
			errCodes: []string{"PASSWORD_NO_UPPER"},
		},
		{
			name:     "password missing lowercase",
			password: "MYSTR0NG!PASS",
			wantErr:  true,
			errCodes: []string{"PASSWORD_NO_LOWER"},
		},
		{
			name:     "password missing digit",
			password: "MyStrong!Pass",
			wantErr:  true,
			errCodes: []string{"PASSWORD_NO_DIGIT"},
		},
		{
			name:     "password missing special character",
			password: "MyStr0ngPass",
			wantErr:  true,
			errCodes: []string{"PASSWORD_NO_SPECIAL"},
		},
		{
			name:     "common password",
			password: "Password123!",
			wantErr:  true,
			errCodes: []string{"COMMON_PASSWORD"},
		},
		{
			name:     "multiple issues",
			password: "abc",
			wantErr:  true,
			errCodes: []string{"PASSWORD_TOO_SHORT", "PASSWORD_NO_UPPER", "PASSWORD_NO_DIGIT", "PASSWORD_NO_SPECIAL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidatePassword("password", tt.password)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidatePassword() expected error but got none")
				} else {
					errors := v.Errors()
					for _, expectedCode := range tt.errCodes {
						found := false
						for _, err := range errors {
							if err.Code == expectedCode {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("ValidatePassword() expected error code %s but got %v", expectedCode, errors)
						}
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidatePassword() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}

func TestValidator_ValidateChannelName(t *testing.T) {
	tests := []struct {
		name        string
		channelName string
		wantErr     bool
		errCode     string
	}{
		{
			name:        "valid channel name",
			channelName: "general",
			wantErr:     false,
		},
		{
			name:        "valid channel name with numbers",
			channelName: "channel123",
			wantErr:     false,
		},
		{
			name:        "valid channel name with underscore",
			channelName: "dev_team",
			wantErr:     false,
		},
		{
			name:        "valid channel name with hyphen",
			channelName: "dev-team",
			wantErr:     false,
		},
		{
			name:        "channel name too long",
			channelName: "verylongchannelnamethatexceedsthelimitverylongchannelnamethatexceedsthelimit",
			wantErr:     true,
			errCode:     "TOO_LONG",
		},
		{
			name:        "channel name with invalid characters",
			channelName: "channel@name",
			wantErr:     true,
			errCode:     "INVALID_CHANNEL_NAME",
		},
		{
			name:        "channel name starting with number",
			channelName: "123channel",
			wantErr:     true,
			errCode:     "INVALID_CHANNEL_START",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateChannelName("channel", tt.channelName)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidateChannelName() expected error but got none")
				} else {
					errors := v.Errors()
					found := false
					for _, err := range errors {
						if err.Code == tt.errCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateChannelName() expected error code %s but got %v", tt.errCode, errors)
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidateChannelName() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}

func TestValidator_ValidateMessageContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		errCode string
	}{
		{
			name:    "valid message",
			content: "Hello, world!",
			wantErr: false,
		},
		{
			name:    "message with emojis",
			content: "Hello! 😊",
			wantErr: false,
		},
		{
			name:    "message too long",
			content: string(make([]byte, 5000)),
			wantErr: true,
			errCode: "TOO_LONG",
		},
		{
			name:    "message with HTML",
			content: "Hello <script>alert('xss')</script>",
			wantErr: true,
			errCode: "HTML_NOT_ALLOWED",
		},
		{
			name:    "message with XSS",
			content: "Hello javascript:alert('xss')",
			wantErr: true,
			errCode: "XSS_DETECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateMessageContent("content", tt.content)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidateMessageContent() expected error but got none")
				} else {
					errors := v.Errors()
					found := false
					for _, err := range errors {
						if err.Code == tt.errCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateMessageContent() expected error code %s but got %v", tt.errCode, errors)
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidateMessageContent() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}

func TestValidator_ValidateFileUpload(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		size         int64
		allowedTypes []string
		wantErr      bool
		errCode      string
	}{
		{
			name:         "valid file",
			filename:     "document.pdf",
			size:         1024,
			allowedTypes: []string{".pdf", ".doc", ".txt"},
			wantErr:      false,
		},
		{
			name:    "file too large",
			filename: "large.pdf",
			size:    20 << 20, // 20MB
			wantErr: true,
			errCode: "FILE_TOO_LARGE",
		},
		{
			name:         "invalid file type",
			filename:     "script.exe",
			size:         1024,
			allowedTypes: []string{".pdf", ".doc", ".txt"},
			wantErr:      true,
			errCode:      "INVALID_FILE_TYPE",
		},
		{
			name:     "dangerous filename",
			filename: "../../../etc/passwd",
			size:     1024,
			wantErr:  true,
			errCode:  "DANGEROUS_FILENAME",
		},
		{
			name:     "empty filename",
			filename: "",
			size:     1024,
			wantErr:  true,
			errCode:  "FILENAME_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.ValidateFileUpload("file", tt.filename, tt.size, tt.allowedTypes)

			if tt.wantErr {
				if !v.HasErrors() {
					t.Errorf("ValidateFileUpload() expected error but got none")
				} else {
					errors := v.Errors()
					found := false
					for _, err := range errors {
						if err.Code == tt.errCode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateFileUpload() expected error code %s but got %v", tt.errCode, errors)
					}
				}
			} else {
				if v.HasErrors() {
					t.Errorf("ValidateFileUpload() unexpected error: %v", v.Errors())
				}
			}
		})
	}
}
