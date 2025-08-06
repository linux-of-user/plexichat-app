package database

import (
	"context"
	"database/sql"
	"fmt"
)

// SaveFile saves a file record to the database
func (d *Database) SaveFile(ctx context.Context, file *File) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO files (id, name, path, size, mime_type, hash, uploaded_by, channel_id, message_id, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			path = excluded.path,
			size = excluded.size,
			mime_type = excluded.mime_type,
			hash = excluded.hash,
			metadata = excluded.metadata
	`

	_, err := d.db.ExecContext(ctx, query,
		file.ID, file.Name, file.Path, file.Size, file.MimeType,
		file.Hash, file.UploadedBy, file.ChannelID, file.MessageID,
		file.CreatedAt, file.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// GetFile retrieves a file by ID
func (d *Database) GetFile(ctx context.Context, fileID string) (*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files WHERE id = ?
	`

	file := &File{}
	err := d.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.ID, &file.Name, &file.Path, &file.Size, &file.MimeType,
		&file.Hash, &file.UploadedBy, &file.ChannelID, &file.MessageID,
		&file.CreatedAt, &file.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found: %s", fileID)
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// GetFileByHash retrieves a file by its hash
func (d *Database) GetFileByHash(ctx context.Context, hash string) (*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files WHERE hash = ?
	`

	file := &File{}
	err := d.db.QueryRowContext(ctx, query, hash).Scan(
		&file.ID, &file.Name, &file.Path, &file.Size, &file.MimeType,
		&file.Hash, &file.UploadedBy, &file.ChannelID, &file.MessageID,
		&file.CreatedAt, &file.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found with hash: %s", hash)
		}
		return nil, fmt.Errorf("failed to get file by hash: %w", err)
	}

	return file, nil
}

// GetChannelFiles retrieves all files in a channel
func (d *Database) GetChannelFiles(ctx context.Context, channelID string, limit, offset int) ([]*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files 
		WHERE channel_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, channelID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query channel files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(&file.ID, &file.Name, &file.Path, &file.Size,
			&file.MimeType, &file.Hash, &file.UploadedBy, &file.ChannelID,
			&file.MessageID, &file.CreatedAt, &file.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// GetUserFiles retrieves all files uploaded by a user
func (d *Database) GetUserFiles(ctx context.Context, userID string, limit, offset int) ([]*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files 
		WHERE uploaded_by = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query user files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(&file.ID, &file.Name, &file.Path, &file.Size,
			&file.MimeType, &file.Hash, &file.UploadedBy, &file.ChannelID,
			&file.MessageID, &file.CreatedAt, &file.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// SearchFiles searches for files by name or mime type
func (d *Database) SearchFiles(ctx context.Context, searchText string, limit int) ([]*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files 
		WHERE name LIKE ? OR mime_type LIKE ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	searchPattern := "%" + searchText + "%"
	rows, err := d.db.QueryContext(ctx, query, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(&file.ID, &file.Name, &file.Path, &file.Size,
			&file.MimeType, &file.Hash, &file.UploadedBy, &file.ChannelID,
			&file.MessageID, &file.CreatedAt, &file.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// GetFilesByType retrieves files by mime type
func (d *Database) GetFilesByType(ctx context.Context, mimeType string, limit, offset int) ([]*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files 
		WHERE mime_type LIKE ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, mimeType+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query files by type: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(&file.ID, &file.Name, &file.Path, &file.Size,
			&file.MimeType, &file.Hash, &file.UploadedBy, &file.ChannelID,
			&file.MessageID, &file.CreatedAt, &file.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// GetRecentFiles retrieves recently uploaded files
func (d *Database) GetRecentFiles(ctx context.Context, limit int) ([]*File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, path, size, mime_type, hash, uploaded_by, 
			   channel_id, message_id, created_at, metadata
		FROM files 
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(&file.ID, &file.Name, &file.Path, &file.Size,
			&file.MimeType, &file.Hash, &file.UploadedBy, &file.ChannelID,
			&file.MessageID, &file.CreatedAt, &file.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// DeleteFile deletes a file record from the database
func (d *Database) DeleteFile(ctx context.Context, fileID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM files WHERE id = ?`
	result, err := d.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", fileID)
	}

	return nil
}

// GetFileStats returns statistics about files
func (d *Database) GetFileStats(ctx context.Context) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total files
	var totalFiles int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files").Scan(&totalFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to count total files: %w", err)
	}
	stats["total_files"] = totalFiles

	// Total storage used
	var totalSize int64
	err = d.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size), 0) FROM files").Scan(&totalSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total size: %w", err)
	}
	stats["total_size"] = totalSize

	// Files by type
	typeQuery := `
		SELECT 
			CASE 
				WHEN mime_type LIKE 'image/%' THEN 'image'
				WHEN mime_type LIKE 'video/%' THEN 'video'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'text/%' THEN 'text'
				WHEN mime_type LIKE 'application/pdf' THEN 'pdf'
				ELSE 'other'
			END as file_type,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as total_size
		FROM files 
		GROUP BY file_type
	`
	rows, err := d.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query file type stats: %w", err)
	}
	defer rows.Close()

	typeStats := make(map[string]map[string]interface{})
	for rows.Next() {
		var fileType string
		var count int
		var size int64
		if err := rows.Scan(&fileType, &count, &size); err != nil {
			return nil, fmt.Errorf("failed to scan type stats: %w", err)
		}
		typeStats[fileType] = map[string]interface{}{
			"count": count,
			"size":  size,
		}
	}
	stats["type_breakdown"] = typeStats

	return stats, nil
}
