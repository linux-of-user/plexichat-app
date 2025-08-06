package main

// Version information for PlexiChat Client
const (
	// Version is the current version of the PlexiChat client
	Version = "b.1.1-97"
	
	// BuildDate is set during build time
	BuildDate = "2024-01-01"
	
	// GitCommit is set during build time
	GitCommit = "unknown"
	
	// GoVersion is the Go version used to build
	GoVersion = "go1.21"
)

// GetVersionInfo returns version information
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_date": BuildDate,
		"git_commit": GitCommit,
		"go_version": GoVersion,
	}
}
