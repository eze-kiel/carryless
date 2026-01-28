package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"
)

// CreateUserPackLabel creates a new user-scoped pack label
func CreateUserPackLabel(db *sql.DB, userID int, name, color string) (*models.UserPackLabel, error) {
	query := `
		INSERT INTO user_pack_labels (user_id, name, color)
		VALUES (?, ?, ?)
	`

	result, err := db.Exec(query, userID, name, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create user pack label: %w", err)
	}

	labelID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get label ID: %w", err)
	}

	label := &models.UserPackLabel{
		ID:     int(labelID),
		UserID: userID,
		Name:   name,
		Color:  color,
	}

	return label, nil
}

// GetUserPackLabels returns all pack labels for a user
func GetUserPackLabels(db *sql.DB, userID int) ([]models.UserPackLabel, error) {
	query := `
		SELECT id, user_id, name, color, created_at, updated_at
		FROM user_pack_labels
		WHERE user_id = ?
		ORDER BY name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user pack labels: %w", err)
	}
	defer rows.Close()

	var labels []models.UserPackLabel
	for rows.Next() {
		var label models.UserPackLabel
		err := rows.Scan(
			&label.ID,
			&label.UserID,
			&label.Name,
			&label.Color,
			&label.CreatedAt,
			&label.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user pack label: %w", err)
		}
		labels = append(labels, label)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user pack labels: %w", err)
	}

	return labels, nil
}

// UpdateUserPackLabel updates an existing user pack label
func UpdateUserPackLabel(db *sql.DB, labelID int, name, color string, userID int) error {
	// First verify the user owns this label
	checkQuery := `SELECT user_id FROM user_pack_labels WHERE id = ?`
	var labelUserID int
	err := db.QueryRow(checkQuery, labelID).Scan(&labelUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label ownership: %w", err)
	}

	if labelUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		UPDATE user_pack_labels
		SET name = ?, color = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, name, color, labelID)
	if err != nil {
		return fmt.Errorf("failed to update user pack label: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("label not found")
	}

	return nil
}

// DeleteUserPackLabel deletes a user pack label and all its assignments
func DeleteUserPackLabel(db *sql.DB, labelID int, userID int) error {
	// First verify the user owns this label
	checkQuery := `SELECT user_id FROM user_pack_labels WHERE id = ?`
	var labelUserID int
	err := db.QueryRow(checkQuery, labelID).Scan(&labelUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label ownership: %w", err)
	}

	if labelUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Delete will cascade to pack_label_assignments due to foreign key
	query := `DELETE FROM user_pack_labels WHERE id = ?`
	result, err := db.Exec(query, labelID)
	if err != nil {
		return fmt.Errorf("failed to delete user pack label: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("label not found")
	}

	return nil
}

// AssignLabelToPack assigns a user pack label to a pack
func AssignLabelToPack(db *sql.DB, packID string, labelID int, userID int) error {
	// Verify user owns the pack
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}
	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Verify user owns the label
	checkQuery := `SELECT user_id FROM user_pack_labels WHERE id = ?`
	var labelUserID int
	err = db.QueryRow(checkQuery, labelID).Scan(&labelUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label ownership: %w", err)
	}

	if labelUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		INSERT INTO pack_label_assignments (pack_id, user_pack_label_id)
		VALUES (?, ?)
	`

	_, err = db.Exec(query, packID, labelID)
	if err != nil {
		return fmt.Errorf("failed to assign label to pack: %w", err)
	}

	return nil
}

// RemoveLabelFromPack removes a label assignment from a pack
func RemoveLabelFromPack(db *sql.DB, packID string, labelID int, userID int) error {
	// Verify user owns the pack
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}
	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		DELETE FROM pack_label_assignments
		WHERE pack_id = ? AND user_pack_label_id = ?
	`

	result, err := db.Exec(query, packID, labelID)
	if err != nil {
		return fmt.Errorf("failed to remove label from pack: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("label assignment not found")
	}

	return nil
}

// GetPackLevelLabels returns all labels assigned to a specific pack
func GetPackLevelLabels(db *sql.DB, packID string) ([]models.UserPackLabel, error) {
	query := `
		SELECT upl.id, upl.user_id, upl.name, upl.color, upl.created_at, upl.updated_at
		FROM user_pack_labels upl
		JOIN pack_label_assignments pla ON upl.id = pla.user_pack_label_id
		WHERE pla.pack_id = ?
		ORDER BY upl.name
	`

	rows, err := db.Query(query, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack level labels: %w", err)
	}
	defer rows.Close()

	var labels []models.UserPackLabel
	for rows.Next() {
		var label models.UserPackLabel
		err := rows.Scan(
			&label.ID,
			&label.UserID,
			&label.Name,
			&label.Color,
			&label.CreatedAt,
			&label.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack level label: %w", err)
		}
		labels = append(labels, label)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack level labels: %w", err)
	}

	return labels, nil
}
