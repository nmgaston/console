// Package plugin provides the interface for DMT plugins to integrate with the core infrastructure.
package plugin

import (
	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Context provides shared DMT infrastructure services to plugins.
type Context struct {
	// Core Infrastructure
	Config   *config.Config   // Configuration management with YAML and env overrides
	Logger   logger.Interface // Structured logging with configurable levels
	Database *db.SQL          // Database abstraction with repository pattern

	// Shared Services
	Usecases *usecase.Usecases // Use cases with transaction management

	// HTTP Infrastructure
	Router *gin.Engine // Main HTTP router for middleware pipeline registration
}

// Plugin defines the interface that all DMT plugins must implement.
type Plugin interface {
	// Name returns the plugin name (e.g., "redfish", "openapi", etc.)
	Name() string

	// Version returns the plugin version for compatibility checks
	Version() string

	// Initialize initializes the plugin with access to shared DMT infrastructure.
	// This is called once during application startup.
	Initialize(ctx *Context) error

	// RegisterRoutes registers the plugin's HTTP routes and middleware.
	// The plugin receives a router group and can register protected/unprotected routes.
	RegisterRoutes(ctx *Context, protected, unprotected *gin.RouterGroup) error

	// RegisterMiddleware allows the plugin to register global middleware.
	// This is called before route registration.
	RegisterMiddleware(ctx *Context) error

	// Shutdown performs cleanup when the application is shutting down.
	Shutdown(ctx *Context) error
}

// Manager manages plugin lifecycle and registration.
type Manager struct {
	plugins []Plugin
	ctx     *Context
}

// NewManager creates a new plugin manager with shared infrastructure context.
func NewManager(cfg *config.Config, log logger.Interface, database *db.SQL, usecases *usecase.Usecases, router *gin.Engine) *Manager {
	return &Manager{
		plugins: make([]Plugin, 0),
		ctx: &Context{
			Config:   cfg,
			Logger:   log,
			Database: database,
			Usecases: usecases,
			Router:   router,
		},
	}
}

// Register adds a plugin to the manager.
func (m *Manager) Register(plugin Plugin) {
	m.plugins = append(m.plugins, plugin)
}

// Initialize initializes all registered plugins.
func (m *Manager) Initialize() error {
	for _, plugin := range m.plugins {
		m.ctx.Logger.Debug("Initializing plugin: " + plugin.Name() + " v" + plugin.Version())

		if err := plugin.Initialize(m.ctx); err != nil {
			return err
		}
	}

	return nil
}

// RegisterMiddleware registers middleware for all plugins.
func (m *Manager) RegisterMiddleware() error {
	for _, plugin := range m.plugins {
		if err := plugin.RegisterMiddleware(m.ctx); err != nil {
			return err
		}
	}

	return nil
}

// RegisterRoutes registers routes for all plugins with protected and unprotected groups.
func (m *Manager) RegisterRoutes(protected, unprotected *gin.RouterGroup) error {
	for _, plugin := range m.plugins {
		m.ctx.Logger.Debug("Registering routes for plugin: " + plugin.Name())

		if err := plugin.RegisterRoutes(m.ctx, protected, unprotected); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown shuts down all plugins gracefully.
func (m *Manager) Shutdown() error {
	for _, plugin := range m.plugins {
		m.ctx.Logger.Debug("Shutting down plugin: " + plugin.Name())

		if err := plugin.Shutdown(m.ctx); err != nil {
			// Log error but continue shutting down other plugins
			m.ctx.Logger.Error("Error shutting down plugin " + plugin.Name() + ": " + err.Error())
		}
	}

	return nil
}
