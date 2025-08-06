package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveChannel saves or updates a channel in the database
func (d *Database) SaveChannel(ctx context.Context, channel *Channel) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO channels (id, name, description, type, private, created_by, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			type = excluded.type,
			private = excluded.private,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := d.db.ExecContext(ctx, query,
		channel.ID, channel.Name, channel.Description, channel.Type,
		channel.Private, channel.CreatedBy, channel.CreatedAt, channel.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	return nil
}

// GetChannel retrieves a channel by ID
func (d *Database) GetChannel(ctx context.Context, channelID string) (*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels WHERE id = ?
	`

	channel := &Channel{}
	err := d.db.QueryRowContext(ctx, query, channelID).Scan(
		&channel.ID, &channel.Name, &channel.Description, &channel.Type,
		&channel.Private, &channel.CreatedBy, &channel.CreatedAt,
		&channel.UpdatedAt, &channel.LastMessage, &channel.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel not found: %s", channelID)
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// GetChannelByName retrieves a channel by name
func (d *Database) GetChannelByName(ctx context.Context, name string) (*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels WHERE name = ?
	`

	channel := &Channel{}
	err := d.db.QueryRowContext(ctx, query, name).Scan(
		&channel.ID, &channel.Name, &channel.Description, &channel.Type,
		&channel.Private, &channel.CreatedBy, &channel.CreatedAt,
		&channel.UpdatedAt, &channel.LastMessage, &channel.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// GetChannels retrieves all channels with pagination
func (d *Database) GetChannels(ctx context.Context, limit, offset int) ([]*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels 
		ORDER BY name
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		err := rows.Scan(&channel.ID, &channel.Name, &channel.Description,
			&channel.Type, &channel.Private, &channel.CreatedBy,
			&channel.CreatedAt, &channel.UpdatedAt, &channel.LastMessage,
			&channel.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// GetPublicChannels retrieves all public channels
func (d *Database) GetPublicChannels(ctx context.Context) ([]*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels 
		WHERE private = FALSE
		ORDER BY name
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query public channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		err := rows.Scan(&channel.ID, &channel.Name, &channel.Description,
			&channel.Type, &channel.Private, &channel.CreatedBy,
			&channel.CreatedAt, &channel.UpdatedAt, &channel.LastMessage,
			&channel.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// GetUserChannels retrieves channels created by a specific user
func (d *Database) GetUserChannels(ctx context.Context, userID string) ([]*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels 
		WHERE created_by = ?
		ORDER BY created_at DESC
	`

	rows, err := d.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		err := rows.Scan(&channel.ID, &channel.Name, &channel.Description,
			&channel.Type, &channel.Private, &channel.CreatedBy,
			&channel.CreatedAt, &channel.UpdatedAt, &channel.LastMessage,
			&channel.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// SearchChannels searches for channels by name or description
func (d *Database) SearchChannels(ctx context.Context, searchText string, limit int) ([]*Channel, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, name, description, type, private, created_by, 
			   created_at, updated_at, last_message, metadata
		FROM channels 
		WHERE (name LIKE ? OR description LIKE ?) AND private = FALSE
		ORDER BY name
		LIMIT ?
	`

	searchPattern := "%" + searchText + "%"
	rows, err := d.db.QueryContext(ctx, query, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel := &Channel{}
		err := rows.Scan(&channel.ID, &channel.Name, &channel.Description,
			&channel.Type, &channel.Private, &channel.CreatedBy,
			&channel.CreatedAt, &channel.UpdatedAt, &channel.LastMessage,
			&channel.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// UpdateChannelLastMessage updates the last message timestamp for a channel
func (d *Database) UpdateChannelLastMessage(ctx context.Context, channelID string, timestamp time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE channels SET last_message = ? WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, timestamp, channelID)
	if err != nil {
		return fmt.Errorf("failed to update channel last message: %w", err)
	}

	return nil
}

// DeleteChannel deletes a channel and all its messages
func (d *Database) DeleteChannel(ctx context.Context, channelID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Start transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all messages in the channel
	_, err = tx.ExecContext(ctx, "DELETE FROM messages WHERE channel_id = ?", channelID)
	if err != nil {
		return fmt.Errorf("failed to delete channel messages: %w", err)
	}

	// Delete all files in the channel
	_, err = tx.ExecContext(ctx, "DELETE FROM files WHERE channel_id = ?", channelID)
	if err != nil {
		return fmt.Errorf("failed to delete channel files: %w", err)
	}

	// Delete the channel
	_, err = tx.ExecContext(ctx, "DELETE FROM channels WHERE id = ?", channelID)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetChannelStats returns statistics about channels
func (d *Database) GetChannelStats(ctx context.Context) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total channels
	var totalChannels int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM channels").Scan(&totalChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to count total channels: %w", err)
	}
	stats["total_channels"] = totalChannels

	// Public vs private channels
	var publicChannels, privateChannels int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM channels WHERE private = FALSE").Scan(&publicChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to count public channels: %w", err)
	}
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM channels WHERE private = TRUE").Scan(&privateChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to count private channels: %w", err)
	}
	stats["public_channels"] = publicChannels
	stats["private_channels"] = privateChannels

	// Channels by type
	typeQuery := `
		SELECT type, COUNT(*) 
		FROM channels 
		GROUP BY type
	`
	rows, err := d.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query channel type stats: %w", err)
	}
	defer rows.Close()

	typeStats := make(map[string]int)
	for rows.Next() {
		var channelType string
		var count int
		if err := rows.Scan(&channelType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type stats: %w", err)
		}
		typeStats[channelType] = count
	}
	stats["type_breakdown"] = typeStats

	return stats, nil
}
