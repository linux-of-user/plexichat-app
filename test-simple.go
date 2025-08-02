package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("PlexiChat Desktop Test")
	fmt.Println("Version: 2.0.0-alpha")
	
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("plexichat-desktop version 2.0.0-alpha")
		case "--help":
			fmt.Println("Usage: test-simple [--version|--help]")
		default:
			fmt.Printf("Unknown argument: %s\n", os.Args[1])
		}
	} else {
		fmt.Println("Use --help for usage information")
	}
}
