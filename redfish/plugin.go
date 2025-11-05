// Package redfish implements the Redfish module for DMT.
package redfish

import (
	"errors"

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

// ErrDevicesCastFailed is returned when the devices use case cannot be cast to the expected type.
var ErrDevicesCastFailed = errors.New("failed to cast devices use case")

// Config holds redfish-specific configuration.
type Config struct {
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
		return ErrDevicesCastFailed
	}

	repo := redfishusecase.NewWsmanComputerSystemRepo(devicesUC)
	computerSystemUC := &redfishusecase.ComputerSystemUseCase{Repo: repo}

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

	// Register the Redfish handlers directly on the main router engine
	// This ensures routes are /redfish/v1/* and not /api/redfish/v1/*
	redfishgenerated.RegisterHandlersWithOptions(router, server, redfishgenerated.GinServerOptions{
		BaseURL:      "",
		ErrorHandler: createErrorHandler(),
	})

	log.Info("Redfish API routes registered successfully")

	return nil
}

// createErrorHandler creates an error handler for OpenAPI-generated routes.
func createErrorHandler() func(*gin.Context, error, int) {
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
