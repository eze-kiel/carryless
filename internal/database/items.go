package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"carryless/internal/models"
)

func CreateItem(db *sql.DB, userID int, item models.Item) (*models.Item, error) {
	query := `
		INSERT INTO items (user_id, category_id, name, note, weight_grams, weight_to_verify, price, brand, model, purchase_date, capacity, capacity_unit, link)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, userID, item.CategoryID, item.Name, item.Note, item.WeightGrams, item.WeightToVerify, item.Price,
		item.Brand, item.Model, item.PurchaseDate, item.Capacity, item.CapacityUnit, item.Link)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get item ID: %w", err)
	}

	item.ID = int(id)
	item.UserID = userID
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	return &item, nil
}

func GetItems(db *sql.DB, userID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.user_id = ?
		ORDER BY c.name, i.name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Convert nullable fields to pointer types
		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	return items, nil
}

func GetItem(db *sql.DB, userID, itemID int) (*models.Item, error) {
	item := &models.Item{}
	category := &models.Category{}
	var brand, model, capacityUnit, link sql.NullString
	var purchaseDate sql.NullTime
	var capacity sql.NullFloat64

	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.id = ? AND i.user_id = ?
	`

	err := db.QueryRow(query, itemID, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.CategoryID,
		&item.Name,
		&item.Note,
		&item.WeightGrams,
		&item.WeightToVerify,
		&item.Price,
		&brand,
		&model,
		&purchaseDate,
		&capacity,
		&capacityUnit,
		&link,
		&item.CreatedAt,
		&item.UpdatedAt,
		&category.ID,
		&category.Name,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("failed to query item: %w", err)
	}

	// Convert nullable fields to pointer types
	if brand.Valid {
		item.Brand = &brand.String
	}
	if model.Valid {
		item.Model = &model.String
	}
	if purchaseDate.Valid {
		item.PurchaseDate = &purchaseDate.Time
	}
	if capacity.Valid {
		item.Capacity = &capacity.Float64
	}
	if capacityUnit.Valid {
		item.CapacityUnit = &capacityUnit.String
	}
	if link.Valid {
		item.Link = &link.String
	}

	item.Category = category
	return item, nil
}

func UpdateItem(db *sql.DB, userID, itemID int, updatedItem models.Item) error {
	query := `
		UPDATE items
		SET category_id = ?, name = ?, note = ?, weight_grams = ?, weight_to_verify = ?, price = ?,
		    brand = ?, model = ?, purchase_date = ?, capacity = ?, capacity_unit = ?, link = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, updatedItem.CategoryID, updatedItem.Name, updatedItem.Note, updatedItem.WeightGrams, updatedItem.WeightToVerify, updatedItem.Price,
		updatedItem.Brand, updatedItem.Model, updatedItem.PurchaseDate, updatedItem.Capacity, updatedItem.CapacityUnit, updatedItem.Link,
		itemID, userID)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found")
	}

	return nil
}

func DeleteItem(db *sql.DB, userID, itemID int) error {
	return DeleteItemWithForce(db, userID, itemID, false)
}

func DeleteItemWithForce(db *sql.DB, userID, itemID int, force bool) error {
	var packCount int
	countQuery := `SELECT COUNT(*) FROM pack_items WHERE item_id = ?`
	err := db.QueryRow(countQuery, itemID).Scan(&packCount)
	if err != nil {
		return fmt.Errorf("failed to check item usage in packs: %w", err)
	}

	if packCount > 0 && !force {
		return fmt.Errorf("cannot delete item used in %d pack(s)", packCount)
	}

	// If force is true and item is in packs, remove it from all packs first
	if force && packCount > 0 {
		removeQuery := `DELETE FROM pack_items WHERE item_id = ?`
		_, err := db.Exec(removeQuery, itemID)
		if err != nil {
			return fmt.Errorf("failed to remove item from packs: %w", err)
		}
	}

	query := `
		DELETE FROM items
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, itemID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found")
	}

	return nil
}

func GetPacksUsingItem(db *sql.DB, userID, itemID int) ([]string, error) {
	query := `
		SELECT p.name 
		FROM packs p 
		JOIN pack_items pi ON p.id = pi.pack_id 
		WHERE pi.item_id = ? AND p.user_id = ?
		ORDER BY p.name
	`
	
	rows, err := db.Query(query, itemID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query packs using item: %w", err)
	}
	defer rows.Close()

	var packNames []string
	for rows.Next() {
		var packName string
		if err := rows.Scan(&packName); err != nil {
			return nil, fmt.Errorf("failed to scan pack name: %w", err)
		}
		packNames = append(packNames, packName)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pack names: %w", err)
	}

	return packNames, nil
}

