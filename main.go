package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/email"
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

	emailService := email.NewService(cfg)
	if emailService.IsEnabled() {
		log.Println("Email service enabled with Mailgun")
	} else {
		log.Println("Email service disabled - Mailgun not configured")
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
		"sub": func(a, b int) int {
			return a - b
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
		"sequence": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i
			}
			return result
		},
		"getLabelForItem": func(labels []models.ItemLabel, itemIndex int) *models.ItemLabel {
			currentIndex := 0
			for _, label := range labels {
				if currentIndex <= itemIndex && itemIndex < currentIndex+label.Count {
					return &label
				}
				currentIndex += label.Count
			}
			return nil
		},
		"timeAgo": func(t time.Time) string {
			now := time.Now()
			duration := now.Sub(t)
			
			if duration.Minutes() < 1 {
				return "Just now"
			} else if duration.Hours() < 1 {
				minutes := int(duration.Minutes())
				if minutes == 1 {
					return "1 minute ago"
				}
				return fmt.Sprintf("%d minutes ago", minutes)
			} else if duration.Hours() < 24 {
				hours := int(duration.Hours())
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			} else if duration.Hours() < 48 {
				return "Yesterday"
			} else if duration.Hours() < 168 { // 7 days
				days := int(duration.Hours() / 24)
				return fmt.Sprintf("%d days ago", days)
			} else {
				return t.Format("Jan 2")
			}
		},
	}

	r.SetFuncMap(funcMap)
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit())

	handlers.SetupRoutes(r, db, emailService)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}