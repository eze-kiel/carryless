package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Initialize(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			note TEXT,
			weight_grams INTEGER NOT NULL,
			price REAL NOT NULL DEFAULT 0,
			purchase_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS packs (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			is_public BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS pack_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pack_id TEXT NOT NULL,
			item_id INTEGER NOT NULL,
			is_worn BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (pack_id) REFERENCES packs(id) ON DELETE CASCADE,
			FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
			UNIQUE(pack_id, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_csrf_tokens_user_id ON csrf_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_csrf_tokens_expires_at ON csrf_tokens(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_items_user_id ON items(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_packs_user_id ON packs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pack_items_pack_id ON pack_items(pack_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pack_items_item_id ON pack_items(item_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	// Handle existing database schema updates
	if err := updatePackItemsSchema(db); err != nil {
		return fmt.Errorf("failed to update pack_items schema: %w", err)
	}

	// Add currency column to users table if it doesn't exist
	if err := addUserCurrencyColumn(db); err != nil {
		return fmt.Errorf("failed to add currency column: %w", err)
	}

	// Remove purchase_date column from items table if it exists
	if err := removePurchaseDateColumn(db); err != nil {
		return fmt.Errorf("failed to remove purchase_date column: %w", err)
	}

	return nil
}

func addUserCurrencyColumn(db *sql.DB) error {
	// Check if currency column exists
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasCurrency := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "currency" {
			hasCurrency = true
			break
		}
	}

	if !hasCurrency {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN currency TEXT DEFAULT '$'")
		if err != nil {
			return err
		}
	}

	return nil
}

func removePurchaseDateColumn(db *sql.DB) error {
	// Check if purchase_date column exists in items table
	rows, err := db.Query("PRAGMA table_info(items)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasPurchaseDate := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "purchase_date" {
			hasPurchaseDate = true
			break
		}
	}

	if hasPurchaseDate {
		// SQLite doesn't support DROP COLUMN, so we need to recreate the table
		migrations := []string{
			`CREATE TABLE IF NOT EXISTS items_temp (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				category_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				note TEXT,
				weight_grams INTEGER DEFAULT 0,
				price REAL DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
			)`,
			`INSERT INTO items_temp (id, user_id, category_id, name, note, weight_grams, price, created_at, updated_at)
			 SELECT id, user_id, category_id, name, note, weight_grams, price, created_at, updated_at FROM items`,
			`DROP TABLE items`,
			`ALTER TABLE items_temp RENAME TO items`,
			`CREATE INDEX IF NOT EXISTS idx_items_user_id ON items(user_id)`,
			`CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id)`,
		}

		for _, migration := range migrations {
			if _, err := db.Exec(migration); err != nil {
				return err
			}
		}
	}

	return nil
}

func updatePackItemsSchema(db *sql.DB) error {
	// Check if count column exists
	rows, err := db.Query("PRAGMA table_info(pack_items)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasCount := false
	hasWornCount := false
	
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "count" {
			hasCount = true
		}
		if name == "worn_count" {
			hasWornCount = true
		}
	}

	// If we need to add columns or remove unique constraint, recreate the table
	if !hasCount || !hasWornCount {
		migrations := []string{
			`CREATE TABLE IF NOT EXISTS pack_items_temp (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				pack_id TEXT NOT NULL,
				item_id INTEGER NOT NULL,
				is_worn BOOLEAN DEFAULT FALSE,
				count INTEGER DEFAULT 1,
				worn_count INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (pack_id) REFERENCES packs(id) ON DELETE CASCADE,
				FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
			)`,
		}

		// Copy data based on what columns exist
		var insertSQL string
		if hasCount && hasWornCount {
			insertSQL = `INSERT INTO pack_items_temp (id, pack_id, item_id, is_worn, count, worn_count, created_at)
						 SELECT id, pack_id, item_id, is_worn, count, worn_count, created_at FROM pack_items`
		} else if hasCount {
			insertSQL = `INSERT INTO pack_items_temp (id, pack_id, item_id, is_worn, count, created_at)
						 SELECT id, pack_id, item_id, is_worn, count, created_at FROM pack_items`
		} else {
			insertSQL = `INSERT INTO pack_items_temp (id, pack_id, item_id, is_worn, created_at)
						 SELECT id, pack_id, item_id, is_worn, created_at FROM pack_items`
		}

		migrations = append(migrations,
			insertSQL,
			`DROP TABLE pack_items`,
			`ALTER TABLE pack_items_temp RENAME TO pack_items`,
			`CREATE INDEX IF NOT EXISTS idx_pack_items_pack_id ON pack_items(pack_id)`,
			`CREATE INDEX IF NOT EXISTS idx_pack_items_item_id ON pack_items(item_id)`,
		)

		for _, migration := range migrations {
			if _, err := db.Exec(migration); err != nil {
				return err
			}
		}
	}

	return nil
}