package handlers

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"carryless/internal/database"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

// Valid capacity units for items
var validCapacityUnits = map[string]bool{
	"mL":    true,
	"L":     true,
	"fl-oz": true,
	"mAh":   true,
}

// isValidCapacityUnit checks if the given unit is valid
func isValidCapacityUnit(unit string) bool {
	return validCapacityUnits[unit]
}

// isValidURL checks if the given string is a valid http/https URL
func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return true
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func handleInventory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	// Check if filtering for verification items
	verifyOnly := c.Query("verify_only") == "true"
	
	var items []models.Item
	var err error
	
	if verifyOnly {
		items, err = database.GetItemsToVerify(db, userID)
	} else {
		items, err = database.GetItems(db, userID)
	}
	
	if err != nil {
		c.HTML(http.StatusInternalServerError, "inventory.html", gin.H{
			"Title": "Inventory - Carryless",
			"User":  user,
			"Error": "Failed to load inventory",
		})
		return
	}

	categories, err := database.GetCategories(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "inventory.html", gin.H{
			"Title": "Inventory - Carryless",
			"User":  user,
			"Error": "Failed to load categories",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "inventory.html", gin.H{
			"Title": "Inventory - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "inventory.html", gin.H{
		"Title":      "Inventory - Carryless",
		"User":       user,
		"Items":      items,
		"Categories": categories,
		"CSRFToken":  csrfToken.Token,
		"VerifyOnly": verifyOnly,
	})
}

func handleNewItemPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	categories, err := database.GetCategories(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_item.html", gin.H{
			"Title": "New Item - Carryless",
			"User":  user,
			"Error": "Failed to load categories",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_item.html", gin.H{
			"Title": "New Item - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "new_item.html", gin.H{
		"Title":      "New Item - Carryless",
		"User":       user,
		"Categories": categories,
		"CSRFToken":  csrfToken.Token,
	})
}

func handleCreateItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	name := strings.TrimSpace(c.PostForm("name"))
	note := strings.TrimSpace(c.PostForm("note"))
	categoryName := strings.TrimSpace(c.PostForm("category_name"))
	weightStr := c.PostForm("weight_grams")
	priceStr := c.PostForm("price")
	weightToVerify := c.PostForm("weight_to_verify") == "on"

	// New optional fields
	brand := strings.TrimSpace(c.PostForm("brand"))
	model := strings.TrimSpace(c.PostForm("model"))
	purchaseDateStr := c.PostForm("purchase_date")
	capacityStr := c.PostForm("capacity")
	capacityUnit := c.PostForm("capacity_unit")
	link := strings.TrimSpace(c.PostForm("link"))

	categories, _ := database.GetCategories(db, userID)

	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Item name is required"
	}
	if len(name) > 200 {
		errors["name"] = "Item name must be less than 200 characters"
	}

	if categoryName == "" {
		errors["category_name"] = "Category is required"
	}
	if len(categoryName) > 100 {
		errors["category_name"] = "Category name must be less than 100 characters"
	}

	weightGrams, err := strconv.Atoi(weightStr)
	if err != nil || weightGrams < 0 {
		errors["weight_grams"] = "Weight must be a positive number"
	}

	price := 0.0
	if priceStr != "" {
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil || price < 0 {
			errors["price"] = "Price must be a positive number"
		}
	}

	// Validate new optional fields
	if len(brand) > 100 {
		errors["brand"] = "Brand must be less than 100 characters"
	}
	if len(model) > 100 {
		errors["model"] = "Model must be less than 100 characters"
	}

	var purchaseDatePtr *time.Time
	if purchaseDateStr != "" {
		t, err := time.Parse("2006-01-02", purchaseDateStr)
		if err != nil {
			errors["purchase_date"] = "Invalid date format"
		} else {
			purchaseDatePtr = &t
		}
	}

	var capacityPtr *float64
	var capacityUnitPtr *string
	if capacityStr != "" {
		cap, err := strconv.ParseFloat(capacityStr, 64)
		if err != nil || cap < 0 {
			errors["capacity"] = "Capacity must be a positive number"
		} else {
			capacityPtr = &cap
			if capacityUnit == "" {
				errors["capacity_unit"] = "Unit is required when capacity is specified"
			} else if !isValidCapacityUnit(capacityUnit) {
				errors["capacity_unit"] = "Invalid capacity unit"
			} else {
				capacityUnitPtr = &capacityUnit
			}
		}
	}

	if link != "" {
		if len(link) > 500 {
			errors["link"] = "Link must be less than 500 characters"
		} else if !isValidURL(link) {
			errors["link"] = "Invalid URL format (must start with http:// or https://)"
		}
	}

	if len(errors) > 0 {
		errorMsg := ""
		for _, v := range errors {
			errorMsg = v
			break
		}
		c.HTML(http.StatusBadRequest, "new_item.html", gin.H{
			"Title":      "New Item - Carryless",
			"User":       user,
			"Categories": categories,
			"Error":      errorMsg,
		})
		return
	}

	// Get or create the category
	category, err := database.GetOrCreateCategory(db, userID, categoryName)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new_item.html", gin.H{
			"Title":      "New Item - Carryless",
			"User":       user,
			"Categories": categories,
			"Error":      "Failed to create or find category",
		})
		return
	}

	// Build optional field pointers
	var brandPtr *string
	if brand != "" {
		brandPtr = &brand
	}
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}
	var linkPtr *string
	if link != "" {
		linkPtr = &link
	}

	item := models.Item{
		CategoryID:     category.ID,
		Name:           name,
		Note:           note,
		WeightGrams:    weightGrams,
		WeightToVerify: weightToVerify,
		Price:          price,
		Brand:          brandPtr,
		Model:          modelPtr,
		PurchaseDate:   purchaseDatePtr,
		Capacity:       capacityPtr,
		CapacityUnit:   capacityUnitPtr,
		Link:           linkPtr,
	}

	_, err = database.CreateItem(db, userID, item)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_item.html", gin.H{
			"Title":      "New Item - Carryless",
			"User":       user,
			"Categories": categories,
			"Error":      "Failed to create item",
		})
		return
	}

	c.Redirect(http.StatusFound, "/inventory")
}

func handleEditItemPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title": "Edit Item - Carryless",
			"User":  user,
			"Error": "Invalid item ID",
		})
		return
	}

	item, err := database.GetItem(db, userID, itemID)
	if err != nil {
		c.HTML(http.StatusNotFound, "edit_item.html", gin.H{
			"Title": "Edit Item - Carryless",
			"User":  user,
			"Error": "Item not found",
		})
		return
	}

	categories, err := database.GetCategories(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_item.html", gin.H{
			"Title": "Edit Item - Carryless",
			"User":  user,
			"Error": "Failed to load categories",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_item.html", gin.H{
			"Title": "Edit Item - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "edit_item.html", gin.H{
		"Title":      "Edit Item - Carryless",
		"User":       user,
		"Item":       item,
		"Categories": categories,
		"CSRFToken":  csrfToken.Token,
	})
}

func handleUpdateItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title": "Edit Item - Carryless",
			"User":  user,
			"Error": "Invalid item ID",
		})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	note := strings.TrimSpace(c.PostForm("note"))
	categoryName := strings.TrimSpace(c.PostForm("category_name"))
	weightStr := c.PostForm("weight_grams")
	priceStr := c.PostForm("price")

	weightToVerify := c.PostForm("weight_to_verify") == "on"

	// New optional fields
	brand := strings.TrimSpace(c.PostForm("brand"))
	model := strings.TrimSpace(c.PostForm("model"))
	purchaseDateStr := c.PostForm("purchase_date")
	capacityStr := c.PostForm("capacity")
	capacityUnit := c.PostForm("capacity_unit")
	link := strings.TrimSpace(c.PostForm("link"))

	categories, _ := database.GetCategories(db, userID)
	currentItem, _ := database.GetItem(db, userID, itemID)

	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Item name is required"
	}
	if len(name) > 200 {
		errors["name"] = "Item name must be less than 200 characters"
	}

	if categoryName == "" {
		errors["category_name"] = "Category is required"
	}
	if len(categoryName) > 100 {
		errors["category_name"] = "Category name must be less than 100 characters"
	}

	weightGrams, err := strconv.Atoi(weightStr)
	if err != nil || weightGrams < 0 {
		errors["weight_grams"] = "Weight must be a positive number"
	}

	price := 0.0
	if priceStr != "" {
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil || price < 0 {
			errors["price"] = "Price must be a positive number"
		}
	}

	// Validate new optional fields
	if len(brand) > 100 {
		errors["brand"] = "Brand must be less than 100 characters"
	}
	if len(model) > 100 {
		errors["model"] = "Model must be less than 100 characters"
	}

	var purchaseDatePtr *time.Time
	if purchaseDateStr != "" {
		t, err := time.Parse("2006-01-02", purchaseDateStr)
		if err != nil {
			errors["purchase_date"] = "Invalid date format"
		} else {
			purchaseDatePtr = &t
		}
	}

	var capacityPtr *float64
	var capacityUnitPtr *string
	if capacityStr != "" {
		cap, err := strconv.ParseFloat(capacityStr, 64)
		if err != nil || cap < 0 {
			errors["capacity"] = "Capacity must be a positive number"
		} else {
			capacityPtr = &cap
			if capacityUnit == "" {
				errors["capacity_unit"] = "Unit is required when capacity is specified"
			} else if !isValidCapacityUnit(capacityUnit) {
				errors["capacity_unit"] = "Invalid capacity unit"
			} else {
				capacityUnitPtr = &capacityUnit
			}
		}
	}

	if link != "" {
		if len(link) > 500 {
			errors["link"] = "Link must be less than 500 characters"
		} else if !isValidURL(link) {
			errors["link"] = "Invalid URL format (must start with http:// or https://)"
		}
	}

	if len(errors) > 0 {
		errorMsg := ""
		for _, v := range errors {
			errorMsg = v
			break
		}
		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title":      "Edit Item - Carryless",
			"User":       user,
			"Item":       currentItem,
			"Categories": categories,
			"Error":      errorMsg,
		})
		return
	}

	// Get or create the category
	category, err := database.GetOrCreateCategory(db, userID, categoryName)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title":      "Edit Item - Carryless",
			"User":       user,
			"Item":       currentItem,
			"Categories": categories,
			"Error":      "Failed to create or find category",
		})
		return
	}

	// Build optional field pointers
	var brandPtr *string
	if brand != "" {
		brandPtr = &brand
	}
	var modelPtr *string
	if model != "" {
		modelPtr = &model
	}
	var linkPtr *string
	if link != "" {
		linkPtr = &link
	}

	item := models.Item{
		CategoryID:     category.ID,
		Name:           name,
		Note:           note,
		WeightGrams:    weightGrams,
		WeightToVerify: weightToVerify,
		Price:          price,
		Brand:          brandPtr,
		Model:          modelPtr,
		PurchaseDate:   purchaseDatePtr,
		Capacity:       capacityPtr,
		CapacityUnit:   capacityUnitPtr,
		Link:           linkPtr,
	}

	err = database.UpdateItem(db, userID, itemID, item)
	if err != nil {
		var errorMsg string
		if strings.Contains(err.Error(), "not found") {
			errorMsg = "Item not found"
		} else {
			errorMsg = "Failed to update item"
		}

		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title":      "Edit Item - Carryless",
			"User":       user,
			"Item":       currentItem,
			"Categories": categories,
			"Error":      errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/inventory")
}

func handleDeleteItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		fmt.Printf("[DEBUG] Delete item failed - Invalid ID: %s, error: %v\n", itemIDStr, err)
		c.Redirect(http.StatusFound, "/inventory?error=invalid_id")
		return
	}

	// Check if this is a force delete request
	force := c.PostForm("force") == "true"
	
	fmt.Printf("[DEBUG] Attempting to delete item ID: %d for user ID: %d (force: %v)\n", itemID, userID, force)

	if force {
		// Force delete - remove from packs and delete item
		err = database.DeleteItemWithForce(db, userID, itemID, true)
		if err != nil {
			fmt.Printf("[DEBUG] Force delete item failed - ID: %d, error: %v\n", itemID, err)
			if strings.Contains(err.Error(), "item not found") {
				c.Redirect(http.StatusFound, "/inventory?error=item_not_found")
			} else {
				c.Redirect(http.StatusFound, "/inventory?error=delete_failed")
			}
			return
		}
		fmt.Printf("[DEBUG] Successfully force deleted item ID: %d\n", itemID)
		c.Redirect(http.StatusFound, "/inventory?success=deleted")
		return
	}

	// Regular delete - check for packs first
	err = database.DeleteItem(db, userID, itemID)
	if err != nil {
		fmt.Printf("[DEBUG] Delete item failed - ID: %d, error: %v\n", itemID, err)
		if strings.Contains(err.Error(), "cannot delete item used in") {
			c.Redirect(http.StatusFound, "/inventory?error=item_in_use")
		} else if strings.Contains(err.Error(), "item not found") {
			c.Redirect(http.StatusFound, "/inventory?error=item_not_found")
		} else {
			c.Redirect(http.StatusFound, "/inventory?error=delete_failed")
		}
		return
	}

	fmt.Printf("[DEBUG] Successfully deleted item ID: %d\n", itemID)
	c.Redirect(http.StatusFound, "/inventory?success=deleted")
}

func handleDuplicateItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=invalid_id")
		return
	}

	// Duplicate the item
	_, err = database.DuplicateItem(db, userID, itemID)
	if err != nil {
		fmt.Printf("[DEBUG] Duplicate item failed - ID: %d, error: %v\n", itemID, err)
		if strings.Contains(err.Error(), "not found") {
			c.Redirect(http.StatusFound, "/inventory?error=item_not_found")
		} else {
			c.Redirect(http.StatusFound, "/inventory?error=duplicate_failed")
		}
		return
	}

	c.Redirect(http.StatusFound, "/inventory?success=duplicated")
}

func handleCheckItemPacks(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Get packs using this item
	packNames, err := database.GetPacksUsingItem(db, userID, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check packs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"packs": packNames})
}

func handleExportInventory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	items, err := database.GetItems(db, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load inventory")
		return
	}

	// Create CSV content
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header (extended with new fields)
	header := []string{"Name", "Category", "Weight (grams)", "Price", "Description", "Brand", "Model", "Purchased", "Capacity", "Capacity Unit", "Link"}
	if err := writer.Write(header); err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate CSV")
		return
	}

	// Write items
	for _, item := range items {
		// Convert optional fields to strings
		brandStr := ""
		if item.Brand != nil {
			brandStr = *item.Brand
		}
		modelStr := ""
		if item.Model != nil {
			modelStr = *item.Model
		}
		purchaseDateStr := ""
		if item.PurchaseDate != nil {
			purchaseDateStr = item.PurchaseDate.Format("2006-01-02")
		}
		capacityStr := ""
		if item.Capacity != nil {
			capacityStr = fmt.Sprintf("%.2f", *item.Capacity)
		}
		capacityUnitStr := ""
		if item.CapacityUnit != nil {
			capacityUnitStr = *item.CapacityUnit
		}
		linkStr := ""
		if item.Link != nil {
			linkStr = *item.Link
		}

		record := []string{
			item.Name,
			item.Category.Name,
			strconv.Itoa(item.WeightGrams),
			fmt.Sprintf("%.2f", item.Price),
			item.Note,
			brandStr,
			modelStr,
			purchaseDateStr,
			capacityStr,
			capacityUnitStr,
			linkStr,
		}
		if err := writer.Write(record); err != nil {
			c.String(http.StatusInternalServerError, "Failed to generate CSV")
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate CSV")
		return
	}

	// Set headers for download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=inventory.csv")
	c.Data(http.StatusOK, "text/csv", buf.Bytes())
}

