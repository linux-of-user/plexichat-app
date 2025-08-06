package main

import (
	"fmt"
	"runtime"
)

// Version information for PlexiChat Client
const (
	// Version is the current version of the PlexiChat client
	Version = "3.0.0-production"

	// BuildDate is set during build time
	BuildDate = "2024-01-01"

	// GitCommit is set during build time
	GitCommit = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = "go1.21"

	// Application metadata
	AppName = "PlexiChat Client"
	AppID   = "com.plexichat.desktop"

	// Update configuration
	UpdateCheckURL  = "https://api.github.com/repos/linux-of-user/plexichat-app/releases/latest"
	DownloadBaseURL = "https://github.com/linux-of-user/plexichat-app/releases/download"
)

// VersionInfo holds complete version information
type VersionInfo struct {
	Version      string `json:"version"`
	BuildDate    string `json:"build_date"`
	GitCommit    string `json:"git_commit"`
	GoVersion    string `json:"go_version"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	AppName      string `json:"app_name"`
	AppID        string `json:"app_id"`
}

// GetVersionInfo returns complete version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:      Version,
		BuildDate:    BuildDate,
		GitCommit:    GitCommit,
		GoVersion:    GoVersion,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		AppName:      AppName,
		AppID:        AppID,
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	return fmt.Sprintf("%s v%s", AppName, Version)
}

// GetFullVersionString returns a detailed version string
func GetFullVersionString() string {
	return fmt.Sprintf("%s v%s\nBuild: %s\nPlatform: %s/%s\nGo: %s\nCommit: %s",
		AppName, Version, BuildDate, runtime.GOOS, runtime.GOARCH, GoVersion, GitCommit)
}
