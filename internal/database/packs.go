package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"

	"github.com/google/uuid"
)

func CreatePack(db *sql.DB, userID int, name string) (*models.Pack, error) {
	return CreatePackWithPublic(db, userID, name, false)
}

func CreatePackWithPublic(db *sql.DB, userID int, name string, isPublic bool) (*models.Pack, error) {
	packID := uuid.New().String()

	query := `
		INSERT INTO packs (id, user_id, name, is_public)
		VALUES (?, ?, ?, ?)
	`

	_, err := db.Exec(query, packID, userID, name, isPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack: %w", err)
	}

	pack := &models.Pack{
		ID:       packID,
		UserID:   userID,
		Name:     name,
		IsPublic: isPublic,
	}

	return pack, nil
}

func GetPacks(db *sql.DB, userID int) ([]models.Pack, error) {
	query := `
		SELECT id, user_id, name, is_public, created_at, updated_at
		FROM packs
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query packs: %w", err)
	}
	defer rows.Close()

	var packs []models.Pack
	for rows.Next() {
		var pack models.Pack
		err := rows.Scan(
			&pack.ID,
			&pack.UserID,
			&pack.Name,
			&pack.IsPublic,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack: %w", err)
		}
		packs = append(packs, pack)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating packs: %w", err)
	}

	return packs, nil
}

func GetPack(db *sql.DB, packID string) (*models.Pack, error) {
	pack := &models.Pack{}
	query := `
		SELECT id, user_id, name, is_public, created_at, updated_at
		FROM packs
		WHERE id = ?
	`

	err := db.QueryRow(query, packID).Scan(
		&pack.ID,
		&pack.UserID,
		&pack.Name,
		&pack.IsPublic,
		&pack.CreatedAt,
		&pack.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pack not found")
		}
		return nil, fmt.Errorf("failed to query pack: %w", err)
	}

	return pack, nil
}

func GetPackWithItems(db *sql.DB, packID string) (*models.Pack, error) {
	pack, err := GetPack(db, packID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT pi.id, pi.pack_id, pi.item_id, pi.is_worn, pi.count, COALESCE(pi.worn_count, 0), pi.created_at,
		       i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.price, i.created_at, i.updated_at,
		       c.id, c.name
		FROM pack_items pi
		INNER JOIN items i ON pi.item_id = i.id
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE pi.pack_id = ?
		ORDER BY c.name, i.name
	`

	rows, err := db.Query(query, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pack items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var packItem models.PackItem
		var item models.Item
		var category models.Category

		err := rows.Scan(
			&packItem.ID,
			&packItem.PackID,
			&packItem.ItemID,
			&packItem.IsWorn,
			&packItem.Count,
			&packItem.WornCount,
			&packItem.CreatedAt,
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.Price,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack item: %w", err)
		}

		item.Category = &category
		packItem.Item = &item
		pack.Items = append(pack.Items, packItem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack items: %w", err)
	}

	return pack, nil
}

func UpdatePack(db *sql.DB, userID int, packID, name string, isPublic bool) error {
	query := `
		UPDATE packs
		SET name = ?, is_public = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, name, isPublic, packID, userID)
	if err != nil {
		return fmt.Errorf("failed to update pack: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pack not found")
	}

	return nil
}

func DeletePack(db *sql.DB, userID int, packID string) error {
	query := `
		DELETE FROM packs
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, packID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete pack: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pack not found")
	}

	return nil
}

func AddItemToPack(db *sql.DB, packID string, itemID int, userID int) error {
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}

	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	_, err = GetItem(db, userID, itemID)
	if err != nil {
		return fmt.Errorf("item not found")
	}

	// Check if item already exists in pack
	var existingID int
	var currentCount int
	checkQuery := `SELECT id, count FROM pack_items WHERE pack_id = ? AND item_id = ?`
	err = db.QueryRow(checkQuery, packID, itemID).Scan(&existingID, &currentCount)
	
	if err == sql.ErrNoRows {
		// Item doesn't exist, insert new with count 1
		insertQuery := `
			INSERT INTO pack_items (pack_id, item_id, count)
			VALUES (?, ?, 1)
		`
		_, err = db.Exec(insertQuery, packID, itemID)
		if err != nil {
			return fmt.Errorf("failed to add item to pack: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing item: %w", err)
	} else {
		// Item exists, increment count
		updateQuery := `UPDATE pack_items SET count = count + 1 WHERE id = ?`
		_, err = db.Exec(updateQuery, existingID)
		if err != nil {
			return fmt.Errorf("failed to increment item count: %w", err)
		}
	}

	return nil
}

