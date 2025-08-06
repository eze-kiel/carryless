package main

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/handlers"
	"carryless/internal/middleware"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	r := gin.Default()

	funcMap := template.FuncMap{
		"jsonify": func(v interface{}) template.JS {
			bytes, _ := json.Marshal(v)
			return template.JS(bytes)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"groupByCategory": func(items []models.PackItem) map[string][]models.PackItem {
			groups := make(map[string][]models.PackItem)
			for _, item := range items {
				category := item.Item.Category.Name
				groups[category] = append(groups[category], item)
			}
			return groups
		},
		"groupItemsByCategory": func(items []models.Item) map[string][]models.Item {
			groups := make(map[string][]models.Item)
			for _, item := range items {
				category := item.Category.Name
				groups[category] = append(groups[category], item)
			}
			return groups
		},
		"redactEmail": func(email string) string {
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				return email // Return original if not a valid email format
			}
			
			prefix := parts[0]
			domain := parts[1]
			
			if len(prefix) <= 2 {
				return email // Return original if prefix too short to redact
			}
			
			// Create redacted prefix: first letter + *** + last letter
			redactedPrefix := string(prefix[0]) + "***" + string(prefix[len(prefix)-1])
			
			return redactedPrefix + "@" + domain
		},
	}

	r.SetFuncMap(funcMap)
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit())

	handlers.SetupRoutes(r, db)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}