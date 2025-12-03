// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"
	"fmt"

	amtBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

const (
	// ErrMsgDeviceNotFound is the error message returned by devices use case when device is not found.
	ErrMsgDeviceNotFound = "DevicesUseCase -  - : "

	// Power action constants for AMT/WSMAN power management.
	powerActionPowerUp    = 2  // CIM Power Management Service - Power On
	powerActionPowerCycle = 5  // Power Cycle (off then on)
	powerActionPowerDown  = 8  // Power Down (soft off)
	powerActionReset      = 10 // Reset (reboot)

	// maxSystemsList is the maximum number of systems to retrieve in a single request.
	maxSystemsList = 100
)

var (
	// ErrGetAllNotImplemented is returned when GetAll is called (not yet implemented).
	ErrGetAllNotImplemented = errors.New("GetAll not implemented")

	// ErrUnsupportedPowerState is returned when an unsupported power state is requested.
	ErrUnsupportedPowerState = errors.New("unsupported power state")

	// ErrBootSettingsNotAvailable is returned when boot settings cannot be retrieved.
	ErrBootSettingsNotAvailable = errors.New("boot settings not available")

	// ErrUnsupportedBootTarget is returned when an unsupported boot target is requested.
	ErrUnsupportedBootTarget = errors.New("unsupported boot target")
)

// WsmanComputerSystemRepo implements ComputerSystemRepository using WSMAN backend.
type WsmanComputerSystemRepo struct {
	usecase *devices.UseCase
	log     logger.Interface
}

// NewWsmanComputerSystemRepo creates a new WSMAN-backed computer system repository.
func NewWsmanComputerSystemRepo(uc *devices.UseCase, log logger.Interface) *WsmanComputerSystemRepo {
	return &WsmanComputerSystemRepo{
		usecase: uc,
		log:     log,
	}
}

// isDeviceNotFoundError checks if the error indicates a device was not found.
func (r *WsmanComputerSystemRepo) isDeviceNotFoundError(err error) bool {
	return err != nil && err.Error() == ErrMsgDeviceNotFound
}

// mapCIMPowerStateToRedfish converts CIM power state to Redfish PowerState.
func (r *WsmanComputerSystemRepo) mapCIMPowerStateToRedfish(cimState int) redfishv1.PowerState {
	switch cimState {
	case redfishv1.CIMPowerStateOn:
		return redfishv1.PowerStateOn
	case redfishv1.CIMPowerStateOffSoft, redfishv1.CIMPowerStateOffHard:
		return redfishv1.PowerStateOff
	default:
		return redfishv1.PowerStateOff // Default to Off for unknown states
	}
}

// extractHardwareInfo extracts manufacturer, model, and serial number from hardware info.
func (r *WsmanComputerSystemRepo) extractHardwareInfo(ctx context.Context, systemID string) (manufacturer, model, serialNumber string) {
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err == nil && hwInfo.CIMChassis.Response != nil {
		if chassisResponse, ok := hwInfo.CIMChassis.Response.(chassis.PackageResponse); ok {
			return chassisResponse.Manufacturer, chassisResponse.Model, chassisResponse.SerialNumber
		}
	}

	return "", "", ""
}

// mapRedfishPowerStateToAction converts Redfish PowerState to WSMAN power action.
func (r *WsmanComputerSystemRepo) mapRedfishPowerStateToAction(state redfishv1.PowerState) (int, error) {
	switch state {
	case redfishv1.PowerStateOn:
		return devices.CIMPMSPowerOn, nil // Power On = 2
	case redfishv1.PowerStateOff:
		return powerActionPowerDown, nil
	case redfishv1.ResetTypeForceOff:
		return powerActionPowerDown, nil
	case redfishv1.ResetTypeForceRestart:
		return powerActionReset, nil
	case redfishv1.ResetTypePowerCycle:
		return powerActionPowerCycle, nil
	default:
		return 0, ErrUnsupportedPowerState
	}
}

// GetAll retrieves all computer system IDs from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetAll(ctx context.Context) ([]string, error) {
	// Get devices from the device use case
	items, err := r.usecase.Get(ctx, maxSystemsList, 0, "")
	if err != nil {
		return nil, err
	}

	// Extract just the GUIDs from devices
	systemIDs := make([]string, 0, len(items))
	for i := range items { // avoid value copy
		device := &items[i]
		if device.GUID == "" {
			continue // Skip devices without GUID
		}

		systemIDs = append(systemIDs, device.GUID)
	}

	return systemIDs, nil
}

