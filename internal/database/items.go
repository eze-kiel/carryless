package database

import (
	"database/sql"
	"fmt"
	"time"

	"carryless/internal/models"
)

func CreateItem(db *sql.DB, userID int, item models.Item) (*models.Item, error) {
	query := `
		INSERT INTO items (user_id, category_id, name, note, weight_grams, price)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, userID, item.CategoryID, item.Name, item.Note, item.WeightGrams, item.Price)
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
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.price, i.created_at, i.updated_at,
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

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan item: %w", err)
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

	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.price, i.created_at, i.updated_at,
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
		&item.Price,
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

	item.Category = category
	return item, nil
}

func UpdateItem(db *sql.DB, userID, itemID int, updatedItem models.Item) error {
	query := `
		UPDATE items
		SET category_id = ?, name = ?, note = ?, weight_grams = ?, price = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, updatedItem.CategoryID, updatedItem.Name, updatedItem.Note, updatedItem.WeightGrams, updatedItem.Price, itemID, userID)
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
	var packCount int
	countQuery := `SELECT COUNT(*) FROM pack_items WHERE item_id = ?`
	err := db.QueryRow(countQuery, itemID).Scan(&packCount)
	if err != nil {
		return fmt.Errorf("failed to check item usage in packs: %w", err)
	}

	if packCount > 0 {
		return fmt.Errorf("cannot delete item used in %d pack(s)", packCount)
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

func GetItemsByCategory(db *sql.DB, userID, categoryID int) ([]models.Item, error) {
	query := `
		SELECT i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams, i.price, i.created_at, i.updated_at,
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

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan item: %w", err)
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