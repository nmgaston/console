// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
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
}

// NewWsmanComputerSystemRepo creates a new WSMAN-backed computer system repository.
func NewWsmanComputerSystemRepo(uc *devices.UseCase) *WsmanComputerSystemRepo {
	return &WsmanComputerSystemRepo{usecase: uc}
}

// GetByID retrieves a computer system by its ID from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetByID(systemID string) (*redfishv1.ComputerSystem, error) {
	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(context.Background(), systemID)
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
	// Build comprehensive ComputerSystem
	system := &redfishv1.ComputerSystem{
		ID:         systemID,
		PowerState: redfishPowerState,
		ODataID:    "/redfish/v1/Systems/" + systemID,
		ODataType:  "#ComputerSystem.v1_22_0.ComputerSystem",
	}

	return system, nil
}

// GetAll retrieves all computer systems from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetAll() ([]*redfishv1.ComputerSystem, error) {
	// Get devices from the devices use case
	items, err := r.usecase.Get(context.Background(), maxSystemsList, 0, "")
	if err != nil {
		return nil, err
	}

	// Convert devices to ComputerSystem entities
	systems := make([]*redfishv1.ComputerSystem, 0, len(items))
	for i := range items { // avoid value copy
		device := &items[i]
		if device.GUID == "" {
			continue // Skip devices without GUID
		}

		// Create basic system info from device
		system := &redfishv1.ComputerSystem{
			ID:         device.GUID,
			Name:       device.Hostname,
			SystemType: redfishv1.SystemTypePhysical, // Assume physical systems
			ODataID:    "/redfish/v1/Systems/" + device.GUID,
			ODataType:  "#ComputerSystem.v1_22_0.ComputerSystem",
		}

		systems = append(systems, system)
	}

	return systems, nil
}

// UpdatePowerState sends a power action command to the specified system via WSMAN.
func (r *WsmanComputerSystemRepo) UpdatePowerState(systemID string, state redfishv1.PowerState) error {
	// First, get the current power state
	currentSystem, err := r.GetByID(systemID)
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

	_, err = r.usecase.SendPowerAction(context.Background(), systemID, action)
	if err != nil && err.Error() == ErrMsgDeviceNotFound {
		return ErrSystemNotFound
	}

	return err
}

// extractSystemInfo extracts manufacturer, model, and serial number from hardware info.
func (r *WsmanComputerSystemRepo) extractSystemInfo(hardwareInfo dto.HardwareInfo) (manufacturer, model, serialNumber string) {
	// Extract from CIMComputerSystemPackage
	if len(hardwareInfo.CIMComputerSystemPackage.Responses) > 0 {
		if responseMap, ok := hardwareInfo.CIMComputerSystemPackage.Responses[0].(map[string]interface{}); ok {
			if mfg, exists := responseMap["Manufacturer"]; exists {
				if mfgStr, ok := mfg.(string); ok {
					manufacturer = mfgStr
				}
			}
			if mod, exists := responseMap["Model"]; exists {
				if modStr, ok := mod.(string); ok {
					model = modStr
				}
			}
			if serial, exists := responseMap["SerialNumber"]; exists {
				if serialStr, ok := serial.(string); ok {
					serialNumber = serialStr
				}
			}
		}
	}

	// Try CIMChassis if ComputerSystemPackage didn't have info
	if manufacturer == "" && len(hardwareInfo.CIMChassis.Responses) > 0 {
		if responseMap, ok := hardwareInfo.CIMChassis.Responses[0].(map[string]interface{}); ok {
			if mfg, exists := responseMap["Manufacturer"]; exists {
				if mfgStr, ok := mfg.(string); ok {
					manufacturer = mfgStr
				}
			}
			if mod, exists := responseMap["Model"]; exists {
				if modStr, ok := mod.(string); ok {
					model = modStr
				}
			}
			if serial, exists := responseMap["SerialNumber"]; exists {
				if serialStr, ok := serial.(string); ok {
					serialNumber = serialStr
				}
			}
		}
	}

	return manufacturer, model, serialNumber
}
