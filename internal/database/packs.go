package database

import (
	"database/sql"
	"fmt"
	"log"

	"carryless/internal/models"

	"github.com/google/uuid"
)

func CreatePack(db *sql.DB, userID int, name string) (*models.Pack, error) {
	return CreatePackWithPublic(db, userID, name, false)
}

func createPackWithTx(tx *sql.Tx, userID int, name string) (*models.Pack, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO packs (id, user_id, name, is_public)
		VALUES (?, ?, ?, ?)
	`

	_, err := tx.Exec(query, id, userID, name, false)
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
	log.Printf("[DUPLICATE] Starting pack duplication - UserID: %d, OriginalPackID: %s", userID, originalPackID)
	
	// Start a database transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[DUPLICATE] Failed to begin transaction: %v", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	log.Printf("[DUPLICATE] Started database transaction")
	
	// Defer rollback in case of error
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DUPLICATE] Panic occurred, rolling back transaction: %v", r)
			tx.Rollback()
			panic(r)
		}
	}()

	// Get the original pack with all its items
	originalPack, err := GetPackWithItems(db, originalPackID)
	if err != nil {
		log.Printf("[DUPLICATE] Failed to get original pack %s: %v", originalPackID, err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to get original pack: %w", err)
	}
	log.Printf("[DUPLICATE] Retrieved original pack '%s' with %d items", originalPack.Name, len(originalPack.Items))

	// Check if user owns the pack
	if originalPack.UserID != userID {
		log.Printf("[DUPLICATE] Unauthorized access attempt - UserID: %d, PackOwnerID: %d", userID, originalPack.UserID)
		tx.Rollback()
		return nil, fmt.Errorf("unauthorized")
	}

	// Create new pack with "Copy" appended to name
	newPackName := originalPack.Name + " Copy"
	log.Printf("[DUPLICATE] Creating new pack with name: '%s'", newPackName)
	newPack, err := createPackWithTx(tx, userID, newPackName)
	if err != nil {
		log.Printf("[DUPLICATE] Failed to create duplicate pack: %v", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to create duplicate pack: %w", err)
	}
	log.Printf("[DUPLICATE] Created new pack with ID: %s", newPack.ID)

	// Copy all labels from original pack to new pack
	// First, get all labels from the original pack
	log.Printf("[DUPLICATE] Starting to copy labels from pack %s", originalPack.ID)
	getLabelsQuery := `SELECT id, name, color FROM pack_labels WHERE pack_id = ?`
	labelRows, err := tx.Query(getLabelsQuery, originalPack.ID)
	if err != nil {
		log.Printf("[DUPLICATE] Failed to query pack labels: %v", err)
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
			log.Printf("[DUPLICATE] Failed to scan pack label: %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to scan pack label: %w", err)
		}

		// Create new label in the new pack
		insertLabelQuery := `INSERT INTO pack_labels (pack_id, name, color) VALUES (?, ?, ?)`
		result, err := tx.Exec(insertLabelQuery, newPack.ID, name, color)
		if err != nil {
			log.Printf("[DUPLICATE] Failed to insert label '%s': %v", name, err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy pack label: %w", err)
		}

		newLabelID, err := result.LastInsertId()
		if err != nil {
			log.Printf("[DUPLICATE] Failed to get new label ID: %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to get new label ID: %w", err)
		}

		labelIDMap[oldLabelID] = int(newLabelID)
		labelCount++
		log.Printf("[DUPLICATE] Copied label '%s' (color: %s) - OldID: %d -> NewID: %d", name, color, oldLabelID, int(newLabelID))
	}
	log.Printf("[DUPLICATE] Successfully copied %d labels", labelCount)

	// Map old pack item IDs to new pack item IDs
	packItemIDMap := make(map[int]int)

	// Copy all items from original pack to new pack
	log.Printf("[DUPLICATE] Starting to copy %d pack items", len(originalPack.Items))
	for i, packItem := range originalPack.Items {
		log.Printf("[DUPLICATE] Copying pack item %d/%d - ItemID: %d, Count: %d, WornCount: %d", i+1, len(originalPack.Items), packItem.ItemID, packItem.Count, packItem.WornCount)
		
		// Insert the pack item with the same count and worn_count
		insertQuery := `
			INSERT INTO pack_items (pack_id, item_id, count, worn_count, is_worn)
			VALUES (?, ?, ?, ?, ?)
		`
		result, err := tx.Exec(insertQuery, newPack.ID, packItem.ItemID, packItem.Count, packItem.WornCount, packItem.IsWorn)
		if err != nil {
			log.Printf("[DUPLICATE] Failed to copy pack item (ItemID: %d): %v", packItem.ItemID, err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy pack items: %w", err)
		}

		newPackItemID, err := result.LastInsertId()
		if err != nil {
			log.Printf("[DUPLICATE] Failed to get new pack item ID: %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to get new pack item ID: %w", err)
		}

		packItemIDMap[packItem.ID] = int(newPackItemID)
		log.Printf("[DUPLICATE] Mapped pack item - OldID: %d -> NewID: %d", packItem.ID, int(newPackItemID))
	}
	log.Printf("[DUPLICATE] Successfully copied all pack items")

	// Copy item label assignments
	log.Printf("[DUPLICATE] Starting to copy item label assignments")
	getItemLabelsQuery := `
		SELECT il.pack_item_id, il.pack_label_id, il.count 
		FROM item_labels il
		JOIN pack_items pi ON il.pack_item_id = pi.id
		WHERE pi.pack_id = ?
	`
	itemLabelRows, err := tx.Query(getItemLabelsQuery, originalPack.ID)
	if err != nil {
		log.Printf("[DUPLICATE] Failed to query item labels: %v", err)
		tx.Rollback()
		return nil, fmt.Errorf("failed to get item labels: %w", err)
	}
	defer itemLabelRows.Close()

	assignmentCount := 0
	for itemLabelRows.Next() {
		var oldPackItemID, oldLabelID, count int
		err := itemLabelRows.Scan(&oldPackItemID, &oldLabelID, &count)
		if err != nil {
			log.Printf("[DUPLICATE] Failed to scan item label: %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to scan item label: %w", err)
		}

		// Get the new IDs from our maps
		newPackItemID, exists := packItemIDMap[oldPackItemID]
		if !exists {
			log.Printf("[DUPLICATE] Warning: Pack item ID %d not found in mapping, skipping label assignment", oldPackItemID)
			continue // Skip if pack item doesn't exist (shouldn't happen)
		}

		newLabelID, exists := labelIDMap[oldLabelID]
		if !exists {
			log.Printf("[DUPLICATE] Warning: Label ID %d not found in mapping, skipping label assignment", oldLabelID)
			continue // Skip if label doesn't exist (shouldn't happen)
		}

		// Insert the item label assignment
		insertItemLabelQuery := `INSERT INTO item_labels (pack_item_id, pack_label_id, count) VALUES (?, ?, ?)`
		_, err = tx.Exec(insertItemLabelQuery, newPackItemID, newLabelID, count)
		if err != nil {
			log.Printf("[DUPLICATE] Failed to copy item label assignment (PackItemID: %d -> %d, LabelID: %d -> %d, Count: %d): %v", oldPackItemID, newPackItemID, oldLabelID, newLabelID, count, err)
			tx.Rollback()
			return nil, fmt.Errorf("failed to copy item label assignment: %w", err)
		}

		assignmentCount++
		log.Printf("[DUPLICATE] Copied item label assignment - PackItemID: %d -> %d, LabelID: %d -> %d, Count: %d", oldPackItemID, newPackItemID, oldLabelID, newLabelID, count)
	}
	log.Printf("[DUPLICATE] Successfully copied %d item label assignments", assignmentCount)

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("[DUPLICATE] Failed to commit transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Printf("[DUPLICATE] Transaction committed successfully")
	log.Printf("[DUPLICATE] Pack duplication completed successfully - New Pack ID: %s, Name: '%s'", newPack.ID, newPack.Name)

	return newPack, nil
}