// GetByID retrieves a computer system by its ID from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetByID(ctx context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	// Get device information from repository
	device, err := r.usecase.GetByID(ctx, systemID, "", false)
	if r.isDeviceNotFoundError(err) {
		return nil, ErrSystemNotFound
	}

	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrSystemNotFound
	}

	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(ctx, systemID)
	if r.isDeviceNotFoundError(err) {
		return nil, ErrSystemNotFound
	}

	if err != nil {
		return nil, err
	}

	// Map the integer power state to Redfish PowerState
	redfishPowerState := r.mapCIMPowerStateToRedfish(powerState.PowerState)

	// Extract hardware info for manufacturer, model, serial number
	manufacturer, model, serialNumber := r.extractHardwareInfo(ctx, systemID)

	// Build comprehensive ComputerSystem
	system := &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         device.Hostname,
		PowerState:   redfishPowerState,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		SystemType:   redfishv1.SystemTypePhysical,
		ODataID:      "/redfish/v1/Systems/" + systemID,
		ODataType:    "#ComputerSystem.v1_22_0.ComputerSystem",
	}

	// Use friendly name if available
	if device.FriendlyName != "" {
		system.Name = device.FriendlyName
	}

	return system, nil
}

// UpdatePowerState sends a power action command to the specified system via WSMAN.
func (r *WsmanComputerSystemRepo) UpdatePowerState(ctx context.Context, systemID string, state redfishv1.PowerState) error {
	// First, get the current power state
	currentSystem, err := r.GetByID(ctx, systemID)
	if err != nil {
		return err
	}

	// Check if the requested state matches the current state
	if currentSystem.PowerState == state {
		return ErrPowerStateConflict
	}

	// Map Redfish power state to WSMAN action
	action, err := r.mapRedfishPowerStateToAction(state)
	if err != nil {
		return err
	}

	// Send power action command
	_, err = r.usecase.SendPowerAction(ctx, systemID, action)
	if r.isDeviceNotFoundError(err) {
		return ErrSystemNotFound
	}

	return err
}

// GetBootSettings retrieves the current boot configuration for a system.
func (r *WsmanComputerSystemRepo) GetBootSettings(ctx context.Context, systemID string) (*generated.ComputerSystemBoot, error) {
	// Get current boot data from AMT via devices use case
	bootData, err := r.usecase.GetBootSettings(ctx, systemID)
	if err != nil {
		if r.isDeviceNotFoundError(err) {
			return nil, ErrSystemNotFound
		}

		r.log.Warn("Failed to get boot data from device", "systemID", systemID, "error", err)

		return nil, ErrBootSettingsNotAvailable
	}

	// Map AMT boot data to Redfish Boot structure
	boot := &generated.ComputerSystemBoot{}

	// Map BootSourceOverrideEnabled
	if bootData.BootMediaIndex == 0 {
		// No override, boot from normal sources
		enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
		_ = enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabledDisabled)
		boot.BootSourceOverrideEnabled = &enabled
	} else {
		// Boot override is active - assume "Once" for AMT
		enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
		_ = enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabledOnce)
		boot.BootSourceOverrideEnabled = &enabled
	}

	// Map boot target based on boot configuration
	target := generated.ComputerSystemBoot_BootSourceOverrideTarget{}

	switch {
	case bootData.BIOSSetup:
		_ = target.FromComputerSystemBootSourceOverrideTarget(generated.BiosSetup)
	case bootData.UseIDER:
		// IDER can be CD or Floppy
		if bootData.IDERBootDevice == 1 {
			_ = target.FromComputerSystemBootSourceOverrideTarget(generated.Cd)
		} else {
			_ = target.FromComputerSystemBootSourceOverrideTarget(generated.Floppy)
		}
	default:
		// Default or PXE boot - would need additional logic to determine exact source
		_ = target.FromComputerSystemBootSourceOverrideTarget(generated.None)
	}

	boot.BootSourceOverrideTarget = &target

	// Map boot mode based on UEFI boot parameters
	mode := generated.ComputerSystemBoot_BootSourceOverrideMode{}

	if bootData.UEFILocalPBABootEnabled || bootData.UEFIHTTPSBootEnabled || len(bootData.UEFIBootParametersArray) > 0 {
		_ = mode.FromComputerSystemBootSourceOverrideMode(generated.UEFI)
	} else {
		_ = mode.FromComputerSystemBootSourceOverrideMode(generated.Legacy)
	}

	boot.BootSourceOverrideMode = &mode

	return boot, nil
}

