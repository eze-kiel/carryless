package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"carryless/internal/database"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*rateLimiter)
	mu      sync.Mutex
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		mu.Lock()
		defer mu.Unlock()

		if limiter, exists := clients[ip]; exists {
			limiter.lastSeen = time.Now()
			if !limiter.limiter.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
				c.Abort()
				return
			}
		} else {
			clients[ip] = &rateLimiter{
				limiter:  rate.NewLimiter(rate.Every(time.Second/20), 20),
				lastSeen: time.Now(),
			}
		}

		cleanupOldClients()
		c.Next()
	}
}

func AuthRateLimit() gin.HandlerFunc {
	authClients := make(map[string]*rateLimiter)
	var authMu sync.Mutex
	
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		authMu.Lock()
		defer authMu.Unlock()

		if limiter, exists := authClients[ip]; exists {
			limiter.lastSeen = time.Now()
			if !limiter.limiter.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Authentication rate limit exceeded"})
				c.Abort()
				return
			}
		} else {
			authClients[ip] = &rateLimiter{
				limiter:  rate.NewLimiter(rate.Every(time.Minute), 5),
				lastSeen: time.Now(),
			}
		}

		// Cleanup old auth clients
		for ip, client := range authClients {
			if time.Since(client.lastSeen) > 30*time.Minute {
				delete(authClients, ip)
			}
		}
		
		c.Next()
	}
}

func cleanupOldClients() {
	for ip, client := range clients {
		if time.Since(client.lastSeen) > 10*time.Minute {
			delete(clients, ip)
		}
	}
}

func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, allowedOrigin := range origins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("csrf_token")
		}

		if token == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token required"})
			c.Abort()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		db, exists := c.Get("db")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
			c.Abort()
			return
		}

		err := database.ValidateCSRFToken(db.(*sql.DB), token, userID.(int))
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AuthRequired(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := database.ValidateSession(db, sessionCookie)
		if err != nil {
			c.SetCookie("session_id", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("db", db)
		c.Next()
	}
}

func AuthOptional(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err == nil {
			user, err := database.ValidateSession(db, sessionCookie)
			if err == nil {
				c.Set("user", user)
				c.Set("user_id", user.ID)
			}
		}
		c.Set("db", db)
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
		c.Next()
	}
}

func LogRequests() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006/01/02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

func AddDBContext(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "db", db)
		c.Request = c.Request.WithContext(ctx)
		c.Set("db", db)
		c.Next()
	}
}

func TrimSpaces() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			for key, values := range c.Request.PostForm {
				for i, value := range values {
					c.Request.PostForm[key][i] = strings.TrimSpace(value)
				}
			}
		}
		c.Next()
	}
}

func AdminRequired(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := database.ValidateSession(db, sessionCookie)
		if err != nil {
			c.SetCookie("session_id", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		if !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("db", db)
		c.Next()
	}
}