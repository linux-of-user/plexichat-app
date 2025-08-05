package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/updater"
)

var (
	checkOnly    bool
	forceUpdate  bool
	autoConfirm  bool
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	Long: `Check for and install updates to the PlexiChat client.

This command will check the GitHub repository for new releases and
optionally download and install them automatically.

Examples:
  # Check for updates only
  plexichat-cli update --check

  # Update with confirmation
  plexichat-cli update

  # Force update without confirmation
  plexichat-cli update --force --yes`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update even if already up to date")
	updateCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm update installation")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	updater := updater.NewUpdater()

	// Check for updates
	logging.Info("Checking for updates...")
	updateInfo, err := updater.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Display current version
	fmt.Printf("Current version: %s\n", updateInfo.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", updateInfo.LatestVersion)

	if !updateInfo.Available && !forceUpdate {
		fmt.Println("✅ You are already running the latest version!")
		return nil
	}

	if updateInfo.Available {
		fmt.Printf("\n🎉 New version available: %s\n", updateInfo.LatestVersion)
		if updateInfo.ReleaseNotes != "" {
			fmt.Printf("\nRelease Notes:\n%s\n", updateInfo.ReleaseNotes)
		}
	}

	// If check-only mode, exit here
	if checkOnly {
		if updateInfo.Available {
			fmt.Println("\n💡 Run 'plexichat-cli update' to install the update")
		}
		return nil
	}

	// Confirm update installation
	if !autoConfirm && !forceUpdate {
		fmt.Print("\nDo you want to install this update? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	// Download update
	fmt.Println("\n📥 Downloading update...")
	downloadPath, err := updater.DownloadUpdate(ctx, updateInfo, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	fmt.Println() // New line after progress

	// Install update
	fmt.Println("🔧 Installing update...")
	if err := updater.InstallUpdate(downloadPath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Println("✅ Update installed successfully!")
	fmt.Println("🔄 Please restart the application to use the new version.")

	return nil
}

// CheckForUpdatesBackground checks for updates in the background
func CheckForUpdatesBackground() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		updater := updater.NewUpdater()
		updateInfo, err := updater.CheckForUpdates(ctx)
		if err != nil {
			logging.Debug("Background update check failed: %v", err)
			return
		}

		if updateInfo.Available {
			logging.Info("New version available: %s (current: %s)", 
				updateInfo.LatestVersion, updateInfo.CurrentVersion)
			logging.Info("Run 'plexichat-cli update' to install the update")
		}
	}()
}

// GetCurrentVersion returns the current application version
func GetCurrentVersion() string {
	return updater.CurrentVersion
}