func handleImportInventory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	// Validate file upload
	file, header, err := c.Request.FormFile("csvFile")
	if err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=no_file")
		return
	}
	defer file.Close()

	// Security validations
	if err := validateCSVFile(file, header); err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=invalid_file")
		return
	}

	// Reset file position after validation
	file.Seek(0, 0)

	// Parse CSV
	items, err := parseCSVFile(file, db, userID)
	if err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=parse_error")
		return
	}

	// Begin transaction for atomic operation
	tx, err := db.Begin()
	if err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=database_error")
		return
	}
	defer tx.Rollback()

	// Delete all existing items
	if err := database.DeleteAllItems(db, userID); err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=delete_error")
		return
	}

	// Insert new items
	for _, item := range items {
		if _, err := database.CreateItem(db, userID, item); err != nil {
			c.Redirect(http.StatusFound, "/inventory?error=import_error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.Redirect(http.StatusFound, "/inventory?error=commit_error")
		return
	}

	c.Redirect(http.StatusFound, "/inventory?success=imported")
}

func validateCSVFile(file multipart.File, header *multipart.FileHeader) error {
	// Check file size (max 10MB)
	if header.Size > 10*1024*1024 {
		return fmt.Errorf("file too large")
	}

	// Validate filename extension
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".csv") {
		return fmt.Errorf("invalid file extension")
	}

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return fmt.Errorf("cannot read file")
	}

	// Check MIME type (should be text/plain or text/csv)
	contentType := http.DetectContentType(buffer)
	if !strings.HasPrefix(contentType, "text/") {
		return fmt.Errorf("invalid file type: %s", contentType)
	}

	// Additional CSV validation - check for common CSV patterns
	content := string(buffer)
	if !strings.Contains(content, ",") && !strings.Contains(content, "\n") {
		return fmt.Errorf("file does not appear to be CSV format")
	}

	return nil
}

func parseCSVFile(file multipart.File, db *sql.DB, userID int) ([]models.Item, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields for backward compatibility

	var items []models.Item
	lineNumber := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV parse error at line %d: %v", lineNumber, err)
		}

		lineNumber++

		// Skip header row
		if lineNumber == 1 {
			continue
		}

		// Limit total rows to prevent DoS
		if lineNumber > 10000 {
			return nil, fmt.Errorf("too many rows (max 10000)")
		}

		// Validate field count (5 = old format, 10 = legacy format with brand, 11 = new format with model)
		if len(record) != 5 && len(record) != 10 && len(record) != 11 {
			return nil, fmt.Errorf("invalid number of fields at line %d (expected 5, 10, or 11, got %d)", lineNumber, len(record))
		}

		name := strings.TrimSpace(record[0])
		categoryName := strings.TrimSpace(record[1])
		weightStr := strings.TrimSpace(record[2])
		priceStr := strings.TrimSpace(record[3])
		note := strings.TrimSpace(record[4])

		// Validate required fields
		if name == "" || categoryName == "" {
			return nil, fmt.Errorf("empty required field at line %d", lineNumber)
		}

		// Validate field lengths
		if len(name) > 255 || len(categoryName) > 100 || len(note) > 1000 {
			return nil, fmt.Errorf("field too long at line %d", lineNumber)
		}

		// Parse weight
		weight, err := strconv.Atoi(weightStr)
		if err != nil || weight < 0 || weight > 100000 {
			return nil, fmt.Errorf("invalid weight at line %d", lineNumber)
		}

		// Parse price
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price < 0 || price > 100000 {
			return nil, fmt.Errorf("invalid price at line %d", lineNumber)
		}

		// Find or create category
		category, err := database.GetOrCreateCategory(db, userID, categoryName)
		if err != nil {
			return nil, fmt.Errorf("failed to get/create category at line %d", lineNumber)
		}

		item := models.Item{
			Name:        name,
			CategoryID:  category.ID,
			WeightGrams: weight,
			Price:       price,
			Note:        note,
		}

		// Parse new optional fields if present (10-field or 11-field format)
		if len(record) >= 10 {
			// Brand (index 5)
			brand := strings.TrimSpace(record[5])
			if brand != "" {
				if len(brand) > 100 {
					return nil, fmt.Errorf("brand too long at line %d", lineNumber)
				}
				item.Brand = &brand
			}

			// Handle 11-field format (with Model) vs 10-field legacy format
			var purchaseDateIdx, capacityIdx, capacityUnitIdx, linkIdx int
			if len(record) == 11 {
				// Model (index 6) - new format
				modelStr := strings.TrimSpace(record[6])
				if modelStr != "" {
					if len(modelStr) > 100 {
						return nil, fmt.Errorf("model too long at line %d", lineNumber)
					}
					item.Model = &modelStr
				}
				purchaseDateIdx = 7
				capacityIdx = 8
				capacityUnitIdx = 9
				linkIdx = 10
			} else {
				// 10-field legacy format (no Model)
				purchaseDateIdx = 6
				capacityIdx = 7
				capacityUnitIdx = 8
				linkIdx = 9
			}

			// Purchase date
			purchaseDateStr := strings.TrimSpace(record[purchaseDateIdx])
			if purchaseDateStr != "" {
				t, err := time.Parse("2006-01-02", purchaseDateStr)
				if err != nil {
					return nil, fmt.Errorf("invalid purchase date format at line %d (expected YYYY-MM-DD)", lineNumber)
				}
				item.PurchaseDate = &t
			}

			// Capacity and Capacity Unit
			capacityStr := strings.TrimSpace(record[capacityIdx])
			capacityUnitStr := strings.TrimSpace(record[capacityUnitIdx])
			if capacityStr != "" {
				cap, err := strconv.ParseFloat(capacityStr, 64)
				if err != nil || cap < 0 {
					return nil, fmt.Errorf("invalid capacity at line %d", lineNumber)
				}
				item.Capacity = &cap
				if capacityUnitStr != "" {
					if !isValidCapacityUnit(capacityUnitStr) {
						return nil, fmt.Errorf("invalid capacity unit at line %d (must be mL, L, fl-oz, or mAh)", lineNumber)
					}
					item.CapacityUnit = &capacityUnitStr
				}
			}

			// Link
			linkStr := strings.TrimSpace(record[linkIdx])
			if linkStr != "" {
				if len(linkStr) > 500 {
					return nil, fmt.Errorf("link too long at line %d", lineNumber)
				}
				if !isValidURL(linkStr) {
					return nil, fmt.Errorf("invalid URL format at line %d", lineNumber)
				}
				item.Link = &linkStr
			}
		}

		items = append(items, item)
	}

	return items, nil
}

