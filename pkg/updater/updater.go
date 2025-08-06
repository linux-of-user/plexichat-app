package updater

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"plexichat-client/pkg/logging"
)

const (
	// GitHub API endpoint for releases
	GitHubAPIURL = "https://api.github.com/repos/linux-of-user/plexichat-app/releases"

	// Current version - this should be updated with each release
	CurrentVersion = "v3.0.0-production"

	// Update check interval
	DefaultCheckInterval = 24 * time.Hour

	// Download timeout
	DownloadTimeout = 10 * time.Minute

	// Backup suffix for old executable
	BackupSuffix = ".backup"
)

// Release represents a GitHub release
type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	DownloadURL    string
	AssetName      string
	Assets         []Asset
}

// Updater handles application updates
type Updater struct {
	currentVersion string
	repoURL        string
	httpClient     *http.Client
	logger         *logging.Logger
}

// NewUpdater creates a new updater instance
func NewUpdater() *Updater {
	return &Updater{
		currentVersion: CurrentVersion,
		repoURL:        GitHubAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// CheckForUpdates checks if a new version is available
func (u *Updater) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	logging.Info("Checking for updates...")

	// Get latest release from GitHub API
	req, err := http.NewRequestWithContext(ctx, "GET", u.repoURL+"/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "PlexiChat-Client/"+CurrentVersion)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	// Check if update is available
	updateInfo := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		Assets:         release.Assets,
	}

	if release.TagName != u.currentVersion {
		updateInfo.Available = true

		// Find appropriate asset for current platform
		assetName := u.getAssetNameForPlatform()
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, assetName) {
				updateInfo.DownloadURL = asset.BrowserDownloadURL
				updateInfo.AssetName = asset.Name
				break
			}
		}

		if updateInfo.DownloadURL == "" {
			return nil, fmt.Errorf("no compatible asset found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}

	return updateInfo, nil
}

// DownloadUpdate downloads the update file
func (u *Updater) DownloadUpdate(ctx context.Context, updateInfo *UpdateInfo, progressCallback func(downloaded, total int64)) (string, error) {
	if !updateInfo.Available {
		return "", fmt.Errorf("no update available")
	}

	logging.Info("Downloading update: %s", updateInfo.AssetName)

	// Create temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, updateInfo.AssetName)

	// Download file
	req, err := http.NewRequestWithContext(ctx, "GET", updateInfo.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Copy with progress tracking
	if progressCallback != nil {
		contentLength := resp.ContentLength
		reader := &progressReader{
			Reader:   resp.Body,
			total:    contentLength,
			callback: progressCallback,
		}
		_, err = io.Copy(out, reader)
	} else {
		_, err = io.Copy(out, resp.Body)
	}

	if err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	logging.Info("Update downloaded to: %s", tempFile)
	return tempFile, nil
}

// InstallUpdate installs the downloaded update
func (u *Updater) InstallUpdate(downloadPath string) error {
	logging.Info("Installing update from: %s", downloadPath)

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create backup of current executable
	backupPath := currentExe + ".backup"
	if err := u.copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Extract and install new version
	if strings.HasSuffix(downloadPath, ".zip") {
		err = u.installFromZip(downloadPath, currentExe)
	} else {
		err = u.installFromBinary(downloadPath, currentExe)
	}

	if err != nil {
		// Restore backup on failure
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up
	os.Remove(backupPath)
	os.Remove(downloadPath)

	logging.Info("Update installed successfully")
	return nil
}

// getAssetNameForPlatform returns the expected asset name for current platform
func (u *Updater) getAssetNameForPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

// installFromZip extracts and installs from a zip file
func (u *Updater) installFromZip(zipPath, targetPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Find the executable in the zip
	var executableFile *zip.File
	executableName := filepath.Base(targetPath)

	for _, file := range reader.File {
		if filepath.Base(file.Name) == executableName {
			executableFile = file
			break
		}
	}

	if executableFile == nil {
		return fmt.Errorf("executable not found in zip file")
	}

	// Extract executable
	src, err := executableFile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Set executable permissions on Unix systems
	if runtime.GOOS != "windows" {
		err = os.Chmod(targetPath, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// installFromBinary installs from a binary file
func (u *Updater) installFromBinary(binaryPath, targetPath string) error {
	return u.copyFile(binaryPath, targetPath)
}

// copyFile copies a file from src to dst
func (u *Updater) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// progressReader tracks download progress
type progressReader struct {
	io.Reader
	total      int64
	downloaded int64
	callback   func(downloaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.downloaded += int64(n)
	if pr.callback != nil {
		pr.callback(pr.downloaded, pr.total)
	}
	return n, err
}

// SelfUpdate performs a self-update of the current executable
func (u *Updater) SelfUpdate(ctx context.Context) error {
	u.logger.Info("Starting self-update process...")

	// Check for updates
	updateInfo, err := u.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.Available {
		u.logger.Info("No updates available")
		return nil
	}

	u.logger.Info("Update available: %s -> %s", CurrentVersion, updateInfo.LatestVersion)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create backup of current executable
	backupPath := execPath + BackupSuffix
	if err := u.createBackup(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Download new version
	tempFile, err := u.downloadUpdate(ctx, updateInfo)
	if err != nil {
		// Restore backup on failure
		u.restoreBackup(backupPath, execPath)
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Replace current executable
	if err := u.replaceExecutable(tempFile, execPath); err != nil {
		// Restore backup on failure
		u.restoreBackup(backupPath, execPath)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	u.logger.Info("Self-update completed successfully")
	return nil
}

// createBackup creates a backup of the current executable
func (u *Updater) createBackup(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// restoreBackup restores the backup executable
func (u *Updater) restoreBackup(backup, target string) error {
	return os.Rename(backup, target)
}

// downloadUpdate downloads the update file
func (u *Updater) downloadUpdate(ctx context.Context, updateInfo *UpdateInfo) (string, error) {
	// Find the appropriate asset for current platform
	var downloadURL string
	expectedName := fmt.Sprintf("plexichat-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	for _, asset := range updateInfo.Assets {
		if strings.Contains(asset.Name, expectedName) ||
			strings.Contains(asset.Name, runtime.GOOS) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no suitable asset found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "plexichat-update-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Download with timeout
	client := &http.Client{Timeout: DownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Copy response to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

// replaceExecutable replaces the current executable with the new one
func (u *Updater) replaceExecutable(newFile, targetFile string) error {
	// On Windows, we need to handle the file being in use
	if runtime.GOOS == "windows" {
		// Move current executable to temp name
		tempName := targetFile + ".old"
		if err := os.Rename(targetFile, tempName); err != nil {
			return err
		}

		// Move new file to target location
		if err := os.Rename(newFile, targetFile); err != nil {
			// Restore original on failure
			os.Rename(tempName, targetFile)
			return err
		}

		// Schedule old file for deletion on next boot
		// This is a Windows-specific approach
		os.Remove(tempName)
	} else {
		// On Unix-like systems, we can replace directly
		if err := os.Rename(newFile, targetFile); err != nil {
			return err
		}

		// Make executable
		if err := os.Chmod(targetFile, 0755); err != nil {
			return err
		}
	}

	return nil
}
