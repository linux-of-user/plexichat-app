package files

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/security"
)

// FileType represents different file types
type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
	FileTypeDocument FileType = "document"
	FileTypeArchive  FileType = "archive"
	FileTypeCode     FileType = "code"
	FileTypeText     FileType = "text"
	FileTypeOther    FileType = "other"
)

// FileStatus represents file processing status
type FileStatus string

const (
	StatusPending    FileStatus = "pending"
	StatusUploading  FileStatus = "uploading"
	StatusProcessing FileStatus = "processing"
	StatusReady      FileStatus = "ready"
	StatusError      FileStatus = "error"
	StatusDeleted    FileStatus = "deleted"
)

// FileInfo represents comprehensive file information
type FileInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"original_name"`
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	Type         FileType               `json:"type"`
	MimeType     string                 `json:"mime_type"`
	Extension    string                 `json:"extension"`
	Status       FileStatus             `json:"status"`
	Hash         string                 `json:"hash"`
	Checksum     string                 `json:"checksum"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	AccessedAt   time.Time              `json:"accessed_at"`
	UploadedBy   string                 `json:"uploaded_by"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	Thumbnail    *ThumbnailInfo         `json:"thumbnail,omitempty"`
	Preview      *PreviewInfo           `json:"preview,omitempty"`
	Versions     []*FileVersion         `json:"versions,omitempty"`
	Permissions  *FilePermissions       `json:"permissions,omitempty"`
	VirusScan    *VirusScanResult       `json:"virus_scan,omitempty"`
}

// ThumbnailInfo represents thumbnail information
type ThumbnailInfo struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
}

// PreviewInfo represents preview information
type PreviewInfo struct {
	Path        string    `json:"path"`
	Type        string    `json:"type"` // image, pdf, text
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	PageCount   int       `json:"page_count,omitempty"`
	Duration    int       `json:"duration,omitempty"` // for videos/audio
	Generated   bool      `json:"generated"`
	GeneratedAt time.Time `json:"generated_at"`
}

// FileVersion represents a file version
type FileVersion struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	Comment   string    `json:"comment,omitempty"`
}

// FilePermissions represents file access permissions
type FilePermissions struct {
	Owner       string     `json:"owner"`
	Group       string     `json:"group"`
	Permissions string     `json:"permissions"` // rwxrwxrwx format
	Public      bool       `json:"public"`
	SharedWith  []string   `json:"shared_with,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// VirusScanResult represents virus scan results
type VirusScanResult struct {
	Scanned   bool      `json:"scanned"`
	Clean     bool      `json:"clean"`
	Threats   []string  `json:"threats,omitempty"`
	Scanner   string    `json:"scanner"`
	ScannedAt time.Time `json:"scanned_at"`
}

// UploadProgress represents upload progress
type UploadProgress struct {
	FileID        string        `json:"file_id"`
	BytesUploaded int64         `json:"bytes_uploaded"`
	TotalBytes    int64         `json:"total_bytes"`
	Percentage    float64       `json:"percentage"`
	Speed         int64         `json:"speed"` // bytes per second
	ETA           time.Duration `json:"eta"`
	StartTime     time.Time     `json:"start_time"`
	LastUpdate    time.Time     `json:"last_update"`
}

