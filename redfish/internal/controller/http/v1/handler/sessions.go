// Package v1 provides Redfish v1 API handlers for SessionService
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/entity"
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

// sessionToRedfishResponse converts entity.Session to generated.SessionSession format.
func sessionToRedfishResponse(s *entity.Session) (*generated.SessionSession, error) {
	context := "/redfish/v1/$metadata#Session.Session"
	odataID := "/redfish/v1/SessionService/Sessions/" + s.ID
	odataType := "#Session.v1_8_0.Session"
	name := "User Session"
	descriptionText := "User Session for " + s.Username

	// Create description using union type
	var description generated.SessionSession_Description
	if err := description.FromResourceDescription(descriptionText); err != nil {
		return nil, err
	}

	// Create SessionType as Redfish
	var sessionType generated.SessionSession_SessionType
	if err := sessionType.FromSessionSessionTypes(generated.Redfish); err != nil {
		return nil, err
	}

	return &generated.SessionSession{
		OdataContext:          &context,
		OdataId:               &odataID,
		OdataType:             &odataType,
		Id:                    s.ID,
		Name:                  name,
		Description:           &description,
		UserName:              &s.Username,
		SessionType:           &sessionType,
		CreatedTime:           &s.CreatedTime,
		ClientOriginIPAddress: &s.ClientIP,
		Password:              nil, // Always null in responses per Redfish spec
		Token:                 nil, // Always null in responses per Redfish spec
	}, nil
}

// buildSessionServiceResponse builds the SessionService response object using generated types
// This method is shared by GET, PATCH, and PUT to avoid duplication
func (r *RedfishServer) buildSessionServiceResponse() (*generated.SessionServiceSessionService, error) {
	context := "/redfish/v1/$metadata#SessionService.SessionService"
	odataID := "/redfish/v1/SessionService"
	odataType := "#SessionService.v1_2_0.SessionService"
	sessionTimeout := int64(1800)
	serviceEnabled := true

	// Create description
	var description generated.SessionServiceSessionService_Description
	if err := description.FromResourceDescription("Session Service for DMT Console Redfish API"); err != nil {
		return nil, fmt.Errorf("failed to create description: %w", err)
	}

	// Create status with union types
	var statusState generated.ResourceStatus_State
	if err := statusState.FromResourceStatusState1("Enabled"); err != nil {
		return nil, fmt.Errorf("failed to create status state: %w", err)
	}

	var statusHealth generated.ResourceStatus_Health
	if err := statusHealth.FromResourceStatusHealth1("OK"); err != nil {
		return nil, fmt.Errorf("failed to create status health: %w", err)
	}

	return &generated.SessionServiceSessionService{
		OdataContext: &context,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           "SessionService",
		Name:         "Session Service",
		Description:  &description,
		Status: &generated.ResourceStatus{
			State:  &statusState,
			Health: &statusHealth,
		},
		ServiceEnabled: &serviceEnabled,
		SessionTimeout: &sessionTimeout,
		Sessions: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/SessionService/Sessions"),
		},
	}, nil
}

// GetRedfishV1SessionService handles GET /redfish/v1/SessionService
func (r *RedfishServer) GetRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to build session service response: %w", err))

		return
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

	members := make([]generated.OdataV4IdRef, 0, len(sessionList))
	for _, session := range sessionList {
		members = append(members, generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/SessionService/Sessions/" + session.ID),
		})
	}

	context := "/redfish/v1/$metadata#SessionCollection.SessionCollection"
	odataID := "/redfish/v1/SessionService/Sessions"
	odataType := "#SessionCollection.SessionCollection"
	etag := "1"
	membersCount := int64(len(members))

	response := generated.SessionCollectionSessionCollection{
		OdataContext:      &context,
		OdataId:           &odataID,
		OdataType:         &odataType,
		OdataEtag:         &etag,
		Name:              "Session Collection",
		Members:           &members,
		MembersOdataCount: &membersCount,
	}

	c.JSON(http.StatusOK, response)
}

// PostRedfishV1SessionServiceSessions handles POST /redfish/v1/SessionService/Sessions
func (r *RedfishServer) PostRedfishV1SessionServiceSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	// Parse request body using generated type
	var request generated.SessionSession

	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequestError(c, "Invalid request body")

		return
	}

	// Validate required fields
	if request.UserName == nil || request.Password == nil {
		BadRequestError(c, "UserName and Password are required")

		return
	}

	// Get client info
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Create session
	session, token, err := r.SessionUC.CreateSession(
		*request.UserName,
		*request.Password,
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

	// Convert to Redfish response format
	response, err := sessionToRedfishResponse(session)
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to build session response: %w", err))

		return
	}

	c.JSON(http.StatusCreated, response)
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

	// Convert to Redfish response format
	response, err := sessionToRedfishResponse(session)
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to build session response: %w", err))

		return
	}

	c.JSON(http.StatusOK, response)
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

// PatchRedfishV1SessionService handles PATCH /redfish/v1/SessionService
func (r *RedfishServer) PatchRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	var req generated.SessionServiceSessionService
	if err := c.BindJSON(&req); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid request body: %v", err))

		return
	}

	// For now, PATCH returns the current state without modifications
	// TODO: Implement PATCH logic to update configurable properties
	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to build session service response: %w", err))

		return
	}

	c.JSON(http.StatusOK, response)
}

// PutRedfishV1SessionService handles PUT /redfish/v1/SessionService
func (r *RedfishServer) PutRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	var req generated.SessionServiceSessionService
	if err := c.BindJSON(&req); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid request body: %v", err))

		return
	}

	// For now, PUT returns the current state without modifications
	// TODO: Implement PUT logic to replace the resource
	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("failed to build session service response: %w", err))

		return
	}

	c.JSON(http.StatusOK, response)
}
