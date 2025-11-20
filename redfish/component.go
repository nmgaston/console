// Package redfish implements the Redfish component for DMT.
package redfish

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	dmtconfig "github.com/device-management-toolkit/console/config"
	dmtusecase "github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishgenerated "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	v1 "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	"github.com/device-management-toolkit/console/redfish/internal/infrastructure/services"
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// ErrDevicesCastFailed is returned when the devices use case cannot be cast to the expected type.
var ErrDevicesCastFailed = errors.New("failed to cast devices use case")

// Constants for Redfish API paths.
const (
	redfishV1Path       = "/redfish/v1/"
	redfishMetadataPath = "/redfish/v1/$metadata"
	redfishV1PathPrefix = "/redfish/v1/"
)

// ComponentConfig holds component-specific configuration.
type ComponentConfig struct {
	Enabled      bool   `yaml:"enabled" env:"REDFISH_ENABLED"`
	AuthRequired bool   `yaml:"auth_required" env:"REDFISH_AUTH_REQUIRED"`
	BaseURL      string `yaml:"base_url" env:"REDFISH_BASE_URL"`
}

// Server implements the OpenAPI ServerInterface using individual handlers.
type Server struct {
	serviceRootHandler    *v1.ServiceRootHandler
	computerSystemHandler *v1.ComputerSystemHandler
	metadataHandler       *v1.MetadataHandler
}

// GetRedfishV1 implements ServerInterface.
func (rs *Server) GetRedfishV1(c *gin.Context) {
	rs.serviceRootHandler.GetRedfishV1(c)
}

// GetRedfishV1Metadata implements ServerInterface.
func (rs *Server) GetRedfishV1Metadata(c *gin.Context) {
	rs.metadataHandler.GetMetadata(c)
}

// GetRedfishV1Systems implements ServerInterface.
func (rs *Server) GetRedfishV1Systems(c *gin.Context) {
	rs.computerSystemHandler.GetComputerSystemCollection(c)
}

// GetRedfishV1SystemsComputerSystemId implements ServerInterface.
//
//nolint:revive // Method name must match OpenAPI generated interface
func (rs *Server) GetRedfishV1SystemsComputerSystemId(c *gin.Context, _ string) {
	rs.computerSystemHandler.GetComputerSystem(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset implements ServerInterface.
//
//nolint:revive // Method name must match OpenAPI generated interface
func (rs *Server) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, _ string) {
	rs.computerSystemHandler.PostComputerSystemReset(c)
}

var (
	server          *Server
	componentConfig *ComponentConfig
	appConfig       *dmtconfig.Config
)

// Initialize initializes the Redfish component with proper dependency injection and layered architecture.
func Initialize(_ *gin.Engine, log logger.Interface, _ *db.SQL, usecases *dmtusecase.Usecases, config *dmtconfig.Config) error {
	log.Info("Initializing Redfish component")

	// Initialize configuration with defaults
	auth := config.Auth
	componentConfig = &ComponentConfig{
		Enabled:      true,
		AuthRequired: !auth.Disabled,
		BaseURL:      "/redfish/v1",
	}

	// External service dependencies
	devicesUC, ok := usecases.Devices.(*devices.UseCase)
	if !ok {
		log.Error("Failed to cast Devices usecase to *devices.UseCase")

		return ErrDevicesCastFailed
	}

	// Infrastructure Services (Cross-cutting concerns)
	accessPolicy := services.CreateAccessPolicyService(log)
	cache := services.CreateCacheService(log)
	audit := services.CreateAuditService(log)
	metrics := services.CreateMetricsService(log)

	// Repository for data access
	repo := redfishusecase.CreateWsmanComputerSystemRepo(devicesUC, log)

	// Business logic layer
	computerSystemUC := redfishusecase.CreateComputerSystemUseCase(
		repo,
		accessPolicy,
		cache,
		audit,
		metrics,
	)

	// Create individual handlers
	serviceRootHandler := v1.CreateServiceRootHandler(log)
	metadataHandler := v1.CreateMetadataHandler(log)
	computerSystemHandler := v1.CreateComputerSystemHandler(computerSystemUC, log)

	// Create server implementing OpenAPI ServerInterface
	server = &Server{
		serviceRootHandler:    serviceRootHandler,
		computerSystemHandler: computerSystemHandler,
		metadataHandler:       metadataHandler,
	}

	// Store config for later use
	appConfig = config

	log.Info("Redfish component initialized successfully")

	return nil
}

// RegisterRoutes registers Redfish API routes with OpenAPI-generated routing.
func RegisterRoutes(ginRouter *gin.Engine, log logger.Interface) error {
	if !componentConfig.Enabled {
		log.Info("Redfish component is disabled, skipping route registration")

		return nil
	}

	log.Info("Registering Redfish API routes")

	// Authentication is handled within OpenAPI middleware for selective endpoint protection

	// Use OpenAPI-generated routing for full spec compliance
	log.Info("Using OpenAPI-generated routing")

	// Create basic auth middleware for protected endpoints
	var basicAuthMiddleware gin.HandlerFunc

	if componentConfig.AuthRequired {
		auth := appConfig.Auth
		adminUsername := auth.AdminUsername
		adminPassword := auth.AdminPassword
		basicAuthMiddleware = v1.BasicAuthValidator(adminUsername, adminPassword)
	}

	redfishgenerated.RegisterHandlersWithOptions(ginRouter, server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: createErrorHandler(),
		Middlewares: []redfishgenerated.MiddlewareFunc{
			// OpenAPI-spec-driven selective authentication
			func(c *gin.Context) {
				path := c.Request.URL.Path

				// Public endpoints as defined in OpenAPI spec (security: [{}])
				if path == redfishV1Path || path == redfishMetadataPath {
					c.Next()

					return
				}

				// Protected endpoints as defined in OpenAPI spec (security: [{"BasicAuth": []}])
				if strings.HasPrefix(path, redfishV1PathPrefix) {
					if componentConfig.AuthRequired && basicAuthMiddleware != nil {
						basicAuthMiddleware(c)
					} else {
						c.Next()
					}

					return
				}

				// Default: no authentication
				c.Next()
			},
		},
	})

	// Enable HandleMethodNotAllowed to return 405 for wrong HTTP methods
	ginRouter.HandleMethodNotAllowed = true

	// Add NoMethod handler for Redfish routes to return 405 with proper error
	ginRouter.NoMethod(func(c *gin.Context) {
		// Only handle Redfish routes
		if len(c.Request.URL.Path) >= 10 && c.Request.URL.Path[:10] == "/redfish/v" {
			v1.MethodNotAllowedError(c)
		}
	})

	log.Info("Redfish API routes registered successfully")

	return nil
}

// createErrorHandler creates an error handler for OpenAPI-generated routes.
// This provides consistent error handling across all endpoints.
func createErrorHandler() func(*gin.Context, error, int) {
	return func(c *gin.Context, err error, _ int) {
		// Use standard error handling for OpenAPI routes
		v1.InternalServerError(c, err)
	}
}
