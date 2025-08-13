package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

func handlePacks(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	packs, err := database.GetPacks(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to load packs",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "packs.html", gin.H{
			"Title": "Packs - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "packs.html", gin.H{
		"Title":     "Packs - Carryless",
		"User":      user,
		"Packs":     packs,
		"CSRFToken": csrfToken.Token,
	})
}

func handleNewPackPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "new_pack.html", gin.H{
		"Title":     "New Pack - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}

func handleCreatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	name := strings.TrimSpace(c.PostForm("name"))
	isPublicStr := c.PostForm("is_public")

	if name == "" {
		c.HTML(http.StatusBadRequest, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Pack name is required",
		})
		return
	}

	if len(name) > 200 {
		c.HTML(http.StatusBadRequest, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Pack name must be less than 200 characters",
		})
		return
	}

	isPublic := isPublicStr == "true" || isPublicStr == "1"

	_, err := database.CreatePackWithPublic(db, userID, name, isPublic)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_pack.html", gin.H{
			"Title": "New Pack - Carryless",
			"User":  user,
			"Error": "Failed to create pack",
		})
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handlePackDetail(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	userID := c.MustGet("user_id").(int)
	user := c.MustGet("user")

	pack, err := database.GetPackWithItems(db, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"Title": "Pack Not Found - Carryless",
				"User":  user,
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}

	if pack.UserID != userID {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	items, err := database.GetItems(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "pack_detail.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Failed to load available items",
		})
		return
	}

	categoryWeights := make(map[string]int)
	categoryWornWeights := make(map[string]int)
	labelWeights := make(map[string]int)
	labelColors := make(map[string]string)
	itemsInPack := make(map[int]bool)
	totalWeight := 0
	totalWornWeight := 0
	totalItemCount := 0

	for _, packItem := range pack.Items {
		categoryName := packItem.Item.Category.Name
		itemsInPack[packItem.Item.ID] = true
		packWeight := packItem.Item.WeightGrams * (packItem.Count - packItem.WornCount)
		wornWeight := packItem.Item.WeightGrams * packItem.WornCount
		totalItemCount += packItem.Count
		
		if packWeight > 0 {
			categoryWeights[categoryName] += packWeight
			totalWeight += packWeight
		}
		if wornWeight > 0 {
			categoryWornWeights[categoryName] += wornWeight
			totalWornWeight += wornWeight
		}
		
		// Calculate label weights using the actual label assignment counts
		for _, itemLabel := range packItem.Labels {
			labelWeights[itemLabel.PackLabel.Name] += packItem.Item.WeightGrams * itemLabel.Count
			labelColors[itemLabel.PackLabel.Name] = itemLabel.PackLabel.Color
		}
	}

	c.HTML(http.StatusOK, "pack_detail.html", gin.H{
		"Title":               "Pack Detail - Carryless",
		"User":                user,
		"Pack":                pack,
		"Items":               items,
		"ItemsInPack":         itemsInPack,
		"CategoryWeights":     categoryWeights,
		"CategoryWornWeights": categoryWornWeights,
		"LabelWeights":        labelWeights,
		"LabelColors":         labelColors,
		"TotalWeight":         totalWeight,
		"TotalWornWeight":     totalWornWeight,
		"TotalItemCount":      totalItemCount,
	})
}

func handlePublicPack(c *gin.Context) {
	packID := c.Param("id")
	db := c.MustGet("db").(*sql.DB)
	
	user, _ := c.Get("user")

	pack, err := database.GetPackWithItems(db, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"Title": "Pack Not Found - Carryless",
				"User":  user,
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "public_pack.html", gin.H{
			"Title": "Pack Detail - Carryless",
			"User":  user,
			"Error": "Failed to load pack",
		})
		return
	}

	if !pack.IsPublic {
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"Title": "Access Denied - Carryless",
			"User":  user,
		})
		return
	}

	categoryWeights := make(map[string]int)
	categoryWornWeights := make(map[string]int)
	labelWeights := make(map[string]int)
	labelColors := make(map[string]string)
	totalWeight := 0
	totalWornWeight := 0
	totalItemCount := 0

	for _, packItem := range pack.Items {
		categoryName := packItem.Item.Category.Name
		packWeight := packItem.Item.WeightGrams * (packItem.Count - packItem.WornCount)
		wornWeight := packItem.Item.WeightGrams * packItem.WornCount
		totalItemCount += packItem.Count
		
		if packWeight > 0 {
			categoryWeights[categoryName] += packWeight
			totalWeight += packWeight
		}
		if wornWeight > 0 {
			categoryWornWeights[categoryName] += wornWeight
			totalWornWeight += wornWeight
		}
		
		// Calculate label weights using the actual label assignment counts
		for _, itemLabel := range packItem.Labels {
			labelWeights[itemLabel.PackLabel.Name] += packItem.Item.WeightGrams * itemLabel.Count
			labelColors[itemLabel.PackLabel.Name] = itemLabel.PackLabel.Color
		}
	}

	var csrfToken string
	if userID, hasUserID := c.Get("user_id"); hasUserID {
		if token, err := database.CreateCSRFToken(db, userID.(int)); err == nil {
			csrfToken = token.Token
		}
	}

	c.HTML(http.StatusOK, "public_pack.html", gin.H{
		"Title":               pack.Name + " - Carryless",
		"User":                user,
		"Pack":                pack,
		"CategoryWeights":     categoryWeights,
		"CategoryWornWeights": categoryWornWeights,
		"LabelWeights":        labelWeights,
		"LabelColors":         labelColors,
		"TotalWeight":         totalWeight,
		"TotalWornWeight":     totalWornWeight,
		"TotalItemCount":      totalItemCount,
		"CSRFToken":           csrfToken,
	})
}

func handleEditPackPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	packID := c.Param("id")

	pack, err := database.GetPack(db, packID)
	if err != nil {
		c.HTML(http.StatusNotFound, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Error": "Pack not found",
		})
		return
	}

	if pack.UserID != userID {
		c.HTML(http.StatusForbidden, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Error": "Access denied",
		})
		return
	}

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "edit_pack.html", gin.H{
		"Title":     "Edit Pack - Carryless",
		"User":      user,
		"Pack":      pack,
		"CSRFToken": csrfToken.Token,
	})
}

func handleUpdatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")
	packID := c.Param("id")

	name := strings.TrimSpace(c.PostForm("name"))
	isPublicStr := c.PostForm("is_public")

	if name == "" {
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Pack name is required",
		})
		return
	}

	if len(name) > 200 {
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": "Pack name must be less than 200 characters",
		})
		return
	}

	isPublic := isPublicStr == "true" || isPublicStr == "1"

	err := database.UpdatePack(db, userID, packID, name, isPublic)
	if err != nil {
		var errorMsg string
		if strings.Contains(err.Error(), "not found") {
			errorMsg = "Pack not found"
		} else {
			errorMsg = "Failed to update pack"
		}
		
		pack, _ := database.GetPack(db, packID)
		c.HTML(http.StatusBadRequest, "edit_pack.html", gin.H{
			"Title": "Edit Pack - Carryless",
			"User":  user,
			"Pack":  pack,
			"Error": errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handleDeletePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	err := database.DeletePack(db, userID, packID)
	if err != nil {
		c.Redirect(http.StatusFound, "/packs")
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handleAddItemToPack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	itemIDStr := c.PostForm("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = database.AddItemToPack(db, packID, itemID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Item already in pack"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to pack"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item added to pack successfully"})
}

func handleRemoveItemFromPack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	err = database.RemoveItemFromPack(db, packID, itemID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item from pack"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item removed from pack successfully"})
}

func handleToggleWorn(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	isWornStr := c.PostForm("is_worn")
	isWorn := isWornStr == "true" || isWornStr == "1"

	err = database.TogglePackItemWorn(db, packID, itemID, userID, isWorn)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worn status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worn status updated successfully"})
}

func handleUpdateWornCount(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	itemIDStr := c.Param("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	wornCountStr := c.PostForm("worn_count")
	wornCount, err := strconv.Atoi(wornCountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worn count"})
		return
	}

	err = database.UpdatePackItemWornCount(db, packID, itemID, userID, wornCount)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pack or item not found"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worn count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worn count updated successfully"})
}

func handleDuplicatePack(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	_, err := database.DuplicatePack(db, userID, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.Redirect(http.StatusFound, "/packs")
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.Redirect(http.StatusFound, "/packs")
			return
		}
		c.Redirect(http.StatusFound, "/packs")
		return
	}

	c.Redirect(http.StatusFound, "/packs")
}

func handleCreatePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	packID := c.Param("id")

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	_, err := database.CreatePackLabel(db, packID, strings.TrimSpace(name), color, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label created successfully"})
}

func handleUpdatePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	name := c.PostForm("name")
	color := c.PostForm("color")
	if color == "" {
		color = "#6b7280" // Default gray color
	}

	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Label name is required"})
		return
	}

	err = database.UpdatePackLabel(db, labelID, strings.TrimSpace(name), color, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label name already exists"})
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label updated successfully"})
}

func handleDeletePackLabel(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.DeletePackLabel(db, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted successfully"})
}

func handleAssignLabelToItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	packItemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	labelID, err := strconv.Atoi(c.PostForm("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.AssignLabelToPackItem(db, packItemID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item or label not found"})
			return
		}
		if strings.Contains(err.Error(), "same pack") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label does not belong to the same pack"})
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Label already assigned to this item"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label assigned successfully"})
}

func handleRemoveLabelFromItem(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	
	packItemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	labelID, err := strconv.Atoi(c.Param("label_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	err = database.RemoveLabelFromPackItem(db, packItemID, labelID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed successfully"})
}