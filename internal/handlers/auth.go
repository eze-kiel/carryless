package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func handleRegister(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	errors := make(map[string]string)

	if len(username) < 3 || len(username) > 30 {
		errors["username"] = "Username must be between 3 and 30 characters"
	}

	if !emailRegex.MatchString(email) {
		errors["email"] = "Please enter a valid email address"
	}

	if len(password) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	}

	if password != confirmPassword {
		errors["confirm_password"] = "Passwords do not match"
	}

	if len(errors) > 0 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Title":   "Register - Carryless",
			"Errors":  errors,
			"Username": username,
			"Email":   email,
		})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	user, err := database.CreateUser(db, username, email, password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if strings.Contains(err.Error(), "username") {
				errors["username"] = "Username already exists"
			} else if strings.Contains(err.Error(), "email") {
				errors["email"] = "Email already exists"
			}
		} else {
			errors["general"] = "Failed to create account. Please try again."
		}

		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Title":   "Register - Carryless",
			"Errors":  errors,
			"Username": username,
			"Email":   email,
		})
		return
	}

	session, err := database.CreateSession(db, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Title":  "Register - Carryless",
			"Errors": map[string]string{"general": "Failed to create session. Please try logging in."},
		})
		return
	}

	c.SetCookie("session_id", session.ID, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

func handleLogin(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")

	errors := make(map[string]string)

	if email == "" {
		errors["email"] = "Email is required"
	}

	if password == "" {
		errors["password"] = "Password is required"
	}

	if len(errors) > 0 {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Title":  "Login - Carryless",
			"Errors": errors,
			"Email":  email,
		})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	user, err := database.AuthenticateUser(db, email, password)
	if err != nil {
		errors["general"] = "Invalid email or password"
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Title":  "Login - Carryless",
			"Errors": errors,
			"Email":  email,
		})
		return
	}

	session, err := database.CreateSession(db, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"Title":  "Login - Carryless",
			"Errors": map[string]string{"general": "Failed to create session. Please try again."},
		})
		return
	}

	c.SetCookie("session_id", session.ID, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

func handleLogout(c *gin.Context) {
	sessionCookie, err := c.Cookie("session_id")
	if err == nil {
		db := c.MustGet("db").(*sql.DB)
		database.DeleteSession(db, sessionCookie)
	}

	c.SetCookie("session_id", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func handleCSRFToken(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)

	token, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token.Token})
}