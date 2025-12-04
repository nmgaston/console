// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
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
	// ErrSystemNotFound is returned when a system is not found.
	ErrSystemNotFound = errors.New("system not found")

	// ErrGetAllNotImplemented is returned when GetAll is called (not yet implemented).
	ErrGetAllNotImplemented = errors.New("GetAll not implemented")

	// ErrUnsupportedPowerState is returned when an unsupported power state is requested.
	ErrUnsupportedPowerState = errors.New("unsupported power state")
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

// mapRedfishResetTypeToAction converts Redfish reset type to WSMAN power action.
func (r *WsmanComputerSystemRepo) mapRedfishResetTypeToAction(resetType redfishv1.PowerState) (int, error) {
	switch resetType {
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
func (r *WsmanComputerSystemRepo) UpdatePowerState(ctx context.Context, systemID string, resetType redfishv1.PowerState) error {
	// Get the current power state for logging and validation
	currentSystem, err := r.GetByID(ctx, systemID)
	if err != nil {
		return err
	}

	// For certain reset types like PowerCycle and ForceRestart, we don't check current state
	// because they are valid operations regardless of current power state
	if resetType != redfishv1.ResetTypePowerCycle && resetType != redfishv1.ResetTypeForceRestart {
		// Check if the requested state matches the current state
		if currentSystem.PowerState == resetType {
			return ErrPowerStateConflict
		}
	}

	// Map Redfish reset type to WSMAN action
	action, err := r.mapRedfishResetTypeToAction(resetType)
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