func GetItemsByCategory(db *sql.DB, userID, categoryID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.user_id = ? AND i.category_id = ?
		ORDER BY i.name
	`

	rows, err := db.Query(query, userID, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items by category: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Convert nullable fields to pointer types
		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	return items, nil
}

func DeleteAllItems(db *sql.DB, userID int) error {
	// First, delete all pack items that reference the user's items
	deletePackItemsQuery := `
		DELETE FROM pack_items 
		WHERE item_id IN (SELECT id FROM items WHERE user_id = ?)
	`
	_, err := db.Exec(deletePackItemsQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to delete pack items: %w", err)
	}

	// Then delete all items for the user
	deleteItemsQuery := `DELETE FROM items WHERE user_id = ?`
	_, err = db.Exec(deleteItemsQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to delete items: %w", err)
	}

	return nil
}

func GetItemsToVerify(db *sql.DB, userID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.weight_to_verify, i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.user_id = ? AND i.weight_to_verify = true
		ORDER BY c.name, i.name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items to verify: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Convert nullable fields to pointer types
		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items to verify: %w", err)
	}

	return items, nil
}

func GetItemsWithEmptyBrand(db *sql.DB, userID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.user_id = ? AND (i.brand IS NULL OR i.brand = '')
		ORDER BY c.name, i.name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items with empty brand: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Convert nullable fields to pointer types
		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items with empty brand: %w", err)
	}

	return items, nil
}

func GetItemsWithEmptyModel(db *sql.DB, userID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.user_id = ? AND (i.model IS NULL OR i.model = '')
		ORDER BY c.name, i.name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items with empty model: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Convert nullable fields to pointer types
		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items with empty model: %w", err)
	}

	return items, nil
}

// GetItemsWithFilters returns items matching the specified filter criteria.
// Multiple filters can be combined (AND logic).
func GetItemsWithFilters(db *sql.DB, userID int, verifyOnly, emptyBrand, emptyModel bool) ([]models.Item, error) {
	// Build WHERE clause dynamically
	conditions := []string{"i.user_id = ?"}
	args := []interface{}{userID}

	if verifyOnly {
		conditions = append(conditions, "COALESCE(i.weight_to_verify, false) = true")
	}
	if emptyBrand {
		conditions = append(conditions, "(i.brand IS NULL OR i.brand = '')")
	}
	if emptyModel {
		conditions = append(conditions, "(i.model IS NULL OR i.model = '')")
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.model, i.purchase_date, i.capacity, i.capacity_unit, i.link,
		       i.created_at, i.updated_at,
		       c.id, c.name
		FROM items i
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE %s
		ORDER BY c.name, i.name
	`, whereClause)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items with filters: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var category models.Category
		var brand, model, capacityUnit, link sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.Name,
			&item.Note,
			&item.WeightGrams,
			&item.WeightToVerify,
			&item.Price,
			&brand,
			&model,
			&purchaseDate,
			&capacity,
			&capacityUnit,
			&link,
			&item.CreatedAt,
			&item.UpdatedAt,
			&category.ID,
			&category.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		if brand.Valid {
			item.Brand = &brand.String
		}
		if model.Valid {
			item.Model = &model.String
		}
		if purchaseDate.Valid {
			item.PurchaseDate = &purchaseDate.Time
		}
		if capacity.Valid {
			item.Capacity = &capacity.Float64
		}
		if capacityUnit.Valid {
			item.CapacityUnit = &capacityUnit.String
		}
		if link.Valid {
			item.Link = &link.String
		}

		item.Category = &category
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items with filters: %w", err)
	}

	return items, nil
}

// DuplicateItem creates a copy of an item with "(duplicate)" appended to the name.
// If a duplicate already exists, it will be named "(duplicate 2)", "(duplicate 3)", etc.
func DuplicateItem(db *sql.DB, userID, itemID int) (*models.Item, error) {
	// Get the original item
	original, err := GetItem(db, userID, itemID)
	if err != nil {
		return nil, err
	}

	// Generate the new name
	baseName := original.Name
	newName := generateDuplicateName(db, userID, baseName)

	// Create the duplicate
	duplicate := models.Item{
		CategoryID:     original.CategoryID,
		Name:           newName,
		Note:           original.Note,
		WeightGrams:    original.WeightGrams,
		WeightToVerify: original.WeightToVerify,
		Price:          original.Price,
		Brand:          original.Brand,
		Model:          original.Model,
		PurchaseDate:   original.PurchaseDate,
		Capacity:       original.Capacity,
		CapacityUnit:   original.CapacityUnit,
		Link:           original.Link,
	}

	return CreateItem(db, userID, duplicate)
}

