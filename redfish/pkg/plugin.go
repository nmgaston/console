// Package redfish provides the public interface for registering the Redfish plugin.
package redfish

import (
	"github.com/device-management-toolkit/console/pkg/plugin"
	redfishplugin "github.com/device-management-toolkit/console/redfish"
)

// NewPlugin creates and returns a new Redfish plugin instance that implements the Plugin interface.
// This is the public entry point for integrating the Redfish module with DMT.
func NewPlugin() plugin.Plugin {
	return redfishplugin.NewPlugin()
}
