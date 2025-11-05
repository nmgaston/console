// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"errors"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

var (
	// ErrInvalidPowerState is returned when an invalid power state is requested.
	ErrInvalidPowerState = errors.New("invalid power state")

	// ErrPowerStateConflict is returned when a power state transition is not allowed.
	ErrPowerStateConflict = errors.New("power state transition not allowed")

	// ErrInvalidResetType is returned when an invalid reset type is requested.
	ErrInvalidResetType = errors.New("invalid reset type")
)

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
type ComputerSystemUseCase struct {
	Repo ComputerSystemRepository
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and converts it to the generated API type.
func (uc *ComputerSystemUseCase) GetComputerSystem(systemID string) (*generated.ComputerSystemComputerSystem, error) {
	system, err := uc.Repo.GetByID(systemID)
	if err != nil {
		return nil, err
	}

	// Convert from entity type to generated API type
	return convertToGeneratedType(system), nil
}

// SetPowerState validates and sets the power state for a ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(id string, resetType generated.ResourceResetType) error {
	// Validate the reset type
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeForceOn,
		generated.ResourceResetTypeFullPowerCycle,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypeGracefulShutdown,
		generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePause,
		generated.ResourceResetTypePushPowerButton,
		generated.ResourceResetTypeResume,
		generated.ResourceResetTypeSuspend:
		// Valid reset types
	default:
		return ErrInvalidResetType
	}

	// Convert generated reset type to entity power state
	powerState := convertToEntityPowerState(resetType)

	// Set the power state
	return uc.Repo.UpdatePowerState(id, powerState)
}

// convertToGeneratedType converts from entity type to generated API type.
func convertToGeneratedType(system *redfishv1.ComputerSystem) *generated.ComputerSystemComputerSystem {
	// This is a simplified conversion - in a real implementation,
	// you would map all the fields properly
	return &generated.ComputerSystemComputerSystem{
		Id:   system.ID,
		Name: system.Name,
	}
}

// convertToEntityPowerState converts from generated reset type to entity power state.
func convertToEntityPowerState(resetType generated.ResourceResetType) redfishv1.PowerState {
	// This is a simplified mapping - in a real implementation,
	// you would handle all the reset types properly
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOn:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeGracefulShutdown:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePause,
		generated.ResourceResetTypePushPowerButton,
		generated.ResourceResetTypeResume,
		generated.ResourceResetTypeSuspend:
		// For all other reset types, return Off as default
		return redfishv1.PowerStateOff
	default:
		return redfishv1.PowerStateOff
	}
}
