package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"

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

	// Add is_admin column to users table if it doesn't exist
	if err := addUserIsAdminColumn(db); err != nil {
		return fmt.Errorf("failed to add is_admin column: %w", err)
	}

	// Create system settings table if it doesn't exist
	if err := createSystemSettingsTable(db); err != nil {
		return fmt.Errorf("failed to create system_settings table: %w", err)
	}

	// Add weight_to_verify column to items table if it doesn't exist
	if err := addItemWeightToVerifyColumn(db); err != nil {
		return fmt.Errorf("failed to add weight_to_verify column: %w", err)
	}

	// Create labels tables if they don't exist
	if err := createLabelsTable(db); err != nil {
		return fmt.Errorf("failed to create labels tables: %w", err)
	}

	// Add note column to packs table if it doesn't exist
	if err := addPackNoteColumn(db); err != nil {
		return fmt.Errorf("failed to add note column to packs: %w", err)
	}

	// Add short_id column to packs table if it doesn't exist
	if err := addPackShortIDColumn(db); err != nil {
		return fmt.Errorf("failed to add short_id column to packs: %w", err)
	}

	// Add is_activated column to users table and create activation_tokens table
	if err := addUserActivationColumn(db); err != nil {
		return fmt.Errorf("failed to add activation column to users: %w", err)
	}

	// Create activation tokens table if it doesn't exist
	if err := createActivationTokensTable(db); err != nil {
		return fmt.Errorf("failed to create activation_tokens table: %w", err)
	}

	// Add last_seen column to users table if it doesn't exist
	if err := addUserLastSeenColumn(db); err != nil {
		return fmt.Errorf("failed to add last_seen column to users: %w", err)
	}

	// Add is_locked column to packs table if it doesn't exist
	if err := addPackIsLockedColumn(db); err != nil {
		return fmt.Errorf("failed to add is_locked column to packs: %w", err)
	}

	// Add optional fields to items table if they don't exist
	if err := addItemOptionalFields(db); err != nil {
		return fmt.Errorf("failed to add optional fields to items: %w", err)
	}

	// Create trips tables if they don't exist
	if err := createTripsTable(db); err != nil {
		return fmt.Errorf("failed to create trips tables: %w", err)
	}

	// Migrate transport steps to use departure/arrival fields
	if err := migrateTransportStepsDepartureArrival(db); err != nil {
		return fmt.Errorf("failed to migrate transport steps: %w", err)
	}

	// Create item_links table if it doesn't exist
	if err := createItemLinksTable(db); err != nil {
		return fmt.Errorf("failed to create item_links table: %w", err)
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

func addUserIsAdminColumn(db *sql.DB) error {
	// Check if is_admin column exists
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasIsAdmin := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "is_admin" {
			hasIsAdmin = true
			break
		}
	}

	if !hasIsAdmin {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT FALSE")
		if err != nil {
			return err
		}
	}

	return nil
}

func createSystemSettingsTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	
	if _, err := db.Exec(query); err != nil {
		return err
	}

	// Insert default value for registration enabled if it doesn't exist
	insertQuery := `INSERT OR IGNORE INTO system_settings (key, value) VALUES ('registration_enabled', 'true')`
	if _, err := db.Exec(insertQuery); err != nil {
		return err
	}

	return nil
}

func addItemWeightToVerifyColumn(db *sql.DB) error {
	// Check if weight_to_verify column exists
	rows, err := db.Query("PRAGMA table_info(items)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasWeightToVerify := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "weight_to_verify" {
			hasWeightToVerify = true
			break
		}
	}

	if !hasWeightToVerify {
		_, err = db.Exec("ALTER TABLE items ADD COLUMN weight_to_verify BOOLEAN DEFAULT FALSE")
		if err != nil {
			return err
		}
	}

	return nil
}

