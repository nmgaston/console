// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
	"github.com/gin-gonic/gin"
)

const (
	// Task state constants from Redfish Task.v1_8_0 specification
	taskStateCompleted = "Completed"

	// Registry message IDs
	msgIDBaseSuccess = "Base.1.22.0.Success"

	// OData metadata constants - Task
	odataContextTask = "/redfish/v1/$metadata#Task.Task"
	odataTypeTask    = "#Task.v1_6_0.Task"
	taskName         = "System Reset Task"
	taskServiceTasks = "/redfish/v1/TaskService/Tasks/"
)

// RedfishServer implements the Redfish API handlers and delegates operations to specialized handlers
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
	Config           *dmtconfig.Config
	Logger           logger.Interface
	Services         []ODataService // Cached OData services loaded from OpenAPI spec
}

// Ensure RedfishServer implements generated.ServerInterface
var _ generated.ServerInterface = (*RedfishServer)(nil)

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr creates a pointer to an int value.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr creates a pointer to an int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr creates a pointer to a bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// Session endpoint stubs - actual implementation in sessions.go
// These methods satisfy the generated.ServerInterface

// GetRedfishV1SessionService handles GET /redfish/v1/SessionService
func (r *RedfishServer) GetRedfishV1SessionService(c *gin.Context) {
	// Implementation delegated to session handler
	c.JSON(501, gin.H{"error": "Not implemented - use session handler"})
}

// GetRedfishV1SessionServiceSessions handles GET /redfish/v1/SessionService/Sessions
func (r *RedfishServer) GetRedfishV1SessionServiceSessions(c *gin.Context) {
	// Implementation delegated to session handler
	c.JSON(501, gin.H{"error": "Not implemented - use session handler"})
}

// PostRedfishV1SessionServiceSessions handles POST /redfish/v1/SessionService/Sessions
func (r *RedfishServer) PostRedfishV1SessionServiceSessions(c *gin.Context) {
	// Implementation delegated to session handler
	c.JSON(501, gin.H{"error": "Not implemented - use session handler"})
}

// GetRedfishV1SessionServiceSessionsSessionId handles GET /redfish/v1/SessionService/Sessions/{SessionId}
func (r *RedfishServer) GetRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	// Implementation delegated to session handler
	c.JSON(501, gin.H{"error": "Not implemented - use session handler"})
}

// DeleteRedfishV1SessionServiceSessionsSessionId handles DELETE /redfish/v1/SessionService/Sessions/{SessionId}
func (r *RedfishServer) DeleteRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	// Implementation delegated to session handler
	c.JSON(501, gin.H{"error": "Not implemented - use session handler"})
}
