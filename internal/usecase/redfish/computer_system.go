// Package redfish provides interfaces for accessing Redfish computer system data.
package redfish

import (
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/redfish/v1"
)

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
type ComputerSystemUseCase struct {
	Repo ComputerSystemRepository
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and populates OData fields.
func (uc *ComputerSystemUseCase) GetComputerSystem(systemID string) (*redfish.ComputerSystem, error) {
	system, err := uc.Repo.GetByID(systemID)
	if err != nil {
		return nil, err
	}

	// Business logic: generate OData fields
	system.ODataID = "/redfish/v1/Systems/" + systemID
	system.ODataType = "#ComputerSystem.v1_22_0.ComputerSystem"

	return system, nil
}

// SetPowerState sets the power state of the ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(id string, state redfish.PowerState) error {
	// Validate state is in the allowed list
	switch state {
	case redfish.PowerStateOn, redfish.PowerStateOff, redfish.ResetTypeForceOff, redfish.ResetTypeForceRestart, redfish.ResetTypePowerCycle:
		// valid
	default:
		return ErrInvalidPowerState
	}

	// Get current system state to check for conflicts
	system, err := uc.Repo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if the requested state change is valid given the current state
	currentState := system.PowerState
	if !isValidStateTransition(currentState, state) {
		return ErrPowerStateConflict
	}

	return uc.Repo.UpdatePowerState(id, state)
}

// isValidStateTransition checks if a power state transition is allowed.
func isValidStateTransition(current, requested redfish.PowerState) bool {
	// If already in the requested state (for simple On/Off states), it's a conflict
	if current == requested {
		return false
	}

	// Additional conflict rules:
	// - Can't power on if already on
	if current == redfish.PowerStateOn && requested == redfish.ResetTypeOn {
		return false
	}
	// - Can't force off or graceful shutdown if already off
	if current == redfish.PowerStateOff && (requested == redfish.ResetTypeForceOff || requested == redfish.PowerStateOff) {
		return false
	}
	// - Can't restart or power cycle if system is off
	if current == redfish.PowerStateOff && (requested == redfish.ResetTypeForceRestart || requested == redfish.ResetTypePowerCycle) {
		return false
	}

	return true
}

// ErrInvalidPowerState is returned when an invalid power state is provided.
var ErrInvalidPowerState = errors.New("invalid power state")

// ErrPowerStateConflict is returned when the requested power state transition is not allowed.
var ErrPowerStateConflict = errors.New("power state conflict")
