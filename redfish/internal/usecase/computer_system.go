// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"context"
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

// GetAll retrieves all ComputerSystem IDs from the repository.
func (uc *ComputerSystemUseCase) GetAll(ctx context.Context) ([]string, error) {
	return uc.Repo.GetAll(ctx)
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and converts it to the generated API type.
func (uc *ComputerSystemUseCase) GetComputerSystem(ctx context.Context, systemID string) (*generated.ComputerSystemComputerSystem, error) {
	// Get device information from repository - this gives us basic device data
	system, err := uc.Repo.GetByID(ctx, systemID)
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
		case redfishv1.ResetTypeForceOff, redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			redfishPowerState = generated.Off // These reset types default to Off state
		default:
			redfishPowerState = generated.Off // Default to Off for unknown states
		}

		powerState = &generated.ComputerSystemComputerSystem_PowerState{}
		if err := powerState.FromResourcePowerState(redfishPowerState); err != nil {
			// Log error but continue with nil power state
			powerState = nil
		}
	}

	// Convert to string pointers for optional fields (using helper function to reduce allocations)
	manufacturer := stringPtrIfNotEmpty(system.Manufacturer)
	model := stringPtrIfNotEmpty(system.Model)
	serialNumber := stringPtrIfNotEmpty(system.SerialNumber)
	description := stringPtrIfNotEmpty(system.Description)
	hostName := stringPtrIfNotEmpty(system.HostName)

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")
	odataType := generated.OdataV4Type("#ComputerSystem.v1_26_0.ComputerSystem")
	odataID := fmt.Sprintf("/redfish/v1/Systems/%s", systemID)

	// Build Status if present
	status := uc.convertStatusToGenerated(system.Status)

	// Convert Description to union type if present
	var descriptionUnion *generated.ComputerSystemComputerSystem_Description
	if description != nil {
		descriptionUnion = &generated.ComputerSystemComputerSystem_Description{}
		if err := descriptionUnion.FromResourceDescription(generated.ResourceDescription(*description)); err != nil {
			// Log error but continue - don't fail the entire request for Description conversion issues
			descriptionUnion = nil
		}
	}

	result := generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           systemID,
		Name:         system.Name,
		Description:  descriptionUnion,
		HostName:     hostName,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		SystemType:   &systemType,
		Status:       status,
	}

	return &result, nil
}

// SetPowerState validates and sets the power state for a ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(ctx context.Context, id string, resetType generated.ResourceResetType) error {
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
	return uc.Repo.UpdatePowerState(ctx, id, powerState)
}

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// stringPtrIfNotEmpty returns a pointer to the string if it's not empty, otherwise nil.
func stringPtrIfNotEmpty(s string) *string {
	if s != "" {
		return &s
	}
	return nil
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

// convertStatusToGenerated converts entity Status to generated ResourceStatus.
func (uc *ComputerSystemUseCase) convertStatusToGenerated(status *redfishv1.Status) *generated.ResourceStatus {
	if status == nil {
		return nil
	}

	var healthPtr *generated.ResourceStatus_Health
	var statePtr *generated.ResourceStatus_State

	// Convert Health if present
	if status.Health != "" {
		healthPtr = uc.convertHealthToGenerated(status.Health)
	}

	// Convert State if present
	if status.State != "" {
		statePtr = uc.convertStateToGenerated(status.State)
	}

	// Only create Status if we have at least one field
	if healthPtr != nil || statePtr != nil {
		return &generated.ResourceStatus{
			Health: healthPtr,
			State:  statePtr,
		}
	}

	return nil
}

// convertHealthToGenerated converts Health string to generated ResourceStatus_Health.
func (uc *ComputerSystemUseCase) convertHealthToGenerated(health string) *generated.ResourceStatus_Health {
	var healthEnum generated.ResourceHealth
	switch health {
	case "OK":
		healthEnum = generated.OK
	case "Warning":
		healthEnum = generated.Warning
	case "Critical":
		healthEnum = generated.Critical
	default:
		return nil // Don't create health if unknown value
	}

	healthObj := generated.ResourceStatus_Health{}
	if err := healthObj.FromResourceHealth(healthEnum); err == nil {
		return &healthObj
	}

	return nil
}

// convertStateToGenerated converts State string to generated ResourceStatus_State.
func (uc *ComputerSystemUseCase) convertStateToGenerated(state string) *generated.ResourceStatus_State {
	var stateEnum generated.ResourceState
	switch state {
	case "Enabled":
		stateEnum = generated.Enabled
	case "Disabled":
		stateEnum = generated.Disabled
	case "StandbyOffline":
		stateEnum = generated.StandbyOffline
	case "StandbySpare":
		stateEnum = generated.StandbySpare
	case "InTest":
		stateEnum = generated.InTest
	case "Starting":
		stateEnum = generated.Starting
	case "Absent":
		stateEnum = generated.Absent
	case "UnavailableOffline":
		stateEnum = generated.UnavailableOffline
	case "Deferring":
		stateEnum = generated.Deferring
	case "Quiesced":
		stateEnum = generated.Quiesced
	case "Updating":
		stateEnum = generated.Updating
	case "Degraded":
		stateEnum = generated.Degraded
	default:
		return nil // Don't create state if unknown value
	}

	stateObj := generated.ResourceStatus_State{}
	if err := stateObj.FromResourceState(stateEnum); err == nil {
		return &stateObj
	}

	return nil
}
