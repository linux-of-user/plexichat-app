package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SaveSession saves a session to the database
func (d *Database) SaveSession(ctx context.Context, session *Session) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at, last_used, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			token = excluded.token,
			expires_at = excluded.expires_at,
			last_used = excluded.last_used,
			ip_address = excluded.ip_address,
			user_agent = excluded.user_agent
	`

	_, err := d.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.Token, session.ExpiresAt,
		session.CreatedAt, session.LastUsed, session.IPAddress, session.UserAgent)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by ID
func (d *Database) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used, ip_address, user_agent
		FROM sessions WHERE id = ?
	`

	session := &Session{}
	err := d.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsed, &session.IPAddress, &session.UserAgent)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionByToken retrieves a session by token
func (d *Database) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used, ip_address, user_agent
		FROM sessions WHERE token = ?
	`

	session := &Session{}
	err := d.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsed, &session.IPAddress, &session.UserAgent)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found for token")
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return session, nil
}

// GetUserSessions retrieves all sessions for a user
func (d *Database) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used, ip_address, user_agent
		FROM sessions 
		WHERE user_id = ?
		ORDER BY last_used DESC
	`

	rows, err := d.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(&session.ID, &session.UserID, &session.Token,
			&session.ExpiresAt, &session.CreatedAt, &session.LastUsed,
			&session.IPAddress, &session.UserAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// GetActiveSessions retrieves all active (non-expired) sessions
func (d *Database) GetActiveSessions(ctx context.Context) ([]*Session, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used, ip_address, user_agent
		FROM sessions 
		WHERE expires_at > CURRENT_TIMESTAMP
		ORDER BY last_used DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(&session.ID, &session.UserID, &session.Token,
			&session.ExpiresAt, &session.CreatedAt, &session.LastUsed,
			&session.IPAddress, &session.UserAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// UpdateSessionLastUsed updates the last used timestamp for a session
func (d *Database) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE sessions SET last_used = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session last used: %w", err)
	}

	return nil
}

// ExtendSession extends the expiration time of a session
func (d *Database) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	newExpiry := time.Now().Add(duration)
	query := `UPDATE sessions SET expires_at = ?, last_used = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, newExpiry, sessionID)
	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	return nil
}

// ValidateSession checks if a session is valid (exists and not expired)
func (d *Database) ValidateSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := d.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Update last used timestamp
	if err := d.UpdateSessionLastUsed(ctx, sessionID); err != nil {
		d.logger.Error("Failed to update session last used: %v", err)
	}

	return session, nil
}

// DeleteSession deletes a session
func (d *Database) DeleteSession(ctx context.Context, sessionID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM sessions WHERE id = ?`
	result, err := d.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (d *Database) DeleteUserSessions(ctx context.Context, userID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := d.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions
func (d *Database) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`
	result, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetSessionStats returns statistics about sessions
func (d *Database) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total sessions
	var totalSessions int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&totalSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to count total sessions: %w", err)
	}
	stats["total_sessions"] = totalSessions

	// Active sessions
	var activeSessions int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE expires_at > CURRENT_TIMESTAMP").Scan(&activeSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to count active sessions: %w", err)
	}
	stats["active_sessions"] = activeSessions

	// Expired sessions
	var expiredSessions int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP").Scan(&expiredSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to count expired sessions: %w", err)
	}
	stats["expired_sessions"] = expiredSessions

	// Sessions by user
	var uniqueUsers int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT user_id) FROM sessions WHERE expires_at > CURRENT_TIMESTAMP").Scan(&uniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count unique users: %w", err)
	}
	stats["unique_active_users"] = uniqueUsers

	// Recent activity (sessions used in last hour)
	oneHourAgo := time.Now().Add(-time.Hour)
	var recentSessions int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE last_used > ?", oneHourAgo).Scan(&recentSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to count recent sessions: %w", err)
	}
	stats["recent_activity"] = recentSessions

	return stats, nil
}
