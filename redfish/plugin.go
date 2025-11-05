// Package redfish implements the Redfish plugin for DMT using the plugin architecture.
package redfish

import (
	"errors"

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
)

// NewPlugin creates a new Redfish plugin instance.
func NewPlugin() *Plugin {
	return &Plugin{
		config: &PluginConfig{
			Enabled:      true,
			AuthRequired: false,
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
	// Create Redfish-specific repository and use case using DMT's device management
	devicesUC, ok := ctx.Usecases.Devices.(*devices.UseCase)
	if !ok {
		return ErrDevicesCastFailed
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

// RegisterRoutes registers Redfish API routes.
func (p *Plugin) RegisterRoutes(ctx *plugin.Context, _, _ *gin.RouterGroup) error {
	if !p.config.Enabled {
		ctx.Logger.Info("Redfish plugin is disabled, skipping route registration")

		return nil
	}

	// Register the Redfish handlers directly on the main router engine
	// This ensures routes are /redfish/v1/* and not /api/redfish/v1/*
	redfishgenerated.RegisterHandlersWithOptions(ctx.Router, p.server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: p.createErrorHandler(),
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
	const (
		httpStatusUnauthorized     = 401
		httpStatusForbidden        = 403
		httpStatusMethodNotAllowed = 405
		httpStatusBadRequest       = 400
	)

	return func(c *gin.Context, err error, statusCode int) {
		switch statusCode {
		case httpStatusUnauthorized:
			redfishhandler.UnauthorizedError(c)
		case httpStatusForbidden:
			redfishhandler.ForbiddenError(c)
		case httpStatusMethodNotAllowed:
			redfishhandler.MethodNotAllowedError(c)
		case httpStatusBadRequest:
			redfishhandler.BadRequestError(c, err.Error(), "Base.1.11.GeneralError", "Check your request body and parameters.", "Critical")
		default:
			redfishhandler.InternalServerError(c, err)
		}
	}
}
