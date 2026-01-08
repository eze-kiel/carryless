package handlers

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"carryless/internal/database"
	"carryless/internal/logger"

	"github.com/gin-gonic/gin"
)

// handleTrips displays all trips for a user
func handleTrips(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	trips, err := database.GetTrips(db, userID)
	if err != nil {
		logger.Error("Failed to get trips", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "trips.html", gin.H{
			"Title": "Trips - Carryless",
			"User":  user,
			"Error": "Failed to load trips",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		logger.Error("Failed to create CSRF token", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "trips.html", gin.H{
			"Title": "Trips - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "trips.html", gin.H{
		"Title":     "Trips - Carryless",
		"User":      user,
		"Trips":     trips,
		"CSRFToken": csrfToken.Token,
	})
}

// handleNewTripPage displays the create trip form
func handleNewTripPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		logger.Error("Failed to create CSRF token", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "new_trip.html", gin.H{
			"Title": "New Trip - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "new_trip.html", gin.H{
		"Title":     "New Trip - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}

// handleCreateTrip creates a new trip
func handleCreateTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.HTML(http.StatusBadRequest, "new_trip.html", gin.H{
			"Title": "New Trip - Carryless",
			"User":  c.MustGet("user"),
			"Error": "Trip name is required",
		})
		return
	}

	if len(name) > 200 {
		c.HTML(http.StatusBadRequest, "new_trip.html", gin.H{
			"Title": "New Trip - Carryless",
			"User":  c.MustGet("user"),
			"Error": "Trip name must be 200 characters or less",
		})
		return
	}

	isPublicStr := c.PostForm("is_public")
	isPublic := isPublicStr == "true"

	trip, err := database.CreateTrip(db, userID, name, isPublic)
	if err != nil {
		logger.Error("Failed to create trip", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "new_trip.html", gin.H{
			"Title": "New Trip - Carryless",
			"User":  c.MustGet("user"),
			"Error": "Failed to create trip",
		})
		return
	}

	c.Redirect(http.StatusFound, "/trips/"+trip.ID)
}

// handleTripDetail displays a trip's details
func handleTripDetail(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	tripID := c.Param("id")

	trip, err := database.GetTripWithDetails(db, tripID)
	if err != nil {
		logger.Error("Failed to get trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Trip Not Found - Carryless",
			"User":  user,
		})
		return
	}

	// Check ownership
	if trip.UserID != userID {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	// Get user's packs for the pack selector
	allPacks, err := database.GetPacks(db, userID)
	if err != nil {
		logger.Error("Failed to get packs", "user_id", userID, "error", err)
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		logger.Error("Failed to create CSRF token", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "trip_detail.html", gin.H{
			"Title": "Trip - Carryless",
			"User":  user,
			"Trip":  trip,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "trip_detail.html", gin.H{
		"Title":     trip.Name + " - Carryless",
		"User":      user,
		"Trip":      trip,
		"AllPacks":  allPacks,
		"CSRFToken": csrfToken.Token,
	})
}

// handleEditTripPage displays the edit trip form
func handleEditTripPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	tripID := c.Param("id")

	trip, err := database.GetTrip(db, tripID)
	if err != nil {
		logger.Error("Failed to get trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Trip Not Found - Carryless",
			"User":  user,
		})
		return
	}

	// Check ownership
	if trip.UserID != userID {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		logger.Error("Failed to create CSRF token", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "edit_trip.html", gin.H{
			"Title": "Edit Trip - Carryless",
			"User":  user,
			"Trip":  trip,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "edit_trip.html", gin.H{
		"Title":     "Edit " + trip.Name + " - Carryless",
		"User":      user,
		"Trip":      trip,
		"CSRFToken": csrfToken.Token,
	})
}

// handleUpdateTrip updates a trip
func handleUpdateTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	tripID := c.Param("id")

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.HTML(http.StatusBadRequest, "edit_trip.html", gin.H{
			"Title": "Edit Trip - Carryless",
			"User":  user,
			"Error": "Trip name is required",
		})
		return
	}

	if len(name) > 200 {
		c.HTML(http.StatusBadRequest, "edit_trip.html", gin.H{
			"Title": "Edit Trip - Carryless",
			"User":  user,
			"Error": "Trip name must be 200 characters or less",
		})
		return
	}

	// Parse optional fields
	var description, location *string
	descriptionStr := strings.TrimSpace(c.PostForm("description"))
	if descriptionStr != "" {
		description = &descriptionStr
	}
	locationStr := strings.TrimSpace(c.PostForm("location"))
	if locationStr != "" {
		location = &locationStr
	}

	// Parse dates
	var startDate, endDate *time.Time
	startDateStr := c.PostForm("start_date")
	if startDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			startDate = &parsedDate
		}
	}
	endDateStr := c.PostForm("end_date")
	if endDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = &parsedDate
		}
	}

	isPublicStr := c.PostForm("is_public")
	isPublic := isPublicStr == "true"

	err := database.UpdateTrip(db, userID, tripID, name, description, location, startDate, endDate, isPublic)
	if err != nil {
		logger.Error("Failed to update trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.HTML(http.StatusInternalServerError, "edit_trip.html", gin.H{
			"Title": "Edit Trip - Carryless",
			"User":  user,
			"Error": "Failed to update trip",
		})
		return
	}

	c.Redirect(http.StatusFound, "/trips/"+tripID)
}

