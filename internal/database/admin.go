package database

import (
	"database/sql"
	"fmt"
	"time"

	"carryless/internal/models"
)

type AdminStats struct {
	TotalUsers  int `json:"total_users"`
	TotalPacks  int `json:"total_packs"`
	ActiveUsers int `json:"active_users"`
}

type UserWithStats struct {
	ID          int            `json:"id"`
	Username    string         `json:"username"`
	Email       string         `json:"email"`
	Currency    string         `json:"currency"`
	IsAdmin     bool           `json:"is_admin"`
	IsActivated bool           `json:"is_activated"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	PackCount   int            `json:"pack_count"`
	LastSeen    sql.NullTime   `json:"last_seen"`
}

func GetAdminStats(db *sql.DB) (*AdminStats, error) {
	stats := &AdminStats{}

	// Get total users
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get total packs
	err = db.QueryRow("SELECT COUNT(*) FROM packs").Scan(&stats.TotalPacks)
	if err != nil {
		return nil, fmt.Errorf("failed to get pack count: %w", err)
	}

	// Get active users (last seen within 30 days)
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT id)
		FROM users
		WHERE last_seen IS NOT NULL
		AND last_seen > datetime('now', '-30 days')
	`).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get active user count: %w", err)
	}

	return stats, nil
}

func GetAllUsers(db *sql.DB) ([]models.User, error) {
	query := `
		SELECT id, username, email, COALESCE(currency, '$'), COALESCE(is_admin, false), COALESCE(is_activated, false), created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()
	
	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Currency,
			&user.IsAdmin,
			&user.IsActivated,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, nil
}

func GetAllUsersWithStats(db *sql.DB) ([]UserWithStats, error) {
	query := `
		SELECT
			u.id,
			u.username,
			u.email,
			COALESCE(u.currency, '$'),
			COALESCE(u.is_admin, false),
			COALESCE(u.is_activated, false),
			u.created_at,
			u.updated_at,
			u.last_seen,
			COUNT(p.id) as pack_count
		FROM users u
		LEFT JOIN packs p ON u.id = p.user_id
		GROUP BY u.id, u.username, u.email, u.currency, u.is_admin, u.is_activated, u.created_at, u.updated_at, u.last_seen
		ORDER BY u.created_at ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users with stats: %w", err)
	}
	defer rows.Close()

	var users []UserWithStats
	for rows.Next() {
		var user UserWithStats
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Currency,
			&user.IsAdmin,
			&user.IsActivated,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastSeen,
			&user.PackCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user with stats: %w", err)
		}
		users = append(users, user)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users with stats: %w", err)
	}
	
	return users, nil
}

func ToggleUserAdmin(db *sql.DB, userID int) error {
	query := `UPDATE users SET is_admin = NOT COALESCE(is_admin, false) WHERE id = ?`
	_, err := db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to toggle admin status: %w", err)
	}
	return nil
}

func BanUser(db *sql.DB, userID int) error {
	// Start a transaction to ensure all operations succeed or fail together
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all user sessions first
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// Delete all user CSRF tokens
	_, err = tx.Exec("DELETE FROM csrf_tokens WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user CSRF tokens: %w", err)
	}

	// Delete all pack_items for user's packs
	_, err = tx.Exec("DELETE FROM pack_items WHERE pack_id IN (SELECT id FROM packs WHERE user_id = ?)", userID)
	if err != nil {
		return fmt.Errorf("failed to delete pack items: %w", err)
	}

	// Delete all user's packs
	_, err = tx.Exec("DELETE FROM packs WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user packs: %w", err)
	}

	// Delete all user's items
	_, err = tx.Exec("DELETE FROM items WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user items: %w", err)
	}

	// Delete all user's categories
	_, err = tx.Exec("DELETE FROM categories WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user categories: %w", err)
	}

	// Finally, delete the user
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func IsRegistrationEnabled(db *sql.DB) (bool, error) {
	var value string
	err := db.QueryRow("SELECT value FROM system_settings WHERE key = 'registration_enabled'").Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil // Default to enabled if setting doesn't exist
		}
		return false, fmt.Errorf("failed to query registration setting: %w", err)
	}
	return value == "true", nil
}

func GetAllAdmins(db *sql.DB) ([]models.User, error) {
	query := `
		SELECT id, username, email, COALESCE(currency, '$'), COALESCE(is_admin, false), COALESCE(is_activated, false), created_at, updated_at
		FROM users
		WHERE COALESCE(is_admin, false) = true
		ORDER BY username ASC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query admin users: %w", err)
	}
	defer rows.Close()
	
	var admins []models.User
	for rows.Next() {
		var admin models.User
		err := rows.Scan(
			&admin.ID,
			&admin.Username,
			&admin.Email,
			&admin.Currency,
			&admin.IsAdmin,
			&admin.IsActivated,
			&admin.CreatedAt,
			&admin.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin user: %w", err)
		}
		admins = append(admins, admin)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating admin users: %w", err)
	}
	
	return admins, nil
}

func ToggleRegistration(db *sql.DB) error {
	query := `UPDATE system_settings SET value = CASE WHEN value = 'true' THEN 'false' ELSE 'true' END, updated_at = CURRENT_TIMESTAMP WHERE key = 'registration_enabled'`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to toggle registration setting: %w", err)
	}
	return nil
}

func ToggleUserActivation(db *sql.DB, userID int) error {
	query := `UPDATE users SET is_activated = NOT COALESCE(is_activated, false), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to toggle user activation: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}