// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/processor"

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
	// Validate input range - CIM power states should be positive values
	if cimState < 0 {
		return redfishv1.PowerStateOff // Invalid negative value defaults to Off
	}

	switch cimState {
	case redfishv1.CIMPowerStateOn:
		return redfishv1.PowerStateOn
	case redfishv1.CIMPowerStateOffSoft, redfishv1.CIMPowerStateOffHard:
		return redfishv1.PowerStateOff
	default:
		return redfishv1.PowerStateOff // Default to Off for unknown states
	}
}

// extractCIMChassisInfo extracts manufacturer, model, and serial number from CIM chassis info.
func (r *WsmanComputerSystemRepo) extractCIMChassisInfo(ctx context.Context, systemID string) (manufacturer, model, serialNumber string) {
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err == nil && hwInfo.CIMChassis.Response != nil {
		if chassisResponse, ok := hwInfo.CIMChassis.Response.(chassis.PackageResponse); ok {
			return chassisResponse.Manufacturer, chassisResponse.Model, chassisResponse.SerialNumber
		}
	}

	return "", "", ""
}

// extractCIMSystemInfo extracts Description and HostName from CIM_ComputerSystemPackage.
// According to redfish-systems.md:
// - Description: CIM_ComputerSystem.Description
// - HostName: CIM_ComputerSystem.DNSHostName
func (r *WsmanComputerSystemRepo) extractCIMSystemInfo(ctx context.Context, systemID string) (description, hostName string) {
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		return "", ""
	}

	// Try to extract from CIM_ComputerSystemPackage
	if hwInfo.CIMComputerSystemPackage.Response == nil {
		return "", ""
	}

	// Helper function to extract values from an item map
	extractFromItem := func(itemMap map[string]interface{}) {
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

	return description, hostName
}

// extractCIMStatusInfo extracts Status information from CIM_Processor data.
// Maps CIM status properties to Redfish Status:
// - Health: CIM_Processor.HealthState (0=Unknown, 5=OK, 10=Degraded, 15=Minor failure, 20=Major failure, 25=Critical failure, 30=Non-recoverable error)
// - State: CIM_Processor.EnabledState (2=Enabled, 3=Disabled, 6=EnabledButOffline, 8=InTest, 9=Deferred)
func (r *WsmanComputerSystemRepo) extractCIMStatusInfo(ctx context.Context, systemID string) *redfishv1.Status {

	var healthState *int
	var enabledState *int

	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		return nil
	}

	// Try to extract from CIM_Processor responses array
	if hwInfo.CIMProcessor.Responses == nil || len(hwInfo.CIMProcessor.Responses) == 0 {
		return nil
	}

	// Extract status values from the first CIM_Processor response
	// In most cases, there should be only one processor entry, but we take the first one
	response := hwInfo.CIMProcessor.Responses[0]

	// Map responses not handled.
	// Handle response as struct
	if procResp, ok := response.(processor.PackageResponse); ok {
		healthVal := int(procResp.HealthState)
		healthState = &healthVal

		enabledVal := int(procResp.EnabledState)
		enabledState = &enabledVal
	} else if respMap, ok := response.(map[string]interface{}); ok {
		// Handle HealthState - can be int or float64
		if healthVal, exists := respMap["HealthState"]; exists && healthState == nil {
			switch v := healthVal.(type) {
			case int:
				healthState = &v
			case float64:
				healthInt := int(v)
				healthState = &healthInt
			}
		}

		// Handle EnabledState - can be int or float64
		if enabledVal, exists := respMap["EnabledState"]; exists && enabledState == nil {
			switch v := enabledVal.(type) {
			case int:
				enabledState = &v
			case float64:
				enabledInt := int(v)
				enabledState = &enabledInt
			}
		}
	} else {
		// Log unexpected response type for debugging - continue processing without status data
		r.log.Warn("Unexpected response type in extractCIMStatusInfo - status may not be populated", "response_type", response)
	}

	// Map CIM values to Redfish Status if we have data
	if healthState == nil && enabledState == nil {
		return nil
	}

	status := &redfishv1.Status{}

	// Map HealthState to Redfish Health - only set if mapping returns non-empty value
	if healthState != nil {
		if health := r.mapCIMHealthStateToRedfish(*healthState); health != "" {
			status.Health = health
		}
	}

	// Map EnabledState to Redfish State - only set if mapping returns non-empty value
	if enabledState != nil {
		if state := r.mapCIMEnabledStateToRedfish(*enabledState); state != "" {
			status.State = state
		}
	}

	// Only return status if we have at least one field set
	if status.Health == "" && status.State == "" {
		return nil
	}

	return status
}

