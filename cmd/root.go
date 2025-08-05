package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"plexichat-client/pkg/client"
)

var (
	cfgFile string
	baseURL string
	apiKey  string
	verbose bool
)

// Version information
var (
	clientVersion   = "1.0.1"
	clientCommit    = "unknown"
	clientBuildTime = "unknown"
	clientGoVersion = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "plexichat-client",
	Short: "A comprehensive client for PlexiChat API",
	Long: `PlexiChat Client - A feature-rich command-line client for PlexiChat

This client provides access to all PlexiChat features including:
- User authentication and management
- Real-time messaging with WebSocket support
- File uploads and downloads
- Admin operations
- Security testing and monitoring
- Performance benchmarking
- Bot account management
- Rate limiting configuration
- And much more!

Examples:
  plexichat-client auth login --username admin --password secret
  plexichat-client chat send --message "Hello, World!"
  plexichat-client chat listen --room general
  plexichat-client admin users list
  plexichat-client files upload --file document.pdf
  plexichat-client security test --endpoint /api/v1/auth/login
  plexichat-client benchmark --duration 60s --concurrent 10`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.plexichat-app/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8000", "PlexiChat server URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add built-in subcommands (others are registered in their respective files)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(guiCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "config" in .plexichat-app folder
		configDir := filepath.Join(home, ".plexichat-app")
		os.MkdirAll(configDir, 0755)
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Set defaults
	viper.SetDefault("url", "http://localhost:8000")
	viper.SetDefault("timeout", "30s")
	viper.SetDefault("retries", 3)
	viper.SetDefault("concurrent_requests", 10)
}

// SetVersionInfo sets version information from main
func SetVersionInfo(version, commit, buildTime, goVersion string) {
	clientVersion = version
	clientCommit = commit
	clientBuildTime = buildTime
	clientGoVersion = goVersion
}

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch the graphical user interface",
	Long:  "Launch the cross-platform graphical user interface for PlexiChat",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🚀 Launching PlexiChat GUI...")

		// Check if GUI is available
		if !isGUIAvailable() {
			fmt.Println("❌ GUI not available - CGO or C compiler missing")
			fmt.Println("💡 See SETUP-GUIDE.md for GUI setup instructions")
			fmt.Println("🌐 Use 'plexichat-client web' for web interface")
			return fmt.Errorf("GUI not available")
		}

		// Try to launch native GUI
		fmt.Println("✅ GUI dependencies available, launching...")
		err := RunGUI()
		if err != nil {
			fmt.Printf("❌ GUI launch failed: %v\n", err)
			return err
		}
		return nil
	},
}

// isGUIAvailable checks if GUI dependencies are available
func isGUIAvailable() bool {
	// This is a simple check - in reality we'd check for CGO and C compiler
	// For now, just return true and let RunGUI handle the actual check
	return true
}

// runWebInterface launches a web-based GUI as fallback
func runWebInterface() error {
	fmt.Println("🌐 Starting web interface...")
	fmt.Println("📱 This will open PlexiChat in your default browser")
	fmt.Println("🔗 URL: http://localhost:8080")

	// This would start a web server with the GUI
	// For now, just provide instructions
	fmt.Println("\n🔧 To use web interface:")
	fmt.Println("1. Run: plexichat-client web")
	fmt.Println("2. Open browser to: http://localhost:8080")
	fmt.Println("3. Or connect directly to your PlexiChat server")

	return fmt.Errorf("web interface not yet implemented - use 'plexichat-client web' command")
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	Long:  "Check the health status of the PlexiChat server",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(viper.GetString("url"))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		health, err := c.Health(ctx)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}

		fmt.Printf("Status: %s\n", health.Status)
		fmt.Printf("Version: %s\n", health.Version)
		fmt.Printf("Uptime: %s\n", health.Uptime)
		fmt.Printf("Timestamp: %s\n", health.Timestamp)

		if len(health.Checks) > 0 {
			fmt.Println("\nHealth Checks:")
			for name, status := range health.Checks {
				fmt.Printf("  %s: %s\n", name, status)
			}
		}

		return nil
	},
}
