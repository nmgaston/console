// Package v1 provides HTTP handlers for Redfish metadata endpoints.
// This implements the HTTP interface for OData metadata resources.
package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// MetadataHandler handles metadata related HTTP requests.
type MetadataHandler struct {
	logger logger.Interface
}

// CreateMetadataHandler creates a new metadata handler
func CreateMetadataHandler(log logger.Interface) *MetadataHandler {
	return &MetadataHandler{
		logger: log,
	}
}

// GetMetadata handles GET /redfish/v1/$metadata
func (h *MetadataHandler) GetMetadata(c *gin.Context) {
	startTime := time.Now()

	h.logger.Info("Processing GetMetadata request",
		"requestID", c.GetString(string(requestIDKey)),
		"userID", c.GetString(string(userIDKey)))

	// Return basic OData metadata document
	// In production, this should return complete CSDL metadata from specification
	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.0">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="Service">
      <EntityContainer Name="Service" Extends="ServiceRoot.v1_0_0.ServiceContainer"/>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	// Add response metadata
	c.Header("X-Request-ID", c.GetString(string(requestIDKey)))
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("Content-Type", "application/xml")
	c.Header("Cache-Control", "max-age=3600") // Cache for 1 hour

	c.String(http.StatusOK, metadata)

	h.logger.Info("GetMetadata request completed successfully",
		"duration", time.Since(startTime))
}
