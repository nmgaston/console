// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"

// ComputerSystemRepository defines the interface for computer system data access.
type ComputerSystemRepository interface {
	GetByID(systemID string) (*redfishv1.ComputerSystem, error)
	GetAll() ([]*redfishv1.ComputerSystem, error)
	UpdatePowerState(systemID string, state redfishv1.PowerState) error
}
