// Package v1 provides HTTP handlers for Redfish Computer System endpoints.
// This implements the HTTP interface for computer system resources.
package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	systemsBasePath = "/redfish/v1/Systems"
)

// ComputerSystemHandler handles Computer System related HTTP requests.
type ComputerSystemHandler struct {
	usecase *usecase.ComputerSystemUseCase
	logger  logger.Interface
}

// CreateComputerSystemHandler creates a new computer system handler
func CreateComputerSystemHandler(
	uc *usecase.ComputerSystemUseCase,
	log logger.Interface,
) *ComputerSystemHandler {
	return &ComputerSystemHandler{
		usecase: uc,
		logger:  log,
	}
}

// GetComputerSystem handles GET /redfish/v1/Systems/{systemId}
func (h *ComputerSystemHandler) GetComputerSystem(c *gin.Context) {
	startTime := time.Now()
	systemID := c.Param("systemId")

	h.logger.Info("Processing GetComputerSystem request",
		"systemID", systemID,
		"requestID", c.GetString(string(requestIDKey)),
		"userID", c.GetString(string(userIDKey)))

	// Create context with userID for use case
	ctx := context.WithValue(c.Request.Context(), userIDKey, c.GetString(string(userIDKey)))

	// Call use case to get computer system
	system, err := h.usecase.GetComputerSystem(ctx, systemID)
	if err != nil {
		h.handleGetSystemError(c, err, systemID)

		return
	}

	// Add response metadata
	c.Header("X-Request-ID", c.GetString(string(requestIDKey)))
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("ETag", fmt.Sprintf(`\"%s-v1\"`, systemID))

	c.JSON(http.StatusOK, system)

	h.logger.Info("GetComputerSystem request completed successfully",
		"systemID", systemID,
		"duration", time.Since(startTime))
}

// GetComputerSystemCollection handles GET /redfish/v1/Systems/
func (h *ComputerSystemHandler) GetComputerSystemCollection(c *gin.Context) {
	startTime := time.Now()

	h.logger.Info("Processing GetComputerSystemCollection request",
		"requestID", c.GetString(string(requestIDKey)),
		"userID", c.GetString(string(userIDKey)))

	// Create context with userID for use case
	ctx := context.WithValue(c.Request.Context(), userIDKey, c.GetString(string(userIDKey)))

	// Call use case to get all systems
	systemIDs, err := h.usecase.GetAll(ctx)
	if err != nil {
		h.logger.Error("Failed to get computer system collection", "error", err)
		InternalServerError(c, err)

		return
	}

	// Convert system IDs to members array
	members := make([]generated.OdataV4IdRef, 0, len(systemIDs))
	for _, systemID := range systemIDs {
		if systemID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr(systemsBasePath + "/" + systemID),
			})
		}
	}

	// Create response following Redfish specification
	response := h.createSystemCollectionResponse(members)

	// Add response metadata
	c.Header("X-Request-ID", c.GetString(string(requestIDKey)))
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("Cache-Control", "max-age=300") // Cache for 5 minutes

	c.JSON(http.StatusOK, response)

	h.logger.Info("GetComputerSystemCollection request completed successfully",
		"count", len(systemIDs),
		"duration", time.Since(startTime))
}

// PostComputerSystemReset handles POST /redfish/v1/Systems/{systemId}/Actions/ComputerSystem.Reset
func (h *ComputerSystemHandler) PostComputerSystemReset(c *gin.Context) {
	startTime := time.Now()
	systemID := c.Param("systemId")

	h.logger.Info("Processing PostComputerSystemReset request",
		"systemID", systemID,
		"requestID", c.GetString(string(requestIDKey)),
		"userID", c.GetString(string(userIDKey)))

	// Parse request body
	var req generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid JSON in reset request", "error", err)
		MalformedJSONError(c)

		return
	}

	// Validate reset type
	if req.ResetType == nil || *req.ResetType == "" {
		h.logger.Warn("Missing ResetType in request")
		PropertyMissingError(c, "ResetType")

		return
	}

	h.logger.Info("Executing system reset",
		"systemID", systemID,
		"resetType", *req.ResetType)

	// Call use case to execute reset
	// Create context with userID for use case
	ctx := context.WithValue(c.Request.Context(), userIDKey, c.GetString(string(userIDKey)))

	err := h.usecase.SetPowerState(ctx, systemID, *req.ResetType)
	if err != nil {
		h.handleResetError(c, err, systemID, string(*req.ResetType))

		return
	}

	// Create task response
	task := h.createResetTaskResponse(systemID, string(*req.ResetType))

	// Add response metadata
	c.Header("X-Request-ID", c.GetString(string(requestIDKey)))
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("Location", fmt.Sprintf("/redfish/v1/TaskService/Tasks/%s", task["Id"]))

	c.JSON(http.StatusAccepted, task)

	h.logger.Info("PostComputerSystemReset request completed successfully",
		"systemID", systemID,
		"resetType", *req.ResetType,
		"duration", time.Since(startTime))
}

