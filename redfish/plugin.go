// Package redfish implements the Redfish plugin for DMT using the plugin architecture.
package redfish

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/plugin"
	redfishgenerated "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishhandler "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// ErrDevicesCastFailed is returned when the devices use case cannot be cast to the expected type.
var ErrDevicesCastFailed = errors.New("failed to cast devices use case")

// Plugin represents the Redfish plugin for DMT.
type Plugin struct {
	server *redfishhandler.RedfishServer
	config *PluginConfig
}

// PluginConfig holds plugin-specific configuration.
type PluginConfig struct {
	Enabled      bool   `yaml:"enabled" env:"REDFISH_ENABLED"`
	AuthRequired bool   `yaml:"auth_required" env:"REDFISH_AUTH_REQUIRED"`
	BaseURL      string `yaml:"base_url" env:"REDFISH_BASE_URL"`
	Version      string
}

const (
	PluginName    = "redfish"
	PluginVersion = "1.0.0"

	// HTTP status codes.
	statusUnauthorized     = 401
	statusForbidden        = 403
	statusBadRequest       = 400
	statusMethodNotAllowed = 405
)

// NewPlugin creates a new Redfish plugin instance.
func NewPlugin() *Plugin {
	return &Plugin{
		config: &PluginConfig{
			Enabled:      true,
			AuthRequired: true,
			BaseURL:      "/redfish/v1",
			Version:      PluginVersion,
		},
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return PluginVersion
}

// Initialize initializes the Redfish plugin with DMT infrastructure.
func (p *Plugin) Initialize(ctx *plugin.Context) error {
	// Override plugin config from environment variables if set
	// This allows runtime configuration without code changes
	if ctx.Config.Disabled {
		p.config.AuthRequired = false // If auth is globally disabled, disable it for Redfish too
	}

	// Create Redfish-specific repository and use case using DMT's device management
	devicesUC, ok := ctx.Usecases.Devices.(*devices.UseCase)
	if !ok {
		ctx.Logger.Error("Failed to cast Devices usecase to *devices.UseCase")

		return nil // Return nil to not block other plugins
	}

	repo := usecase.NewWsmanComputerSystemRepo(devicesUC)
	computerSystemUC := &usecase.ComputerSystemUseCase{Repo: repo}

	// Initialize the Redfish server with shared infrastructure
	p.server = &redfishhandler.RedfishServer{
		ComputerSystemUC: computerSystemUC,
	}

	ctx.Logger.Info("Redfish plugin initialized successfully")

	return nil
}

// RegisterMiddleware registers Redfish-specific middleware.
func (p *Plugin) RegisterMiddleware(ctx *plugin.Context) error {
	ctx.Logger.Info("Registering Redfish middleware")

	// Redfish plugin doesn't need global middleware
	// Authentication middleware is handled per-route in RegisterRoutes
	return nil
}

// RegisterRoutes registers Redfish API routes with OpenAPI-spec-driven authentication.
func (p *Plugin) RegisterRoutes(ctx *plugin.Context, _, _ *gin.RouterGroup) error {
	if !p.config.Enabled {
		ctx.Logger.Info("Redfish plugin is disabled, skipping route registration")

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
	ctx.Router.HandleMethodNotAllowed = true

	// Register the Redfish handlers directly on the main router engine
	// This ensures routes are /redfish/v1/* and not /api/redfish/v1/*
	redfishgenerated.RegisterHandlersWithOptions(ctx.Router, p.server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: p.createErrorHandler(),
	})

	// Add NoMethod handler for Redfish routes to return 405 with proper error
	ctx.Router.NoMethod(func(c *gin.Context) {
		// Only handle Redfish routes
		if len(c.Request.URL.Path) >= 10 && c.Request.URL.Path[:10] == "/redfish/v" {
			redfishhandler.MethodNotAllowedError(c)
		}
	})

	ctx.Logger.Info("Redfish API routes registered successfully")

	return nil
}

// Shutdown performs cleanup for the Redfish plugin.
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	ctx.Logger.Info("Shutting down Redfish plugin")
	// No specific cleanup needed for Redfish plugin
	return nil
}

// createErrorHandler creates an error handler for OpenAPI-generated routes.
func (p *Plugin) createErrorHandler() func(*gin.Context, error, int) {
	return func(c *gin.Context, err error, statusCode int) {
		switch statusCode {
		case statusUnauthorized:
			redfishhandler.UnauthorizedError(c)
		case statusForbidden:
			redfishhandler.ForbiddenError(c)
		case statusMethodNotAllowed:
			redfishhandler.MethodNotAllowedError(c)
		case statusBadRequest:
			redfishhandler.BadRequestError(c, err.Error())
		default:
			redfishhandler.InternalServerError(c, err)
		}
	}
}
