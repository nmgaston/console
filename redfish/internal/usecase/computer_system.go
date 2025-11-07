// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"errors"
	"fmt"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

var (
	// ErrInvalidPowerState is returned when an invalid power state is requested.
	ErrInvalidPowerState = errors.New("invalid power state")

	// ErrPowerStateConflict is returned when a power state transition is not allowed.
	ErrPowerStateConflict = errors.New("power state transition not allowed")

	// ErrInvalidResetType is returned when an invalid reset type is provided.
	ErrInvalidResetType = errors.New("invalid reset type")
)

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
type ComputerSystemUseCase struct {
	Repo ComputerSystemRepository
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and converts it to the generated API type.
func (uc *ComputerSystemUseCase) GetComputerSystem(systemID string) (*generated.ComputerSystemComputerSystem, error) {
	// Get device information from repository - this gives us basic device data
	system, err := uc.Repo.GetByID(systemID)
	if err != nil {
		return nil, err
	}

	// Build the generated type directly with available information
	// Create power state
	var powerState *generated.ComputerSystemComputerSystem_PowerState
	if system.PowerState != "" {
		var redfishPowerState generated.ResourcePowerState
		switch system.PowerState {
		case redfishv1.PowerStateOn:
			redfishPowerState = generated.On
		case redfishv1.PowerStateOff:
			redfishPowerState = generated.Off
		default:
			redfishPowerState = generated.Off // Default to Off for unknown states
		}

		powerState = &generated.ComputerSystemComputerSystem_PowerState{}
		powerState.FromResourcePowerState(redfishPowerState)
	}

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")
	odataType := generated.OdataV4Type("#ComputerSystem.v1_22_0.ComputerSystem")
	odataId := generated.OdataV4Id(fmt.Sprintf("/redfish/v1/Systems/%s", systemID))

	result := generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataId,
		OdataType:    &odataType,
		Id:           generated.ResourceId(systemID),
		PowerState:   powerState,
		SystemType:   &systemType,
	}

	return &result, nil
}

// GetAll retrieves all ComputerSystems from the repository.
func (uc *ComputerSystemUseCase) GetAll() ([]*redfishv1.ComputerSystem, error) {
	return uc.Repo.GetAll()
}

// SetPowerState validates and sets the power state for a ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(id string, resetType generated.ResourceResetType) error {
	// Validate the reset type
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeForceOn,
		generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypeGracefulShutdown,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle,
		generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton,
		generated.ResourceResetTypePause,
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
func convertToGeneratedType(cs *redfishv1.ComputerSystem) generated.ComputerSystemComputerSystem {
	// Convert power state
	var powerState *generated.ComputerSystemComputerSystem_PowerState
	if cs.PowerState != "" {
		// Map our power state to Redfish power state using entity constants
		var redfishPowerState generated.ResourcePowerState
		switch cs.PowerState {
		case redfishv1.PowerStateOn:
			redfishPowerState = generated.On
		case redfishv1.PowerStateOff:
			redfishPowerState = generated.Off
		default:
			redfishPowerState = generated.Off // Default to Off for unknown states
		}

		powerState = &generated.ComputerSystemComputerSystem_PowerState{}
		powerState.FromResourcePowerState(redfishPowerState)
	}

	// Convert to string pointers for optional fields
	var manufacturer, model, serialNumber *string
	if cs.Manufacturer != "" {
		manufacturer = &cs.Manufacturer
	}
	if cs.Model != "" {
		model = &cs.Model
	}
	if cs.SerialNumber != "" {
		serialNumber = &cs.SerialNumber
	}

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData context
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")

	// Create OData type
	odataType := generated.OdataV4Type("#ComputerSystem.v1_22_0.ComputerSystem")

	// Create OData ID
	odataId := generated.OdataV4Id(fmt.Sprintf("/redfish/v1/Systems/%s", cs.ID))

	return generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataId,
		OdataType:    &odataType,
		Id:           generated.ResourceId(cs.ID),
		Name:         generated.ResourceName(cs.Name),
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		SystemType:   &systemType,
	}
}

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
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
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle:
		return redfishv1.PowerStateOff // Will cycle to On
	case generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypePause,
		generated.ResourceResetTypeSuspend:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeResume:
		return redfishv1.PowerStateOn
	default:
		return redfishv1.PowerStateOff
	}
}
