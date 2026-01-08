package database

import (
	"database/sql"
	"fmt"
	"time"

	"carryless/internal/logger"
	"carryless/internal/models"

	"github.com/google/uuid"
)

// Helper function to generate short ID for trips (reusing logic from packs)
func generateTripShortID(db *sql.DB) (string, error) {
	return generateShortID(db) // Reuse existing function
}

// Helper function to update trip timestamp
func updateTripTimestamp(db *sql.DB, tripID string) error {
	query := `UPDATE trips SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, tripID)
	return err
}

// CreateTrip creates a new trip
func CreateTrip(db *sql.DB, userID int, name string, isPublic bool) (*models.Trip, error) {
	tripID := uuid.New().String()

	var shortID sql.NullString
	if isPublic {
		shortIDValue, err := generateTripShortID(db)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short ID: %w", err)
		}
		shortID = sql.NullString{String: shortIDValue, Valid: true}
	}

	query := `
		INSERT INTO trips (id, user_id, name, is_public, short_id)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, tripID, userID, name, isPublic, shortID)
	if err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	trip := &models.Trip{
		ID:         tripID,
		UserID:     userID,
		Name:       name,
		IsPublic:   isPublic,
		IsArchived: false,
		ShortID:    shortID.String,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return trip, nil
}

// GetTrips returns all trips for a user
func GetTrips(db *sql.DB, userID int) ([]models.Trip, error) {
	query := `
		SELECT
			id, user_id, name,
			COALESCE(description, ''),
			COALESCE(location, ''),
			start_date, end_date,
			COALESCE(notes, ''),
			is_public, is_archived,
			COALESCE(short_id, ''),
			created_at, updated_at
		FROM trips
		WHERE user_id = ?
		ORDER BY is_archived ASC, created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trips: %w", err)
	}
	defer rows.Close()

	var trips []models.Trip
	for rows.Next() {
		var trip models.Trip
		var description, location, notes, shortID string
		var startDate, endDate sql.NullTime

		err := rows.Scan(
			&trip.ID, &trip.UserID, &trip.Name,
			&description, &location,
			&startDate, &endDate,
			&notes,
			&trip.IsPublic, &trip.IsArchived,
			&shortID,
			&trip.CreatedAt, &trip.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}

		// Handle nullable fields
		if description != "" {
			trip.Description = &description
		}
		if location != "" {
			trip.Location = &location
		}
		if startDate.Valid {
			trip.StartDate = &startDate.Time
		}
		if endDate.Valid {
			trip.EndDate = &endDate.Time
		}
		if notes != "" {
			trip.Notes = &notes
		}
		if shortID != "" {
			trip.ShortID = shortID
		}

		trips = append(trips, trip)
	}

	return trips, nil
}

// GetTrip returns a single trip by ID
func GetTrip(db *sql.DB, tripID string) (*models.Trip, error) {
	query := `
		SELECT
			id, user_id, name,
			COALESCE(description, ''),
			COALESCE(location, ''),
			start_date, end_date,
			COALESCE(notes, ''),
			COALESCE(gpx_data, ''),
			is_public, is_archived,
			COALESCE(short_id, ''),
			created_at, updated_at
		FROM trips
		WHERE id = ?
	`

	var trip models.Trip
	var description, location, notes, gpxData, shortID string
	var startDate, endDate sql.NullTime

	err := db.QueryRow(query, tripID).Scan(
		&trip.ID, &trip.UserID, &trip.Name,
		&description, &location,
		&startDate, &endDate,
		&notes, &gpxData,
		&trip.IsPublic, &trip.IsArchived,
		&shortID,
		&trip.CreatedAt, &trip.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trip not found")
		}
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Handle nullable fields
	if description != "" {
		trip.Description = &description
	}
	if location != "" {
		trip.Location = &location
	}
	if startDate.Valid {
		trip.StartDate = &startDate.Time
	}
	if endDate.Valid {
		trip.EndDate = &endDate.Time
	}
	if notes != "" {
		trip.Notes = &notes
	}
	if gpxData != "" {
		trip.GPXData = &gpxData
	}
	if shortID != "" {
		trip.ShortID = shortID
	}

	return &trip, nil
}

// GetTripByShortID returns a public trip by its short ID
func GetTripByShortID(db *sql.DB, shortID string) (*models.Trip, error) {
	query := `
		SELECT
			id, user_id, name,
			COALESCE(description, ''),
			COALESCE(location, ''),
			start_date, end_date,
			COALESCE(notes, ''),
			COALESCE(gpx_data, ''),
			is_public, is_archived,
			COALESCE(short_id, ''),
			created_at, updated_at
		FROM trips
		WHERE short_id = ? AND is_public = TRUE
	`

	var trip models.Trip
	var description, location, notes, gpxData, shortIDVal string
	var startDate, endDate sql.NullTime

	err := db.QueryRow(query, shortID).Scan(
		&trip.ID, &trip.UserID, &trip.Name,
		&description, &location,
		&startDate, &endDate,
		&notes, &gpxData,
		&trip.IsPublic, &trip.IsArchived,
		&shortIDVal,
		&trip.CreatedAt, &trip.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trip not found")
		}
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Handle nullable fields
	if description != "" {
		trip.Description = &description
	}
	if location != "" {
		trip.Location = &location
	}
	if startDate.Valid {
		trip.StartDate = &startDate.Time
	}
	if endDate.Valid {
		trip.EndDate = &endDate.Time
	}
	if notes != "" {
		trip.Notes = &notes
	}
	if gpxData != "" {
		trip.GPXData = &gpxData
	}
	if shortIDVal != "" {
		trip.ShortID = shortIDVal
	}

	return &trip, nil
}

// GetTripWithDetails returns a trip with all related data (packs, checklist, transport steps)
func GetTripWithDetails(db *sql.DB, tripID string) (*models.Trip, error) {
	trip, err := GetTrip(db, tripID)
	if err != nil {
		return nil, err
	}

	// Load associated packs
	packs, err := GetTripPacks(db, tripID)
	if err != nil {
		logger.Error("Failed to load trip packs", "trip_id", tripID, "error", err)
	} else {
		trip.Packs = packs
	}

	// Load checklist items
	checklistItems, err := GetChecklistItems(db, tripID)
	if err != nil {
		logger.Error("Failed to load checklist items", "trip_id", tripID, "error", err)
	} else {
		trip.ChecklistItems = checklistItems
	}

	// Load transport steps
	transportSteps, err := GetTransportSteps(db, tripID)
	if err != nil {
		logger.Error("Failed to load transport steps", "trip_id", tripID, "error", err)
	} else {
		trip.TransportSteps = transportSteps
	}

	return trip, nil
}

// UpdateTrip updates a trip's fields
func UpdateTrip(db *sql.DB, userID int, tripID string, name string, description, location *string, startDate, endDate *time.Time, isPublic bool) error {
	// First check ownership
	var ownerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("trip not found")
		}
		return fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if ownerID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Check if we need to generate a short_id
	var currentShortID sql.NullString
	err = db.QueryRow("SELECT short_id FROM trips WHERE id = ?", tripID).Scan(&currentShortID)
	if err != nil {
		return fmt.Errorf("failed to get current short_id: %w", err)
	}

	// Generate short_id if making public and doesn't have one
	if isPublic && !currentShortID.Valid {
		shortIDValue, err := generateTripShortID(db)
		if err != nil {
			return fmt.Errorf("failed to generate short ID: %w", err)
		}
		currentShortID = sql.NullString{String: shortIDValue, Valid: true}
	}

	query := `
		UPDATE trips
		SET name = ?, description = ?, location = ?, start_date = ?, end_date = ?,
		    is_public = ?, short_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, name, description, location, startDate, endDate, isPublic, currentShortID, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// UpdateTripNotes updates the notes field of a trip
func UpdateTripNotes(db *sql.DB, userID int, tripID string, notes *string) error {
	query := `
		UPDATE trips
		SET notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, notes, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to update trip notes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// DeleteTrip deletes a trip and all related data (cascade)
func DeleteTrip(db *sql.DB, userID int, tripID string) error {
	query := `DELETE FROM trips WHERE id = ? AND user_id = ?`

	result, err := db.Exec(query, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// ArchiveTrip toggles the archive status of a trip
func ArchiveTrip(db *sql.DB, userID int, tripID string, archive bool) error {
	query := `
		UPDATE trips
		SET is_archived = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, archive, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to archive trip: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// Pack Association Functions

// AddPackToTrip associates a pack with a trip
func AddPackToTrip(db *sql.DB, tripID, packID string, userID int) error {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("trip not found")
		}
		return fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Verify pack ownership
	var packOwnerID int
	err = db.QueryRow("SELECT user_id FROM packs WHERE id = ?", packID).Scan(&packOwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("pack not found")
		}
		return fmt.Errorf("failed to check pack ownership: %w", err)
	}

	if packOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		INSERT INTO trip_packs (trip_id, pack_id)
		VALUES (?, ?)
	`

	_, err = db.Exec(query, tripID, packID)
	if err != nil {
		return fmt.Errorf("failed to add pack to trip: %w", err)
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// RemovePackFromTrip removes a pack association from a trip
func RemovePackFromTrip(db *sql.DB, tripID, packID string, userID int) error {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		return fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `DELETE FROM trip_packs WHERE trip_id = ? AND pack_id = ?`

	result, err := db.Exec(query, tripID, packID)
	if err != nil {
		return fmt.Errorf("failed to remove pack from trip: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pack association not found")
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// GetTripPacks returns all packs associated with a trip
func GetTripPacks(db *sql.DB, tripID string) ([]models.Pack, error) {
	query := `
		SELECT p.id, p.user_id, p.name, COALESCE(p.note, ''), p.is_public,
		       COALESCE(p.is_locked, FALSE), COALESCE(p.short_id, ''),
		       p.created_at, p.updated_at
		FROM packs p
		INNER JOIN trip_packs tp ON p.id = tp.pack_id
		WHERE tp.trip_id = ?
		ORDER BY p.name
	`

	rows, err := db.Query(query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trip packs: %w", err)
	}
	defer rows.Close()

	var packs []models.Pack
	for rows.Next() {
		var pack models.Pack
		err := rows.Scan(
			&pack.ID, &pack.UserID, &pack.Name, &pack.Note,
			&pack.IsPublic, &pack.IsLocked, &pack.ShortID,
			&pack.CreatedAt, &pack.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pack: %w", err)
		}

		packs = append(packs, pack)
	}

	return packs, nil
}

// Checklist Functions

// GetChecklistItems returns all checklist items for a trip
func GetChecklistItems(db *sql.DB, tripID string) ([]models.TripChecklistItem, error) {
	query := `
		SELECT id, trip_id, content, is_checked, sort_order, created_at, updated_at
		FROM trip_checklist_items
		WHERE trip_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := db.Query(query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query checklist items: %w", err)
	}
	defer rows.Close()

	var items []models.TripChecklistItem
	for rows.Next() {
		var item models.TripChecklistItem
		err := rows.Scan(
			&item.ID, &item.TripID, &item.Content,
			&item.IsChecked, &item.SortOrder,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checklist item: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}

// AddChecklistItem adds a new checklist item to a trip
func AddChecklistItem(db *sql.DB, tripID string, content string, userID int) (*models.TripChecklistItem, error) {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Get max sort_order
	var maxSortOrder int
	err = db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM trip_checklist_items WHERE trip_id = ?", tripID).Scan(&maxSortOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sort order: %w", err)
	}

	query := `
		INSERT INTO trip_checklist_items (trip_id, content, is_checked, sort_order)
		VALUES (?, ?, FALSE, ?)
	`

	result, err := db.Exec(query, tripID, content, maxSortOrder+1)
	if err != nil {
		return nil, fmt.Errorf("failed to add checklist item: %w", err)
	}

	itemID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	item := &models.TripChecklistItem{
		ID:        int(itemID),
		TripID:    tripID,
		Content:   content,
		IsChecked: false,
		SortOrder: maxSortOrder + 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return item, nil
}

// UpdateChecklistItem updates a checklist item's content and checked status
func UpdateChecklistItem(db *sql.DB, itemID int, content string, isChecked bool, userID int) error {
	// Verify ownership via trip
	var tripOwnerID int
	err := db.QueryRow(`
		SELECT t.user_id
		FROM trips t
		INNER JOIN trip_checklist_items tci ON t.id = tci.trip_id
		WHERE tci.id = ?
	`, itemID).Scan(&tripOwnerID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		UPDATE trip_checklist_items
		SET content = ?, is_checked = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, content, isChecked, itemID)
	if err != nil {
		return fmt.Errorf("failed to update checklist item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("checklist item not found")
	}

	return nil
}

// ToggleChecklistItem toggles the checked status of a checklist item
func ToggleChecklistItem(db *sql.DB, itemID int, userID int) error {
	// Verify ownership via trip
	var tripOwnerID int
	var tripID string
	err := db.QueryRow(`
		SELECT t.user_id, t.id
		FROM trips t
		INNER JOIN trip_checklist_items tci ON t.id = tci.trip_id
		WHERE tci.id = ?
	`, itemID).Scan(&tripOwnerID, &tripID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		UPDATE trip_checklist_items
		SET is_checked = NOT is_checked, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.Exec(query, itemID)
	if err != nil {
		return fmt.Errorf("failed to toggle checklist item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("checklist item not found")
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// DeleteChecklistItem deletes a checklist item
func DeleteChecklistItem(db *sql.DB, itemID int, userID int) error {
	// Verify ownership and get trip_id
	var tripOwnerID int
	var tripID string
	err := db.QueryRow(`
		SELECT t.user_id, t.id
		FROM trips t
		INNER JOIN trip_checklist_items tci ON t.id = tci.trip_id
		WHERE tci.id = ?
	`, itemID).Scan(&tripOwnerID, &tripID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `DELETE FROM trip_checklist_items WHERE id = ?`

	result, err := db.Exec(query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete checklist item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("checklist item not found")
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// ReorderChecklistItems updates the sort order of checklist items
func ReorderChecklistItems(db *sql.DB, tripID string, itemIDs []int, userID int) error {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		return fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Update sort order for each item
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `UPDATE trip_checklist_items SET sort_order = ? WHERE id = ? AND trip_id = ?`

	for i, itemID := range itemIDs {
		_, err := tx.Exec(query, i, itemID, tripID)
		if err != nil {
			return fmt.Errorf("failed to update sort order: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// Transport Timeline Functions

// GetTransportSteps returns all transport steps for a trip
func GetTransportSteps(db *sql.DB, tripID string) ([]models.TripTransportStep, error) {
	query := `
		SELECT id, trip_id, journey_type, step_order, departure_place,
		       departure_datetime, arrival_place, arrival_datetime, transport_type, transport_number, notes, created_at
		FROM trip_transport_steps
		WHERE trip_id = ?
		ORDER BY journey_type, step_order ASC
	`

	rows, err := db.Query(query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transport steps: %w", err)
	}
	defer rows.Close()

	var steps []models.TripTransportStep
	for rows.Next() {
		var step models.TripTransportStep
		var departureDatetime, arrivalDatetime sql.NullTime
		var arrivalPlace, transportType, transportNumber, notes sql.NullString

		err := rows.Scan(
			&step.ID, &step.TripID, &step.JourneyType, &step.StepOrder,
			&step.DeparturePlace, &departureDatetime, &arrivalPlace, &arrivalDatetime, &transportType, &transportNumber, &notes,
			&step.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transport step: %w", err)
		}

		// Handle nullable fields
		if departureDatetime.Valid {
			step.DepartureDatetime = &departureDatetime.Time
		}
		if arrivalPlace.Valid {
			step.ArrivalPlace = &arrivalPlace.String
		}
		if arrivalDatetime.Valid {
			step.ArrivalDatetime = &arrivalDatetime.Time
		}
		if transportType.Valid {
			step.TransportType = &transportType.String
		}
		if transportNumber.Valid {
			step.TransportNumber = &transportNumber.String
		}
		if notes.Valid {
			step.Notes = &notes.String
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// AddTransportStep adds a new transport step to a trip
func AddTransportStep(db *sql.DB, tripID string, journeyType string, departurePlace string, departureDatetime *time.Time, arrivalPlace *string, arrivalDatetime *time.Time, transportType, transportNumber, notes *string, userID int) (*models.TripTransportStep, error) {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Validate journey_type
	if journeyType != "outbound" && journeyType != "return" {
		return nil, fmt.Errorf("invalid journey_type: must be 'outbound' or 'return'")
	}

	// Get max step_order for this journey type
	var maxStepOrder int
	err = db.QueryRow("SELECT COALESCE(MAX(step_order), -1) FROM trip_transport_steps WHERE trip_id = ? AND journey_type = ?", tripID, journeyType).Scan(&maxStepOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get max step order: %w", err)
	}

	query := `
		INSERT INTO trip_transport_steps (trip_id, journey_type, step_order, departure_place, departure_datetime, arrival_place, arrival_datetime, transport_type, transport_number, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(query, tripID, journeyType, maxStepOrder+1, departurePlace, departureDatetime, arrivalPlace, arrivalDatetime, transportType, transportNumber, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to add transport step: %w", err)
	}

	stepID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	step := &models.TripTransportStep{
		ID:                int(stepID),
		TripID:            tripID,
		JourneyType:       journeyType,
		StepOrder:         maxStepOrder + 1,
		DeparturePlace:    departurePlace,
		DepartureDatetime: departureDatetime,
		ArrivalPlace:      arrivalPlace,
		ArrivalDatetime:   arrivalDatetime,
		TransportType:     transportType,
		TransportNumber:   transportNumber,
		Notes:             notes,
		CreatedAt:         time.Now(),
	}

	return step, nil
}

// UpdateTransportStep updates a transport step
func UpdateTransportStep(db *sql.DB, stepID int, departurePlace string, departureDatetime *time.Time, arrivalPlace *string, arrivalDatetime *time.Time, transportType, transportNumber, notes *string, userID int) error {
	// Verify ownership via trip
	var tripOwnerID int
	var tripID string
	err := db.QueryRow(`
		SELECT t.user_id, t.id
		FROM trips t
		INNER JOIN trip_transport_steps tts ON t.id = tts.trip_id
		WHERE tts.id = ?
	`, stepID).Scan(&tripOwnerID, &tripID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `
		UPDATE trip_transport_steps
		SET departure_place = ?, departure_datetime = ?, arrival_place = ?, arrival_datetime = ?, transport_type = ?, transport_number = ?, notes = ?
		WHERE id = ?
	`

	result, err := db.Exec(query, departurePlace, departureDatetime, arrivalPlace, arrivalDatetime, transportType, transportNumber, notes, stepID)
	if err != nil {
		return fmt.Errorf("failed to update transport step: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transport step not found")
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// DeleteTransportStep deletes a transport step
func DeleteTransportStep(db *sql.DB, stepID int, userID int) error {
	// Verify ownership and get trip_id
	var tripOwnerID int
	var tripID string
	err := db.QueryRow(`
		SELECT t.user_id, t.id
		FROM trips t
		INNER JOIN trip_transport_steps tts ON t.id = tts.trip_id
		WHERE tts.id = ?
	`, stepID).Scan(&tripOwnerID, &tripID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	query := `DELETE FROM trip_transport_steps WHERE id = ?`

	result, err := db.Exec(query, stepID)
	if err != nil {
		return fmt.Errorf("failed to delete transport step: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transport step not found")
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// ReorderTransportSteps updates the step order of transport steps for a journey type
func ReorderTransportSteps(db *sql.DB, tripID, journeyType string, stepIDs []int, userID int) error {
	// Verify trip ownership
	var tripOwnerID int
	err := db.QueryRow("SELECT user_id FROM trips WHERE id = ?", tripID).Scan(&tripOwnerID)
	if err != nil {
		return fmt.Errorf("failed to check trip ownership: %w", err)
	}

	if tripOwnerID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Update step order for each step
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `UPDATE trip_transport_steps SET step_order = ? WHERE id = ? AND trip_id = ? AND journey_type = ?`

	for i, stepID := range stepIDs {
		_, err := tx.Exec(query, i, stepID, tripID, journeyType)
		if err != nil {
			return fmt.Errorf("failed to update step order: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update trip timestamp
	updateTripTimestamp(db, tripID)

	return nil
}

// GPX Functions

// UpdateTripGPX updates the GPX data for a trip
func UpdateTripGPX(db *sql.DB, userID int, tripID string, gpxData string) error {
	query := `
		UPDATE trips
		SET gpx_data = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, gpxData, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to update GPX data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}

// DeleteTripGPX removes the GPX data from a trip
func DeleteTripGPX(db *sql.DB, userID int, tripID string) error {
	query := `
		UPDATE trips
		SET gpx_data = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`

	result, err := db.Exec(query, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete GPX data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found or unauthorized")
	}

	return nil
}
