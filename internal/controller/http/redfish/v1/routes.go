// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"github.com/gin-gonic/gin"
)

// SetupRedfishV1Routes sets up the Redfish v1 routes on the main router
func SetupRedfishV1Routes(router *gin.Engine) {
	// Create a new Redfish server instance
	redfishServer := NewRedfishServer()

	// Create a route group for Redfish v1 API
	v1Group := router.Group("/redfish/v1")

	// Register the handlers with options
	RegisterHandlersWithOptions(v1Group, redfishServer, GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"code":    "Base.1.11.GeneralError",
					"message": err.Error(),
				},
			})
		},
	})
}

// SetupRedfishV1RoutesProtected sets up the Redfish v1 routes with JWT protection at /redfish/v1
func SetupRedfishV1RoutesProtected(router *gin.Engine, jwtMiddleware gin.HandlerFunc) {
	// Create a new Redfish server instance
	redfishServer := NewRedfishServer()

	// Create a route group for Redfish v1 API with JWT middleware
	v1Group := router.Group("/redfish/v1", jwtMiddleware)

	// Register the handlers with options
	RegisterHandlersWithOptions(v1Group, redfishServer, GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"code":    "Base.1.11.GeneralError",
					"message": err.Error(),
				},
			})
		},
	})
}
