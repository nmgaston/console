// Package v1 provides Redfish v1 API middleware.
package v1

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
)

// RedfishJWTAuthMiddleware returns a Gin middleware that validates JWT tokens
// and returns Redfish-compliant error responses for authentication failures.
func RedfishJWTAuthMiddleware(jwtKey string, verifier *oidc.IDTokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

		if tokenString == "" {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// if clientID is set, use the oidc verifier
		if config.ConsoleConfig.ClientID != "" && verifier != nil {
			_, err := verifier.Verify(c.Request.Context(), tokenString)
			if err != nil {
				UnauthorizedError(c)
				c.Abort()

				return
			}
		} else {
			claims := &jwt.MapClaims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(jwtKey), nil
			})

			if err != nil || !token.Valid {
				UnauthorizedError(c)
				c.Abort()

				return
			}
		}

		c.Next()
	}
}

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
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			UnauthorizedError(c)
			c.Abort()
			return
		}

		username, password := parts[0], parts[1]

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1

		if usernameMatch && passwordMatch {
			c.Next()
		} else {
			UnauthorizedError(c)
			c.Abort()
		}
	}
}
