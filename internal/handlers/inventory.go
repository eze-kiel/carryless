package handlers

import (
	"database/sql"
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