func handleBulkEditItems(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	// Parse item IDs (comma-separated)
	itemIDsStr := c.PostForm("item_ids")
	if itemIDsStr == "" {
		c.Redirect(http.StatusFound, "/inventory?error=no_items_selected")
		return
	}

	itemIDs, err := parseItemIDs(itemIDsStr)
	if err != nil || len(itemIDs) == 0 {
		c.Redirect(http.StatusFound, "/inventory?error=invalid_item_ids")
		return
	}

	// Build updates map - only include fields where apply_X is checked
	updates := make(map[string]interface{})

	// Category
	if c.PostForm("apply_category") == "1" {
		categoryName := strings.TrimSpace(c.PostForm("category_name"))
		if categoryName == "" {
			c.Redirect(http.StatusFound, "/inventory?error=category_required")
			return
		}
		if len(categoryName) > 100 {
			c.Redirect(http.StatusFound, "/inventory?error=category_too_long")
			return
		}
		category, err := database.GetOrCreateCategory(db, userID, categoryName)
		if err != nil {
			c.Redirect(http.StatusFound, "/inventory?error=category_error")
			return
		}
		updates["category_id"] = category.ID
	}

	// Brand
	if c.PostForm("apply_brand") == "1" {
		brand := strings.TrimSpace(c.PostForm("brand"))
		if brand == "" {
			updates["brand"] = nil // Clear the field
		} else {
			if len(brand) > 100 {
				c.Redirect(http.StatusFound, "/inventory?error=brand_too_long")
				return
			}
			updates["brand"] = brand
		}
	}

	// Model
	if c.PostForm("apply_model") == "1" {
		model := strings.TrimSpace(c.PostForm("model"))
		if model == "" {
			updates["model"] = nil // Clear the field
		} else {
			if len(model) > 100 {
				c.Redirect(http.StatusFound, "/inventory?error=model_too_long")
				return
			}
			updates["model"] = model
		}
	}

	// Note/Description
	if c.PostForm("apply_note") == "1" {
		note := strings.TrimSpace(c.PostForm("note"))
		updates["note"] = note // Can be empty to clear
	}

	// Weight
	if c.PostForm("apply_weight") == "1" {
		weightStr := c.PostForm("weight_grams")
		weight, err := strconv.Atoi(weightStr)
		if err != nil || weight < 0 {
			c.Redirect(http.StatusFound, "/inventory?error=invalid_weight")
			return
		}
		updates["weight_grams"] = weight
	}

	// Weight needs verification
	if c.PostForm("apply_weight_to_verify") == "1" {
		weightToVerify := c.PostForm("weight_to_verify") == "1"
		updates["weight_to_verify"] = weightToVerify
	}

	// Capacity and Capacity Unit
	if c.PostForm("apply_capacity") == "1" {
		capacityStr := c.PostForm("capacity")
		capacityUnit := c.PostForm("capacity_unit")

		if capacityStr == "" {
			updates["capacity"] = nil
			updates["capacity_unit"] = nil
		} else {
			cap, err := strconv.ParseFloat(capacityStr, 64)
			if err != nil || cap < 0 {
				c.Redirect(http.StatusFound, "/inventory?error=invalid_capacity")
				return
			}
			updates["capacity"] = cap

			if capacityUnit == "" {
				c.Redirect(http.StatusFound, "/inventory?error=capacity_unit_required")
				return
			}
			if !isValidCapacityUnit(capacityUnit) {
				c.Redirect(http.StatusFound, "/inventory?error=invalid_capacity_unit")
				return
			}
			updates["capacity_unit"] = capacityUnit
		}
	}

	// Price
	if c.PostForm("apply_price") == "1" {
		priceStr := c.PostForm("price")
		if priceStr == "" {
			updates["price"] = 0.0
		} else {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil || price < 0 {
				c.Redirect(http.StatusFound, "/inventory?error=invalid_price")
				return
			}
			updates["price"] = price
		}
	}

	// Purchase Date
	if c.PostForm("apply_purchase_date") == "1" {
		purchaseDateStr := c.PostForm("purchase_date")
		if purchaseDateStr == "" {
			updates["purchase_date"] = nil
		} else {
			t, err := time.Parse("2006-01-02", purchaseDateStr)
			if err != nil {
				c.Redirect(http.StatusFound, "/inventory?error=invalid_date")
				return
			}
			updates["purchase_date"] = t
		}
	}

	// Link
	if c.PostForm("apply_link") == "1" {
		link := strings.TrimSpace(c.PostForm("link"))
		if link == "" {
			updates["link"] = nil
		} else {
			if len(link) > 500 {
				c.Redirect(http.StatusFound, "/inventory?error=link_too_long")
				return
			}
			if !isValidURL(link) {
				c.Redirect(http.StatusFound, "/inventory?error=invalid_url")
				return
			}
			updates["link"] = link
		}
	}

	// Check if any fields were selected
	if len(updates) == 0 {
		c.Redirect(http.StatusFound, "/inventory?error=no_fields_selected")
		return
	}

	// Call database function
	err = database.BulkUpdateItems(db, userID, itemIDs, updates)
	if err != nil {
		fmt.Printf("[DEBUG] Bulk update failed: %v\n", err)
		c.Redirect(http.StatusFound, "/inventory?error=bulk_update_failed")
		return
	}

	c.Redirect(http.StatusFound, "/inventory?success=bulk_updated")
}

