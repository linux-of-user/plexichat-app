package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plexichat-client/pkg/logging"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the local SQLite database
type Database struct {
	db     *sql.DB
	logger *logging.Logger
	mu     sync.RWMutex
	path   string
}

// Message represents a chat message in the database
type Message struct {
	ID          int64      `json:"id" db:"id"`
	ChannelID   string     `json:"channel_id" db:"channel_id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Username    string     `json:"username" db:"username"`
	Content     string     `json:"content" db:"content"`
	MessageType string     `json:"message_type" db:"message_type"`
	Timestamp   time.Time  `json:"timestamp" db:"timestamp"`
	EditedAt    *time.Time `json:"edited_at,omitempty" db:"edited_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	Metadata    string     `json:"metadata" db:"metadata"`
	Attachments string     `json:"attachments" db:"attachments"`
}

// User represents a user in the database
type User struct {
	ID          string    `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Email       string    `json:"email" db:"email"`
	Avatar      string    `json:"avatar" db:"avatar"`
	Status      string    `json:"status" db:"status"`
	LastSeen    time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Metadata    string    `json:"metadata" db:"metadata"`
}

// Channel represents a chat channel in the database
type Channel struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Type        string     `json:"type" db:"type"`
	Private     bool       `json:"private" db:"private"`
	CreatedBy   string     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastMessage *time.Time `json:"last_message,omitempty" db:"last_message"`
	Metadata    string     `json:"metadata" db:"metadata"`
}

// File represents a file in the database
type File struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Path       string    `json:"path" db:"path"`
	Size       int64     `json:"size" db:"size"`
	MimeType   string    `json:"mime_type" db:"mime_type"`
	Hash       string    `json:"hash" db:"hash"`
	UploadedBy string    `json:"uploaded_by" db:"uploaded_by"`
	ChannelID  string    `json:"channel_id" db:"channel_id"`
	MessageID  *int64    `json:"message_id,omitempty" db:"message_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	Metadata   string    `json:"metadata" db:"metadata"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	LastUsed  time.Time `json:"last_used" db:"last_used"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	database := &Database{
		db:     db,
		logger: logging.NewLogger(logging.INFO, nil, true),
		path:   dbPath,
	}

	// Initialize database schema
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initSchema creates the database tables
func (d *Database) initSchema() error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL,
		email TEXT,
		avatar TEXT,
		status TEXT DEFAULT 'offline',
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT DEFAULT '{}'
	);

	-- Channels table
	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		type TEXT DEFAULT 'text',
		private BOOLEAN DEFAULT FALSE,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_message DATETIME,
		metadata TEXT DEFAULT '{}',
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		username TEXT NOT NULL,
		content TEXT NOT NULL,
		message_type TEXT DEFAULT 'text',
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		edited_at DATETIME,
		deleted_at DATETIME,
		metadata TEXT DEFAULT '{}',
		attachments TEXT DEFAULT '[]',
		FOREIGN KEY (channel_id) REFERENCES channels(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Files table
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		hash TEXT NOT NULL,
		uploaded_by TEXT NOT NULL,
		channel_id TEXT,
		message_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT DEFAULT '{}',
		FOREIGN KEY (uploaded_by) REFERENCES users(id),
		FOREIGN KEY (channel_id) REFERENCES channels(id),
		FOREIGN KEY (message_id) REFERENCES messages(id)
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		user_agent TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_messages_channel_timestamp ON messages(channel_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_messages_user_timestamp ON messages(user_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_files_channel ON files(channel_id);
	CREATE INDEX IF NOT EXISTS idx_files_user ON files(uploaded_by);
	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

	-- Triggers for updated_at
	CREATE TRIGGER IF NOT EXISTS update_users_timestamp 
		AFTER UPDATE ON users
		BEGIN
			UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_channels_timestamp 
		AFTER UPDATE ON channels
		BEGIN
			UPDATE channels SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`

	_, err := d.db.Exec(schema)
	return err
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.db.Ping()
}

// GetStats returns database statistics
func (d *Database) GetStats() (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]interface{})

	// Get table counts
	tables := []string{"users", "channels", "messages", "files", "sessions"}
	for _, table := range tables {
		var count int
		err := d.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count %s: %w", table, err)
		}
		stats[table+"_count"] = count
	}

	// Get database size
	var size int64
	if stat, err := os.Stat(d.path); err == nil {
		size = stat.Size()
	}
	stats["database_size"] = size

	return stats, nil
}

// SaveMessage saves a message to the database
func (d *Database) SaveMessage(ctx context.Context, msg *Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO messages (channel_id, user_id, username, content, message_type, timestamp, metadata, attachments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.ExecContext(ctx, query,
		msg.ChannelID, msg.UserID, msg.Username, msg.Content,
		msg.MessageType, msg.Timestamp, msg.Metadata, msg.Attachments)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get message ID: %w", err)
	}
	msg.ID = id

	// Update channel last message time
	_, err = d.db.ExecContext(ctx, "UPDATE channels SET last_message = ? WHERE id = ?",
		msg.Timestamp, msg.ChannelID)
	if err != nil {
		d.logger.Error("Failed to update channel last message: %v", err)
	}

	return nil
}

// GetMessages retrieves messages from a channel
func (d *Database) GetMessages(ctx context.Context, channelID string, limit, offset int) ([]*Message, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, channel_id, user_id, username, content, message_type,
			   timestamp, edited_at, deleted_at, metadata, attachments
		FROM messages
		WHERE channel_id = ? AND deleted_at IS NULL
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, channelID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Username,
			&msg.Content, &msg.MessageType, &msg.Timestamp, &msg.EditedAt,
			&msg.DeletedAt, &msg.Metadata, &msg.Attachments)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// UpdateMessage updates an existing message
func (d *Database) UpdateMessage(ctx context.Context, messageID int64, content string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE messages SET content = ?, edited_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, content, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// DeleteMessage soft deletes a message
func (d *Database) DeleteMessage(ctx context.Context, messageID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// SearchMessages searches for messages containing the given text
func (d *Database) SearchMessages(ctx context.Context, searchText string, limit int) ([]*Message, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, channel_id, user_id, username, content, message_type,
			   timestamp, edited_at, deleted_at, metadata, attachments
		FROM messages
		WHERE content LIKE ? AND deleted_at IS NULL
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, "%"+searchText+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Username,
			&msg.Content, &msg.MessageType, &msg.Timestamp, &msg.EditedAt,
			&msg.DeletedAt, &msg.Metadata, &msg.Attachments)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
