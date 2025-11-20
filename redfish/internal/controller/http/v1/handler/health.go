// Package v1 provides HTTP handlers for health check endpoints.
// This implements the HTTP interface for application health monitoring.
package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// HealthHandler handles health check related HTTP requests.
type HealthHandler struct {
	logger logger.Interface
}

// CreateHealthHandler creates a new health handler
func CreateHealthHandler(log logger.Interface) *HealthHandler {
	return &HealthHandler{
		logger: log,
	}
}

// GetHealth handles GET /health
func (h *HealthHandler) GetHealth(c *gin.Context) {
	startTime := time.Now()

	h.logger.Debug("Processing health check request")

	// Simple health check response
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "redfish-api",
		"version":   "1.0.0",
	}

	// Add response metadata
	c.Header("X-Response-Time", time.Since(startTime).String())
	c.Header("Cache-Control", "no-cache")

	c.JSON(http.StatusOK, health)

	h.logger.Debug("Health check completed", "duration", time.Since(startTime))
}
