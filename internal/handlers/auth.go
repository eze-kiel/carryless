package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"carryless/internal/config"
	"carryless/internal/database"
	emailService "carryless/internal/email"
	"carryless/internal/logger"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func handleRegister(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	
	// Check if registration is enabled
	registrationEnabled, err := database.IsRegistrationEnabled(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Title": "Register - Carryless",
			"Error": "Unable to check registration status",
		})
		return
	}
	
	if !registrationEnabled {
		c.HTML(http.StatusForbidden, "register.html", gin.H{
			"Title":               "Register - Carryless",
			"RegistrationEnabled": false,
			"Error":               "Registration has been disabled by an administrator",
		})
		return
	}

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
			"Title":               "Register - Carryless",
			"Errors":              errors,
			"Username":            username,
			"Email":               email,
			"RegistrationEnabled": true,
		})
		return
	}

	user, err := database.CreateUser(db, username, email, password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			errors["general"] = "An account with those credentials already exists"
		} else {
			errors["general"] = "Failed to create account. Please try again."
		}

		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Title":               "Register - Carryless",
			"Errors":              errors,
			"Username":            "",
			"Email":               "",
			"RegistrationEnabled": true,
		})
		return
	}

	// Create activation token
	activationToken, err := database.CreateActivationToken(db, user.ID)
	if err != nil {
		logger.Error("Failed to create activation token",
			"email", user.Email,
			"user_id", user.ID,
			"error", err)
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Title":               "Register - Carryless",
			"Errors":              map[string]string{"general": "Failed to complete registration. Please try again."},
			"RegistrationEnabled": true,
		})
		return
	}

	emailSvc, _ := c.Get("email_service")
	if service, ok := emailSvc.(*emailService.Service); ok && service.IsEnabled() {
		if err := service.SendWelcomeEmail(user, activationToken.Token); err != nil {
			logger.Warn("Failed to send welcome email",
				"email", user.Email,
				"user_id", user.ID,
				"error", err)
		}
		
		// Send notification to all admins about the new user registration
		admins, err := database.GetAllAdmins(db)
		if err != nil {
			logger.Error("Failed to get admin users for notification", "error", err)
		} else {
			logger.Debug("Found admin users for notification", "count", len(admins))
			for _, admin := range admins {
				logger.Debug("Sending admin notification",
					"admin_email", admin.Email,
					"admin_id", admin.ID,
					"new_user_id", user.ID)
				go func(adminUser models.User) {
					if err := service.SendAdminNotificationEmail(&adminUser, user); err != nil {
						logger.Warn("Failed to send admin notification email",
							"admin_email", adminUser.Email,
							"admin_id", adminUser.ID,
							"error", err)
					} else {
						logger.Debug("Successfully queued admin notification email",
							"admin_email", adminUser.Email,
							"admin_id", adminUser.ID)
					}
				}(admin)
			}
		}
	}

	// Redirect to a success page instead of logging in the user
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Title":               "Registration Complete - Carryless",
		"Success":             "Registration successful! Please check your email and click the activation link to complete your account setup.",
		"RegistrationEnabled": true,
	})
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

	cfg := c.MustGet("config").(*config.Config)
	session, err := database.CreateSession(db, user.ID, cfg.SessionDuration)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"Title":  "Login - Carryless",
			"Errors": map[string]string{"general": "Failed to create session. Please try again."},
		})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	// Set cookie expiry to match session duration
	cookieMaxAge := int(cfg.SessionDuration.Seconds())
	c.SetCookie("session_id", session.ID, cookieMaxAge, "/", "", true, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

func handleLogout(c *gin.Context) {
	sessionCookie, err := c.Cookie("session_id")
	if err == nil {
		db := c.MustGet("db").(*sql.DB)
		database.DeleteSession(db, sessionCookie)
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session_id", "", -1, "/", "", true, true)
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

func handleActivate(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "activation_result.html", gin.H{
			"Title":   "Invalid Activation Link - Carryless",
			"Success": false,
			"Message": "Invalid activation link. Please check the link in your email and try again.",
		})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	
	// Validate the activation token
	user, err := database.ValidateActivationToken(db, token)
	if err != nil {
		logger.Warn("Failed to validate activation token",
			"token", token,
			"error", err)
		c.HTML(http.StatusBadRequest, "activation_result.html", gin.H{
			"Title":   "Activation Failed - Carryless",
			"Success": false,
			"Message": "This activation link is invalid or has expired. Please register again or contact support.",
		})
		return
	}

	// Check if user is already activated
	if user.IsActivated {
		c.HTML(http.StatusOK, "activation_result.html", gin.H{
			"Title":   "Already Activated - Carryless",
			"Success": true,
			"Message": "Your account is already activated! You can now log in to access all features.",
		})
		return
	}

	// Activate the user
	err = database.ActivateUser(db, user.ID, token)
	if err != nil {
		logger.Error("Failed to activate user",
			"user_id", user.ID,
			"token", token,
			"error", err)
		c.HTML(http.StatusInternalServerError, "activation_result.html", gin.H{
			"Title":   "Activation Error - Carryless",
			"Success": false,
			"Message": "There was an error activating your account. Please try again or contact support.",
		})
		return
	}

	logger.Info("User successfully activated",
		"email", user.Email,
		"user_id", user.ID)
	
	// Success - user is now activated
	c.HTML(http.StatusOK, "activation_result.html", gin.H{
		"Title":   "Account Activated - Carryless",
		"Success": true,
		"Message": "Congratulations! Your account has been successfully activated. You can now log in and start using all features of Carryless.",
		"ShowLoginButton": true,
	})
}