// handleDeleteTrip deletes a trip
func handleDeleteTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	err := database.DeleteTrip(db, userID, tripID)
	if err != nil {
		logger.Error("Failed to delete trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete trip"})
		return
	}

	c.Redirect(http.StatusFound, "/trips")
}

// handleArchiveTrip toggles the archive status of a trip
func handleArchiveTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	archiveStr := c.PostForm("is_archived")
	isArchived := archiveStr == "true"

	err := database.ArchiveTrip(db, userID, tripID, isArchived)
	if err != nil {
		logger.Error("Failed to archive trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive trip"})
		return
	}

	c.Redirect(http.StatusFound, "/trips")
}

// Pack Association Handlers

// handleAddPackToTrip adds a pack to a trip
func handleAddPackToTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	packID := c.PostForm("pack_id")
	if packID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pack ID is required"})
		return
	}

	err := database.AddPackToTrip(db, tripID, packID, userID)
	if err != nil {
		logger.Error("Failed to add pack to trip", "user_id", userID, "trip_id", tripID, "pack_id", packID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add pack to trip"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleRemovePackFromTrip removes a pack from a trip
func handleRemovePackFromTrip(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")
	packID := c.Param("pack_id")

	err := database.RemovePackFromTrip(db, tripID, packID, userID)
	if err != nil {
		logger.Error("Failed to remove pack from trip", "user_id", userID, "trip_id", tripID, "pack_id", packID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove pack from trip"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Checklist Handlers

// handleAddChecklistItem adds a checklist item to a trip
func handleAddChecklistItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	var req struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	item, err := database.AddChecklistItem(db, tripID, content, userID)
	if err != nil {
		logger.Error("Failed to add checklist item", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add checklist item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// handleUpdateChecklistItem updates a checklist item
func handleUpdateChecklistItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var req struct {
		Content   string `json:"content"`
		IsChecked bool   `json:"is_checked"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	err = database.UpdateChecklistItem(db, itemID, content, req.IsChecked, userID)
	if err != nil {
		logger.Error("Failed to update checklist item", "user_id", userID, "item_id", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update checklist item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleDeleteChecklistItem deletes a checklist item
func handleDeleteChecklistItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = database.DeleteChecklistItem(db, itemID, userID)
	if err != nil {
		logger.Error("Failed to delete checklist item", "user_id", userID, "item_id", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete checklist item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleToggleChecklistItem toggles a checklist item's checked status
func handleToggleChecklistItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = database.ToggleChecklistItem(db, itemID, userID)
	if err != nil {
		logger.Error("Failed to toggle checklist item", "user_id", userID, "item_id", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle checklist item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleReorderChecklist reorders checklist items
func handleReorderChecklist(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	var req struct {
		ItemIDs []int `json:"item_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := database.ReorderChecklistItems(db, tripID, req.ItemIDs, userID)
	if err != nil {
		logger.Error("Failed to reorder checklist items", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder checklist items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Transport Timeline Handlers

// handleAddTransportStep adds a transport step to a trip
func handleAddTransportStep(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	var req struct {
		JourneyType       string  `json:"journey_type"`
		DeparturePlace    string  `json:"departure_place"`
		DepartureDatetime *string `json:"departure_datetime"`
		ArrivalPlace      *string `json:"arrival_place"`
		ArrivalDatetime   *string `json:"arrival_datetime"`
		TransportType     *string `json:"transport_type"`
		TransportNumber   *string `json:"transport_number"`
		Notes             *string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	departurePlace := strings.TrimSpace(req.DeparturePlace)
	if departurePlace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Departure place is required"})
		return
	}

	if req.JourneyType != "outbound" && req.JourneyType != "return" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid journey type"})
		return
	}

	// Parse datetimes
	var departureDatetime *time.Time
	if req.DepartureDatetime != nil && *req.DepartureDatetime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.DepartureDatetime)
		if err == nil {
			departureDatetime = &parsedTime
		}
	}

	var arrivalDatetime *time.Time
	if req.ArrivalDatetime != nil && *req.ArrivalDatetime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ArrivalDatetime)
		if err == nil {
			arrivalDatetime = &parsedTime
		}
	}

	// Trim arrival place if provided
	var arrivalPlace *string
	if req.ArrivalPlace != nil {
		trimmed := strings.TrimSpace(*req.ArrivalPlace)
		if trimmed != "" {
			arrivalPlace = &trimmed
		}
	}

	step, err := database.AddTransportStep(db, tripID, req.JourneyType, departurePlace, departureDatetime, arrivalPlace, arrivalDatetime, req.TransportType, req.TransportNumber, req.Notes, userID)
	if err != nil {
		logger.Error("Failed to add transport step", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add transport step"})
		return
	}

	c.JSON(http.StatusOK, step)
}

// handleUpdateTransportStep updates a transport step
func handleUpdateTransportStep(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	stepID, err := strconv.Atoi(c.Param("step_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID"})
		return
	}

	var req struct {
		DeparturePlace    string  `json:"departure_place"`
		DepartureDatetime *string `json:"departure_datetime"`
		ArrivalPlace      *string `json:"arrival_place"`
		ArrivalDatetime   *string `json:"arrival_datetime"`
		TransportType     *string `json:"transport_type"`
		TransportNumber   *string `json:"transport_number"`
		Notes             *string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	departurePlace := strings.TrimSpace(req.DeparturePlace)
	if departurePlace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Departure place is required"})
		return
	}

	// Parse datetimes
	var departureDatetime *time.Time
	if req.DepartureDatetime != nil && *req.DepartureDatetime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.DepartureDatetime)
		if err == nil {
			departureDatetime = &parsedTime
		}
	}

	var arrivalDatetime *time.Time
	if req.ArrivalDatetime != nil && *req.ArrivalDatetime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ArrivalDatetime)
		if err == nil {
			arrivalDatetime = &parsedTime
		}
	}

	// Trim arrival place if provided
	var arrivalPlace *string
	if req.ArrivalPlace != nil {
		trimmed := strings.TrimSpace(*req.ArrivalPlace)
		if trimmed != "" {
			arrivalPlace = &trimmed
		}
	}

	err = database.UpdateTransportStep(db, stepID, departurePlace, departureDatetime, arrivalPlace, arrivalDatetime, req.TransportType, req.TransportNumber, req.Notes, userID)
	if err != nil {
		logger.Error("Failed to update transport step", "user_id", userID, "step_id", stepID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transport step"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleDeleteTransportStep deletes a transport step
func handleDeleteTransportStep(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	stepID, err := strconv.Atoi(c.Param("step_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID"})
		return
	}

	err = database.DeleteTransportStep(db, stepID, userID)
	if err != nil {
		logger.Error("Failed to delete transport step", "user_id", userID, "step_id", stepID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transport step"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleReorderTransportSteps reorders transport steps
func handleReorderTransportSteps(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	var req struct {
		JourneyType string `json:"journey_type"`
		StepIDs     []int  `json:"step_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := database.ReorderTransportSteps(db, tripID, req.JourneyType, req.StepIDs, userID)
	if err != nil {
		logger.Error("Failed to reorder transport steps", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder transport steps"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GPX Handlers

// handleUploadGPX uploads a GPX file to a trip
func handleUploadGPX(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	file, err := c.FormFile("gpx_file")
	if err != nil {
		logger.Error("Failed to get file from form", "user_id", userID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Check file size (5MB limit)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".gpx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only .gpx files are allowed"})
		return
	}

	// Read file content
	fileContent, err := file.Open()
	if err != nil {
		logger.Error("Failed to open file", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer fileContent.Close()

	gpxData, err := io.ReadAll(fileContent)
	if err != nil {
		logger.Error("Failed to read file content", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Store GPX data
	err = database.UpdateTripGPX(db, userID, tripID, string(gpxData))
	if err != nil {
		logger.Error("Failed to update trip GPX", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save GPX data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleDeleteGPX deletes GPX data from a trip
func handleDeleteGPX(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	err := database.DeleteTripGPX(db, userID, tripID)
	if err != nil {
		logger.Error("Failed to delete trip GPX", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete GPX data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleDownloadGPX downloads the GPX file for a trip
func handleDownloadGPX(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	// Get trip with GPX data
	trip, err := database.GetTrip(db, tripID)
	if err != nil {
		logger.Error("Failed to get trip", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		return
	}

	// Check ownership
	if trip.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if GPX data exists
	if trip.GPXData == nil || *trip.GPXData == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No GPX data available"})
		return
	}

	// Generate filename from trip name
	filename := trip.Name
	if filename == "" {
		filename = "trip"
	}
	// Sanitize filename
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ToLower(filename) + ".gpx"

	// Set headers for file download
	c.Header("Content-Type", "application/gpx+xml")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/gpx+xml", []byte(*trip.GPXData))
}

// handlePublicDownloadGPX downloads the GPX file for a public trip
func handlePublicDownloadGPX(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	shortID := c.Param("id")

	// Get trip by short ID
	trip, err := database.GetTripByShortID(db, shortID)
	if err != nil {
		logger.Error("Failed to get trip", "short_id", shortID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		return
	}

	// Check if trip is public
	if !trip.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "This trip is not public"})
		return
	}

	// Check if GPX data exists
	if trip.GPXData == nil || *trip.GPXData == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No GPX data available"})
		return
	}

	// Generate filename from trip name
	filename := trip.Name
	if filename == "" {
		filename = "trip"
	}
	// Sanitize filename
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ToLower(filename) + ".gpx"

	// Set headers for file download
	c.Header("Content-Type", "application/gpx+xml")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/gpx+xml", []byte(*trip.GPXData))
}

// handlePublicTripByShortID displays a public trip by its short ID
func handlePublicTripByShortID(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	shortID := c.Param("id")
	user, _ := c.Get("user")

	trip, err := database.GetTripByShortID(db, shortID)
	if err != nil {
		logger.Error("Failed to get public trip", "short_id", shortID, "error", err)
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Trip Not Found - Carryless",
			"User":  user,
		})
		return
	}

	// Load trip details
	tripWithDetails, err := database.GetTripWithDetails(db, trip.ID)
	if err != nil {
		logger.Error("Failed to get trip details", "trip_id", trip.ID, "error", err)
		tripWithDetails = trip
	}

	c.HTML(http.StatusOK, "public_trip.html", gin.H{
		"Title": tripWithDetails.Name + " - Carryless",
		"User":  user,
		"Trip":  tripWithDetails,
	})
}

// handleUpdateTripNotes updates the notes field of a trip (JSON endpoint)
func handleUpdateTripNotes(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	tripID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	notesStr := strings.TrimSpace(req.Notes)
	var notes *string
	if notesStr != "" {
		notes = &notesStr
	}

	err := database.UpdateTripNotes(db, userID, tripID, notes)
	if err != nil {
		logger.Error("Failed to update trip notes", "user_id", userID, "trip_id", tripID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
