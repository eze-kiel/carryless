package handlers

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"carryless/internal/database"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

func handleInventory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	items, err := database.GetItems(db, userID)
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
	categoryIDStr := c.PostForm("category_id")
	weightStr := c.PostForm("weight_grams")
	priceStr := c.PostForm("price")
	categories, _ := database.GetCategories(db, userID)

	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Item name is required"
	}
	if len(name) > 200 {
		errors["name"] = "Item name must be less than 200 characters"
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		errors["category_id"] = "Invalid category"
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

	_, err = database.GetCategory(db, userID, categoryID)
	if err != nil {
		c.HTML(http.StatusBadRequest, "new_item.html", gin.H{
			"Title":      "New Item - Carryless",
			"User":       user,
			"Categories": categories,
			"Error":      "Category not found",
		})
		return
	}

	item := models.Item{
		CategoryID:   categoryID,
		Name:         name,
		Note:         note,
		WeightGrams:  weightGrams,
		Price:        price,
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
	categoryIDStr := c.PostForm("category_id")
	weightStr := c.PostForm("weight_grams")
	priceStr := c.PostForm("price")

	categories, _ := database.GetCategories(db, userID)
	currentItem, _ := database.GetItem(db, userID, itemID)

	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Item name is required"
	}
	if len(name) > 200 {
		errors["name"] = "Item name must be less than 200 characters"
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		errors["category_id"] = "Invalid category"
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

	_, err = database.GetCategory(db, userID, categoryID)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_item.html", gin.H{
			"Title":      "Edit Item - Carryless",
			"User":       user,
			"Item":       currentItem,
			"Categories": categories,
			"Error":      "Category not found",
		})
		return
	}

	item := models.Item{
		CategoryID:   categoryID,
		Name:         name,
		Note:         note,
		WeightGrams:  weightGrams,
		Price:        price,
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
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	err = database.DeleteItem(db, userID, itemID)
	if err != nil {
		c.Redirect(http.StatusFound, "/inventory")
		return
	}

	c.Redirect(http.StatusFound, "/inventory")
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
	
	// Write header
	header := []string{"Name", "Category", "Weight (grams)", "Price", "Note"}
	if err := writer.Write(header); err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate CSV")
		return
	}

	// Write items
	for _, item := range items {
		record := []string{
			item.Name,
			item.Category.Name,
			strconv.Itoa(item.WeightGrams),
			fmt.Sprintf("%.2f", item.Price),
			item.Note,
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
	reader.FieldsPerRecord = 5 // Expect exactly 5 fields

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

		// Validate and sanitize record
		if len(record) != 5 {
			return nil, fmt.Errorf("invalid number of fields at line %d", lineNumber)
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

		items = append(items, item)
	}

	return items, nil
}