// mapCIMHealthStateToRedfish converts CIM HealthState to Redfish Health string
func (r *WsmanComputerSystemRepo) mapCIMHealthStateToRedfish(healthState int) string {
	// Validate input range based on CIM HealthState specification
	if healthState < 0 || healthState > 30 {
		return "" // Invalid range
	}

	switch healthState {
	case 0: // Unknown - For systems reporting unknown health, omit the Health field
		return "" // Let field be omitted when health is unknown
	case 5: // OK
		return "OK"
	case 10: // Degraded/Warning
		return "Warning"
	case 15: // Minor failure
		return "Warning"
	case 20: // Major failure
		return "Critical"
	case 25: // Critical failure
		return "Critical"
	case 30: // Non-recoverable error
		return "Critical"
	default: // Other unrecognized values
		return "" // No fallback - let field be omitted
	}
}

// mapCIMEnabledStateToRedfish converts CIM EnabledState to Redfish State string
func (r *WsmanComputerSystemRepo) mapCIMEnabledStateToRedfish(enabledState int) string {
	// Validate input range based on CIM EnabledState specification (0-32767)
	if enabledState < 0 || enabledState > 32767 {
		return "" // Invalid range
	}

	switch enabledState {
	case 1: // Other
		return "Enabled"
	case 2: // Enabled
		return "Enabled"
	case 3: // Disabled
		return "Disabled"
	case 4: // ShuttingDown
		return "Disabled"
	case 5: // NotApplicable
		return "Enabled"
	case 6: // EnabledButOffline
		return "StandbyOffline"
	case 7: // InTest
		return "InTest"
	case 8: // Deferred
		return "Disabled"
	case 9: // Quiesce
		return "Quiesced"
	case 10: // Starting
		return "Starting"
	default: // Unknown or unrecognized values
		return "" // No fallback - let field be omitted
	}
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
		if device.GUID != "" { // Only append non-empty GUIDs
			systemIDs = append(systemIDs, device.GUID)
		}
	}

	return systemIDs, nil
}

// GetByID retrieves a computer system by its ID from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetByID(ctx context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	// Get device information from repository
	device, err := r.usecase.GetByID(ctx, systemID, "", true)
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
	manufacturer, model, serialNumber := r.extractCIMChassisInfo(ctx, systemID)

	// Extract additional CIM properties for new Redfish v1.26.0 support
	description, hostName := r.extractCIMSystemInfo(ctx, systemID)

	// Extract Status information from CIM_Processor data
	status := r.extractCIMStatusInfo(ctx, systemID)

	// Build comprehensive ComputerSystem with v1.26.0 properties
	system := &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         device.Hostname,
		Status:       status,
		PowerState:   redfishPowerState,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		SystemType:   redfishv1.SystemTypePhysical,
		ODataID:      "/redfish/v1/Systems/" + systemID,
		ODataType:    "#ComputerSystem.v1_26_0.ComputerSystem",
	}

	// Only set Description if we have actual CIM data
	if description != "" {
		system.Description = description
	}

	// Only set HostName if we have actual CIM data
	if hostName != "" {
		system.HostName = hostName
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
