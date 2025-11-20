// Package v1 provides HTTP route orchestration for Redfish v1 API.
// This file implements the HTTP routing layer with proper dependency management.
package v1

import (
	"github.com/gin-gonic/gin"

	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// Router orchestrates HTTP routing with proper dependency injection.
// This represents the main composition root where all handlers are configured.
type Router struct {
	config                *dmtconfig.Config
	logger                logger.Interface
	computerSystemHandler *ComputerSystemHandler
	serviceRootHandler    *ServiceRootHandler
	healthHandler         *HealthHandler
	metadataHandler       *MetadataHandler
}

// CreateRouter creates a new router with all handlers properly configured.
func CreateRouter(
	config *dmtconfig.Config,
	log logger.Interface,
	computerSystemUC *usecase.ComputerSystemUseCase,
) *Router {
	// Create handlers for each resource type
	computerSystemHandler := CreateComputerSystemHandler(computerSystemUC, log)
	serviceRootHandler := CreateServiceRootHandler(log)
	healthHandler := CreateHealthHandler(log)
	metadataHandler := CreateMetadataHandler(log)

	return &Router{
		config:                config,
		logger:                log,
		computerSystemHandler: computerSystemHandler,
		serviceRootHandler:    serviceRootHandler,
		healthHandler:         healthHandler,
		metadataHandler:       metadataHandler,
	}
}

// SetupRoutes configures the Gin router with Redfish API routes and middleware.
// This method implements the main route configuration and applies proper middleware.
func (r *Router) SetupRoutes(engine *gin.Engine) {
	r.logger.Info("Setting up Redfish API routes")

	// Apply global middleware
	r.applyGlobalMiddleware(engine)

	// Health endpoint (outside of /redfish namespace)
	engine.GET("/health", r.healthHandler.GetHealth)

	// Redfish v1 API routes
	v1 := engine.Group("/redfish/v1")
	{
		// Apply Redfish-specific middleware
		r.applyRedfishMiddleware(v1)

		// Service root endpoints
		v1.GET("/", r.serviceRootHandler.GetServiceRoot)
		v1.GET("", r.serviceRootHandler.GetRedfishV1) // Handle both with and without trailing slash

		// Metadata endpoint
		v1.GET("/$metadata", r.metadataHandler.GetMetadata)

		// Systems endpoints
		v1.GET("/Systems", r.computerSystemHandler.GetComputerSystemCollection)
		v1.GET("/Systems/", r.computerSystemHandler.GetComputerSystemCollection)
		v1.GET("/Systems/:systemId", r.computerSystemHandler.GetComputerSystem)
		v1.POST("/Systems/:systemId/Actions/ComputerSystem.Reset", r.computerSystemHandler.PostComputerSystemReset)

		// Note: Only Systems endpoints are supported in this implementation
	}

	r.logger.Info("Redfish API routes registered successfully")
}

// applyGlobalMiddleware applies middleware to all routes
func (r *Router) applyGlobalMiddleware(_ *gin.Engine) {
	// Apply global middleware here (CORS, rate limiting, etc.)
	// This handles cross-cutting concerns for all endpoints
}

// applyRedfishMiddleware applies Redfish-specific middleware
func (r *Router) applyRedfishMiddleware(group *gin.RouterGroup) {
	// Apply middleware for cross-cutting concerns (excluding authentication which is handled at component level)
	// Authentication is handled at the component level for selective protection of endpoints
	group.Use(RequestIDMiddleware())
	// Note: AuthenticationMiddleware() removed - handled at component level for selective authentication
	group.Use(LoggingMiddleware(r.logger))
	group.Use(ErrorHandlingMiddleware(r.logger))
}

// Helper functions for backward compatibility and utility

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