func RemoveItemFromPack(db *sql.DB, packID string, itemID, userID int) error {
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}

	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Check current count
	var currentCount int
	var packItemID int
	checkQuery := `SELECT id, count FROM pack_items WHERE pack_id = ? AND item_id = ?`
	err = db.QueryRow(checkQuery, packID, itemID).Scan(&packItemID, &currentCount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("item not found in pack")
	} else if err != nil {
		return fmt.Errorf("failed to check item count: %w", err)
	}

	if currentCount <= 1 {
		// Delete the item completely if count is 1 or less
		deleteQuery := `DELETE FROM pack_items WHERE id = ?`
		_, err = db.Exec(deleteQuery, packItemID)
		if err != nil {
			return fmt.Errorf("failed to remove item from pack: %w", err)
		}
	} else {
		// Decrement the count
		updateQuery := `UPDATE pack_items SET count = count - 1 WHERE id = ?`
		_, err = db.Exec(updateQuery, packItemID)
		if err != nil {
			return fmt.Errorf("failed to decrement item count: %w", err)
		}
	}

	return nil
}

func UpdatePackItemWornCount(db *sql.DB, packID string, itemID, userID int, wornCount int) error {
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}

	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Get current count to validate worn_count
	var currentCount int
	var packItemID int
	checkQuery := `SELECT id, count FROM pack_items WHERE pack_id = ? AND item_id = ?`
	err = db.QueryRow(checkQuery, packID, itemID).Scan(&packItemID, &currentCount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("item not found in pack")
	} else if err != nil {
		return fmt.Errorf("failed to check item: %w", err)
	}

	// Validate worn count doesn't exceed total count
	if wornCount < 0 {
		wornCount = 0
	}
	if wornCount > currentCount {
		wornCount = currentCount
	}

	// Update worn_count and is_worn flag
	isWorn := wornCount > 0
	updateQuery := `UPDATE pack_items SET worn_count = ?, is_worn = ? WHERE id = ?`
	_, err = db.Exec(updateQuery, wornCount, isWorn, packItemID)
	if err != nil {
		return fmt.Errorf("failed to update worn count: %w", err)
	}

	return nil
}

func TogglePackItemWorn(db *sql.DB, packID string, itemID, userID int, isWorn bool) error {
	pack, err := GetPack(db, packID)
	if err != nil {
		return err
	}

	if pack.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Get current count to determine worn_count
	var currentCount int
	var packItemID int
	checkQuery := `SELECT id, count FROM pack_items WHERE pack_id = ? AND item_id = ?`
	err = db.QueryRow(checkQuery, packID, itemID).Scan(&packItemID, &currentCount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("item not found in pack")
	} else if err != nil {
		return fmt.Errorf("failed to check item: %w", err)
	}

	// For checkbox behavior (count = 1), set worn_count to 0 or 1
	// For counter behavior (count > 1), this shouldn't be called, but handle gracefully
	var wornCount int
	if isWorn {
		wornCount = currentCount // Set all items as worn
	} else {
		wornCount = 0 // Set no items as worn
	}

	updateQuery := `UPDATE pack_items SET is_worn = ?, worn_count = ? WHERE id = ?`
	_, err = db.Exec(updateQuery, isWorn, wornCount, packItemID)
	if err != nil {
		return fmt.Errorf("failed to update worn status: %w", err)
	}

	return nil
}

func DuplicatePack(db *sql.DB, userID int, originalPackID string) (*models.Pack, error) {
	// Get the original pack with all its items
	originalPack, err := GetPackWithItems(db, originalPackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original pack: %w", err)
	}

	// Check if user owns the pack
	if originalPack.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Create new pack with "Copy" appended to name
	newPackName := originalPack.Name + " Copy"
	newPack, err := CreatePack(db, userID, newPackName)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicate pack: %w", err)
	}

	// Copy all items from original pack to new pack
	for _, packItem := range originalPack.Items {
		// Insert the pack item with the same count and worn_count
		insertQuery := `
			INSERT INTO pack_items (pack_id, item_id, count, worn_count, is_worn)
			VALUES (?, ?, ?, ?, ?)
		`
		_, err = db.Exec(insertQuery, newPack.ID, packItem.ItemID, packItem.Count, packItem.WornCount, packItem.IsWorn)
		if err != nil {
			// If duplication fails, clean up by deleting the created pack
			DeletePack(db, userID, newPack.ID)
			return nil, fmt.Errorf("failed to copy pack items: %w", err)
		}
	}

	return newPack, nil
}