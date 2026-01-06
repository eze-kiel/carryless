package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"carryless/internal/logger"
	"carryless/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetUserByID(db *sql.DB, userID int) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, COALESCE(currency, '$'), COALESCE(is_admin, false),
		       COALESCE(is_activated, false), created_at, updated_at
		FROM users
		WHERE id = ?
	`

	err := db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Currency,
		&user.IsAdmin,
		&user.IsActivated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return user, nil
}

func CreateUser(db *sql.DB, username, email, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if this is the first user
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	isAdmin := userCount == 0 // First user becomes admin

	query := `
		INSERT INTO users (username, email, password_hash, is_admin, is_activated)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, username, email, string(hashedPassword), isAdmin, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	user := &models.User{
		ID:           int(id),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		IsAdmin:      isAdmin,
		IsActivated:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return user, nil
}

func AuthenticateUser(db *sql.DB, email, password string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, COALESCE(is_admin, false), COALESCE(is_activated, false), created_at, updated_at
		FROM users
		WHERE email = ?
	`

	err := db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.IsActivated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func CreateSession(db *sql.DB, userID int, sessionDuration time.Duration) (*models.Session, error) {
	sessionID, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	expiresAt := time.Now().Add(sessionDuration)

	query := `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return session, nil
}

func ValidateSession(db *sql.DB, sessionID string, sessionDuration time.Duration) (*models.User, error) {
	user := &models.User{}
	var lastSeen sql.NullTime
	query := `
		SELECT u.id, u.username, u.email, COALESCE(u.currency, '$'), COALESCE(u.is_admin, false), COALESCE(u.is_activated, false), u.created_at, u.updated_at, u.last_seen
		FROM users u
		INNER JOIN sessions s ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > CURRENT_TIMESTAMP
	`

	err := db.QueryRow(query, sessionID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Currency,
		&user.IsAdmin,
		&user.IsActivated,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastSeen,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	// Update last_seen if it's been more than 5 minutes since the last update
	now := time.Now()
	shouldUpdateLastSeen := !lastSeen.Valid || now.Sub(lastSeen.Time) > 5*time.Minute

	if shouldUpdateLastSeen {
		updateLastSeenQuery := `UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = ?`
		_, err = db.Exec(updateLastSeenQuery, user.ID)
		if err != nil {
			// Log but don't fail the request if we can't update last_seen
			logger.Warn("Failed to update last_seen",
				"user_id", user.ID,
				"error", err)
		}
	}

	err = RenewSession(db, sessionID, sessionDuration)
	if err != nil {
		logger.Warn("Failed to renew session",
			"session_id", sessionID,
			"error", err)
	}

	return user, nil
}

func VerifyPassword(db *sql.DB, userID int, password string) error {
	var hashedPassword string
	query := "SELECT password_hash FROM users WHERE id = ?"
	err := db.QueryRow(query, userID).Scan(&hashedPassword)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}

func UpdatePassword(db *sql.DB, userID int, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	query := "UPDATE users SET password_hash = ? WHERE id = ?"
	_, err = db.Exec(query, string(hashedPassword), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func UpdateUserCurrency(db *sql.DB, userID int, currency string) error {
	query := "UPDATE users SET currency = ? WHERE id = ?"
	_, err := db.Exec(query, currency, userID)
	if err != nil {
		return fmt.Errorf("failed to update currency: %w", err)
	}

	return nil
}

func UpdateUsername(db *sql.DB, userID int, username string) error {
	query := "UPDATE users SET username = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := db.Exec(query, username, userID)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}

	return nil
}

func RenewSession(db *sql.DB, sessionID string, sessionDuration time.Duration) error {
	// Implementing sliding window - always extend on activity
	now := time.Now()
	newExpiresAt := now.Add(sessionDuration)

	updateQuery := `UPDATE sessions SET expires_at = ? WHERE id = ?`
	_, err := db.Exec(updateQuery, newExpiresAt, sessionID)
	if err != nil {
		return fmt.Errorf("failed to renew session: %w", err)
	}

	return nil
}

func DeleteSession(db *sql.DB, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func CleanupExpiredSessions(db *sql.DB) error {
	query := `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

func CreateCSRFToken(db *sql.DB, userID int) (*models.CSRFToken, error) {
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	query := `
		INSERT INTO csrf_tokens (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(query, token, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSRF token: %w", err)
	}

	csrfToken := &models.CSRFToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return csrfToken, nil
}

func ValidateCSRFToken(db *sql.DB, token string, userID int) error {
	query := `
		SELECT 1
		FROM csrf_tokens
		WHERE token = ? AND user_id = ? AND expires_at > CURRENT_TIMESTAMP
	`

	var exists int
	err := db.QueryRow(query, token, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("CSRF token not found or expired")
		}
		return fmt.Errorf("failed to validate CSRF token: %w", err)
	}

	query = `DELETE FROM csrf_tokens WHERE token = ?`
	_, err = db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete used CSRF token: %w", err)
	}

	return nil
}

func CleanupExpiredCSRFTokens(db *sql.DB) error {
	query := `DELETE FROM csrf_tokens WHERE expires_at < CURRENT_TIMESTAMP`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired CSRF tokens: %w", err)
	}
	return nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CreateActivationToken(db *sql.DB, userID int) (*models.ActivationToken, error) {
	tokenUUID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	query := `
		INSERT INTO activation_tokens (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`

	_, err := db.Exec(query, tokenUUID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create activation token: %w", err)
	}

	token := &models.ActivationToken{
		Token:     tokenUUID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return token, nil
}

func ValidateActivationToken(db *sql.DB, token string) (*models.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, COALESCE(u.is_admin, false), COALESCE(u.is_activated, false), u.created_at, u.updated_at
		FROM users u
		JOIN activation_tokens at ON u.id = at.user_id
		WHERE at.token = ? AND at.expires_at > CURRENT_TIMESTAMP
	`

	user := &models.User{}
	err := db.QueryRow(query, token).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.IsActivated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("activation token not found or expired")
		}
		return nil, fmt.Errorf("failed to validate activation token: %w", err)
	}

	return user, nil
}

func ActivateUser(db *sql.DB, userID int, token string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updateUserQuery := `UPDATE users SET is_activated = TRUE WHERE id = ?`
	_, err = tx.Exec(updateUserQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	deleteTokenQuery := `DELETE FROM activation_tokens WHERE token = ?`
	_, err = tx.Exec(deleteTokenQuery, token)
	if err != nil {
		return fmt.Errorf("failed to delete activation token: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit activation: %w", err)
	}

	return nil
}

func ResendActivationToken(db *sql.DB, userID int) (*models.ActivationToken, error) {
	// Start a transaction to ensure atomicity
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete any existing activation tokens for this user
	deleteQuery := `DELETE FROM activation_tokens WHERE user_id = ?`
	_, err = tx.Exec(deleteQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old activation tokens: %w", err)
	}

	// Generate new activation token
	tokenUUID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Insert new token
	insertQuery := `
		INSERT INTO activation_tokens (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`
	_, err = tx.Exec(insertQuery, tokenUUID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create new activation token: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	token := &models.ActivationToken{
		Token:     tokenUUID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return token, nil
}

func CleanupExpiredActivationTokens(db *sql.DB) error {
	query := `DELETE FROM activation_tokens WHERE expires_at < CURRENT_TIMESTAMP`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired activation tokens: %w", err)
	}
	return nil
}