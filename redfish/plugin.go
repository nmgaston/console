// Package redfish implements the Redfish module for DMT.
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
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// ErrDevicesCastFailed is returned when the devices use case cannot be cast to the expected type.
var ErrDevicesCastFailed = errors.New("failed to cast devices use case")

// Plugin represents the Redfish plugin for DMT.
type Plugin struct {
	server *v1.RedfishServer
	config *PluginConfig
}

// PluginConfig holds plugin-specific configuration.
type PluginConfig struct {
	Enabled      bool   `yaml:"enabled" env:"REDFISH_ENABLED"`
	AuthRequired bool   `yaml:"auth_required" env:"REDFISH_AUTH_REQUIRED"`
	BaseURL      string `yaml:"base_url" env:"REDFISH_BASE_URL"`
}

const (
	ModuleName    = "redfish"
	ModuleVersion = "1.0.0"

	// HTTP status codes.
	statusBadRequest       = 400
	statusUnauthorized     = 401
	statusForbidden        = 403
	statusMethodNotAllowed = 405
)

var (
	server       *v1.RedfishServer
	moduleConfig *PluginConfig
)

// Initialize initializes the Redfish module with DMT infrastructure.
func Initialize(_ *gin.Engine, log logger.Interface, _ *db.SQL, usecases *dmtusecase.Usecases, _ *dmtconfig.Config) error {
	// Initialize configuration with defaults
	moduleConfig = &PluginConfig{
		Enabled:      true,
		AuthRequired: true,
		BaseURL:      "/redfish/v1",
	}

	// Create Redfish-specific repository and use case using DMT's device management
	devicesUC, ok := usecases.Devices.(*devices.UseCase)
	if !ok {
		log.Error("Failed to cast Devices usecase to *devices.UseCase")

		return nil // Return nil to not block other plugins
	}

	repo := redfishusecase.NewWsmanComputerSystemRepo(devicesUC, log)
	computerSystemUC := &redfishusecase.ComputerSystemUseCase{Repo: repo}

	// Initialize the Redfish server with shared infrastructure
	server = &v1.RedfishServer{
		ComputerSystemUC: computerSystemUC,
	}

	log.Info("Redfish module initialized successfully")

	return nil
}

// RegisterRoutes registers Redfish API routes.
func RegisterRoutes(router *gin.Engine, log logger.Interface) error {
	if !moduleConfig.Enabled {
		log.Info("Redfish module is disabled, skipping route registration")

		return nil
	}

	if p.config.AuthRequired {
		// Apply Basic Auth middleware to OpenAPI-defined protected endpoints
		adminUsername := ctx.Config.AdminUsername
		adminPassword := ctx.Config.AdminPassword
		basicAuthMiddleware := redfishhandler.BasicAuthValidator(adminUsername, adminPassword)

		// Register handlers with OpenAPI-spec-compliant middleware
		redfishgenerated.RegisterHandlersWithOptions(ctx.Router, p.server, redfishgenerated.GinServerOptions{
			BaseURL:      "",
			ErrorHandler: p.createErrorHandler(),
			Middlewares: []redfishgenerated.MiddlewareFunc{
				// OpenAPI-spec-driven selective authentication
				func(c *gin.Context) {
					path := c.Request.URL.Path

					// Public endpoints as defined in OpenAPI spec (security: [{}])
					if path == "/redfish/v1/" || path == "/redfish/v1/$metadata" {
						c.Next()

						return
					}

					// Protected endpoints as defined in OpenAPI spec (security: [{"BasicAuth": []}])
					if strings.HasPrefix(path, "/redfish/v1/") {
						basicAuthMiddleware(c)

						return
					}

					// Default: no authentication
					c.Next()
				},
			},
		})

		ctx.Logger.Info("Redfish API routes registered with OpenAPI-spec-driven Basic Auth")
	} else {
		// Register without authentication (all endpoints public)
		redfishgenerated.RegisterHandlersWithOptions(ctx.Router, p.server, redfishgenerated.GinServerOptions{
			BaseURL:      "",
			ErrorHandler: p.createErrorHandler(),
		})

		ctx.Logger.Info("Redfish API routes registered without authentication")
	}

	// Enable HandleMethodNotAllowed to return 405 for wrong HTTP methods
	router.HandleMethodNotAllowed = true

	// Add NoMethod handler for Redfish routes to return 405 with proper error
	router.NoMethod(func(c *gin.Context) {
		// Only handle Redfish routes
		if len(c.Request.URL.Path) >= 10 && c.Request.URL.Path[:10] == "/redfish/v" {
			v1.MethodNotAllowedError(c)
		}
	})

	log.Info("Redfish API routes registered successfully")

	return nil
}

// createErrorHandler creates an error handler for OpenAPI-generated routes.
func (p *Plugin) createErrorHandler() func(*gin.Context, error, int) {
	return func(c *gin.Context, err error, statusCode int) {
		switch statusCode {
		case statusUnauthorized:
			v1.UnauthorizedError(c)
		case statusForbidden:
			v1.ForbiddenError(c)
		case statusMethodNotAllowed:
			v1.MethodNotAllowedError(c)
		case statusBadRequest:
			v1.BadRequestError(c, err.Error())
		default:
			v1.InternalServerError(c, err)
		}
	}
}
