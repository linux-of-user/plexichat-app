package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveUser saves or updates a user in the database
func (d *Database) SaveUser(ctx context.Context, user *User) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO users (id, username, display_name, email, avatar, status, last_seen, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			display_name = excluded.display_name,
			email = excluded.email,
			avatar = excluded.avatar,
			status = excluded.status,
			last_seen = excluded.last_seen,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := d.db.ExecContext(ctx, query,
		user.ID, user.Username, user.DisplayName, user.Email,
		user.Avatar, user.Status, user.LastSeen, user.CreatedAt, user.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID
func (d *Database) GetUser(ctx context.Context, userID string) (*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, username, display_name, email, avatar, status, 
			   last_seen, created_at, updated_at, metadata
		FROM users WHERE id = ?
	`

	user := &User{}
	err := d.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.Email,
		&user.Avatar, &user.Status, &user.LastSeen, &user.CreatedAt,
		&user.UpdatedAt, &user.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (d *Database) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, username, display_name, email, avatar, status, 
			   last_seen, created_at, updated_at, metadata
		FROM users WHERE username = ?
	`

	user := &User{}
	err := d.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.Email,
		&user.Avatar, &user.Status, &user.LastSeen, &user.CreatedAt,
		&user.UpdatedAt, &user.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUsers retrieves all users with pagination
func (d *Database) GetUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, username, display_name, email, avatar, status, 
			   last_seen, created_at, updated_at, metadata
		FROM users 
		ORDER BY username
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Username, &user.DisplayName,
			&user.Email, &user.Avatar, &user.Status, &user.LastSeen,
			&user.CreatedAt, &user.UpdatedAt, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// UpdateUserStatus updates a user's status
func (d *Database) UpdateUserStatus(ctx context.Context, userID, status string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE users SET status = ?, last_seen = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, status, userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

// UpdateUserLastSeen updates a user's last seen timestamp
func (d *Database) UpdateUserLastSeen(ctx context.Context, userID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update user last seen: %w", err)
	}

	return nil
}

// SearchUsers searches for users by username or display name
func (d *Database) SearchUsers(ctx context.Context, searchText string, limit int) ([]*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, username, display_name, email, avatar, status, 
			   last_seen, created_at, updated_at, metadata
		FROM users 
		WHERE username LIKE ? OR display_name LIKE ?
		ORDER BY username
		LIMIT ?
	`

	searchPattern := "%" + searchText + "%"
	rows, err := d.db.QueryContext(ctx, query, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Username, &user.DisplayName,
			&user.Email, &user.Avatar, &user.Status, &user.LastSeen,
			&user.CreatedAt, &user.UpdatedAt, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetOnlineUsers retrieves users who are currently online
func (d *Database) GetOnlineUsers(ctx context.Context) ([]*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Consider users online if they were active in the last 5 minutes
	threshold := time.Now().Add(-5 * time.Minute)

	query := `
		SELECT id, username, display_name, email, avatar, status, 
			   last_seen, created_at, updated_at, metadata
		FROM users 
		WHERE status = 'online' AND last_seen > ?
		ORDER BY last_seen DESC
	`

	rows, err := d.db.QueryContext(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query online users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Username, &user.DisplayName,
			&user.Email, &user.Avatar, &user.Status, &user.LastSeen,
			&user.CreatedAt, &user.UpdatedAt, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// DeleteUser soft deletes a user (marks as inactive)
func (d *Database) DeleteUser(ctx context.Context, userID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Instead of deleting, we update the status to 'deleted'
	query := `UPDATE users SET status = 'deleted', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetUserStats returns statistics about users
func (d *Database) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total users
	var totalUsers int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE status != 'deleted'").Scan(&totalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	stats["total_users"] = totalUsers

	// Online users
	threshold := time.Now().Add(-5 * time.Minute)
	var onlineUsers int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE status = 'online' AND last_seen > ?", threshold).Scan(&onlineUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count online users: %w", err)
	}
	stats["online_users"] = onlineUsers

	// Users by status
	statusQuery := `
		SELECT status, COUNT(*) 
		FROM users 
		WHERE status != 'deleted'
		GROUP BY status
	`
	rows, err := d.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query user status stats: %w", err)
	}
	defer rows.Close()

	statusStats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status stats: %w", err)
		}
		statusStats[status] = count
	}
	stats["status_breakdown"] = statusStats

	return stats, nil
}
