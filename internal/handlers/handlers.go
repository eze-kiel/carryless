package handlers

import (
	"database/sql"
	"net/http"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/email"
	"carryless/internal/logger"
	"carryless/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, db *sql.DB, emailService *email.Service, cfg *config.Config) {
	r.Use(middleware.LogRequests())
	r.Use(middleware.SecurityHeaders(cfg))
	r.Use(middleware.AddDBContext(db))
	r.Use(addEmailServiceContext(emailService))
	r.Use(addConfigContext(cfg))
	r.Use(middleware.TrimSpaces())

	r.GET("/", middleware.AuthOptional(db, cfg), handleHome)
	r.GET("/terms", middleware.AuthOptional(db, cfg), handleTermsPage)
	r.GET("/privacy", middleware.AuthOptional(db, cfg), handlePrivacyPage)
	r.GET("/register", handleRegisterPage)
	r.POST("/register", middleware.AuthRateLimit(cfg), handleRegister)
	r.GET("/login", handleLoginPage)
	r.POST("/login", middleware.AuthRateLimit(cfg), handleLogin)
	r.POST("/logout", middleware.AuthRequired(db, cfg), handleLogout)
	r.GET("/activate/:token", middleware.ActivationRateLimit(cfg), middleware.AddDBContext(db), handleActivate)

	protected := r.Group("/")
	protected.Use(middleware.AuthRequired(db, cfg))
	protected.Use(middleware.CSRF(cfg))
	{
		protected.GET("/dashboard", handleDashboard)
		protected.GET("/account", handleAccountPage)
		protected.POST("/account/password", handleChangePassword)
		protected.POST("/account/currency", handleChangeCurrency)
		protected.POST("/account/username", handleChangeUsername)
		protected.GET("/api/csrf-token", handleCSRFToken)
	}

	// Routes that require activation (content creation/modification)
	activated := r.Group("/")
	activated.Use(middleware.AuthRequired(db, cfg))
	activated.Use(middleware.RequireActivation())
	activated.Use(middleware.CSRF(cfg))
	{
		activated.GET("/inventory", handleInventory)
		activated.GET("/inventory/export", handleExportInventory)
		activated.POST("/inventory/import", handleImportInventory)
		activated.GET("/inventory/items/new", handleNewItemPage)
		activated.POST("/inventory/items", handleCreateItem)
		activated.GET("/inventory/items/:id/edit", handleEditItemPage)
		activated.POST("/inventory/items/:id", handleUpdateItem)
		activated.GET("/inventory/items/:id/packs", handleCheckItemPacks)
		activated.POST("/inventory/items/:id/delete", handleDeleteItem)
		activated.POST("/inventory/items/:id/duplicate", handleDuplicateItem)
		activated.POST("/inventory/items/bulk-edit", handleBulkEditItems)
		activated.POST("/inventory/items/bulk-delete", handleBulkDeleteItems)
		activated.PATCH("/api/items/:id", handlePatchItem)

		activated.GET("/categories", handleCategories)
		activated.GET("/categories/new", handleNewCategoryPage)
		activated.POST("/categories", handleCreateCategory)
		activated.GET("/categories/:id/edit", handleEditCategoryPage)
		activated.POST("/categories/:id", handleUpdateCategory)
		activated.GET("/categories/:id/items", handleCheckCategoryItems)
		activated.POST("/categories/:id/delete", handleDeleteCategory)

		activated.GET("/packs", handlePacks)
		activated.GET("/packs/new", handleNewPackPage)
		activated.POST("/packs", handleCreatePack)
		activated.GET("/packs/:id", handlePackDetail)
		activated.GET("/packs/:id/edit", handleEditPackPage)
		activated.POST("/packs/:id", handleUpdatePack)
		activated.POST("/packs/:id/delete", handleDeletePack)
		activated.POST("/packs/:id/duplicate", handleDuplicatePack)
		activated.POST("/packs/:id/items", handleAddItemToPack)
		activated.DELETE("/packs/:id/items/:item_id", handleRemoveItemFromPack)
		activated.PUT("/packs/:id/items/:item_id/worn", handleToggleWorn)
		activated.PUT("/packs/:id/items/:item_id/worn-count", handleUpdateWornCount)
		activated.POST("/packs/:id/lock", handleTogglePackLock)

		activated.POST("/packs/:id/labels", handleCreatePackLabel)
		activated.POST("/packs/:id/labels/:label_id", handleUpdatePackLabel)
		activated.DELETE("/packs/:id/labels/:label_id", handleDeletePackLabel)
		activated.POST("/packs/:id/items/:item_id/labels", handleAssignLabelToItem)
		activated.DELETE("/packs/:id/items/:item_id/labels/:label_id", handleRemoveLabelFromItem)

		// Trip routes
		activated.GET("/trips", handleTrips)
		activated.GET("/trips/new", handleNewTripPage)
		activated.POST("/trips", handleCreateTrip)
		activated.GET("/trips/:id", handleTripDetail)
		activated.GET("/trips/:id/edit", handleEditTripPage)
		activated.POST("/trips/:id", handleUpdateTrip)
		activated.POST("/trips/:id/delete", handleDeleteTrip)
		activated.POST("/trips/:id/archive", handleArchiveTrip)

		// Pack associations
		activated.POST("/trips/:id/packs", handleAddPackToTrip)
		activated.DELETE("/trips/:id/packs/:pack_id", handleRemovePackFromTrip)

		// Checklist API
		activated.POST("/trips/:id/checklist", handleAddChecklistItem)
		activated.PUT("/trips/:id/checklist/:item_id", handleUpdateChecklistItem)
		activated.DELETE("/trips/:id/checklist/:item_id", handleDeleteChecklistItem)
		activated.POST("/trips/:id/checklist/:item_id/toggle", handleToggleChecklistItem)
		activated.POST("/trips/:id/checklist/reorder", handleReorderChecklist)

		// Transport timeline API
		activated.POST("/trips/:id/transport", handleAddTransportStep)
		activated.PUT("/trips/:id/transport/:step_id", handleUpdateTransportStep)
		activated.DELETE("/trips/:id/transport/:step_id", handleDeleteTransportStep)
		activated.POST("/trips/:id/transport/reorder", handleReorderTransportSteps)

		// GPX upload
		activated.POST("/trips/:id/gpx", handleUploadGPX)
		activated.DELETE("/trips/:id/gpx", handleDeleteGPX)
		activated.GET("/trips/:id/gpx/download", handleDownloadGPX)
	}

	// Autosave routes that need new CSRF tokens returned after each request
	autosave := r.Group("/")
	autosave.Use(middleware.AuthRequired(db, cfg))
	autosave.Use(middleware.RequireActivation())
	autosave.Use(middleware.CSRFWithRenewal(cfg))
	{
		autosave.POST("/packs/:id/note", handleUpdatePackNote)
		autosave.POST("/trips/:id/notes", handleUpdateTripNotes)
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AdminRequired(db, cfg))
	admin.Use(middleware.CSRF(cfg))
	{
		admin.GET("/", handleAdminPanel)
		admin.POST("/users/:id/toggle-admin", handleToggleUserAdmin)
		admin.POST("/users/:id/toggle-activation", handleToggleUserActivation)
		admin.POST("/users/:id/resend-activation", handleResendActivationEmail)
		admin.POST("/users/:id/ban", handleBanUser)
		admin.POST("/toggle-registration", handleToggleRegistration)
	}

	r.GET("/p/:id", middleware.AuthOptional(db, cfg), handlePublicPackByShortID)
	r.GET("/p/:id/checklist", middleware.AuthOptional(db, cfg), handlePackChecklistByShortID)
	r.GET("/p/packs/:id", middleware.AuthOptional(db, cfg), handlePublicPack)
	r.GET("/packs/:id/checklist", middleware.AuthOptional(db, cfg), handlePackChecklist)

	// Public trip route
	r.GET("/t/:id", middleware.AuthOptional(db, cfg), handlePublicTripByShortID)
	r.GET("/t/:id/gpx/download", middleware.AuthOptional(db, cfg), handlePublicDownloadGPX)

	r.NoRoute(handle404)
}

