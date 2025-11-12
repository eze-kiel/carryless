package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"

	"carryless/internal/logger"
	"carryless/internal/models"

	"github.com/google/uuid"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Helper function to update pack timestamp when items are modified
func updatePackTimestamp(db *sql.DB, packID string) error {
	query := `UPDATE packs SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, packID)
	return err
}

func generateShortID(db *sql.DB) (string, error) {
	const idLength = 8
	const maxRetries = 10

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Generate random 8-character ID
		b := make([]byte, idLength)
		for i := range b {
			num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			if err != nil {
				return "", fmt.Errorf("failed to generate random number: %w", err)
			}
			b[i] = charset[num.Int64()]
		}
		
		shortID := string(b)
		
		// Check if this ID already exists
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM packs WHERE short_id = ?)", shortID).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("failed to check short ID existence: %w", err)
		}
		
		if !exists {
			return shortID, nil
		}
	}
	
	return "", fmt.Errorf("failed to generate unique short ID after %d attempts", maxRetries)
}

func CreatePack(db *sql.DB, userID int, name string) (*models.Pack, error) {
	return CreatePackWithPublic(db, userID, name, false)
}

func createPackWithTx(tx *sql.Tx, userID int, name string) (*models.Pack, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO packs (id, user_id, name, note, is_public)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(query, id, userID, name, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack: %w", err)
	}

	pack := &models.Pack{
		ID:       id,
		UserID:   userID,
		Name:     name,
		IsPublic: false,
	}

	return pack, nil
}

func CreatePackWithPublic(db *sql.DB, userID int, name string, isPublic bool) (*models.Pack, error) {
	packID := uuid.New().String()
	
	var shortID sql.NullString
	if isPublic {
		shortIDValue, err := generateShortID(db)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short ID: %w", err)
		}
		shortID = sql.NullString{String: shortIDValue, Valid: true}
	}

	query := `
		INSERT INTO packs (id, user_id, name, note, is_public, short_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, packID, userID, name, "", isPublic, shortID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pack: %w", err)
	}

	pack := &models.Pack{
		ID:       packID,
		UserID:   userID,
		Name:     name,
		IsPublic: isPublic,
		ShortID:  shortID.String,
	}

	return pack, nil
}

