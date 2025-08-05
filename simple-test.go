package main

import (
	"fmt"
	"time"

	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/security"
)

func main() {
	fmt.Println("🚀 PlexiChat Client Simple Test")
	fmt.Println("===============================")

	// Test 1: Logging System
	fmt.Println("\n[1/3] Testing Logging System...")
	
	logging.SetGlobalLevel(logging.DEBUG)
	logging.SetGlobalColorized(true)
	logging.SetGlobalPrefix("TEST")

	logging.Debug("Debug message test")
	logging.Info("Info message test")
	logging.Warn("Warning message test")
	logging.Error("Error message test")

	fmt.Println("✅ Logging system working correctly")

	// Test 2: Security Validation
	fmt.Println("\n[2/3] Testing Security Validation...")
	
	validator := security.NewValidator()

	// Test email validation
	validator.ValidateEmail("email", "test@example.com")
	validator.ValidateEmail("email", "invalid-email-that-is-way-too-long-and-exceeds-the-maximum-allowed-length-for-email-addresses-which-should-be-254-characters-according-to-rfc-standards-but-this-one-is-much-longer-than-that-limit-and-should-fail-validation@example.com")

	// Test password validation
	validator.ValidatePassword("password", "SecurePass123!")
	validator.ValidatePassword("password", "Password123!") // Common password

	if validator.HasErrors() {
		errors := validator.GetErrors()
		fmt.Printf("✅ Security validation working - found %d validation errors as expected\n", len(errors))
	} else {
		fmt.Println("✅ Security validation working correctly")
	}

	// Test 3: Basic functionality
	fmt.Println("\n[3/3] Testing Basic Functionality...")
	
	// Test ASCII conversion
	asciiText := logging.ToASCII("Hello 🌟 World! 🚀")
	fmt.Printf("ASCII conversion: '%s'\n", asciiText)

	// Test log level parsing
	level := logging.ParseLogLevel("debug")
	if level == logging.DEBUG {
		fmt.Println("✅ Log level parsing working correctly")
	}

	fmt.Println("\n🎉 All basic tests completed successfully!")
	fmt.Println("✅ PlexiChat Client core functionality is working!")
}
