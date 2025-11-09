// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"
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
	if err != nil {
		if err.Error() == ErrMsgDeviceNotFound {
			return nil, ErrSystemNotFound
		}

		return nil, err
	}

	if device == nil {
		return nil, ErrSystemNotFound
	}

	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(ctx, systemID)
	if err != nil {
		if err.Error() == ErrMsgDeviceNotFound {
			return nil, ErrSystemNotFound
		}

		return nil, err
	}

	// Map the integer power state to Redfish PowerState
	var redfishPowerState redfishv1.PowerState

	switch powerState.PowerState {
	case redfishv1.CIMPowerStateOn:
		redfishPowerState = redfishv1.PowerStateOn
	case redfishv1.CIMPowerStateOffSoft:
		redfishPowerState = redfishv1.PowerStateOff
	case redfishv1.CIMPowerStateOffHard:
		redfishPowerState = redfishv1.PowerStateOff
	default:
		redfishPowerState = redfishv1.PowerStateOff // Default to Off for unknown states
	}

	// Try to get hardware info for manufacturer, model, serial number
	var manufacturer, model, serialNumber string

	// Get hardware info from devices use case
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err == nil && hwInfo.CIMChassis.Response != nil {
		if chassisResponse, ok := hwInfo.CIMChassis.Response.(chassis.PackageResponse); ok {
			manufacturer, model, serialNumber = chassisResponse.Manufacturer, chassisResponse.Model, chassisResponse.SerialNumber
		}
	}

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

	var action int

	switch state {
	case redfishv1.PowerStateOn:
		action = devices.CIMPMSPowerOn // Power On = 2
	case redfishv1.PowerStateOff:
		action = powerActionPowerDown
	case redfishv1.ResetTypeForceOff:
		action = powerActionPowerDown
	case redfishv1.ResetTypeForceRestart:
		action = powerActionReset
	case redfishv1.ResetTypePowerCycle:
		action = powerActionPowerCycle
	default:
		return ErrUnsupportedPowerState
	}

	_, err = r.usecase.SendPowerAction(ctx, systemID, action)
	if err != nil && err.Error() == ErrMsgDeviceNotFound {
		return ErrSystemNotFound
	}

	return err
}
