package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"
)

func CreateCategory(db *sql.DB, userID int, name string) (*models.Category, error) {
	query := `
		INSERT INTO categories (user_id, name)
		VALUES (?, ?)
	`

	result, err := db.Exec(query, userID, name)
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
		Name:   name,
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
	var itemCount int
	countQuery := `SELECT COUNT(*) FROM items WHERE category_id = ? AND user_id = ?`
	err := db.QueryRow(countQuery, categoryID, userID).Scan(&itemCount)
	if err != nil {
		return fmt.Errorf("failed to check items in category: %w", err)
	}

	if itemCount > 0 {
		return fmt.Errorf("cannot delete category with %d items", itemCount)
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