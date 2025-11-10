// Package redfish implements the Redfish module for DMT.
package redfish

import (
	"github.com/gin-gonic/gin"

	dmtconfig "github.com/device-management-toolkit/console/config"
	dmtusecase "github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishgenerated "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishhandler "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
)

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
}

const (
	ModuleName    = "redfish"
	ModuleVersion = "1.0.0"
)

var (
	server       *redfishhandler.RedfishServer
	moduleConfig *Config
)

// Initialize initializes the Redfish module with DMT infrastructure.
func Initialize(_ *gin.Engine, log logger.Interface, _ *db.SQL, usecases *dmtusecase.Usecases, _ *dmtconfig.Config) error {
	// Initialize configuration with defaults
	moduleConfig = &Config{
		Enabled:      true,
		AuthRequired: false,
		BaseURL:      "/redfish/v1",
	}

	// Create Redfish-specific repository and use case using DMT's device management
	devicesUC, ok := usecases.Devices.(*devices.UseCase)
	if !ok {
		ctx.Logger.Error("Failed to cast Devices usecase to *devices.UseCase")

		return nil // Return nil to not block other plugins
	}

	repo := usecase.NewWsmanComputerSystemRepo(devicesUC, ctx.Logger)
	computerSystemUC := &usecase.ComputerSystemUseCase{Repo: repo}

	// Initialize the Redfish server with shared infrastructure
	server = &redfishhandler.RedfishServer{
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

	// Enable HandleMethodNotAllowed to return 405 for wrong HTTP methods
	ctx.Router.HandleMethodNotAllowed = true

	// Register the Redfish handlers directly on the main router engine
	// This ensures routes are /redfish/v1/* and not /api/redfish/v1/*
	redfishgenerated.RegisterHandlersWithOptions(router, server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: createErrorHandler(),
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
