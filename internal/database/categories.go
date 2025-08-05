package database

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"carryless/internal/models"
)

// normalizeCategoryName converts category name to title case (first letter uppercase, rest lowercase)
func normalizeCategoryName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	
	// Convert to lowercase first
	name = strings.ToLower(name)
	
	// Capitalize the first letter
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	
	return string(runes)
}

func CreateCategory(db *sql.DB, userID int, name string) (*models.Category, error) {
	// Normalize the category name to title case
	normalizedName := normalizeCategoryName(name)
	
	query := `
		INSERT INTO categories (user_id, name)
		VALUES (?, ?)
	`

	result, err := db.Exec(query, userID, normalizedName)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get category ID: %w", err)
	}

	category := &models.Category{
		ID:     int(id),
		UserID: userID,
		Name:   normalizedName,
	}

	return category, nil
}

func GetCategories(db *sql.DB, userID int) ([]models.Category, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM categories
		WHERE user_id = ?
		ORDER BY name
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID,
			&category.UserID,
			&category.Name,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return categories, nil
}

func GetCategory(db *sql.DB, userID, categoryID int) (*models.Category, error) {
	category := &models.Category{}
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM categories
		WHERE id = ? AND user_id = ?
	`

	err := db.QueryRow(query, categoryID, userID).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to query category: %w", err)
	}

	return category, nil
}

func UpdateCategory(db *sql.DB, userID, categoryID int, name string) error {
	query := `
		UPDATE categories
		SET name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, name, categoryID, userID)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

func DeleteCategory(db *sql.DB, userID, categoryID int) error {
	return DeleteCategoryWithForce(db, userID, categoryID, false)
}

func DeleteCategoryWithForce(db *sql.DB, userID, categoryID int, force bool) error {
	var itemCount int
	countQuery := `SELECT COUNT(*) FROM items WHERE category_id = ? AND user_id = ?`
	err := db.QueryRow(countQuery, categoryID, userID).Scan(&itemCount)
	if err != nil {
		return fmt.Errorf("failed to check items in category: %w", err)
	}

	if itemCount > 0 && !force {
		return fmt.Errorf("cannot delete category with %d items", itemCount)
	}

	// If force is true and category has items, delete all items first
	if force && itemCount > 0 {
		// First remove items from any packs
		removeFromPacksQuery := `
			DELETE FROM pack_items 
			WHERE item_id IN (SELECT id FROM items WHERE category_id = ? AND user_id = ?)
		`
		_, err := db.Exec(removeFromPacksQuery, categoryID, userID)
		if err != nil {
			return fmt.Errorf("failed to remove items from packs: %w", err)
		}

		// Then delete all items in the category
		deleteItemsQuery := `DELETE FROM items WHERE category_id = ? AND user_id = ?`
		_, err = db.Exec(deleteItemsQuery, categoryID, userID)
		if err != nil {
			return fmt.Errorf("failed to delete items in category: %w", err)
		}
	}

	query := `
		DELETE FROM categories
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, categoryID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

func GetItemsInCategory(db *sql.DB, userID, categoryID int) ([]models.ItemInfo, error) {
	query := `
		SELECT name, note 
		FROM items 
		WHERE category_id = ? AND user_id = ?
		ORDER BY name
	`
	
	rows, err := db.Query(query, categoryID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items in category: %w", err)
	}
	defer rows.Close()

	var items []models.ItemInfo
	for rows.Next() {
		var item models.ItemInfo
		if err := rows.Scan(&item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	return items, nil
}

func GetOrCreateCategory(db *sql.DB, userID int, name string) (*models.Category, error) {
	// Normalize the input name for consistent searching and creation
	normalizedName := normalizeCategoryName(name)
	
	// First try to get existing category (case-insensitive)
	query := `SELECT id, user_id, name FROM categories WHERE user_id = ? AND LOWER(name) = LOWER(?)`
	var category models.Category
	err := db.QueryRow(query, userID, normalizedName).Scan(&category.ID, &category.UserID, &category.Name)
	
	if err == nil {
		// Category exists, return the existing one
		return &category, nil
	}
	
	if err != sql.ErrNoRows {
		// Real error occurred
		return nil, fmt.Errorf("failed to query category: %w", err)
	}
	
	// Category doesn't exist, create it with normalized case (Title case)
	return CreateCategory(db, userID, normalizedName)
}