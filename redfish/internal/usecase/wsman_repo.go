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

// extractCIMSystemInfo extracts Description, UUID, and HostName from CIM_ComputerSystemPackage.
// According to redfish-systems.md:
// - Description: CIM_ComputerSystem.Description
// - UUID: CIM_ComputerSystem.UUID
// - HostName: CIM_ComputerSystem.DNSHostName
func (r *WsmanComputerSystemRepo) extractCIMSystemInfo(ctx context.Context, systemID string) (description, uuid, hostName string) {
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		return "", "", ""
	}

	// Try to extract from CIM_ComputerSystemPackage
	if hwInfo.CIMComputerSystemPackage.Response != nil {
		// Helper function to extract values from an item map
		extractFromItem := func(itemMap map[string]interface{}) {
			if uuidVal, ok := itemMap["UUID"].(string); ok && uuidVal != "" && uuid == "" {
				uuid = uuidVal
			}
			if descVal, ok := itemMap["Description"].(string); ok && descVal != "" && description == "" {
				description = descVal
			}
			if hostnameVal, ok := itemMap["DNSHostName"].(string); ok && hostnameVal != "" && hostName == "" {
				hostName = hostnameVal
			}
		}

		// Helper function to process items array
		processItemsArray := func(items []interface{}) {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					extractFromItem(itemMap)
				}
			}
		}

		// Try multiple response structures to handle different WSMAN response formats
		if responseMap, ok := hwInfo.CIMComputerSystemPackage.Response.(map[string]interface{}); ok {
			// Try PullResponse -> Items structure
			if pullResponse, ok := responseMap["PullResponse"].(map[string]interface{}); ok {
				if items, ok := pullResponse["Items"].([]interface{}); ok {
					processItemsArray(items)
				}
			}

			// Try direct Items array
			if items, ok := responseMap["Items"].([]interface{}); ok {
				processItemsArray(items)
			}

			// Try Body -> PullResponse -> Items (another common structure)
			if body, ok := responseMap["Body"].(map[string]interface{}); ok {
				if pullResponse, ok := body["PullResponse"].(map[string]interface{}); ok {
					if items, ok := pullResponse["Items"].([]interface{}); ok {
						processItemsArray(items)
					}
				}
			}

			// Try direct extraction if response is a single CIM_ComputerSystem object
			extractFromItem(responseMap)
		}

		// Try if Response is directly an array of items
		if itemsArray, ok := hwInfo.CIMComputerSystemPackage.Response.([]interface{}); ok {
			processItemsArray(itemsArray)
		}
	}

	return description, uuid, hostName
}

// extractBIOSVersion extracts BIOS version from CIM_BIOSElement.
// According to redfish-systems.md: BiosVersion: CIM_BIOSElement.Version
func (r *WsmanComputerSystemRepo) extractBIOSVersion(ctx context.Context, systemID string) string {
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		if r.log != nil {
			r.log.Debug("Failed to get hardware info for BIOS version extraction", "systemID", systemID, "error", err)
		}
		return ""
	}

	// Try to extract from CIM_BIOSElement
	if hwInfo.CIMBIOSElement.Response != nil {
		// The Response contains CIM_BIOSElement data with Version field
		if responseMap, ok := hwInfo.CIMBIOSElement.Response.(map[string]interface{}); ok {
			// Look for PullResponse -> Items array structure
			if pullResponse, ok := responseMap["PullResponse"].(map[string]interface{}); ok {
				if items, ok := pullResponse["Items"].([]interface{}); ok {
					for _, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if version, ok := itemMap["Version"].(string); ok && version != "" {
								if r.log != nil {
									r.log.Debug("Extracted BIOS version", "systemID", systemID, "version", version)
								}
								return version
							}
						}
					}
				}
			}
			// Also try direct Items array structure (alternative format)
			if items, ok := responseMap["Items"].([]interface{}); ok {
				for _, item := range items {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if version, ok := itemMap["Version"].(string); ok && version != "" {
							if r.log != nil {
								r.log.Debug("Extracted BIOS version from direct items", "systemID", systemID, "version", version)
							}
							return version
						}
					}
				}
			}
		}
	}

	return ""
}

