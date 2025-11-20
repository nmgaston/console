// Package v1 provides HTTP handlers for Redfish service root endpoints.
// This implements the HTTP interface for service root resources.
package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// ServiceRootHandler handles service root related HTTP requests.
type ServiceRootHandler struct {
	logger logger.Interface
}

// CreateServiceRootHandler creates a new service root handler
func CreateServiceRootHandler(log logger.Interface) *ServiceRootHandler {
	return &ServiceRootHandler{
		logger: log,
	}
}

// GetServiceRoot handles GET /redfish/v1/
func (h *ServiceRootHandler) GetServiceRoot(c *gin.Context) {
	startTime := time.Now()

	h.logger.Info("Processing GetServiceRoot request",
		"requestID", c.GetString(string(requestIDKey)),
		"userID", c.GetString(string(userIDKey)))

	// Create service root response (static data for now)
	response := h.createServiceRootResponse()

	// Add response metadata
	c.Header("X-Request-ID", c.GetString(string(requestIDKey)))
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("Cache-Control", "max-age=300") // Cache for 5 minutes

	c.JSON(http.StatusOK, response)

	h.logger.Info("GetServiceRoot request completed successfully",
		"duration", time.Since(startTime))
}

// GetRedfishV1 handles GET /redfish/v1 (alias for service root)
func (h *ServiceRootHandler) GetRedfishV1(c *gin.Context) {
	h.GetServiceRoot(c)
}

// createServiceRootResponse creates a Redfish service root response
func (h *ServiceRootHandler) createServiceRootResponse() *generated.ServiceRootServiceRoot {
	// OData fields
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ServiceRoot.ServiceRoot")
	odataType := generated.OdataV4Type("#ServiceRoot.v1_15_1.ServiceRoot")
	odataID := "/redfish/v1/"

	// Create links to main resources
	systems := "/redfish/v1/Systems"

	// UUID for this service instance
	// NOTE: In production, this should be persistent across restarts
	uuid := "12345678-1234-5678-9abc-123456789012"

	// Version information
	version := "1.18.0"

	return &generated.ServiceRootServiceRoot{
		OdataContext: &odataContext,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           "RootService",
		Name:         "Root Service",
		UUID:         &uuid,

		// Protocol information
		RedfishVersion: &version,

		// Links to main resources
		Systems: &generated.OdataV4IdRef{
			OdataId: &systems,
		},
	}
}
