// Package v1 provides Redfish v1 API middleware.
package v1

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Context key type for type-safe context values
type contextKey string

const (
	userIDKey    contextKey = "userID"
	requestIDKey contextKey = "requestId"
)

const requestIDRandomBytesLength = 4

const expectedCredentialParts = 2

// BasicAuthValidator validates HTTP Basic Authentication
func BasicAuthValidator(expectedUsername, expectedPassword string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if !strings.HasPrefix(authHeader, "Basic ") {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Extract and decode credentials
		credentials := strings.TrimPrefix(authHeader, "Basic ")

		decoded, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Split username:password
		parts := strings.SplitN(string(decoded), ":", expectedCredentialParts)
		if len(parts) != expectedCredentialParts {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		username, password := parts[0], parts[1]

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1

		if usernameMatch && passwordMatch {
			// Set authenticated user in context for logging and audit
			c.Set(string(userIDKey), username)
			c.Next()
		} else {
			UnauthorizedError(c)
			c.Abort()
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Generate a unique request ID for tracing
		requestID := generateRequestID()
		c.Set(string(requestIDKey), requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})
}

// AuthenticationMiddleware provides Basic Authentication
// Note: This is currently not used as authentication is handled at component level
// for selective endpoint protection
func AuthenticationMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This middleware is disabled - authentication handled at component level
		// for proper selective authentication of endpoints
		c.Set(string(userIDKey), "admin") // Placeholder - not used in production
		c.Next()
	})
}

// LoggingMiddleware provides request logging
func LoggingMiddleware(_ interface{}) gin.HandlerFunc {
	// FUTURE: Implement proper structured logging with the provided logger
	// For now, use default Gin logging
	return gin.LoggerWithWriter(gin.DefaultWriter)
}

// ErrorHandlingMiddleware provides centralized error handling
func ErrorHandlingMiddleware(_ interface{}) gin.HandlerFunc {
	// FUTURE: Implement custom recovery handler with proper logging
	// For now, use default Gin recovery
	return gin.Recovery()
}

// generateRequestID generates a unique request ID for tracing
func generateRequestID() string {
	// Use timestamp + random component for uniqueness
	timestamp := time.Now().UnixNano()

	// Generate random bytes
	randomBytes := make([]byte, requestIDRandomBytesLength)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp only if random generation fails
		randomBytes = []byte{0, 0, 0, 0}
	}

	// Create request ID: req-<timestamp>-<random>
	return fmt.Sprintf("req-%d-%x", timestamp, randomBytes)
}
