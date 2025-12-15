// Package v1 provides Redfish v1 API handlers for SessionService
package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// SessionHandler handles Redfish SessionService requests
type SessionHandler struct {
	sessionUseCase *sessions.UseCase
	config         *config.Config
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(useCase *sessions.UseCase, cfg *config.Config) *SessionHandler {
	return &SessionHandler{
		sessionUseCase: useCase,
		config:         cfg,
	}
}

// GetSessionService returns the SessionService resource
// GET /redfish/v1/SessionService
func (h *SessionHandler) GetSessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionCount, _ := h.sessionUseCase.GetSessionCount()

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

// ListSessions returns the collection of active sessions
// GET /redfish/v1/SessionService/Sessions
func (h *SessionHandler) ListSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionList, err := h.sessionUseCase.ListSessions()
	if err != nil {
		InternalServerError(c, errors.New("failed to list sessions"))
		return
	}

	members := make([]map[string]interface{}, 0, len(sessionList))
	for _, session := range sessionList {
		members = append(members, map[string]interface{}{
			"@odata.id": "/redfish/v1/SessionService/Sessions/" + session.ID,
		})
	}

	response := map[string]interface{}{
		"@odata.context":        "/redfish/v1/$metadata#SessionCollection.SessionCollection",
		"@odata.id":             "/redfish/v1/SessionService/Sessions",
		"@odata.type":           "#SessionCollection.SessionCollection",
		"Name":                  "Session Collection",
		"Members":               members,
		"Members@odata.count":   len(members),
	}

	c.JSON(http.StatusOK, response)
}

// CreateSession creates a new session
// POST /redfish/v1/SessionService/Sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
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
	session, token, err := h.sessionUseCase.CreateSession(
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

// GetSession returns details of a specific session
// GET /redfish/v1/SessionService/Sessions/{SessionId}
func (h *SessionHandler) GetSession(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionID := c.Param("SessionId")
	if sessionID == "" {
		BadRequestError(c, "Session ID required")
		return
	}

	session, err := h.sessionUseCase.GetSession(sessionID)
	if err != nil {
		if err == sessions.ErrSessionNotFound || err == sessions.ErrSessionExpired {
			NotFoundError(c, "Session", sessionID)
			return
		}

		InternalServerError(c, errors.New("failed to retrieve session"))
		return
	}

	c.JSON(http.StatusOK, session.ToRedfishResponse())
}

// DeleteSession terminates a session (logout)
// DELETE /redfish/v1/SessionService/Sessions/{SessionId}
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionID := c.Param("SessionId")
	if sessionID == "" {
		BadRequestError(c, "Session ID required")
		return
	}

	err := h.sessionUseCase.DeleteSession(sessionID)
	if err != nil {
		if err == sessions.ErrSessionNotFound {
			NotFoundError(c, "Session", sessionID)
			return
		}

		InternalServerError(c, errors.New("failed to delete session"))
		return
	}

	c.Status(http.StatusNoContent)
}

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
