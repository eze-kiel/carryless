package database

import (
	"database/sql"
	"fmt"
	"time"

	"carryless/internal/models"
)

func CreateItem(db *sql.DB, userID int, item models.Item) (*models.Item, error) {
	query := `
		INSERT INTO items (user_id, category_id, name, note, weight_grams, weight_to_verify, price, brand, purchase_date, capacity, capacity_unit, link)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, userID, item.CategoryID, item.Name, item.Note, item.WeightGrams, item.WeightToVerify, item.Price,
		item.Brand, item.PurchaseDate, item.Capacity, item.CapacityUnit, item.Link)
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
		       i.brand, i.purchase_date, i.capacity, i.capacity_unit, i.link,
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
		var brand, capacityUnit, link sql.NullString
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
	var brand, capacityUnit, link sql.NullString
	var purchaseDate sql.NullTime
	var capacity sql.NullFloat64

	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, COALESCE(i.weight_to_verify, false), i.price,
		       i.brand, i.purchase_date, i.capacity, i.capacity_unit, i.link,
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
		    brand = ?, purchase_date = ?, capacity = ?, capacity_unit = ?, link = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, updatedItem.CategoryID, updatedItem.Name, updatedItem.Note, updatedItem.WeightGrams, updatedItem.WeightToVerify, updatedItem.Price,
		updatedItem.Brand, updatedItem.PurchaseDate, updatedItem.Capacity, updatedItem.CapacityUnit, updatedItem.Link,
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
		       i.brand, i.purchase_date, i.capacity, i.capacity_unit, i.link,
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
		var brand, capacityUnit, link sql.NullString
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
		       i.brand, i.purchase_date, i.capacity, i.capacity_unit, i.link,
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
		var brand, capacityUnit, link sql.NullString
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