// FileManagerConfig represents file manager configuration
type FileManagerConfig struct {
	StorageDir         string                 `json:"storage_dir"`
	ThumbnailDir       string                 `json:"thumbnail_dir"`
	PreviewDir         string                 `json:"preview_dir"`
	TempDir            string                 `json:"temp_dir"`
	MaxFileSize        int64                  `json:"max_file_size"`
	AllowedTypes       []string               `json:"allowed_types"`
	BlockedTypes       []string               `json:"blocked_types"`
	GenerateThumbnails bool                   `json:"generate_thumbnails"`
	GeneratePreviews   bool                   `json:"generate_previews"`
	VirusScanEnabled   bool                   `json:"virus_scan_enabled"`
	VersioningEnabled  bool                   `json:"versioning_enabled"`
	MaxVersions        int                    `json:"max_versions"`
	CompressionEnabled bool                   `json:"compression_enabled"`
	EncryptionEnabled  bool                   `json:"encryption_enabled"`
	CleanupInterval    time.Duration          `json:"cleanup_interval"`
	RetentionDays      int                    `json:"retention_days"`
	ChunkSize          int64                  `json:"chunk_size"`
	ConcurrentUploads  int                    `json:"concurrent_uploads"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// FileManager manages file operations
type FileManager struct {
	config    *FileManagerConfig
	files     map[string]*FileInfo
	uploads   map[string]*UploadProgress
	storage   *FileStorage
	processor *FileProcessor
	logger    *logging.Logger
	mu        sync.RWMutex
	uploadSem chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewFileManager creates a new file manager
func NewFileManager(config *FileManagerConfig) *FileManager {
	if config == nil {
		config = &FileManagerConfig{
			StorageDir:         "storage/files",
			ThumbnailDir:       "storage/thumbnails",
			PreviewDir:         "storage/previews",
			TempDir:            "storage/temp",
			MaxFileSize:        100 * 1024 * 1024, // 100MB
			AllowedTypes:       []string{"image/*", "text/*", "application/pdf"},
			GenerateThumbnails: true,
			GeneratePreviews:   true,
			VirusScanEnabled:   false,
			VersioningEnabled:  true,
			MaxVersions:        10,
			CompressionEnabled: false,
			EncryptionEnabled:  false,
			CleanupInterval:    24 * time.Hour,
			RetentionDays:      30,
			ChunkSize:          1024 * 1024, // 1MB
			ConcurrentUploads:  5,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	fm := &FileManager{
		config:    config,
		files:     make(map[string]*FileInfo),
		uploads:   make(map[string]*UploadProgress),
		storage:   NewFileStorage(config),
		processor: NewFileProcessor(config),
		logger:    logging.NewLogger(logging.INFO, nil, true),
		uploadSem: make(chan struct{}, config.ConcurrentUploads),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create directories
	fm.createDirectories()

	// Start background tasks
	go fm.cleanupRoutine()

	return fm
}

// UploadFile uploads a file
func (fm *FileManager) UploadFile(ctx context.Context, reader io.Reader, filename string, metadata map[string]interface{}) (*FileInfo, error) {
	// Security validation
	filename = security.SanitizeInput(filename)
	if filename == "" {
		return nil, fmt.Errorf("invalid filename after sanitization")
	}

	// Check for malicious filename patterns
	if security.ContainsMaliciousContent(filename) {
		return nil, fmt.Errorf("filename contains potentially malicious content")
	}

	// Validate metadata for security
	if metadata != nil {
		if err := security.ValidateRequestBody(metadata); err != nil {
			return nil, fmt.Errorf("metadata validation failed: %w", err)
		}
	}

	// Validate file
	if err := fm.validateFile(filename); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Generate file ID
	fileID := generateFileID()

	// Create file info
	fileInfo := &FileInfo{
		ID:           fileID,
		Name:         filename,
		OriginalName: filename,
		Type:         fm.detectFileType(filename),
		MimeType:     fm.detectMimeType(filename),
		Extension:    strings.ToLower(filepath.Ext(filename)),
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Tags:         make([]string, 0),
		Metadata:     metadata,
	}

	// Create upload progress
	progress := &UploadProgress{
		FileID:     fileID,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	fm.mu.Lock()
	fm.files[fileID] = fileInfo
	fm.uploads[fileID] = progress
	fm.mu.Unlock()

	// Acquire upload semaphore
	select {
	case fm.uploadSem <- struct{}{}:
		defer func() { <-fm.uploadSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Upload file
	if err := fm.performUpload(ctx, reader, fileInfo, progress); err != nil {
		fileInfo.Status = StatusError
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	// Process file
	if err := fm.processFile(ctx, fileInfo); err != nil {
		fm.logger.Error("File processing failed: %v", err)
		// Don't fail the upload, just log the error
	}

	fileInfo.Status = StatusReady
	fileInfo.UpdatedAt = time.Now()

	// Save file info
	if err := fm.storage.SaveFileInfo(fileInfo); err != nil {
		fm.logger.Error("Failed to save file info: %v", err)
	}

	// Clean up upload progress
	fm.mu.Lock()
	delete(fm.uploads, fileID)
	fm.mu.Unlock()

	fm.logger.Info("File uploaded successfully: %s (%s)", filename, fileID)
	return fileInfo, nil
}

// DownloadFile downloads a file
func (fm *FileManager) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	fm.mu.RLock()
	fileInfo, exists := fm.files[fileID]
	fm.mu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("file not found: %s", fileID)
	}

	if fileInfo.Status != StatusReady {
		return nil, nil, fmt.Errorf("file not ready: %s", fileInfo.Status)
	}

	// Update access time
	fileInfo.AccessedAt = time.Now()

	// Open file
	reader, err := fm.storage.OpenFile(fileInfo.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	fm.logger.Debug("File download started: %s", fileID)
	return reader, fileInfo, nil
}

// GetFileInfo retrieves file information
func (fm *FileManager) GetFileInfo(fileID string) (*FileInfo, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fileInfo, exists := fm.files[fileID]
	return fileInfo, exists
}

// ListFiles lists files with optional filters
func (fm *FileManager) ListFiles(filters map[string]interface{}) []*FileInfo {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var result []*FileInfo

	for _, fileInfo := range fm.files {
		if fm.matchesFilters(fileInfo, filters) {
			result = append(result, fileInfo)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// DeleteFile deletes a file
func (fm *FileManager) DeleteFile(fileID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fileInfo, exists := fm.files[fileID]
	if !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// Delete physical file
	if err := fm.storage.DeleteFile(fileInfo.Path); err != nil {
		fm.logger.Error("Failed to delete physical file: %v", err)
	}

	// Delete thumbnail
	if fileInfo.Thumbnail != nil {
		if err := fm.storage.DeleteFile(fileInfo.Thumbnail.Path); err != nil {
			fm.logger.Error("Failed to delete thumbnail: %v", err)
		}
	}

	// Delete preview
	if fileInfo.Preview != nil {
		if err := fm.storage.DeleteFile(fileInfo.Preview.Path); err != nil {
			fm.logger.Error("Failed to delete preview: %v", err)
		}
	}

	// Delete versions
	for _, version := range fileInfo.Versions {
		if err := fm.storage.DeleteFile(version.Path); err != nil {
			fm.logger.Error("Failed to delete version: %v", err)
		}
	}

	// Update status
	fileInfo.Status = StatusDeleted
	fileInfo.UpdatedAt = time.Now()

	// Remove from memory
	delete(fm.files, fileID)

	// Delete from storage
	if err := fm.storage.DeleteFileInfo(fileID); err != nil {
		fm.logger.Error("Failed to delete file info: %v", err)
	}

	fm.logger.Info("File deleted: %s", fileID)
	return nil
}

// GetUploadProgress returns upload progress for a file
func (fm *FileManager) GetUploadProgress(fileID string) (*UploadProgress, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	progress, exists := fm.uploads[fileID]
	return progress, exists
}

// GetThumbnail returns thumbnail for a file
func (fm *FileManager) GetThumbnail(fileID string) (io.ReadCloser, error) {
	fm.mu.RLock()
	fileInfo, exists := fm.files[fileID]
	fm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}

	if fileInfo.Thumbnail == nil {
		return nil, fmt.Errorf("thumbnail not available for file: %s", fileID)
	}

	return fm.storage.OpenFile(fileInfo.Thumbnail.Path)
}

// GetPreview returns preview for a file
func (fm *FileManager) GetPreview(fileID string) (io.ReadCloser, error) {
	fm.mu.RLock()
	fileInfo, exists := fm.files[fileID]
	fm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}

	if fileInfo.Preview == nil {
		return nil, fmt.Errorf("preview not available for file: %s", fileID)
	}

	return fm.storage.OpenFile(fileInfo.Preview.Path)
}

// UpdateMetadata updates file metadata
func (fm *FileManager) UpdateMetadata(fileID string, metadata map[string]interface{}) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fileInfo, exists := fm.files[fileID]
	if !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// Update metadata
	for key, value := range metadata {
		fileInfo.Metadata[key] = value
	}

	fileInfo.UpdatedAt = time.Now()

	// Save to storage
	if err := fm.storage.SaveFileInfo(fileInfo); err != nil {
		return fmt.Errorf("failed to save file info: %w", err)
	}

	fm.logger.Debug("File metadata updated: %s", fileID)
	return nil
}

// AddTags adds tags to a file
func (fm *FileManager) AddTags(fileID string, tags []string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fileInfo, exists := fm.files[fileID]
	if !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// Add new tags
	for _, tag := range tags {
		found := false
		for _, existingTag := range fileInfo.Tags {
			if existingTag == tag {
				found = true
				break
			}
		}
		if !found {
			fileInfo.Tags = append(fileInfo.Tags, tag)
		}
	}

	fileInfo.UpdatedAt = time.Now()

	// Save to storage
	if err := fm.storage.SaveFileInfo(fileInfo); err != nil {
		return fmt.Errorf("failed to save file info: %w", err)
	}

	fm.logger.Debug("Tags added to file: %s", fileID)
	return nil
}

// GetStats returns file manager statistics
func (fm *FileManager) GetStats() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_files":    len(fm.files),
		"active_uploads": len(fm.uploads),
	}

	// Count by type
	typeCounts := make(map[FileType]int)
	statusCounts := make(map[FileStatus]int)
	var totalSize int64

	for _, fileInfo := range fm.files {
		typeCounts[fileInfo.Type]++
		statusCounts[fileInfo.Status]++
		totalSize += fileInfo.Size
	}

	stats["by_type"] = typeCounts
	stats["by_status"] = statusCounts
	stats["total_size"] = totalSize
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)

	return stats
}

// Helper methods
func (fm *FileManager) createDirectories() {
	dirs := []string{
		fm.config.StorageDir,
		fm.config.ThumbnailDir,
		fm.config.PreviewDir,
		fm.config.TempDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fm.logger.Error("Failed to create directory %s: %v", dir, err)
		}
	}
}

func (fm *FileManager) validateFile(filename string) error {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))

	// Check blocked types
	for _, blocked := range fm.config.BlockedTypes {
		if strings.Contains(blocked, ext) || blocked == "*" {
			return fmt.Errorf("file type not allowed: %s", ext)
		}
	}

	// Check allowed types if specified
	if len(fm.config.AllowedTypes) > 0 {
		allowed := false
		mimeType := fm.detectMimeType(filename)

		for _, allowedType := range fm.config.AllowedTypes {
			if strings.HasSuffix(allowedType, "*") {
				prefix := strings.TrimSuffix(allowedType, "*")
				if strings.HasPrefix(mimeType, prefix) {
					allowed = true
					break
				}
			} else if allowedType == mimeType || strings.Contains(allowedType, ext) {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("file type not allowed: %s", ext)
		}
	}

	return nil
}

func (fm *FileManager) detectFileType(filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case strings.Contains(".jpg.jpeg.png.gif.bmp.webp.svg", ext):
		return FileTypeImage
	case strings.Contains(".mp4.avi.mov.wmv.flv.webm.mkv", ext):
		return FileTypeVideo
	case strings.Contains(".mp3.wav.flac.aac.ogg.wma", ext):
		return FileTypeAudio
	case strings.Contains(".pdf.doc.docx.xls.xlsx.ppt.pptx.txt.rtf", ext):
		return FileTypeDocument
	case strings.Contains(".zip.rar.7z.tar.gz.bz2", ext):
		return FileTypeArchive
	case strings.Contains(".js.ts.go.py.java.cpp.c.h.css.html.xml.json.yaml.yml", ext):
		return FileTypeCode
	case strings.Contains(".txt.md.log.cfg.conf.ini", ext):
		return FileTypeText
	default:
		return FileTypeOther
	}
}

func (fm *FileManager) detectMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

func (fm *FileManager) performUpload(ctx context.Context, reader io.Reader, fileInfo *FileInfo, progress *UploadProgress) error {
	fileInfo.Status = StatusUploading

	// Create file path
	filePath := filepath.Join(fm.config.StorageDir, fileInfo.ID+fileInfo.Extension)
	fileInfo.Path = filePath

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create hash writers
	md5Hash := md5.New()
	sha256Hash := sha256.New()
	multiWriter := io.MultiWriter(file, md5Hash, sha256Hash)

	// Copy with progress tracking
	buffer := make([]byte, fm.config.ChunkSize)
	var totalBytes int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := reader.Read(buffer)
		if n > 0 {
			if _, writeErr := multiWriter.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("failed to write file: %w", writeErr)
			}

			totalBytes += int64(n)

			// Update progress
			fm.updateProgress(progress, totalBytes)

			// Check file size limit
			if fm.config.MaxFileSize > 0 && totalBytes > fm.config.MaxFileSize {
				return fmt.Errorf("file size exceeds limit: %d > %d", totalBytes, fm.config.MaxFileSize)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Set file info
	fileInfo.Size = totalBytes
	fileInfo.Hash = hex.EncodeToString(md5Hash.Sum(nil))
	fileInfo.Checksum = hex.EncodeToString(sha256Hash.Sum(nil))

	return nil
}

func (fm *FileManager) updateProgress(progress *UploadProgress, bytesUploaded int64) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(progress.StartTime)

	progress.BytesUploaded = bytesUploaded
	if progress.TotalBytes > 0 {
		progress.Percentage = float64(bytesUploaded) / float64(progress.TotalBytes) * 100
	}

	if elapsed.Seconds() > 0 {
		progress.Speed = int64(float64(bytesUploaded) / elapsed.Seconds())
		if progress.Speed > 0 && progress.TotalBytes > 0 {
			remaining := progress.TotalBytes - bytesUploaded
			progress.ETA = time.Duration(float64(remaining) / float64(progress.Speed) * float64(time.Second))
		}
	}

	progress.LastUpdate = now
}

func (fm *FileManager) processFile(ctx context.Context, fileInfo *FileInfo) error {
	fileInfo.Status = StatusProcessing

	// Generate thumbnail
	if fm.config.GenerateThumbnails && fm.canGenerateThumbnail(fileInfo.Type) {
		if thumbnail, err := fm.processor.GenerateThumbnail(ctx, fileInfo); err != nil {
			fm.logger.Error("Failed to generate thumbnail: %v", err)
		} else {
			fileInfo.Thumbnail = thumbnail
		}
	}

	// Generate preview
	if fm.config.GeneratePreviews && fm.canGeneratePreview(fileInfo.Type) {
		if preview, err := fm.processor.GeneratePreview(ctx, fileInfo); err != nil {
			fm.logger.Error("Failed to generate preview: %v", err)
		} else {
			fileInfo.Preview = preview
		}
	}

	// Virus scan
	if fm.config.VirusScanEnabled {
		if scanResult, err := fm.processor.ScanForViruses(ctx, fileInfo); err != nil {
			fm.logger.Error("Failed to scan for viruses: %v", err)
		} else {
			fileInfo.VirusScan = scanResult
			if !scanResult.Clean {
				return fmt.Errorf("file contains threats: %v", scanResult.Threats)
			}
		}
	}

	return nil
}

func (fm *FileManager) canGenerateThumbnail(fileType FileType) bool {
	return fileType == FileTypeImage || fileType == FileTypeVideo || fileType == FileTypeDocument
}

func (fm *FileManager) canGeneratePreview(fileType FileType) bool {
	return fileType == FileTypeImage || fileType == FileTypeDocument || fileType == FileTypeText
}

func (fm *FileManager) matchesFilters(fileInfo *FileInfo, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "type":
			if string(fileInfo.Type) != value {
				return false
			}
		case "status":
			if string(fileInfo.Status) != value {
				return false
			}
		case "extension":
			if fileInfo.Extension != value {
				return false
			}
		case "tag":
			found := false
			for _, tag := range fileInfo.Tags {
				if tag == value {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func (fm *FileManager) cleanupRoutine() {
	ticker := time.NewTicker(fm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.performCleanup()
		case <-fm.ctx.Done():
			return
		}
	}
}

func (fm *FileManager) performCleanup() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	now := time.Now()
	cutoff := now.AddDate(0, 0, -fm.config.RetentionDays)
	var toDelete []string

	for id, fileInfo := range fm.files {
		// Delete old files
		if fileInfo.CreatedAt.Before(cutoff) && fileInfo.Status == StatusDeleted {
			toDelete = append(toDelete, id)
		}
	}

	// Delete files
	for _, id := range toDelete {
		delete(fm.files, id)
		if err := fm.storage.DeleteFileInfo(id); err != nil {
			fm.logger.Error("Failed to delete file info: %v", err)
		}
	}

	if len(toDelete) > 0 {
		fm.logger.Info("Cleaned up %d old files", len(toDelete))
	}
}

// Shutdown gracefully shuts down the file manager
func (fm *FileManager) Shutdown() {
	fm.logger.Info("Shutting down file manager...")
	fm.cancel()

	// Close storage
	if fm.storage != nil {
		fm.storage.Close()
	}

	fm.logger.Info("File manager shutdown complete")
}

// Helper functions
func generateFileID() string {
	return fmt.Sprintf("file_%d", time.Now().UnixNano())
}
