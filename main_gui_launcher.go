package main

import (
"runtime"
)

// Version information - will be set during build
var (
guiVersion   = "dev"
guiCommit    = "unknown"
guiBuildTime = "unknown"
)

func init() {
version = guiVersion
commit = guiCommit
buildTime = guiBuildTime
}

func main() {
// Set version information for GUI
_ = runtime.Version()

// Launch GUI
app := NewPlexiChatApp()
app.Run()
}