// UpdateBootSettings updates the boot configuration for a system.
func (r *WsmanComputerSystemRepo) UpdateBootSettings(ctx context.Context, systemID string, boot *generated.ComputerSystemBoot) error {
	// Get current boot data to preserve settings
	bootData, err := r.usecase.GetBootSettings(ctx, systemID)
	if err != nil {
		if r.isDeviceNotFoundError(err) {
			return ErrSystemNotFound
		}

		return fmt.Errorf("failed to get current boot data: %w", err)
	}

	// Create new boot settings based on current data
	newBootData := r.createBootDataRequest(bootData)

	// Parse and apply boot target
	bootSource, err := r.applyBootTarget(boot, &newBootData)
	if err != nil {
		return err
	}

	// Apply boot mode if specified
	r.applyBootMode(boot, systemID)

	// Use devices use case methods to configure boot
	if err := r.usecase.SetBootSettings(ctx, systemID, newBootData); err != nil {
		return fmt.Errorf("failed to set boot settings: %w", err)
	}

	// Set boot order if we have a boot source
	if bootSource != "" {
		if err := r.usecase.ChangeBootOrder(ctx, systemID, bootSource); err != nil {
			return fmt.Errorf("failed to change boot order: %w", err)
		}
	}

	r.log.Info("Boot settings updated successfully",
		"systemID", systemID,
		"target", boot.BootSourceOverrideTarget,
		"enabled", boot.BootSourceOverrideEnabled,
		"mode", boot.BootSourceOverrideMode,
	)

	return nil
}

// createBootDataRequest creates a new boot data request from current settings.
func (r *WsmanComputerSystemRepo) createBootDataRequest(bootData amtBoot.BootSettingDataResponse) amtBoot.BootSettingDataRequest {
	return amtBoot.BootSettingDataRequest{
		BIOSLastStatus:         bootData.BIOSLastStatus,
		BIOSPause:              false,
		BIOSSetup:              false,
		BootMediaIndex:         0,
		ConfigurationDataReset: false,
		ElementName:            bootData.ElementName,
		EnforceSecureBoot:      bootData.EnforceSecureBoot,
		FirmwareVerbosity:      bootData.FirmwareVerbosity,
		ForcedProgressEvents:   false,
		InstanceID:             bootData.InstanceID,
		LockKeyboard:           false,
		LockPowerButton:        false,
		LockResetButton:        false,
		LockSleepButton:        false,
		OptionsCleared:         true,
		OwningEntity:           bootData.OwningEntity,
		ReflashBIOS:            false,
		UseIDER:                false,
		UseSOL:                 bootData.UseSOL,
		UseSafeMode:            false,
		UserPasswordBypass:     false,
		SecureErase:            false,
	}
}

// applyBootTarget applies the boot target to the boot data and returns the boot source.
func (r *WsmanComputerSystemRepo) applyBootTarget(boot *generated.ComputerSystemBoot, newBootData *amtBoot.BootSettingDataRequest) (string, error) {
	if boot.BootSourceOverrideTarget == nil {
		return "", nil
	}

	target, err := boot.BootSourceOverrideTarget.AsComputerSystemBootSourceOverrideTarget()
	if err != nil {
		return "", nil
	}

	switch target {
	case generated.BiosSetup:
		newBootData.BIOSSetup = true

		return "", nil // Clear boot order for BIOS setup
	case generated.Pxe:
		return string(cimBoot.PXE), nil
	case generated.Cd:
		newBootData.UseIDER = true
		newBootData.IDERBootDevice = 1 // CD-ROM

		return string(cimBoot.CD), nil
	case generated.Floppy:
		newBootData.UseIDER = true
		newBootData.IDERBootDevice = 0 // Floppy

		return "", nil
	case generated.Hdd, generated.None:
		return "", nil // Default boot or clear override
	case generated.Usb:
		return "", ErrUnsupportedBootTarget
	case generated.Diags, generated.Recovery, generated.RemoteDrive, generated.SDCard,
		generated.UefiBootNext, generated.UefiHttp, generated.UefiShell, generated.UefiTarget, generated.Utilities:
		return "", ErrUnsupportedBootTarget
	default:
		return "", ErrUnsupportedBootTarget
	}
}

// applyBootMode logs the requested boot mode.
func (r *WsmanComputerSystemRepo) applyBootMode(boot *generated.ComputerSystemBoot, systemID string) {
	if boot.BootSourceOverrideMode == nil {
		return
	}

	mode, err := boot.BootSourceOverrideMode.AsComputerSystemBootSourceOverrideMode()
	if err != nil {
		return
	}

	switch mode {
	case generated.UEFI:
		r.log.Info("UEFI boot mode requested", "systemID", systemID)
	case generated.Legacy:
		r.log.Info("Legacy boot mode requested", "systemID", systemID)
	}
}