// extractAMTUUID extracts the system UUID from AMT General Settings.
// AMT General Settings contains the real AMT system UUID that matches the rpc amtinfo command.
func (r *WsmanComputerSystemRepo) extractAMTUUID(ctx context.Context, systemID string) string {
	// Try to get AMT General Settings which contains the actual AMT UUID
	generalSettings, err := r.usecase.GetGeneralSettings(ctx, systemID)
	if err != nil {
		return ""
	}

	// Extract UUID from AMT General Settings body
	if generalSettings.Body != nil {
		if bodyMap, ok := generalSettings.Body.(map[string]interface{}); ok {
			// Look for UUID in the general settings body
			if uuid, ok := bodyMap["UUID"].(string); ok && uuid != "" {
				return uuid
			}

			// Try AMT_GeneralSettings structure
			if amtSettings, ok := bodyMap["AMT_GeneralSettings"].(map[string]interface{}); ok {
				if uuid, ok := amtSettings["UUID"].(string); ok && uuid != "" {
					return uuid
				}
			}

			// Try GetResponse -> AMT_GeneralSettings structure
			if getResponse, ok := bodyMap["GetResponse"].(map[string]interface{}); ok {
				if amtSettings, ok := getResponse["AMT_GeneralSettings"].(map[string]interface{}); ok {
					if uuid, ok := amtSettings["UUID"].(string); ok && uuid != "" {
						return uuid
					}
				}
			}
		}
	}

	return ""
}

// mapRedfishPowerStateToAction converts Redfish PowerState to WSMAN power action.
// Note: Graceful operations (GracefulShutdown, GracefulRestart) should use
// IPS_PowerManagementService.RequestOSPowerSavingStateChange instead of CIM power management.
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
	// Note: GracefulShutdown and GracefulRestart are handled in the computer_system.go usecase
	// using IPS_PowerManagementService.RequestOSPowerSavingStateChange
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

	// Extract additional CIM properties for new Redfish v1.26.0 support
	description, uuid, hostName := r.extractCIMSystemInfo(ctx, systemID)
	biosVersion := r.extractBIOSVersion(ctx, systemID)

	// If CIM extraction didn't get UUID, try AMT General Settings
	if uuid == "" {
		uuid = r.extractAMTUUID(ctx, systemID)
	}

	// Build comprehensive ComputerSystem with v1.26.0 properties
	system := &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         device.Hostname,
		Description:  description,
		UUID:         uuid,
		HostName:     hostName,
		BiosVersion:  biosVersion,
		PowerState:   redfishPowerState,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		SystemType:   redfishv1.SystemTypePhysical,
		ODataID:      "/redfish/v1/Systems/" + systemID,
		ODataType:    "#ComputerSystem.v1_26_0.ComputerSystem",
	}

	// Use friendly name if available
	if device.FriendlyName != "" {
		system.Name = device.FriendlyName
	}

	// Add Status information based on device state and power state
	// According to redfish-systems.md:
	// - Status.State: CIM_ComputerSystem.EnabledState
	// - Status.Health: CIM_ComputerSystem.HealthState
	system.Status = &redfishv1.Status{
		State:  r.mapDeviceStateToRedfishState(device, powerState),
		Health: r.mapDeviceHealthToRedfishHealth(device),
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

// mapDeviceStateToRedfishState maps device and power state to Redfish Status.State.
func (r *WsmanComputerSystemRepo) mapDeviceStateToRedfishState(device interface{}, powerState interface{}) string {
	// This would map CIM_ComputerSystem.EnabledState to Redfish Status.State
	// For now, use power state as a proxy
	return "Enabled" // Default to Enabled
}

// mapDeviceHealthToRedfishHealth maps device health information to Redfish Status.Health.
func (r *WsmanComputerSystemRepo) mapDeviceHealthToRedfishHealth(device interface{}) string {
	// This would map CIM_ComputerSystem.HealthState to Redfish Status.Health
	// For now, default to OK
	return "OK" // Default to OK
}
