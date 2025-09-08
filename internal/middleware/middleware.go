package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type clientTracker struct {
	errors404    []time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

var (
	clients      = make(map[string]*rateLimiter)
	mu           sync.Mutex
	trackers     = make(map[string]*clientTracker)
	trackersMu   sync.Mutex
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

func ActivationRateLimit() gin.HandlerFunc {
	activationClients := make(map[string]*rateLimiter)
	var activationMu sync.Mutex
	
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		activationMu.Lock()
		defer activationMu.Unlock()

		if limiter, exists := activationClients[ip]; exists {
			limiter.lastSeen = time.Now()
			if !limiter.limiter.Allow() {
				c.HTML(http.StatusTooManyRequests, "activation_result.html", gin.H{
					"Title":   "Too Many Requests - Carryless",
					"Success": false,
					"Message": "Too many activation attempts. Please wait before trying again.",
				})
				c.Abort()
				return
			}
		} else {
			activationClients[ip] = &rateLimiter{
				limiter:  rate.NewLimiter(rate.Every(time.Minute*5), 3),
				lastSeen: time.Now(),
			}
		}

		// Cleanup old activation clients
		for ip, client := range activationClients {
			if time.Since(client.lastSeen) > 30*time.Minute {
				delete(activationClients, ip)
			}
		}
		
		c.Next()
	}
}

func IPBlocker() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		trackersMu.Lock()
		tracker, exists := trackers[ip]
		trackersMu.Unlock()
		
		if exists && time.Now().Before(tracker.blockedUntil) {
			c.HTML(http.StatusForbidden, "blocked.html", gin.H{
				"Title":   "Access Blocked - Carryless",
				"Message": "Your IP has been temporarily blocked due to excessive invalid requests. Please try again later.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

func Track404AndBlock() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		if c.Writer.Status() == http.StatusNotFound {
			ip := c.ClientIP()
			now := time.Now()
			
			trackersMu.Lock()
			defer trackersMu.Unlock()
			
			tracker, exists := trackers[ip]
			if !exists {
				tracker = &clientTracker{
					errors404: make([]time.Time, 0),
					lastSeen:  now,
				}
				trackers[ip] = tracker
			}
			
			tracker.lastSeen = now
			tracker.errors404 = append(tracker.errors404, now)
			
			// Remove 404 errors older than 5 minutes
			cutoff := now.Add(-5 * time.Minute)
			validErrors := make([]time.Time, 0)
			for _, errorTime := range tracker.errors404 {
				if errorTime.After(cutoff) {
					validErrors = append(validErrors, errorTime)
				}
			}
			tracker.errors404 = validErrors
			
			// Check if we should block this IP
			if len(tracker.errors404) >= 10 {
				tracker.blockedUntil = now.Add(15 * time.Minute)
				tracker.errors404 = make([]time.Time, 0) // Reset counter
				log.Printf("Blocked IP %s for 15 minutes due to %d 404 errors in 5 minutes", ip, len(validErrors))
			}
			
			// Cleanup old trackers
			for trackerIP, trackerData := range trackers {
				if time.Since(trackerData.lastSeen) > 30*time.Minute && time.Now().After(trackerData.blockedUntil) {
					delete(trackers, trackerIP)
				}
			}
		}
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

func AuthRequired(db *sql.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := database.ValidateSession(db, sessionCookie, cfg.SessionDuration, cfg.SessionExtensionThreshold)
		if err != nil {
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("session_id", "", -1, "/", "", true, true)
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

func AuthOptional(db *sql.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err == nil {
			user, err := database.ValidateSession(db, sessionCookie, cfg.SessionDuration, cfg.SessionExtensionThreshold)
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
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; img-src 'self' data:; font-src 'self' https://cdnjs.cloudflare.com")
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

func RequireActivation() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		userModel := user.(*models.User)
		if !userModel.IsActivated {
			c.HTML(http.StatusForbidden, "activation_required.html", gin.H{
				"Title": "Account Activation Required - Carryless",
				"User":  userModel,
				"Message": "Please check your email and click the activation link to complete your account setup before accessing this feature.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AdminRequired(db *sql.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := database.ValidateSession(db, sessionCookie, cfg.SessionDuration, cfg.SessionExtensionThreshold)
		if err != nil {
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("session_id", "", -1, "/", "", true, true)
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