func handleBulkDeleteItems(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	// Parse item IDs (comma-separated)
	itemIDsStr := c.PostForm("item_ids")
	if itemIDsStr == "" {
		c.Redirect(http.StatusFound, "/inventory?error=no_items_selected")
		return
	}

	itemIDs, err := parseItemIDs(itemIDsStr)
	if err != nil || len(itemIDs) == 0 {
		c.Redirect(http.StatusFound, "/inventory?error=invalid_item_ids")
		return
	}

	// Call database function
	deleted, err := database.BulkDeleteItems(db, userID, itemIDs)
	if err != nil {
		fmt.Printf("[DEBUG] Bulk delete failed: %v\n", err)
		c.Redirect(http.StatusFound, "/inventory?error=bulk_delete_failed")
		return
	}

	fmt.Printf("[DEBUG] Bulk deleted %d items\n", deleted)
	c.Redirect(http.StatusFound, "/inventory?success=bulk_deleted")
}

// parseItemIDs parses a comma-separated string of item IDs into a slice of integers
func parseItemIDs(itemIDsStr string) ([]int, error) {
	parts := strings.Split(itemIDsStr, ",")
	itemIDs := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid item ID: %s", part)
		}
		if id <= 0 {
			return nil, fmt.Errorf("invalid item ID: %d", id)
		}
		itemIDs = append(itemIDs, id)
	}

	return itemIDs, nil
}