// generateDuplicateName generates a unique duplicate name for an item
func generateDuplicateName(db *sql.DB, userID int, baseName string) string {
	// First try "Name (duplicate)"
	candidateName := baseName + " (duplicate)"
	if !itemNameExists(db, userID, candidateName) {
		return candidateName
	}

	// Try "Name (duplicate 2)", "Name (duplicate 3)", etc.
	for i := 2; i < 1000; i++ {
		candidateName = fmt.Sprintf("%s (duplicate %d)", baseName, i)
		if !itemNameExists(db, userID, candidateName) {
			return candidateName
		}
	}

	// Fallback with timestamp if somehow we hit 1000 duplicates
	return fmt.Sprintf("%s (duplicate %d)", baseName, time.Now().Unix())
}

// itemNameExists checks if an item with the given name exists for the user
func itemNameExists(db *sql.DB, userID int, name string) bool {
	var count int
	query := `SELECT COUNT(*) FROM items WHERE user_id = ? AND name = ?`
	err := db.QueryRow(query, userID, name).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// BulkDeleteItems deletes multiple items atomically.
// Returns the number of items deleted.
func BulkDeleteItems(db *sql.DB, userID int, itemIDs []int) (int, error) {
	if len(itemIDs) == 0 {
		return 0, fmt.Errorf("no items specified")
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build placeholders for item IDs
	placeholders := make([]string, len(itemIDs))
	idArgs := make([]interface{}, len(itemIDs))
	for i, id := range itemIDs {
		placeholders[i] = "?"
		idArgs[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	// First, verify ALL items belong to the user
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM items WHERE user_id = ? AND id IN (%s)",
		placeholderStr,
	)

	countArgs := append([]interface{}{userID}, idArgs...)
	var count int
	err = tx.QueryRow(countQuery, countArgs...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to verify item ownership: %w", err)
	}

	if count != len(itemIDs) {
		return 0, fmt.Errorf("some items not found or not owned by user (found %d of %d)", count, len(itemIDs))
	}

	// Remove items from all packs first
	removePackItemsQuery := fmt.Sprintf(
		"DELETE FROM pack_items WHERE item_id IN (%s)",
		placeholderStr,
	)
	_, err = tx.Exec(removePackItemsQuery, idArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to remove items from packs: %w", err)
	}

	// Delete the items
	deleteQuery := fmt.Sprintf(
		"DELETE FROM items WHERE user_id = ? AND id IN (%s)",
		placeholderStr,
	)
	deleteArgs := append([]interface{}{userID}, idArgs...)

	result, err := tx.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk delete items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// BulkUpdateItems updates multiple items atomically with the specified field updates.
// The updates map should contain column names as keys and their new values.
// All updates happen in a single transaction - either all succeed or all fail.
func BulkUpdateItems(db *sql.DB, userID int, itemIDs []int, updates map[string]interface{}) error {
	if len(itemIDs) == 0 || len(updates) == 0 {
		return fmt.Errorf("no items or updates specified")
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build placeholders for item IDs
	placeholders := make([]string, len(itemIDs))
	idArgs := make([]interface{}, len(itemIDs))
	for i, id := range itemIDs {
		placeholders[i] = "?"
		idArgs[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	// First, verify ALL items belong to the user
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM items WHERE user_id = ? AND id IN (%s)",
		placeholderStr,
	)

	countArgs := append([]interface{}{userID}, idArgs...)
	var count int
	err = tx.QueryRow(countQuery, countArgs...).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify item ownership: %w", err)
	}

	if count != len(itemIDs) {
		return fmt.Errorf("some items not found or not owned by user (found %d of %d)", count, len(itemIDs))
	}

	// Build dynamic UPDATE query
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	updateArgs := []interface{}{}

	for field, value := range updates {
		setClauses = append(setClauses, field+" = ?")
		updateArgs = append(updateArgs, value)
	}

	// Add WHERE clause args (userID first, then item IDs)
	updateArgs = append(updateArgs, userID)
	updateArgs = append(updateArgs, idArgs...)

	updateQuery := fmt.Sprintf(
		"UPDATE items SET %s WHERE user_id = ? AND id IN (%s)",
		strings.Join(setClauses, ", "),
		placeholderStr,
	)

	result, err := tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		return fmt.Errorf("failed to bulk update items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if int(rowsAffected) != len(itemIDs) {
		return fmt.Errorf("expected to update %d items, but updated %d", len(itemIDs), rowsAffected)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}