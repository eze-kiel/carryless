package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"carryless/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func CreateUser(db *sql.DB, username, email, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES (?, ?, ?)
	`

	result, err := db.Exec(query, username, email, string(hashedPassword))
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
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return user, nil
}

func AuthenticateUser(db *sql.DB, email, password string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	err := db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
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

func CreateSession(db *sql.DB, userID int) (*models.Session, error) {
	sessionID, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)

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

func ValidateSession(db *sql.DB, sessionID string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT u.id, u.username, u.email, COALESCE(u.currency, '$'), u.created_at, u.updated_at
		FROM users u
		INNER JOIN sessions s ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > CURRENT_TIMESTAMP
	`

	err := db.QueryRow(query, sessionID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Currency,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to validate session: %w", err)
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