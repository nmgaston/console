// Package v1 provides Redfish v1 API handlers for SessionService
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// SessionAuthMiddleware validates X-Auth-Token header
// This middleware integrates DMT JWT tokens with Redfish session authentication
func SessionAuthMiddleware(sessionUseCase *sessions.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for X-Auth-Token header (Redfish standard)
		token := c.GetHeader("X-Auth-Token")

		// Fallback to Authorization: Bearer for compatibility
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Validate token
		session, err := sessionUseCase.ValidateToken(token)
		if err != nil {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Store session in context for handlers
		c.Set("session", session)
		c.Set("username", session.Username)

		c.Next()
	}
}

// RedfishServer session endpoint implementations
// These methods are part of RedfishServer to satisfy the generated.ServerInterface

// GetRedfishV1SessionService handles GET /redfish/v1/SessionService
func (r *RedfishServer) GetRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionCount, _ := r.SessionUC.GetSessionCount()

	response := map[string]interface{}{
		"@odata.context": "/redfish/v1/$metadata#SessionService.SessionService",
		"@odata.id":      "/redfish/v1/SessionService",
		"@odata.type":    "#SessionService.v1_1_9.SessionService",
		"Id":             "SessionService",
		"Name":           "Session Service",
		"Description":    "Session Service for DMT Console Redfish API",
		"Status": map[string]interface{}{
			"State":  "Enabled",
			"Health": "OK",
		},
		"ServiceEnabled": true,
		"SessionTimeout": 1800,
		"Sessions": map[string]interface{}{
			"@odata.id": "/redfish/v1/SessionService/Sessions",
		},
		"SessionsCount": sessionCount,
	}

	c.JSON(http.StatusOK, response)
}

// GetRedfishV1SessionServiceSessions handles GET /redfish/v1/SessionService/Sessions
func (r *RedfishServer) GetRedfishV1SessionServiceSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionList, err := r.SessionUC.ListSessions()
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to list sessions: %w", err))

		return
	}

	members := make([]map[string]interface{}, 0, len(sessionList))
	for _, session := range sessionList {
		members = append(members, map[string]interface{}{
			"@odata.id": "/redfish/v1/SessionService/Sessions/" + session.ID,
		})
	}

	response := map[string]interface{}{
		"@odata.context":      "/redfish/v1/$metadata#SessionCollection.SessionCollection",
		"@odata.id":           "/redfish/v1/SessionService/Sessions",
		"@odata.type":         "#SessionCollection.SessionCollection",
		"Name":                "Session Collection",
		"Members":             members,
		"Members@odata.count": len(members),
	}

	c.JSON(http.StatusOK, response)
}

// PostRedfishV1SessionServiceSessions handles POST /redfish/v1/SessionService/Sessions
func (r *RedfishServer) PostRedfishV1SessionServiceSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	// Parse request body
	var request struct {
		UserName string `json:"UserName" binding:"required"`
		Password string `json:"Password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequestError(c, "Invalid request body")

		return
	}

	// Get client info
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Create session
	session, token, err := r.SessionUC.CreateSession(
		request.UserName,
		request.Password,
		clientIP,
		userAgent,
	)
	if err != nil {
		UnauthorizedError(c)

		return
	}

	// Set response headers
	c.Header("X-Auth-Token", token)
	c.Header("Location", "/redfish/v1/SessionService/Sessions/"+session.ID)

	// Return session resource
	c.JSON(http.StatusCreated, session.ToRedfishResponse())
}

// GetRedfishV1SessionServiceSessionsSessionId handles GET /redfish/v1/SessionService/Sessions/{SessionId}.
//
//nolint:revive // Method name must match OpenAPI-generated interface
func (r *RedfishServer) GetRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	SetRedfishHeaders(c)

	if sessionId == "" {
		BadRequestError(c, "Session ID required")

		return
	}

	session, err := r.SessionUC.GetSession(sessionId)
	if err != nil {
		if errors.Is(err, sessions.ErrSessionNotFound) || errors.Is(err, sessions.ErrSessionExpired) {
			NotFoundError(c, "Session", sessionId)

			return
		}

		InternalServerError(c, fmt.Errorf("failed to retrieve session: %w", err))

		return
	}

	c.JSON(http.StatusOK, session.ToRedfishResponse())
}

// DeleteRedfishV1SessionServiceSessionsSessionId handles DELETE /redfish/v1/SessionService/Sessions/{SessionId}.
//
//nolint:revive // Method name must match OpenAPI-generated interface
func (r *RedfishServer) DeleteRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	SetRedfishHeaders(c)

	if sessionId == "" {
		BadRequestError(c, "Session ID required")

		return
	}

	err := r.SessionUC.DeleteSession(sessionId)
	if err != nil {
		if errors.Is(err, sessions.ErrSessionNotFound) {
			NotFoundError(c, "Session", sessionId)

			return
		}

		InternalServerError(c, fmt.Errorf("failed to delete session: %w", err))

		return
	}

	c.Status(http.StatusNoContent)
}
