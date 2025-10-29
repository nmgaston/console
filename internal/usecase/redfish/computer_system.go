// Package redfish provides interfaces for accessing Redfish computer system data.
package redfish

import "github.com/device-management-toolkit/console/internal/entity/redfish/v1"

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
type ComputerSystemUseCase struct {
	repo ComputerSystemRepository
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and populates OData fields.
func (uc *ComputerSystemUseCase) GetComputerSystem(systemID string) (*redfish.ComputerSystem, error) {
	system, err := uc.repo.GetByID(systemID)
	if err != nil {
		return nil, err
	}

	// Business logic: generate OData fields
	system.ODataID = "/redfish/v1/Systems/" + systemID
	system.ODataType = "#ComputerSystem.v1_22_0.ComputerSystem"

	return system, nil
}
