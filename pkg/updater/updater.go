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
	CurrentVersion = "v1.0.0"
	
	// Update check interval
	DefaultCheckInterval = 24 * time.Hour
)

// Release represents a GitHub release
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
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
}

// Updater handles application updates
type Updater struct {
	currentVersion string
	repoURL        string
	httpClient     *http.Client
}

// NewUpdater creates a new updater instance
func NewUpdater() *Updater {
	return &Updater{
		currentVersion: CurrentVersion,
		repoURL:        GitHubAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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