func handleHome(c *gin.Context) {
	user, exists := c.Get("user")
	if exists {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	var csrfToken string
	if userID, hasUserID := c.Get("user_id"); hasUserID {
		db := c.MustGet("db").(*sql.DB)
		if token, err := database.CreateCSRFToken(db, userID.(int)); err == nil {
			csrfToken = token.Token
		}
	}
	
	c.HTML(http.StatusOK, "home.html", gin.H{
		"Title":     "Carryless - Outdoor Gear Catalog",
		"User":      user,
		"CSRFToken": csrfToken,
	})
}

func handleRegisterPage(c *gin.Context) {
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
	
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Title":               "Register - Carryless",
		"RegistrationEnabled": registrationEnabled,
	})
}

func handleLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title": "Login - Carryless",
	})
}

func handleTermsPage(c *gin.Context) {
	user, _ := c.Get("user")
	c.HTML(http.StatusOK, "terms.html", gin.H{
		"Title": "Terms of Service - Carryless",
		"User":  user,
	})
}

func handlePrivacyPage(c *gin.Context) {
	user, _ := c.Get("user")
	c.HTML(http.StatusOK, "privacy.html", gin.H{
		"Title": "Privacy Policy - Carryless",
		"User":  user,
	})
}

func addEmailServiceContext(emailService *email.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("email_service", emailService)
		c.Next()
	}
}

func addConfigContext(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}
}

func handle404(c *gin.Context) {
	user, _ := c.Get("user")
	c.HTML(http.StatusNotFound, "404.html", gin.H{
		"Title": "Page Not Found - Carryless",
		"User":  user,
	})
}

func handleDashboard(c *gin.Context) {
	logger.Debug("Dashboard handler started")

	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	logger.Debug("Dashboard request", "user_id", userID)

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		logger.Error("Failed to create CSRF token", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"Title": "Dashboard - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}
	logger.Debug("CSRF token created successfully")

	// Get user statistics
	logger.Debug("Fetching user statistics", "user_id", userID)
	stats, err := database.GetUserStats(db, userID)
	if err != nil {
		logger.Error("Failed to get user stats", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"Title": "Dashboard - Carryless",
			"User":  user,
			"Error": "Failed to load dashboard statistics",
		})
		return
	}
	logger.Debug("User stats fetched",
		"user_id", userID,
		"total_packs", stats.TotalPacks,
		"total_items", stats.TotalItems)

	// Get recent packs
	logger.Debug("Fetching recent packs", "user_id", userID)
	recentPacks, err := database.GetRecentPacks(db, userID, 3)
	if err != nil {
		logger.Error("Failed to get recent packs", "user_id", userID, "error", err)
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"Title": "Dashboard - Carryless",
			"User":  user,
			"Error": "Failed to load recent packs",
		})
		return
	}
	logger.Debug("Recent packs fetched",
		"user_id", userID,
		"pack_count", len(recentPacks))

	logger.Debug("Rendering dashboard template", "user_id", userID)
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":       "Dashboard - Carryless",
		"User":        user,
		"CSRFToken":   csrfToken.Token,
		"Stats":       stats,
		"RecentPacks": recentPacks,
	})
	logger.Debug("Dashboard template rendered successfully", "user_id", userID)
}