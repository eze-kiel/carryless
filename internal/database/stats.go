package database

import (
	"database/sql"
	"fmt"
	"time"

	"carryless/internal/models"
)

type UserStats struct {
	TotalPacks      int     `json:"total_packs"`
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalWeight     int     `json:"total_weight"`
	ItemsToVerify   int     `json:"items_to_verify"`
	LightestPack    *models.Pack `json:"lightest_pack,omitempty"`
	LightestWeight  int     `json:"lightest_weight"`
}

type RecentPack struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IsPublic    bool      `json:"is_public"`
	ShortID     string    `json:"short_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	ItemCount   int       `json:"item_count"`
	PackWeight  int       `json:"pack_weight"`
	WornWeight  int       `json:"worn_weight"`
	TotalWeight int       `json:"total_weight"`
}

func GetUserStats(db *sql.DB, userID int) (*UserStats, error) {
	stats := &UserStats{}
	
	// Get total packs
	err := db.QueryRow("SELECT COUNT(*) FROM packs WHERE user_id = ?", userID).Scan(&stats.TotalPacks)
	if err != nil {
		return nil, fmt.Errorf("failed to get pack count: %w", err)
	}
	
	// Get total items
	err = db.QueryRow("SELECT COUNT(*) FROM items WHERE user_id = ?", userID).Scan(&stats.TotalItems)
	if err != nil {
		return nil, fmt.Errorf("failed to get item count: %w", err)
	}
	
	// Get total categories
	err = db.QueryRow("SELECT COUNT(*) FROM categories WHERE user_id = ?", userID).Scan(&stats.TotalCategories)
	if err != nil {
		return nil, fmt.Errorf("failed to get category count: %w", err)
	}
	
	// Get total weight of all items
	err = db.QueryRow("SELECT COALESCE(SUM(weight_grams), 0) FROM items WHERE user_id = ?", userID).Scan(&stats.TotalWeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get total weight: %w", err)
	}
	
	// Get items needing weight verification
	err = db.QueryRow("SELECT COUNT(*) FROM items WHERE user_id = ? AND weight_to_verify = true", userID).Scan(&stats.ItemsToVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to get items to verify count: %w", err)
	}
	
	// Get lightest pack
	lightestQuery := `
		SELECT 
			p.id, 
			p.name,
			COALESCE(SUM(CASE WHEN pi.is_worn = 0 THEN i.weight_grams * pi.count ELSE 0 END), 0) as pack_weight
		FROM packs p
		LEFT JOIN pack_items pi ON p.id = pi.pack_id
		LEFT JOIN items i ON pi.item_id = i.id
		WHERE p.user_id = ?
		GROUP BY p.id, p.name
		HAVING COUNT(pi.item_id) > 0
		ORDER BY pack_weight ASC
		LIMIT 1
	`
	
	var lightestPack models.Pack
	var lightestWeight int
	err = db.QueryRow(lightestQuery, userID).Scan(&lightestPack.ID, &lightestPack.Name, &lightestWeight)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get lightest pack: %w", err)
	}
	if err == nil {
		stats.LightestPack = &lightestPack
		stats.LightestWeight = lightestWeight
	}
	
	return stats, nil
}

func GetRecentPacks(db *sql.DB, userID int, limit int) ([]RecentPack, error) {
	query := `
		SELECT 
			p.id, 
			p.name, 
			p.is_public,
			COALESCE(p.short_id, ''),
			p.updated_at,
			COUNT(DISTINCT pi.item_id) as item_count,
			COALESCE(SUM(CASE WHEN pi.is_worn = 0 THEN i.weight_grams * pi.count ELSE 0 END), 0) as pack_weight,
			COALESCE(SUM(CASE WHEN pi.is_worn = 1 THEN i.weight_grams * pi.count ELSE 0 END), 0) as worn_weight
		FROM packs p
		LEFT JOIN pack_items pi ON p.id = pi.pack_id
		LEFT JOIN items i ON pi.item_id = i.id
		WHERE p.user_id = ?
		GROUP BY p.id, p.name, p.is_public, p.short_id, p.updated_at
		ORDER BY p.updated_at DESC
		LIMIT ?
	`
	
	rows, err := db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent packs: %w", err)
	}
	defer rows.Close()
	
	var recentPacks []RecentPack
	for rows.Next() {
		var pack RecentPack
		err := rows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.IsPublic,
			&pack.ShortID,
			&pack.UpdatedAt,
			&pack.ItemCount,
			&pack.PackWeight,
			&pack.WornWeight,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent pack: %w", err)
		}
		pack.TotalWeight = pack.PackWeight + pack.WornWeight
		recentPacks = append(recentPacks, pack)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent packs: %w", err)
	}
	
	return recentPacks, nil
}