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

	// ErrInvalidBootSettings is returned when invalid boot settings are provided.
	ErrInvalidBootSettings = errors.New("invalid boot settings")

	// ErrSystemNotFound is returned when a system is not found.
	ErrSystemNotFound = errors.New("system not found")
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

	// Convert to string pointers for optional fields
	var manufacturer, model, serialNumber *string
	if system.Manufacturer != "" {
		manufacturer = &system.Manufacturer
	}

	if system.Model != "" {
		model = &system.Model
	}

	if system.SerialNumber != "" {
		serialNumber = &system.SerialNumber
	}

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")
	odataType := generated.OdataV4Type("#ComputerSystem.v1_22_0.ComputerSystem")
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

	// Fetch boot settings
	boot, err := uc.Repo.GetBootSettings(ctx, systemID)
	if err != nil {
		// Log error but don't fail the entire request - boot settings may not be available
		boot = nil
	}

	result := generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           systemID,
		Name:         system.Name,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		SystemType:   &systemType,
		Status:       status,
		Boot:         boot,
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

// UpdateBootSettings updates the boot configuration for a ComputerSystem.
func (uc *ComputerSystemUseCase) UpdateBootSettings(ctx context.Context, systemID string, boot *generated.ComputerSystemBoot) error {
	if boot == nil {
		return nil // Nothing to update
	}

	// Validate all boot settings
	if err := uc.validateBootSettings(boot); err != nil {
		return err
	}

	// Update boot settings in repository
	return uc.Repo.UpdateBootSettings(ctx, systemID, boot)
}

// validateBootSettings validates all boot configuration fields.
func (uc *ComputerSystemUseCase) validateBootSettings(boot *generated.ComputerSystemBoot) error {
	if err := uc.validateBootTargetField(boot.BootSourceOverrideTarget); err != nil {
		return err
	}

	if err := uc.validateBootEnabledField(boot.BootSourceOverrideEnabled); err != nil {
		return err
	}

	return uc.validateBootModeField(boot.BootSourceOverrideMode)
}

// validateBootTargetField validates the boot source override target field.
func (uc *ComputerSystemUseCase) validateBootTargetField(targetField *generated.ComputerSystemBoot_BootSourceOverrideTarget) error {
	if targetField == nil {
		return nil
	}

	target, err := targetField.AsComputerSystemBootSourceOverrideTarget()
	if err != nil {
		return nil
	}

	return validateBootTarget(target)
}

// validateBootEnabledField validates the boot source override enabled field.
func (uc *ComputerSystemUseCase) validateBootEnabledField(enabledField *generated.ComputerSystemBoot_BootSourceOverrideEnabled) error {
	if enabledField == nil {
		return nil
	}

	enabled, err := enabledField.AsComputerSystemBootSourceOverrideEnabled()
	if err != nil {
		return nil
	}

	return validateBootEnabled(enabled)
}

// validateBootModeField validates the boot source override mode field.
func (uc *ComputerSystemUseCase) validateBootModeField(modeField *generated.ComputerSystemBoot_BootSourceOverrideMode) error {
	if modeField == nil {
		return nil
	}

	mode, err := modeField.AsComputerSystemBootSourceOverrideMode()
	if err != nil {
		return nil
	}

	return validateBootMode(mode)
}

// validateBootTarget validates the boot source override target.
func validateBootTarget(target generated.ComputerSystemBootSourceOverrideTarget) error {
	switch target {
	case generated.BiosSetup, generated.Cd, generated.Diags, generated.Floppy,
		generated.Hdd, generated.None, generated.Pxe, generated.Recovery,
		generated.RemoteDrive, generated.SDCard, generated.UefiBootNext,
		generated.UefiHttp, generated.UefiShell, generated.UefiTarget,
		generated.Usb, generated.Utilities:
		return nil
	default:
		return fmt.Errorf("%w: invalid boot target %s", ErrInvalidBootSettings, target)
	}
}

// validateBootEnabled validates the boot source override enabled setting.
func validateBootEnabled(enabled generated.ComputerSystemBootSourceOverrideEnabled) error {
	switch enabled {
	case generated.ComputerSystemBootSourceOverrideEnabledContinuous,
		generated.ComputerSystemBootSourceOverrideEnabledDisabled,
		generated.ComputerSystemBootSourceOverrideEnabledOnce:
		return nil
	default:
		return fmt.Errorf("%w: invalid boot enabled setting %s", ErrInvalidBootSettings, enabled)
	}
}

// validateBootMode validates the boot source override mode.
func validateBootMode(mode generated.ComputerSystemBootSourceOverrideMode) error {
	switch mode {
	case generated.Legacy, generated.UEFI:
		return nil
	default:
		return fmt.Errorf("%w: invalid boot mode %s", ErrInvalidBootSettings, mode)
	}
}