func createLabelsTable(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS pack_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pack_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT DEFAULT '#6b7280',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (pack_id) REFERENCES packs(id) ON DELETE CASCADE,
			UNIQUE(pack_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS item_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pack_item_id INTEGER NOT NULL,
			pack_label_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (pack_item_id) REFERENCES pack_items(id) ON DELETE CASCADE,
			FOREIGN KEY (pack_label_id) REFERENCES pack_labels(id) ON DELETE CASCADE,
			UNIQUE(pack_item_id, pack_label_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pack_labels_pack_id ON pack_labels(pack_id)`,
		`CREATE INDEX IF NOT EXISTS idx_item_labels_pack_item_id ON item_labels(pack_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_item_labels_pack_label_id ON item_labels(pack_label_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	// Check if count column exists in item_labels table
	var hasCount bool
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('item_labels') WHERE name='count'").Scan(&hasCount)
	if err != nil {
		return err
	}

	if !hasCount {
		// Drop the unique constraint by recreating the table with count column
		migrations := []string{
			`CREATE TABLE item_labels_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				pack_item_id INTEGER NOT NULL,
				pack_label_id INTEGER NOT NULL,
				count INTEGER DEFAULT 1,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (pack_item_id) REFERENCES pack_items(id) ON DELETE CASCADE,
				FOREIGN KEY (pack_label_id) REFERENCES pack_labels(id) ON DELETE CASCADE
			)`,
			`INSERT INTO item_labels_new (id, pack_item_id, pack_label_id, count, created_at) 
			 SELECT id, pack_item_id, pack_label_id, 1, created_at FROM item_labels`,
			`DROP TABLE item_labels`,
			`ALTER TABLE item_labels_new RENAME TO item_labels`,
			`CREATE INDEX IF NOT EXISTS idx_item_labels_pack_item_id ON item_labels(pack_item_id)`,
			`CREATE INDEX IF NOT EXISTS idx_item_labels_pack_label_id ON item_labels(pack_label_id)`,
		}

		for _, migration := range migrations {
			if _, err := db.Exec(migration); err != nil {
				return err
			}
		}
	}

	return nil
}

func addPackNoteColumn(db *sql.DB) error {
	// Check if note column exists
	rows, err := db.Query("PRAGMA table_info(packs)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasNote := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "note" {
			hasNote = true
			break
		}
	}

	if !hasNote {
		// Add note column to packs table
		_, err = db.Exec("ALTER TABLE packs ADD COLUMN note TEXT")
		if err != nil {
			return err
		}
	}

	return nil
}

func addPackShortIDColumn(db *sql.DB) error {
	// Check if short_id column exists
	rows, err := db.Query("PRAGMA table_info(packs)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasShortID := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "short_id" {
			hasShortID = true
			break
		}
	}

	if !hasShortID {
		// Add short_id column to packs table (without UNIQUE constraint initially)
		_, err = db.Exec("ALTER TABLE packs ADD COLUMN short_id TEXT")
		if err != nil {
			return err
		}

		// Generate short IDs for existing public packs
		err = migrateExistingPublicPacks(db)
		if err != nil {
			return err
		}

		// Now add the UNIQUE constraint and index
		_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_packs_short_id_unique ON packs(short_id) WHERE short_id IS NOT NULL AND short_id != ''")
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateExistingPublicPacks(db *sql.DB) error {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const idLength = 8
	
	// Get all public packs without short_id
	query := `SELECT id FROM packs WHERE is_public = 1 AND (short_id IS NULL OR short_id = '')`
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query public packs: %w", err)
	}
	defer rows.Close()

	var packIDs []string
	for rows.Next() {
		var packID string
		if err := rows.Scan(&packID); err != nil {
			return fmt.Errorf("failed to scan pack ID: %w", err)
		}
		packIDs = append(packIDs, packID)
	}

	// Generate short IDs for each pack
	for _, packID := range packIDs {
		shortID, err := generateUniqueShortID(db, charset, idLength)
		if err != nil {
			return fmt.Errorf("failed to generate short ID for pack %s: %w", packID, err)
		}

		// Update the pack with the new short ID
		updateQuery := `UPDATE packs SET short_id = ? WHERE id = ?`
		_, err = db.Exec(updateQuery, shortID, packID)
		if err != nil {
			return fmt.Errorf("failed to update pack %s with short ID: %w", packID, err)
		}
	}

	return nil
}

func generateUniqueShortID(db *sql.DB, charset string, idLength int) (string, error) {
	const maxRetries = 10

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Generate random ID
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

func addUserActivationColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasActivated := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "is_activated" {
			hasActivated = true
			break
		}
	}

	if !hasActivated {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN is_activated BOOLEAN DEFAULT FALSE")
		if err != nil {
			return err
		}
		
		_, err = db.Exec("UPDATE users SET is_activated = TRUE")
		if err != nil {
			return err
		}
	}

	return nil
}

func createActivationTokensTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS activation_tokens (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`
	
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	indexQuery := `CREATE INDEX IF NOT EXISTS idx_activation_tokens_user_id ON activation_tokens(user_id)`
	_, err = db.Exec(indexQuery)
	if err != nil {
		return err
	}

	expireIndexQuery := `CREATE INDEX IF NOT EXISTS idx_activation_tokens_expires_at ON activation_tokens(expires_at)`
	_, err = db.Exec(expireIndexQuery)
	if err != nil {
		return err
	}

	return nil
}

func addUserLastSeenColumn(db *sql.DB) error {
	// Check if last_seen column exists
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasLastSeen := false
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk string
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}
		if name == "last_seen" {
			hasLastSeen = true
			break
		}
	}

	if !hasLastSeen {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN last_seen DATETIME DEFAULT NULL")
		if err != nil {
			return err
		}
	}

	return nil
}

func addPackIsLockedColumn(db *sql.DB) error {
	// Check if is_locked column exists
	rows, err := db.Query("PRAGMA table_info(packs)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasIsLocked := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "is_locked" {
			hasIsLocked = true
			break
		}
	}

	if !hasIsLocked {
		// Add is_locked column to packs table
		_, err = db.Exec("ALTER TABLE packs ADD COLUMN is_locked BOOLEAN DEFAULT FALSE")
		if err != nil {
			return err
		}
	}

	return nil
}

func addItemOptionalFields(db *sql.DB) error {
	// Check which columns exist in items table
	rows, err := db.Query("PRAGMA table_info(items)")
	if err != nil {
		return err
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		existingColumns[name] = true
	}

	// Define columns to add with their SQL definitions
	columnsToAdd := map[string]string{
		"brand":         "ALTER TABLE items ADD COLUMN brand TEXT",
		"model":         "ALTER TABLE items ADD COLUMN model TEXT",
		"purchase_date": "ALTER TABLE items ADD COLUMN purchase_date DATETIME",
		"capacity":      "ALTER TABLE items ADD COLUMN capacity REAL",
		"capacity_unit": "ALTER TABLE items ADD COLUMN capacity_unit TEXT",
		"link":          "ALTER TABLE items ADD COLUMN link TEXT",
	}

	// Add each missing column
	for column, sql := range columnsToAdd {
		if !existingColumns[column] {
			if _, err := db.Exec(sql); err != nil {
				return fmt.Errorf("failed to add %s column: %w", column, err)
			}
		}
	}

	return nil
}

func createTripsTable(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS trips (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			location TEXT,
			start_date DATETIME,
			end_date DATETIME,
			notes TEXT,
			gpx_data TEXT,
			is_public BOOLEAN DEFAULT FALSE,
			is_archived BOOLEAN DEFAULT FALSE,
			short_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS trip_packs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trip_id TEXT NOT NULL,
			pack_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (pack_id) REFERENCES packs(id) ON DELETE CASCADE,
			UNIQUE(trip_id, pack_id)
		)`,
		`CREATE TABLE IF NOT EXISTS trip_checklist_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trip_id TEXT NOT NULL,
			content TEXT NOT NULL,
			is_checked BOOLEAN DEFAULT FALSE,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS trip_transport_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trip_id TEXT NOT NULL,
			journey_type TEXT NOT NULL CHECK(journey_type IN ('outbound', 'return')),
			step_order INTEGER NOT NULL,
			place_name TEXT NOT NULL,
			datetime DATETIME,
			transport_type TEXT CHECK(transport_type IN ('train', 'plane', 'bus', 'other')),
			transport_number TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_trips_short_id ON trips(short_id) WHERE short_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_trips_user_id ON trips(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_packs_trip_id ON trip_packs(trip_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_packs_pack_id ON trip_packs(pack_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_trip_id ON trip_checklist_items(trip_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_transport_steps_trip_id ON trip_transport_steps(trip_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	return nil
}

func createItemLinksTable(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS item_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_item_id INTEGER NOT NULL,
			linked_item_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_item_id) REFERENCES items(id) ON DELETE CASCADE,
			FOREIGN KEY (linked_item_id) REFERENCES items(id) ON DELETE CASCADE,
			UNIQUE(parent_item_id, linked_item_id),
			CHECK(parent_item_id != linked_item_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_item_links_parent ON item_links(parent_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_item_links_linked ON item_links(linked_item_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	return nil
}

// migrateTransportStepsDepartureArrival migrates transport steps to use departure/arrival fields
func migrateTransportStepsDepartureArrival(db *sql.DB) error {
	// Check if migration is needed by looking for the old place_name column
	var hasOldSchema bool
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('trip_transport_steps')
		WHERE name = 'place_name'
	`).Scan(&hasOldSchema)
	if err != nil {
		return err
	}

	// If we don't have the old schema, migration already done
	if !hasOldSchema {
		return nil
	}

	migrations := []string{
		// Add new columns
		`ALTER TABLE trip_transport_steps ADD COLUMN departure_place TEXT`,
		`ALTER TABLE trip_transport_steps ADD COLUMN departure_datetime DATETIME`,
		`ALTER TABLE trip_transport_steps ADD COLUMN arrival_place TEXT`,
		`ALTER TABLE trip_transport_steps ADD COLUMN arrival_datetime DATETIME`,
		// Copy data from old columns to new columns
		`UPDATE trip_transport_steps SET departure_place = place_name, departure_datetime = datetime`,
		// Create new table with correct schema
		`CREATE TABLE trip_transport_steps_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trip_id TEXT NOT NULL,
			journey_type TEXT NOT NULL CHECK(journey_type IN ('outbound', 'return')),
			step_order INTEGER NOT NULL,
			departure_place TEXT NOT NULL,
			departure_datetime DATETIME,
			arrival_place TEXT,
			arrival_datetime DATETIME,
			transport_type TEXT CHECK(transport_type IN ('train', 'plane', 'bus', 'other')),
			transport_number TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`,
		// Copy data to new table
		`INSERT INTO trip_transport_steps_new (id, trip_id, journey_type, step_order, departure_place, departure_datetime, arrival_place, arrival_datetime, transport_type, transport_number, notes, created_at)
		 SELECT id, trip_id, journey_type, step_order, departure_place, departure_datetime, arrival_place, arrival_datetime, transport_type, transport_number, notes, created_at
		 FROM trip_transport_steps`,
		// Drop old table
		`DROP TABLE trip_transport_steps`,
		// Rename new table
		`ALTER TABLE trip_transport_steps_new RENAME TO trip_transport_steps`,
		// Recreate index
		`CREATE INDEX IF NOT EXISTS idx_trip_transport_steps_trip_id ON trip_transport_steps(trip_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	return nil
}