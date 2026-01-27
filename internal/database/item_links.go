package database

import (
	"database/sql"
	"fmt"

	"carryless/internal/models"
)

// CreateItemLink creates a link between two items
// Validates ownership, prevents self-links, and checks for circular references
func CreateItemLink(db *sql.DB, userID, parentItemID, linkedItemID int) error {
	// Prevent self-links
	if parentItemID == linkedItemID {
		return fmt.Errorf("cannot link an item to itself")
	}

	// Verify both items belong to the user
	var parentOwner, linkedOwner int
	err := db.QueryRow("SELECT user_id FROM items WHERE id = ?", parentItemID).Scan(&parentOwner)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("parent item not found")
		}
		return err
	}
	if parentOwner != userID {
		return fmt.Errorf("parent item does not belong to user")
	}

	err = db.QueryRow("SELECT user_id FROM items WHERE id = ?", linkedItemID).Scan(&linkedOwner)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("linked item not found")
		}
		return err
	}
	if linkedOwner != userID {
		return fmt.Errorf("linked item does not belong to user")
	}

	// Check for circular reference (if linkedItemID already has parentItemID as a linked item)
	if err := ValidateNoCircularReference(db, parentItemID, linkedItemID); err != nil {
		return err
	}

	// Create the link
	_, err = db.Exec(
		"INSERT INTO item_links (parent_item_id, linked_item_id) VALUES (?, ?)",
		parentItemID, linkedItemID,
	)
	if err != nil {
		return fmt.Errorf("failed to create item link: %w", err)
	}

	return nil
}

// DeleteItemLink removes a link between two items
func DeleteItemLink(db *sql.DB, userID, parentItemID, linkedItemID int) error {
	// Verify the parent item belongs to the user
	var owner int
	err := db.QueryRow("SELECT user_id FROM items WHERE id = ?", parentItemID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("parent item not found")
		}
		return err
	}
	if owner != userID {
		return fmt.Errorf("parent item does not belong to user")
	}

	result, err := db.Exec(
		"DELETE FROM item_links WHERE parent_item_id = ? AND linked_item_id = ?",
		parentItemID, linkedItemID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete item link: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("link not found")
	}

	return nil
}

// GetLinkedItems returns the linked items with full Item data for a parent item
func GetLinkedItems(db *sql.DB, parentItemID int) ([]models.ItemLink, error) {
	query := `
		SELECT
			il.id, il.parent_item_id, il.linked_item_id, il.created_at,
			i.id, i.user_id, i.category_id, i.name, i.note, i.weight_grams,
			i.weight_to_verify, i.price, i.brand, i.model, i.purchase_date,
			i.capacity, i.capacity_unit, i.link, i.created_at, i.updated_at,
			c.id, c.name
		FROM item_links il
		JOIN items i ON il.linked_item_id = i.id
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE il.parent_item_id = ?
		ORDER BY i.name
	`

	rows, err := db.Query(query, parentItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.ItemLink
	for rows.Next() {
		var link models.ItemLink
		var item models.Item
		var category models.Category
		var categoryID sql.NullInt64
		var categoryName sql.NullString
		var brand, model, capacityUnit, itemLink sql.NullString
		var purchaseDate sql.NullTime
		var capacity sql.NullFloat64

		err := rows.Scan(
			&link.ID, &link.ParentItemID, &link.LinkedItemID, &link.CreatedAt,
			&item.ID, &item.UserID, &item.CategoryID, &item.Name, &item.Note, &item.WeightGrams,
			&item.WeightToVerify, &item.Price, &brand, &model, &purchaseDate,
			&capacity, &capacityUnit, &itemLink, &item.CreatedAt, &item.UpdatedAt,
			&categoryID, &categoryName,
		)
		if err != nil {
			return nil, err
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
		if itemLink.Valid {
			item.Link = &itemLink.String
		}

		if categoryID.Valid {
			category.ID = int(categoryID.Int64)
			category.Name = categoryName.String
			item.Category = &category
		}

		link.LinkedItem = &item
		links = append(links, link)
	}

	return links, nil
}

// GetLinkedItemIDs returns just the IDs of linked items for a parent item
// Used for pack operations when we only need the IDs
func GetLinkedItemIDs(db *sql.DB, parentItemID int) ([]int, error) {
	query := `SELECT linked_item_id FROM item_links WHERE parent_item_id = ?`

	rows, err := db.Query(query, parentItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GetItemsLinkedCount returns a map of item IDs to their linked items count for a user
func GetItemsLinkedCount(db *sql.DB, userID int) (map[int]int, error) {
	query := `
		SELECT il.parent_item_id, COUNT(*) as count
		FROM item_links il
		JOIN items i ON il.parent_item_id = i.id
		WHERE i.user_id = ?
		GROUP BY il.parent_item_id
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var itemID, count int
		if err := rows.Scan(&itemID, &count); err != nil {
			return nil, err
		}
		counts[itemID] = count
	}

	return counts, nil
}

// ValidateNoCircularReference checks that creating a link from parentID to linkedID
// won't create a circular reference (i.e., linkedID doesn't already link back to parentID)
func ValidateNoCircularReference(db *sql.DB, parentID, linkedID int) error {
	// Check if linkedID has parentID as one of its linked items (direct circular ref)
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM item_links WHERE parent_item_id = ? AND linked_item_id = ?)",
		linkedID, parentID,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("circular reference detected: item %d already links to item %d", linkedID, parentID)
	}

	return nil
}

// HasLinkedItems checks if an item has any linked items
func HasLinkedItems(db *sql.DB, itemID int) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM item_links WHERE parent_item_id = ?)",
		itemID,
	).Scan(&exists)
	return exists, err
}