func GetPacks(db *sql.DB, userID int) ([]models.Pack, error) {
	query := `
		SELECT id, user_id, name, COALESCE(note, ''), is_public, COALESCE(is_locked, FALSE), COALESCE(short_id, ''), created_at, updated_at
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
			&pack.Note,
			&pack.IsPublic,
			&pack.IsLocked,
			&pack.ShortID,
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
		SELECT id, user_id, name, COALESCE(note, ''), is_public, COALESCE(is_locked, FALSE), COALESCE(short_id, ''), created_at, updated_at
		FROM packs
		WHERE id = ?
	`

	err := db.QueryRow(query, packID).Scan(
		&pack.ID,
		&pack.UserID,
		&pack.Name,
		&pack.Note,
		&pack.IsPublic,
		&pack.IsLocked,
		&pack.ShortID,
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

func GetPackByShortID(db *sql.DB, shortID string) (*models.Pack, error) {
	pack := &models.Pack{}
	query := `
		SELECT id, user_id, name, COALESCE(note, ''), is_public, COALESCE(is_locked, FALSE), COALESCE(short_id, ''), created_at, updated_at
		FROM packs
		WHERE short_id = ?
	`

	err := db.QueryRow(query, shortID).Scan(
		&pack.ID,
		&pack.UserID,
		&pack.Name,
		&pack.Note,
		&pack.IsPublic,
		&pack.IsLocked,
		&pack.ShortID,
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

	// Get pack labels
	labels, err := GetPackLabels(db, packID, pack.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pack labels: %w", err)
	}
	pack.Labels = labels

	query := `
		SELECT pi.id, pi.pack_id, pi.item_id, pi.is_worn, pi.count, COALESCE(pi.worn_count, 0), pi.created_at,
		       i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.weight_to_verify, i.price, i.created_at, i.updated_at,
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
			&item.WeightToVerify,
			&item.Price,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack item: %w", err)
		}

		// Get labels for this pack item
		itemLabels, err := GetPackItemLabels(db, packItem.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get pack item labels: %w", err)
		}

		item.Category = &category
		packItem.Item = &item
		packItem.Labels = itemLabels
		pack.Items = append(pack.Items, packItem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack items: %w", err)
	}

	return pack, nil
}

func UpdatePack(db *sql.DB, userID int, packID, name string, isPublic bool) error {
	// First, get the current pack to check if it's being made public and needs a short ID
	currentPack, err := GetPack(db, packID)
	if err != nil {
		return fmt.Errorf("failed to get current pack: %w", err)
	}
	
	if currentPack.UserID != userID {
		return fmt.Errorf("pack not found")
	}

	// Generate short ID if pack is being made public and doesn't have one
	var shortIDToSet sql.NullString
	if isPublic && currentPack.ShortID == "" {
		shortIDValue, err := generateShortID(db)
		if err != nil {
			return fmt.Errorf("failed to generate short ID: %w", err)
		}
		shortIDToSet = sql.NullString{String: shortIDValue, Valid: true}
	} else if currentPack.ShortID != "" {
		shortIDToSet = sql.NullString{String: currentPack.ShortID, Valid: true}
	}

	query := `
		UPDATE packs
		SET name = ?, is_public = ?, short_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, name, isPublic, shortIDToSet, packID, userID)
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

func UpdatePackNote(db *sql.DB, userID int, packID, note string) error {
	query := `
		UPDATE packs
		SET note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, note, packID, userID)
	if err != nil {
		return fmt.Errorf("failed to update pack note: %w", err)
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

	// Update pack timestamp since items were modified
	if err := updatePackTimestamp(db, packID); err != nil {
		return fmt.Errorf("failed to update pack timestamp: %w", err)
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

	// Update pack timestamp since items were modified
	if err := updatePackTimestamp(db, packID); err != nil {
		return fmt.Errorf("failed to update pack timestamp: %w", err)
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

	// Update pack timestamp since items were modified
	if err := updatePackTimestamp(db, packID); err != nil {
		return fmt.Errorf("failed to update pack timestamp: %w", err)
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

	// Update pack timestamp since items were modified
	if err := updatePackTimestamp(db, packID); err != nil {
		return fmt.Errorf("failed to update pack timestamp: %w", err)
	}

	return nil
}

func TogglePackLock(db *sql.DB, userID int, packID string, isLocked bool) error {
	query := `
		UPDATE packs
		SET is_locked = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, isLocked, packID, userID)
	if err != nil {
		return fmt.Errorf("failed to update pack lock status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pack not found or unauthorized")
	}

	return nil
}

func DuplicatePack(db *sql.DB, userID int, originalPackID string) (*models.Pack, error) {
	logger.Debug("Starting pack duplication",
		"user_id", userID,
		"original_pack_id", originalPackID)
	
	// Start a database transaction
	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	logger.Debug("Started database transaction")
	
	// Defer rollback in case of error
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic occurred, rolling back transaction", "panic", r)
			tx.Rollback()
			panic(r)
		}
	}()

	// Get the original pack with all its items
	originalPack, err := GetPackWithItems(db, originalPackID)
	if err != nil {
		logger.Error("Failed to get original pack",
			"pack_id", originalPackID,
			"error", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to get original pack: %w", err)
	}
	logger.Debug("Retrieved original pack",
		"pack_name", originalPack.Name,
		"item_count", len(originalPack.Items))

	// Check if user owns the pack
	if originalPack.UserID != userID {
		logger.Warn("Unauthorized pack duplication attempt",
			"user_id", userID,
			"pack_owner_id", originalPack.UserID)
		tx.Rollback()
		return nil, fmt.Errorf("unauthorized")
	}

	// Create new pack with "Copy" appended to name
	newPackName := originalPack.Name + " Copy"
	logger.Debug("Creating new pack", "pack_name", newPackName)
	newPack, err := createPackWithTx(tx, userID, newPackName)
	if err != nil {
		logger.Error("Failed to create duplicate pack", "error", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to create duplicate pack: %w", err)
	}
	logger.Info("Created new pack", "pack_id", newPack.ID)

	// Copy all labels from original pack to new pack
	// First, get all labels from the original pack
	logger.Debug("Starting to copy labels", "pack_id", originalPack.ID)
	getLabelsQuery := `SELECT id, name, color FROM pack_labels WHERE pack_id = ?`
	labelRows, err := tx.Query(getLabelsQuery, originalPack.ID)
	if err != nil {
		logger.Error("Failed to query pack labels", "error", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to get pack labels: %w", err)
	}
	defer labelRows.Close()

	// Map old label IDs to new label IDs
	labelIDMap := make(map[int]int)
	labelCount := 0
	for labelRows.Next() {
		var oldLabelID int
		var name, color string
		err := labelRows.Scan(&oldLabelID, &name, &color)
		if err != nil {
			logger.Error("Failed to scan pack label", "error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to scan pack label: %w", err)
		}

		// Create new label in the new pack
		insertLabelQuery := `INSERT INTO pack_labels (pack_id, name, color) VALUES (?, ?, ?)`
		result, err := tx.Exec(insertLabelQuery, newPack.ID, name, color)
		if err != nil {
			logger.Error("Failed to insert label",
				"label_name", name,
				"error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy pack label: %w", err)
		}

		newLabelID, err := result.LastInsertId()
		if err != nil {
			logger.Error("Failed to get new label ID", "error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to get new label ID: %w", err)
		}

		labelIDMap[oldLabelID] = int(newLabelID)
		labelCount++
		logger.Debug("Copied label",
			"label_name", name,
			"color", color,
			"old_id", oldLabelID,
			"new_id", int(newLabelID))
	}
	logger.Info("Successfully copied labels", "count", labelCount)

	// Map old pack item IDs to new pack item IDs
	packItemIDMap := make(map[int]int)

	// Copy all items from original pack to new pack
	logger.Debug("Starting to copy pack items", "count", len(originalPack.Items))
	for i, packItem := range originalPack.Items {
		logger.Debug("Copying pack item",
			"index", i+1,
			"total", len(originalPack.Items),
			"item_id", packItem.ItemID,
			"count", packItem.Count,
			"worn_count", packItem.WornCount)

		// Insert the pack item with the same count and worn_count
		insertQuery := `
			INSERT INTO pack_items (pack_id, item_id, count, worn_count, is_worn)
			VALUES (?, ?, ?, ?, ?)
		`
		result, err := tx.Exec(insertQuery, newPack.ID, packItem.ItemID, packItem.Count, packItem.WornCount, packItem.IsWorn)
		if err != nil {
			logger.Error("Failed to copy pack item",
				"item_id", packItem.ItemID,
				"error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy pack items: %w", err)
		}

		newPackItemID, err := result.LastInsertId()
		if err != nil {
			logger.Error("Failed to get new pack item ID", "error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to get new pack item ID: %w", err)
		}

		packItemIDMap[packItem.ID] = int(newPackItemID)
		logger.Debug("Mapped pack item",
			"old_id", packItem.ID,
			"new_id", int(newPackItemID))
	}
	logger.Info("Successfully copied all pack items")

	// Copy item label assignments
	logger.Debug("Starting to copy item label assignments")
	getItemLabelsQuery := `
		SELECT il.pack_item_id, il.pack_label_id, il.count
		FROM item_labels il
		JOIN pack_items pi ON il.pack_item_id = pi.id
		WHERE pi.pack_id = ?
	`
	itemLabelRows, err := tx.Query(getItemLabelsQuery, originalPack.ID)
	if err != nil {
		logger.Error("Failed to query item labels", "error", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to get item labels: %w", err)
	}
	defer itemLabelRows.Close()

	assignmentCount := 0
	for itemLabelRows.Next() {
		var oldPackItemID, oldLabelID, count int
		err := itemLabelRows.Scan(&oldPackItemID, &oldLabelID, &count)
		if err != nil {
			logger.Error("Failed to scan item label", "error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to scan item label: %w", err)
		}

		// Get the new IDs from our maps
		newPackItemID, exists := packItemIDMap[oldPackItemID]
		if !exists {
			logger.Warn("Pack item ID not found in mapping, skipping label assignment",
				"pack_item_id", oldPackItemID)
			continue // Skip if pack item doesn't exist (shouldn't happen)
		}

		newLabelID, exists := labelIDMap[oldLabelID]
		if !exists {
			logger.Warn("Label ID not found in mapping, skipping label assignment",
				"label_id", oldLabelID)
			continue // Skip if label doesn't exist (shouldn't happen)
		}

		// Insert the item label assignment
		insertItemLabelQuery := `INSERT INTO item_labels (pack_item_id, pack_label_id, count) VALUES (?, ?, ?)`
		_, err = tx.Exec(insertItemLabelQuery, newPackItemID, newLabelID, count)
		if err != nil {
			logger.Error("Failed to copy item label assignment",
				"old_pack_item_id", oldPackItemID,
				"new_pack_item_id", newPackItemID,
				"old_label_id", oldLabelID,
				"new_label_id", newLabelID,
				"count", count,
				"error", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy item label assignment: %w", err)
		}

		assignmentCount++
		logger.Debug("Copied item label assignment",
			"old_pack_item_id", oldPackItemID,
			"new_pack_item_id", newPackItemID,
			"old_label_id", oldLabelID,
			"new_label_id", newLabelID,
			"count", count)
	}
	logger.Info("Successfully copied item label assignments", "count", assignmentCount)

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		logger.Error("Failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	logger.Debug("Transaction committed successfully")
	logger.Info("Pack duplication completed successfully",
		"new_pack_id", newPack.ID,
		"pack_name", newPack.Name)

	return newPack, nil
}