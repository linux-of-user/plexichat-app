package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"plexichat-client/pkg/updater"
)

var (
	showBuildInfo bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for the PlexiChat client.

This command shows the current version, build information,
and optionally checks for available updates.

Examples:
  # Show version
  plexichat-cli version

  # Show detailed build information
  plexichat-cli version --build`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolVar(&showBuildInfo, "build", false, "Show detailed build information")
}

func runVersion(cmd *cobra.Command, args []string) error {
	version := updater.CurrentVersion
	
	fmt.Printf("PlexiChat Client %s\n", version)
	
	if showBuildInfo {
		fmt.Printf("\nBuild Information:\n")
		fmt.Printf("  Version:      %s\n", version)
		fmt.Printf("  Go Version:   %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Compiler:     %s\n", runtime.Compiler)
		
		// Show features
		fmt.Printf("\nFeatures:\n")
		fmt.Printf("  ✅ CLI Interface\n")
		fmt.Printf("  ✅ GUI Interface\n")
		fmt.Printf("  ✅ Real-time Messaging\n")
		fmt.Printf("  ✅ Auto-Update\n")
		fmt.Printf("  ✅ Configuration Management\n")
		fmt.Printf("  ✅ Security Validation\n")
		fmt.Printf("  ✅ ASCII Logging\n")
	}
	
	return nil
}
