package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requestIDMiddleware adds a unique request ID to each request
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// structuredLoggingMiddleware logs requests in structured format
func structuredLoggingMiddleware(config LoggingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		requestID := c.GetString("requestID")

		if config.Structured {
			fmt.Printf(`{"time":"%s","request_id":"%s","method":"%s","path":"%s","status":%d,"latency":"%v"}%s`,
				start.Format(time.RFC3339), requestID, method, path, statusCode, latency, "\n")
		}
	}
}

// corsMiddleware handles CORS
func corsMiddleware(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowed := false
		for _, allowedOrigin := range config.AllowOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			if len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// apiKeyAuthMiddleware validates API key
func apiKeyAuthMiddleware(config APIKeyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(config.HeaderName)
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "missing_api_key"}})
			c.Abort()
			return
		}
		if !slices.Contains(config.Keys, apiKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "invalid_api_key"}})
			c.Abort()
			return
		}
		c.Set("authenticated", true)
		c.Next()
	}
}

// jwtAuthMiddleware validates JWT tokens
func jwtAuthMiddleware(_ JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "missing_token"}})
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "invalid_token_format"}})
			c.Abort()
			return
		}
		c.Set("authenticated", true)
		c.Next()
	}
}
