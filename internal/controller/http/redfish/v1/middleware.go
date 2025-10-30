// Package v1 provides Redfish v1 API middleware.
package v1

import (
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/device-management-toolkit/console/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
