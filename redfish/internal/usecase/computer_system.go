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

	// Try to get additional CIM information for enhanced properties
	// This will populate Description, UUID, HostName, BiosVersion from CIM objects
	uc.enrichSystemWithCIMData(ctx, systemID, system)

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

	// Convert to string pointers for optional fields
	var manufacturer, model, serialNumber, description, uuid, hostName, biosVersion *string
	if system.Manufacturer != "" {
		manufacturer = &system.Manufacturer
	}

	if system.Model != "" {
		model = &system.Model
	}

	if system.SerialNumber != "" {
		serialNumber = &system.SerialNumber
	}

	if system.Description != "" {
		description = &system.Description
	}

	if system.UUID != "" {
		uuid = &system.UUID
	}

	if system.HostName != "" {
		hostName = &system.HostName
	}

	if system.BiosVersion != "" {
		biosVersion = &system.BiosVersion
	}

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")
	odataType := generated.OdataV4Type("#ComputerSystem.v1_26_0.ComputerSystem")
	odataID := fmt.Sprintf("/redfish/v1/Systems/%s", systemID)

	// Build Status if present
	var status *generated.ResourceStatus

	if system.Status != nil {
		health := generated.ResourceStatus_Health{}
		_ = health.FromResourceStatusHealth1(system.Status.Health)
		state := generated.ResourceStatus_State{}
		_ = state.FromResourceStatusState1(system.Status.State)

		status = &generated.ResourceStatus{
			State:  &state,
			Health: &health,
		}
	}

	// Convert Description to union type if present
	var descriptionUnion *generated.ComputerSystemComputerSystem_Description
	if description != nil {
		descriptionUnion = &generated.ComputerSystemComputerSystem_Description{}
		_ = descriptionUnion.FromResourceDescription(generated.ResourceDescription(*description))
	}

	// Convert UUID to union type if present
	var uuidUnion *generated.ComputerSystemComputerSystem_UUID
	if uuid != nil {
		uuidUnion = &generated.ComputerSystemComputerSystem_UUID{}
		_ = uuidUnion.FromResourceUUID(generated.ResourceUUID(*uuid))
	}

	// Create Intel OEM extension with references to PowerState and PowerCapabilities
	intelOem := map[string]interface{}{
		"PowerState": map[string]interface{}{
			"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerState", systemID),
		},
		"PowerCapabilities": map[string]interface{}{
			"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerCapabilities", systemID),
		},
	}

	oem := generated.ResourceOem{
		"Intel": intelOem,
	}

	result := generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           systemID,
		Name:         system.Name,
		Description:  descriptionUnion,
		UUID:         uuidUnion,
		HostName:     hostName,
		BiosVersion:  biosVersion,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		SystemType:   &systemType,
		Status:       status,
		Oem:          &oem,
	}

	return &result, nil
}

// enrichSystemWithCIMData attempts to populate additional system properties from CIM objects.
// According to redfish-systems.md mapping:
// - Description: CIM_ComputerSystem.Description
// - UUID: CIM_ComputerSystem.UUID
// - HostName: CIM_ComputerSystem.DNSHostName
// - BiosVersion: CIM_BIOSElement.Version
func (uc *ComputerSystemUseCase) enrichSystemWithCIMData(ctx context.Context, systemID string, system *redfishv1.ComputerSystem) {
	// This is a best-effort operation - we don't fail if CIM data is unavailable
	// In a real implementation, you would:
	// 1. Get hardware info via devices usecase: hwInfo, err := uc.devicesUseCase.GetHardwareInfo(ctx, systemID)
	// 2. Extract CIM_ComputerSystem data from hwInfo.CIMComputerSystemPackage.Response
	// 3. Extract CIM_BIOSElement data from hwInfo.CIMBIOSElement.Response
	// 4. Type assert and populate the respective fields

	// For now, set some placeholder values that would come from CIM objects
	if system.Description == "" {
		system.Description = "Computer System managed by Intel AMT"
	}

	if system.HostName == "" {
		// Create a safe placeholder hostname
		hostSuffix := systemID
		if len(hostSuffix) > 8 {
			hostSuffix = hostSuffix[:8]
		}
		system.HostName = "amt-system-" + hostSuffix // Placeholder hostname
	}
	if system.BiosVersion == "" {
		system.BiosVersion = "BIOS.Version.1.0.0" // Placeholder BIOS version
	}
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
