// Package redfish provides interfaces for accessing Redfish computer system data.
package redfish

import "github.com/device-management-toolkit/console/internal/entity/redfish/v1"

// ComputerSystemRepository defines the interface for computer system data access.
type ComputerSystemRepository interface {
	GetByID(systemID string) (*redfish.ComputerSystem, error)
	GetAll() ([]*redfish.ComputerSystem, error)
}
