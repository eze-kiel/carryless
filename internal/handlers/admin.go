package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"carryless/internal/database"
	"carryless/internal/email"
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
	
	// Check if registration is enabled
	registrationEnabled, err := database.IsRegistrationEnabled(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get registration status"})
		return
	}
	
	// Generate CSRF token
	csrfToken, err := database.CreateCSRFToken(db, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF token"})
		return
	}
	
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":               "Admin Panel - Carryless",
		"User":                user,
		"Stats":               stats,
		"Users":               users,
		"RegistrationEnabled": registrationEnabled,
		"CSRFToken":           csrfToken.Token,
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

func handleToggleRegistration(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	
	err := database.ToggleRegistration(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle registration"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Registration setting toggled successfully"})
}

func handleToggleUserActivation(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	
	// Get user ID from URL parameter
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	
	err = database.ToggleUserActivation(db, userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle user activation"})
		}
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "User activation status toggled successfully"})
}

func handleResendActivationEmail(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	emailService := c.MustGet("email_service").(*email.Service)

	// Get user ID from URL parameter
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if email service is enabled
	if !emailService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Email service is not configured"})
		return
	}

	// Get the user details
	targetUser, err := database.GetUserByID(db, userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details"})
		}
		return
	}

	// Check if user is already activated
	if targetUser.IsActivated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already activated"})
		return
	}

	// Generate new activation token
	activationToken, err := database.ResendActivationToken(db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new activation token"})
		return
	}

	// Send the activation email
	err = emailService.SendWelcomeEmail(targetUser, activationToken.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send activation email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activation email resent successfully"})
}