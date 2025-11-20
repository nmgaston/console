// Package v1 provides HTTP handlers for Redfish Computer Systems endpoints.
package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// Systems-specific OData metadata constants
	systemsOdataContextCollection = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	systemsOdataIDCollection      = "/redfish/v1/Systems"
	systemsOdataTypeCollection    = "#ComputerSystemCollection.ComputerSystemCollection"
	systemsCollectionTitle        = "Computer System Collection"
	systemsCollectionDescription  = "Collection of Computer Systems"

	// Systems path patterns
	systemsBasePath = "/redfish/v1/Systems/"
)

// SystemsHandler handles HTTP requests for Computer Systems resources.
type SystemsHandler struct {
	computerSystemUC ComputerSystemUseCase
	logger           logger.Interface
}

// ComputerSystemUseCase defines the interface for computer systems operations.
type ComputerSystemUseCase interface {
	GetAll(ctx context.Context) ([]string, error)
	GetComputerSystem(ctx context.Context, systemID string) (*generated.ComputerSystemComputerSystem, error)
}

// NewSystemsHandler creates a new systems handler with its dependencies.
func NewSystemsHandler(computerSystemUC ComputerSystemUseCase, log logger.Interface) *SystemsHandler {
	return &SystemsHandler{
		computerSystemUC: computerSystemUC,
		logger:           log,
	}
}

// GetSystemsCollection handles GET /redfish/v1/Systems
func (h *SystemsHandler) GetSystemsCollection(c *gin.Context) {
	ctx := c.Request.Context()

	systemIDs, err := h.computerSystemUC.GetAll(ctx)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to retrieve computer systems collection", "error", err)
		}

		InternalServerError(c, err)

		return
	}

	members := h.transformToMembers(systemIDs)
	collection := h.buildSystemsCollectionResponse(members)

	c.JSON(http.StatusOK, collection)
}

// GetSystemByID handles GET /redfish/v1/Systems/{systemId}
func (h *SystemsHandler) GetSystemByID(c *gin.Context) {
	ctx := c.Request.Context()
	systemID := c.Param("ComputerSystemId")

	if systemID == "" {
		BadRequestError(c, "Computer system ID is required")

		return
	}

	system, err := h.computerSystemUC.GetComputerSystem(ctx, systemID)
	if err != nil {
		h.handleGetSystemError(c, err, systemID)

		return
	}

	c.JSON(http.StatusOK, system)
}

// transformToMembers converts system IDs to OData member references.
func (h *SystemsHandler) transformToMembers(systemIDs []string) []generated.OdataV4IdRef {
	members := make([]generated.OdataV4IdRef, 0, len(systemIDs))
	for _, systemID := range systemIDs {
		if systemID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr(systemsBasePath + systemID),
			})
		}
	}

	return members
}

// buildSystemsCollectionResponse constructs the systems collection response.
func (h *SystemsHandler) buildSystemsCollectionResponse(members []generated.OdataV4IdRef) generated.ComputerSystemCollectionComputerSystemCollection {
	return generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr(systemsOdataContextCollection),
		OdataId:           StringPtr(systemsOdataIDCollection),
		OdataType:         StringPtr(systemsOdataTypeCollection),
		Name:              systemsCollectionTitle,
		Description:       CreateDescription(systemsCollectionDescription, h.logger),
		MembersOdataCount: Int64Ptr(int64(len(members))),
		Members:           &members,
	}
}

// handleGetSystemError handles errors from GetComputerSystem operations.
func (h *SystemsHandler) handleGetSystemError(c *gin.Context, err error, systemID string) {
	switch {
	case errors.Is(err, usecase.ErrSystemNotFound):
		NotFoundError(c, "System", systemID)
	default:
		if h.logger != nil {
			h.logger.Error("Failed to retrieve computer system",
				"systemID", systemID,
				"error", err)
		}

		InternalServerError(c, err)
	}
}
