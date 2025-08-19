package handlers

import (
	"database/sql"
	"net/http"

	"carryless/internal/database"
	"carryless/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, db *sql.DB) {
	r.Use(middleware.LogRequests())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.AddDBContext(db))
	r.Use(middleware.TrimSpaces())

	r.GET("/", middleware.AuthOptional(db), handleHome)
	r.GET("/register", handleRegisterPage)
	r.POST("/register", handleRegister)
	r.GET("/login", handleLoginPage)
	r.POST("/login", handleLogin)
	r.POST("/logout", middleware.AuthRequired(db), handleLogout)

	protected := r.Group("/")
	protected.Use(middleware.AuthRequired(db))
	protected.Use(middleware.CSRF())
	{
		protected.GET("/dashboard", handleDashboard)
		protected.GET("/inventory", handleInventory)
		protected.GET("/inventory/export", handleExportInventory)
		protected.POST("/inventory/import", handleImportInventory)
		protected.GET("/inventory/items/new", handleNewItemPage)
		protected.POST("/inventory/items", handleCreateItem)
		protected.GET("/inventory/items/:id/edit", handleEditItemPage)
		protected.POST("/inventory/items/:id", handleUpdateItem)
		protected.GET("/inventory/items/:id/packs", handleCheckItemPacks)
		protected.POST("/inventory/items/:id/delete", handleDeleteItem)
		
		protected.GET("/categories", handleCategories)
		protected.GET("/categories/new", handleNewCategoryPage)
		protected.POST("/categories", handleCreateCategory)
		protected.GET("/categories/:id/edit", handleEditCategoryPage)
		protected.POST("/categories/:id", handleUpdateCategory)
		protected.GET("/categories/:id/items", handleCheckCategoryItems)
		protected.POST("/categories/:id/delete", handleDeleteCategory)
		
		protected.GET("/packs", handlePacks)
		protected.GET("/packs/new", handleNewPackPage)
		protected.POST("/packs", handleCreatePack)
		protected.GET("/packs/:id", handlePackDetail)
		protected.GET("/packs/:id/edit", handleEditPackPage)
		protected.POST("/packs/:id", handleUpdatePack)
		protected.POST("/packs/:id/delete", handleDeletePack)
		protected.POST("/packs/:id/duplicate", handleDuplicatePack)
		protected.POST("/packs/:id/note", handleUpdatePackNote)
		protected.POST("/packs/:id/items", handleAddItemToPack)
		protected.DELETE("/packs/:id/items/:item_id", handleRemoveItemFromPack)
		protected.PUT("/packs/:id/items/:item_id/worn", handleToggleWorn)
		protected.PUT("/packs/:id/items/:item_id/worn-count", handleUpdateWornCount)
		
		protected.POST("/packs/:id/labels", handleCreatePackLabel)
		protected.POST("/packs/:id/labels/:label_id", handleUpdatePackLabel)
		protected.DELETE("/packs/:id/labels/:label_id", handleDeletePackLabel)
		protected.POST("/packs/:id/items/:item_id/labels", handleAssignLabelToItem)
		protected.DELETE("/packs/:id/items/:item_id/labels/:label_id", handleRemoveLabelFromItem)
		
		protected.GET("/account", handleAccountPage)
		protected.POST("/account/password", handleChangePassword)
		protected.POST("/account/currency", handleChangeCurrency)
		
		protected.GET("/api/csrf-token", handleCSRFToken)
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AdminRequired(db))
	admin.Use(middleware.CSRF())
	{
		admin.GET("/", handleAdminPanel)
		admin.POST("/users/:id/toggle-admin", handleToggleUserAdmin)
		admin.POST("/users/:id/ban", handleBanUser)
		admin.POST("/toggle-registration", handleToggleRegistration)
	}

	r.GET("/p/packs/:id", middleware.AuthOptional(db), handlePublicPack)
	r.GET("/packs/:id/checklist", middleware.AuthOptional(db), handlePackChecklist)
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

func handleDashboard(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	db := c.MustGet("db").(*sql.DB)
	user := c.MustGet("user")

	csrfToken, err := database.CreateCSRFToken(db, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"Title": "Dashboard - Carryless",
			"User":  user,
			"Error": "Failed to generate security token",
		})
		return
	}
	
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":     "Dashboard - Carryless",
		"User":      user,
		"CSRFToken": csrfToken.Token,
	})
}