package main

import (
	"flag"
	"fmt"
)

func main() {
	var (
		version = flag.Bool("version", false, "Show version information")
		help    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *help {
		fmt.Println("PlexiChat GUI Client")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  plexichat-gui [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Println("PlexiChat GUI Client v3.0.0-production")
		fmt.Println("Build: 2024-01-01")
		fmt.Println("Go version: go1.21")
		return
	}

	fmt.Println("PlexiChat GUI Client v3.0.0-production")
	fmt.Println("GUI interface not yet implemented.")
	fmt.Println("Use the CLI version: plexichat --help")
}
