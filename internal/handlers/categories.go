package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

func handleCategories(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	categories, err := database.GetCategories(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "categories.html", gin.H{
			"Title": "Categories - Carryless",
			"User":  user,
			"Error": "Failed to load categories",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "categories.html", gin.H{
			"Title": "Categories - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "categories.html", gin.H{
		"Title":      "Categories - Carryless",
		"User":       user,
		"Categories": categories,
		"CSRFToken":  csrfToken.Token,
	})
}

func handleCreateCategory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	name := strings.TrimSpace(c.PostForm("name"))

	if name == "" {
		c.HTML(http.StatusBadRequest, "new_category.html", gin.H{
			"Title": "New Category - Carryless",
			"User":  user,
			"Error": "Category name is required",
		})
		return
	}

	if len(name) > 100 {
		c.HTML(http.StatusBadRequest, "new_category.html", gin.H{
			"Title": "New Category - Carryless",
			"User":  user,
			"Error": "Category name must be less than 100 characters",
		})
		return
	}

	_, err := database.CreateCategory(db, userID, name)
	if err != nil {
		var errorMsg string
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			errorMsg = "Category already exists"
		} else {
			errorMsg = "Failed to create category"
		}
		c.HTML(http.StatusBadRequest, "new_category.html", gin.H{
			"Title": "New Category - Carryless",
			"User":  user,
			"Error": errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/categories")
}

func handleUpdateCategory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	categoryIDStr := c.Param("id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_category.html", gin.H{
			"Title": "Edit Category - Carryless",
			"User":  user,
			"Error": "Invalid category ID",
		})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))

	if name == "" {
		category, _ := database.GetCategory(db, userID, categoryID)
		c.HTML(http.StatusBadRequest, "edit_category.html", gin.H{
			"Title":    "Edit Category - Carryless",
			"User":     user,
			"Category": category,
			"Error":    "Category name is required",
		})
		return
	}

	if len(name) > 100 {
		category, _ := database.GetCategory(db, userID, categoryID)
		c.HTML(http.StatusBadRequest, "edit_category.html", gin.H{
			"Title":    "Edit Category - Carryless",
			"User":     user,
			"Category": category,
			"Error":    "Category name must be less than 100 characters",
		})
		return
	}

	err = database.UpdateCategory(db, userID, categoryID, name)
	if err != nil {
		var errorMsg string
		if strings.Contains(err.Error(), "not found") {
			errorMsg = "Category not found"
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			errorMsg = "Category name already exists"
		} else {
			errorMsg = "Failed to update category"
		}
		
		category, _ := database.GetCategory(db, userID, categoryID)
		c.HTML(http.StatusBadRequest, "edit_category.html", gin.H{
			"Title":    "Edit Category - Carryless",
			"User":     user,
			"Category": category,
			"Error":    errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/categories")
}

func handleNewCategoryPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_category.html", gin.H{
			"Title": "New Category - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "new_category.html", gin.H{
		"Title":     "New Category - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}

func handleEditCategoryPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	categoryIDStr := c.Param("id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "edit_category.html", gin.H{
			"Title": "Edit Category - Carryless",
			"User":  user,
			"Error": "Invalid category ID",
		})
		return
	}

	category, err := database.GetCategory(db, userID, categoryID)
	if err != nil {
		c.HTML(http.StatusNotFound, "edit_category.html", gin.H{
			"Title": "Edit Category - Carryless",
			"User":  user,
			"Error": "Category not found",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_category.html", gin.H{
			"Title": "Edit Category - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "edit_category.html", gin.H{
		"Title":     "Edit Category - Carryless",
		"User":      user,
		"Category":  category,
		"CSRFToken": csrfToken.Token,
	})
}

func handleDeleteCategory(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	categoryIDStr := c.Param("id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		fmt.Printf("[DEBUG] Delete category failed - Invalid ID: %s, error: %v\n", categoryIDStr, err)
		c.Redirect(http.StatusFound, "/categories?error=invalid_id")
		return
	}

	// Check if this is a force delete request
	force := c.PostForm("force") == "true"
	
	fmt.Printf("[DEBUG] Attempting to delete category ID: %d for user ID: %d (force: %v)\n", categoryID, userID, force)

	if force {
		// Force delete - remove all items and delete category
		err = database.DeleteCategoryWithForce(db, userID, categoryID, true)
		if err != nil {
			fmt.Printf("[DEBUG] Force delete category failed - ID: %d, error: %v\n", categoryID, err)
			if strings.Contains(err.Error(), "category not found") {
				c.Redirect(http.StatusFound, "/categories?error=category_not_found")
			} else {
				c.Redirect(http.StatusFound, "/categories?error=delete_failed")
			}
			return
		}
		fmt.Printf("[DEBUG] Successfully force deleted category ID: %d\n", categoryID)
		c.Redirect(http.StatusFound, "/categories?success=deleted")
		return
	}

	// Regular delete - check for items first
	err = database.DeleteCategory(db, userID, categoryID)
	if err != nil {
		fmt.Printf("[DEBUG] Delete category failed - ID: %d, error: %v\n", categoryID, err)
		if strings.Contains(err.Error(), "cannot delete category with") {
			c.Redirect(http.StatusFound, "/categories?error=category_has_items")
		} else if strings.Contains(err.Error(), "category not found") {
			c.Redirect(http.StatusFound, "/categories?error=category_not_found")
		} else {
			c.Redirect(http.StatusFound, "/categories?error=delete_failed")
		}
		return
	}

	fmt.Printf("[DEBUG] Successfully deleted category ID: %d\n", categoryID)
	c.Redirect(http.StatusFound, "/categories?success=deleted")
}

func handleCheckCategoryItems(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	categoryIDStr := c.Param("id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Get items in this category
	itemNames, err := database.GetItemsInCategory(db, userID, categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": itemNames})
}