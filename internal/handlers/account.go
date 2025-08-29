package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

func handleAccountPage(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}

	c.HTML(http.StatusOK, "account.html", gin.H{
		"Title":     "Account - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}

func handleChangePassword(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	currentPassword := strings.TrimSpace(c.PostForm("current_password"))
	newPassword := strings.TrimSpace(c.PostForm("new_password"))
	confirmPassword := strings.TrimSpace(c.PostForm("confirm_password"))

	// Validate inputs
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		c.HTML(http.StatusBadRequest, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "All password fields are required",
		})
		return
	}

	if newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "New passwords do not match",
		})
		return
	}

	if len(newPassword) < 8 {
		c.HTML(http.StatusBadRequest, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "New password must be at least 8 characters long",
		})
		return
	}

	// Verify current password
	err := database.VerifyPassword(db, userID, currentPassword)
	if err != nil {
		c.HTML(http.StatusBadRequest, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "Current password is incorrect",
		})
		return
	}

	// Update password
	err = database.UpdatePassword(db, userID, newPassword)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "Failed to update password",
		})
		return
	}

	c.HTML(http.StatusOK, "account.html", gin.H{
		"Title":   "Account - Carryless",
		"User":    user,
		"Success": "Password updated successfully",
	})
}

func handleChangeCurrency(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	currency := strings.TrimSpace(c.PostForm("currency"))

	// Validate currency
	validCurrencies := map[string]bool{
		"$": true,  // USD
		"€": true,  // EUR
		"¥": true,  // JPY
		"£": true,  // GBP
		"₹": true,  // INR
		"₩": true,  // KRW
		"¢": true,  // cents
		"R": true,  // ZAR
	}

	if !validCurrencies[currency] {
		c.HTML(http.StatusBadRequest, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "Invalid currency selected",
		})
		return
	}

	// Update currency
	err := database.UpdateUserCurrency(db, userID, currency)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "account.html", gin.H{
			"Title": "Account - Carryless",
			"User":  user,
			"Error": "Failed to update currency",
		})
		return
	}

	c.HTML(http.StatusOK, "account.html", gin.H{
		"Title":   "Account - Carryless",
		"User":    user,
		"Success": "Currency updated successfully",
	})
}