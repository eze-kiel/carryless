package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"carryless/internal/database"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

func handleAdminPanel(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user").(*models.User)
	
	// Get admin statistics
	stats, err := database.GetAdminStats(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin stats"})
		return
	}
	
	// Get all users with pack counts
	users, err := database.GetAllUsersWithStats(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}
	
	// Generate CSRF token
	csrfToken, err := database.CreateCSRFToken(db, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF token"})
		return
	}
	
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":      "Admin Panel - Carryless",
		"User":       user,
		"Stats":      stats,
		"Users":      users,
		"CSRFToken":  csrfToken.Token,
	})
}

func handleToggleUserAdmin(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user").(*models.User)
	
	// Get user ID from URL parameter
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	
	// Prevent admin from removing their own admin status
	if userID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot modify your own admin status"})
		return
	}
	
	err = database.ToggleUserAdmin(db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle admin status"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "User admin status toggled successfully"})
}

func handleBanUser(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user").(*models.User)
	
	// Get user ID from URL parameter
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	
	// Prevent admin from banning themselves
	if userID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot ban yourself"})
		return
	}
	
	err = database.BanUser(db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "User banned successfully"})
}