// Helper methods

func (h *ComputerSystemHandler) handleGetSystemError(c *gin.Context, err error, systemID string) {
	switch {
	case errors.Is(err, usecase.ErrSystemNotFound):
		h.logger.Warn("Computer system not found", "systemID", systemID)
		NotFoundError(c, "System")
	default:
		h.logger.Error("Unexpected use case error", "error", err)
		InternalServerError(c, err)
	}
}

func (h *ComputerSystemHandler) handleResetError(c *gin.Context, err error, systemID, resetType string) {
	switch {
	case errors.Is(err, usecase.ErrSystemNotFound):
		h.logger.Warn("Computer system not found for reset", "systemID", systemID)
		NotFoundError(c, systemID)
	case errors.Is(err, usecase.ErrInvalidResetType):
		h.logger.Warn("Invalid reset type", "resetType", resetType)
		BadRequestError(c, fmt.Sprintf("Invalid reset type: %s", resetType))
	case errors.Is(err, usecase.ErrPowerStateConflict):
		h.logger.Warn("Power state conflict", "resetType", resetType)
		PowerStateConflictError(c, resetType)
	case errors.Is(err, usecase.ErrUnsupportedPowerState):
		h.logger.Warn("Unsupported power state", "resetType", resetType)
		BadRequestError(c, fmt.Sprintf("Unsupported power state: %s", resetType))
	default:
		h.logger.Error("Unexpected use case error during reset", "error", err)
		InternalServerError(c, err)
	}
}

func (h *ComputerSystemHandler) createSystemCollectionResponse(members []generated.OdataV4IdRef) *generated.ComputerSystemCollectionComputerSystemCollection {
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection")
	odataType := generated.OdataV4Type("#ComputerSystemCollection.ComputerSystemCollection")
	odataID := systemsBasePath

	return &generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      &odataContext,
		OdataId:           &odataID,
		OdataType:         &odataType,
		Name:              "Computer System Collection",
		Description:       CreateDescription("Collection of Computer Systems", h.logger),
		MembersOdataCount: Int64Ptr(int64(len(members))),
		Members:           &members,
	}
}

// CreateDescription creates a Description from a string using ResourceDescription.
// If an error occurs during description creation, it logs the error and returns nil.
// This allows the calling code to continue with a nil description while ensuring
// the error is captured for debugging purposes.
func CreateDescription(desc string, lgr logger.Interface) *generated.ComputerSystemCollectionComputerSystemCollection_Description {
	description := &generated.ComputerSystemCollectionComputerSystemCollection_Description{}
	if err := description.FromResourceDescription(desc); err != nil {
		if lgr != nil {
			lgr.Error("Failed to create description from resource description",
				"error", err,
				"input", desc)
		}

		return nil
	}

	return description
}

func (h *ComputerSystemHandler) createResetTaskResponse(systemID, resetType string) map[string]interface{} {
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	return map[string]interface{}{
		"@odata.context": "/redfish/v1/$metadata#Task.Task",
		"@odata.id":      fmt.Sprintf("/redfish/v1/TaskService/Tasks/%s", taskID),
		"@odata.type":    "#Task.v1_6_0.Task",
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   fmt.Sprintf("System %s reset with type %s completed successfully", systemID, resetType),
				"MessageId": "Base.1.22.0.Success",
				"Severity":  "OK",
			},
		},
		"Name":       "System Reset Task",
		"StartTime":  now,
		"TaskState":  "Completed",
		"TaskStatus": "OK",
	}
}
