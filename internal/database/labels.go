package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"
)

func CreatePackLabel(db *sql.DB, packID string, name, color string, userID int) (*models.PackLabel, error) {
	pack, err := GetPack(db, packID)
	if err != nil {
		return nil, err
	}

	if pack.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	query := `
		INSERT INTO pack_labels (pack_id, name, color)
		VALUES (?, ?, ?)
	`

	result, err := db.Exec(query, packID, name, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack label: %w", err)
	}

	labelID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get label ID: %w", err)
	}

	label := &models.PackLabel{
		ID:     int(labelID),
		PackID: packID,
		Name:   name,
		Color:  color,
	}

	return label, nil
}

func GetPackLabels(db *sql.DB, packID string, userID int) ([]models.PackLabel, error) {
	pack, err := GetPack(db, packID)
	if err != nil {
		return nil, err
	}

	if pack.UserID != userID && !pack.IsPublic {
		return nil, fmt.Errorf("unauthorized")
	}

	query := `
		SELECT id, pack_id, name, color, created_at, updated_at
		FROM pack_labels
		WHERE pack_id = ?
		ORDER BY name
	`

	rows, err := db.Query(query, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack labels: %w", err)
	}
	defer rows.Close()

	var labels []models.PackLabel
	for rows.Next() {
		var label models.PackLabel
		err := rows.Scan(
			&label.ID,
			&label.PackID,
			&label.Name,
			&label.Color,
			&label.CreatedAt,
			&label.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack label: %w", err)
		}
		labels = append(labels, label)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack labels: %w", err)
	}

	return labels, nil
}

func UpdatePackLabel(db *sql.DB, labelID int, name, color string, userID int) error {
	// First verify the user owns the pack this label belongs to
	checkQuery := `
		SELECT p.user_id
		FROM pack_labels pl
		JOIN packs p ON pl.pack_id = p.id
		WHERE pl.id = ?
	`
	
	var packUserID int
	err := db.QueryRow(checkQuery, labelID).Scan(&packUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label ownership: %w", err)
	}

	if packUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		UPDATE pack_labels
		SET name = ?, color = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, name, color, labelID)
	if err != nil {
		return fmt.Errorf("failed to update pack label: %w", err)
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

func DeletePackLabel(db *sql.DB, labelID int, userID int) error {
	// First verify the user owns the pack this label belongs to
	checkQuery := `
		SELECT p.user_id
		FROM pack_labels pl
		JOIN packs p ON pl.pack_id = p.id
		WHERE pl.id = ?
	`
	
	var packUserID int
	err := db.QueryRow(checkQuery, labelID).Scan(&packUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label ownership: %w", err)
	}

	if packUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `DELETE FROM pack_labels WHERE id = ?`

	result, err := db.Exec(query, labelID)
	if err != nil {
		return fmt.Errorf("failed to delete pack label: %w", err)
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

func AssignLabelToPackItem(db *sql.DB, packItemID, labelID int, userID int) error {
	// Verify user owns both the pack item and the label
	checkQuery := `
		SELECT p.user_id, p.id as pack_id
		FROM pack_items pi
		JOIN packs p ON pi.pack_id = p.id
		WHERE pi.id = ?
	`
	
	var packUserID int
	var packID string
	err := db.QueryRow(checkQuery, packItemID).Scan(&packUserID, &packID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("pack item not found")
		}
		return fmt.Errorf("failed to check pack item ownership: %w", err)
	}

	if packUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Verify the label belongs to the same pack
	labelCheckQuery := `SELECT pack_id FROM pack_labels WHERE id = ?`
	var labelPackID string
	err = db.QueryRow(labelCheckQuery, labelID).Scan(&labelPackID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label not found")
		}
		return fmt.Errorf("failed to check label: %w", err)
	}

	if labelPackID != packID {
		return fmt.Errorf("label does not belong to the same pack")
	}

	// Check if assignment already exists and increment count, or create new one
	var existingCount int
	countQuery := `SELECT COALESCE(count, 0) FROM item_labels WHERE pack_item_id = ? AND pack_label_id = ?`
	err = db.QueryRow(countQuery, packItemID, labelID).Scan(&existingCount)
	
	if err == sql.ErrNoRows {
		// Create new assignment
		query := `INSERT INTO item_labels (pack_item_id, pack_label_id, count) VALUES (?, ?, 1)`
		_, err = db.Exec(query, packItemID, labelID)
		if err != nil {
			return fmt.Errorf("failed to assign label to item: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing assignment: %w", err)
	} else {
		// Increment existing count
		query := `UPDATE item_labels SET count = count + 1 WHERE pack_item_id = ? AND pack_label_id = ?`
		_, err = db.Exec(query, packItemID, labelID)
		if err != nil {
			return fmt.Errorf("failed to increment label count: %w", err)
		}
	}

	return nil
}

func RemoveLabelFromPackItem(db *sql.DB, packItemID, labelID int, userID int) error {
	// Verify user owns the pack item
	checkQuery := `
		SELECT p.user_id
		FROM pack_items pi
		JOIN packs p ON pi.pack_id = p.id
		WHERE pi.id = ?
	`
	
	var packUserID int
	err := db.QueryRow(checkQuery, packItemID).Scan(&packUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("pack item not found")
		}
		return fmt.Errorf("failed to check pack item ownership: %w", err)
	}

	if packUserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Check current count and decrement or delete
	var currentCount int
	countQuery := `SELECT count FROM item_labels WHERE pack_item_id = ? AND pack_label_id = ?`
	err = db.QueryRow(countQuery, packItemID, labelID).Scan(&currentCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("label assignment not found")
		}
		return fmt.Errorf("failed to check label count: %w", err)
	}

	if currentCount <= 1 {
		// Delete the assignment if count is 1 or less
		query := `DELETE FROM item_labels WHERE pack_item_id = ? AND pack_label_id = ?`
		_, err = db.Exec(query, packItemID, labelID)
		if err != nil {
			return fmt.Errorf("failed to remove label from item: %w", err)
		}
	} else {
		// Decrement the count
		query := `UPDATE item_labels SET count = count - 1 WHERE pack_item_id = ? AND pack_label_id = ?`
		_, err = db.Exec(query, packItemID, labelID)
		if err != nil {
			return fmt.Errorf("failed to decrement label count: %w", err)
		}
	}

	return nil
}

func GetPackItemLabels(db *sql.DB, packItemID int) ([]models.ItemLabel, error) {
	query := `
		SELECT il.id, il.pack_item_id, il.pack_label_id, il.count, il.created_at,
		       pl.id, pl.pack_id, pl.name, pl.color, pl.created_at, pl.updated_at
		FROM item_labels il
		JOIN pack_labels pl ON il.pack_label_id = pl.id
		WHERE il.pack_item_id = ?
		ORDER BY pl.name
	`

	rows, err := db.Query(query, packItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack item labels: %w", err)
	}
	defer rows.Close()

	var itemLabels []models.ItemLabel
	for rows.Next() {
		var itemLabel models.ItemLabel
		var packLabel models.PackLabel
		err := rows.Scan(
			&itemLabel.ID,
			&itemLabel.PackItemID,
			&itemLabel.PackLabelID,
			&itemLabel.Count,
			&itemLabel.CreatedAt,
			&packLabel.ID,
			&packLabel.PackID,
			&packLabel.Name,
			&packLabel.Color,
			&packLabel.CreatedAt,
			&packLabel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack item label: %w", err)
		}
		
		itemLabel.PackLabel = &packLabel
		itemLabels = append(itemLabels, itemLabel)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack item labels: %w", err)
	}

	return itemLabels, nil
}