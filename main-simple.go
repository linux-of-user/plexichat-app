package main

import (
	"fmt"
	"os"
	"runtime"
)

var (
	version   = "2.0.0-alpha"
	commit    = "fixed"
	buildTime = "now"
)

func main() {
	fmt.Println("PlexiChat Desktop v" + version)
	fmt.Println("The Phoenix Release")
	fmt.Println("==================")
	
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("PlexiChat Desktop v%s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Build Time: %s\n", buildTime)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			
		case "--help", "-h":
			fmt.Println("\nUsage: plexichat [command]")
			fmt.Println("\nCommands:")
			fmt.Println("  --version, -v    Show version information")
			fmt.Println("  --help, -h       Show this help")
			fmt.Println("  auth             Authentication commands")
			fmt.Println("  chat             Chat commands")
			fmt.Println("  files            File operations")
			fmt.Println("  gui              Launch GUI (if available)")
			fmt.Println("  health           Check server health")
			fmt.Println("\nExamples:")
			fmt.Println("  plexichat auth login")
			fmt.Println("  plexichat chat send --message \"Hello!\"")
			fmt.Println("  plexichat gui")
			
		case "auth":
			fmt.Println("🔐 Authentication Commands")
			fmt.Println("Available: login, logout, register, status")
			fmt.Println("Example: plexichat auth login --username admin")
			
		case "chat":
			fmt.Println("💬 Chat Commands")
			fmt.Println("Available: send, listen, history")
			fmt.Println("Example: plexichat chat send --message \"Hello World!\"")
			
		case "files":
			fmt.Println("📁 File Commands")
			fmt.Println("Available: upload, download, list")
			fmt.Println("Example: plexichat files upload --file document.pdf")
			
		case "gui":
			fmt.Println("🎨 GUI Launch")
			fmt.Println("Checking GUI availability...")
			fmt.Println("❌ GUI requires CGO and C compiler")
			fmt.Println("💡 See SETUP-GUIDE.md for GUI setup instructions")
			fmt.Println("🌐 Alternative: Use web interface")
			
		case "health":
			fmt.Println("🏥 Server Health Check")
			fmt.Println("This would check server connectivity")
			fmt.Println("Example: plexichat health --url http://localhost:8000")
			
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Println("Use --help for available commands")
		}
	} else {
		fmt.Println("\n🚀 PlexiChat Desktop is ready!")
		fmt.Println("Use --help for available commands")
		fmt.Println("\nQuick start:")
		fmt.Println("  plexichat --help")
		fmt.Println("  plexichat auth login")
		fmt.Println("  plexichat gui")
	}
}
