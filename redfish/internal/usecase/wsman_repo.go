// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
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

// CIMObjectType represents different types of CIM objects
type CIMObjectType string

const (
	CIMObjectChassis               CIMObjectType = "chassis"
	CIMObjectComputerSystemPackage CIMObjectType = "computersystem"
)

// PropertyExtractor defines a function signature for custom property transformation
type PropertyExtractor func(interface{}) interface{}

// CIMPropertyConfig defines the configuration for extracting a property from CIM data
type CIMPropertyConfig struct {
	CIMObject    CIMObjectType     // Which CIM object to extract from
	CIMProperty  string            // The property name in the CIM object
	StructField  string            // Field name when response is a struct (optional, defaults to CIMProperty)
	Transformer  PropertyExtractor // Optional transformation function
	UseFirstItem bool              // For array responses, use first item (default: true)
}

// CIMExtractorFramework provides a generic framework for extracting properties from CIM objects
type CIMExtractorFramework struct {
	repo *WsmanComputerSystemRepo
}

// WsmanComputerSystemRepo implements ComputerSystemRepository using WSMAN backend.
type WsmanComputerSystemRepo struct {
	usecase *devices.UseCase
	log     logger.Interface
}

// Forward declarations for transformer functions
var healthStateTransformer PropertyExtractor
var enabledStateTransformer PropertyExtractor

// allCIMConfigs defines the complete set of CIM property configurations for computer system data extraction.
// This global configuration is used by GetByID to extract all necessary properties in a single call.
// Transformers are statically assigned for computer system status properties.
var allCIMConfigs = []CIMPropertyConfig{
	// Chassis properties
	{CIMObject: CIMObjectChassis, CIMProperty: "Manufacturer", UseFirstItem: true},
	{CIMObject: CIMObjectChassis, CIMProperty: "Model", UseFirstItem: true},
	{CIMObject: CIMObjectChassis, CIMProperty: "SerialNumber", UseFirstItem: true},
	// ComputerSystem properties
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "Description", UseFirstItem: true},
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "DNSHostName", UseFirstItem: true},
	// Computer System status properties with static transformer functions
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "HealthState", UseFirstItem: true, Transformer: healthStateTransformer},
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "EnabledState", UseFirstItem: true, Transformer: enabledStateTransformer},
}

// NewWsmanComputerSystemRepo creates a new WSMAN-backed computer system repository.
func init() {
	healthStateTransformer = func(value interface{}) interface{} {
		var healthState int
		switch v := value.(type) {
		case int:
			healthState = v
		case float64:
			healthState = int(v)
		default:
			return nil
		}
		// Use existing mapping function logic
		if healthState < 0 || healthState > 30 {
			return "" // Invalid range
		}
		switch healthState {
		case 0:
			return "" // Unknown
		case 5:
			return "OK"
		case 10, 15:
			return "Warning"
		case 20, 25:
			return "Critical"
		case 30:
			return "Critical"
		default:
			return ""
		}
	}

	enabledStateTransformer = func(value interface{}) interface{} {
		var enabledState int
		switch v := value.(type) {
		case int:
			enabledState = v
		case float64:
			enabledState = int(v)
		default:
			return nil
		}
		// Use existing mapping function logic
		if enabledState < 0 || enabledState > 32767 {
			return "" // Invalid range
		}
		switch enabledState {
		case 1, 2, 5:
			return "Enabled"
		case 3, 4, 8:
			return "Disabled"
		case 6:
			return "StandbyOffline"
		case 7:
			return "InTest"
		case 9:
			return "Quiesced"
		case 10:
			return "Starting"
		default:
			return ""
		}
	}
}

// NewWsmanComputerSystemRepo creates a new WSMAN-backed computer system repository.
func NewWsmanComputerSystemRepo(uc *devices.UseCase, log logger.Interface) *WsmanComputerSystemRepo {
	return &WsmanComputerSystemRepo{
		usecase: uc,
		log:     log,
	}
}

// newCIMExtractor creates a new CIM property extraction framework
func (r *WsmanComputerSystemRepo) newCIMExtractor() *CIMExtractorFramework {
	return &CIMExtractorFramework{repo: r}
}

// extractCIMData extracts multiple CIM properties in a single call using the configured extraction framework
func (r *WsmanComputerSystemRepo) extractCIMData(ctx context.Context, systemID string, configs []CIMPropertyConfig) map[string]interface{} {
	extractor := r.newCIMExtractor()
	return extractor.ExtractMultipleProperties(ctx, systemID, configs)
}

// ExtractProperty extracts a single property from CIM data based on configuration
func (f *CIMExtractorFramework) ExtractProperty(ctx context.Context, systemID string, config CIMPropertyConfig) interface{} {
	hwInfo, err := f.repo.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		return nil
	}

	return f.ExtractPropertyFromHardwareInfo(hwInfo, config)
}

// ExtractPropertyFromHardwareInfo extracts a single property from pre-fetched hardware info
func (f *CIMExtractorFramework) ExtractPropertyFromHardwareInfo(hwInfo dto.HardwareInfo, config CIMPropertyConfig) interface{} {
	var response interface{}

	// Select the appropriate CIM object
	switch config.CIMObject {
	case CIMObjectChassis:
		if hwInfo.CIMChassis.Response != nil {
			response = hwInfo.CIMChassis.Response
		}
	case CIMObjectComputerSystemPackage:
		if hwInfo.CIMComputerSystemPackage.Response != nil {
			response = hwInfo.CIMComputerSystemPackage.Response
		}
	default:
		f.repo.log.Warn("Unknown CIM object type", "type", config.CIMObject, "property", config.CIMProperty)
		return nil
	}

	if response == nil {
		return nil
	}

	// Extract the property value
	value := f.extractFromResponse(response, config)

	// Apply transformation if provided
	if config.Transformer != nil && value != nil {
		if transformed := config.Transformer(value); transformed != nil {
			return transformed
		}
		// If transformer returns nil, log warning and return original value
		f.repo.log.Warn("Transformer returned nil", "property", config.CIMProperty, "original_value", value)
	}

	return value
}

// extractFromResponse handles both struct and map response formats
func (f *CIMExtractorFramework) extractFromResponse(response interface{}, config CIMPropertyConfig) interface{} {
	// Try specific type handling for known CIM structs first
	if value := f.extractFromSpecificTypes(response, config); value != nil {
		return value
	}

	// Fall back to map access for generic structures
	return f.extractFromMap(response, config)
}

// extractFromSpecificTypes handles known CIM struct types with specific type assertions
func (f *CIMExtractorFramework) extractFromSpecificTypes(response interface{}, config CIMPropertyConfig) interface{} {
	switch config.CIMObject {
	case CIMObjectChassis:
		if chassisResp, ok := response.(chassis.PackageResponse); ok {
			switch config.CIMProperty {
			case "Manufacturer":
				return chassisResp.Manufacturer
			case "Model":
				return chassisResp.Model
			case "SerialNumber":
				return chassisResp.SerialNumber
			case "Version":
				return chassisResp.Version
			}
		}
		// Note: CIMObjectComputerSystemPackage doesn't have a specific struct type in the CIM messages
		// It uses generic map structures, so it will fall back to map extraction
	}
	return nil
}

// extractFromMap extracts property from map or nested map structures
func (f *CIMExtractorFramework) extractFromMap(response interface{}, config CIMPropertyConfig) interface{} {
	// Early return if response is nil or empty
	if response == nil {
		return nil
	}

	// Helper function to extract from a single map item
	extractFromItem := func(itemMap map[string]interface{}) interface{} {
		if itemMap == nil || len(itemMap) == 0 {
			return nil
		}
		if value, exists := itemMap[config.CIMProperty]; exists {
			return value
		}
		return nil
	}

	// Helper function to process items array
	processItemsArray := func(items []interface{}) interface{} {
		if items == nil || len(items) == 0 {
			return nil
		}
		// Limit iterations to prevent hanging on large arrays
		maxItems := 10
		for i, item := range items {
			if i >= maxItems {
				break
			}
			if itemMap, ok := item.(map[string]interface{}); ok {
				if value := extractFromItem(itemMap); value != nil {
					return value
				}
			}
		}
		return nil
	}

	// Handle map response
	if responseMap, ok := response.(map[string]interface{}); ok {
		// Try PullResponse -> Items structure
		if pullResponse, ok := responseMap["PullResponse"].(map[string]interface{}); ok {
			if items, ok := pullResponse["Items"].([]interface{}); ok {
				if value := processItemsArray(items); value != nil {
					return value
				}
			}
		}

		// Try direct Items array
		if items, ok := responseMap["Items"].([]interface{}); ok {
			if value := processItemsArray(items); value != nil {
				return value
			}
		}

		// Try Body -> PullResponse -> Items structure
		if body, ok := responseMap["Body"].(map[string]interface{}); ok {
			if pullResponse, ok := body["PullResponse"].(map[string]interface{}); ok {
				if items, ok := pullResponse["Items"].([]interface{}); ok {
					if value := processItemsArray(items); value != nil {
						return value
					}
				}
			}
		}

		// Try direct extraction if response is a single object
		if value := extractFromItem(responseMap); value != nil {
			return value
		}
	}

	// Handle array response directly
	if itemsArray, ok := response.([]interface{}); ok {
		if value := processItemsArray(itemsArray); value != nil {
			return value
		}
	}

	return nil
}

// ExtractMultipleProperties extracts multiple properties in a single call for efficiency
func (f *CIMExtractorFramework) ExtractMultipleProperties(ctx context.Context, systemID string, configs []CIMPropertyConfig) map[string]interface{} {
	results := make(map[string]interface{})

	// Get hardware info only once to avoid multiple WSMAN calls
	hwInfo, err := f.repo.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		f.repo.log.Error("Failed to get hardware info", "systemID", systemID, "error", err)
		return results
	}

	for _, config := range configs {
		if value := f.ExtractPropertyFromHardwareInfo(hwInfo, config); value != nil {
			results[config.CIMProperty] = value
		}
	}

	return results
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

// buildStatusFromCIMData creates a Redfish Status object from extracted CIM health and state data
func (r *WsmanComputerSystemRepo) buildStatusFromCIMData(cimData map[string]interface{}) *redfishv1.Status {
	health, hasHealth := cimData["HealthState"].(string)
	state, hasState := cimData["EnabledState"].(string)

	if (!hasHealth || health == "") && (!hasState || state == "") {
		return nil
	}

	status := &redfishv1.Status{}
	if hasHealth && health != "" {
		status.Health = health
	}
	if hasState && state != "" {
		status.State = state
	}
	return status
}

// buildComputerSystemFromCIMData creates a ComputerSystem entity from CIM data only
func (r *WsmanComputerSystemRepo) buildComputerSystemFromCIMData(systemID string, powerState redfishv1.PowerState, cimData map[string]interface{}) *redfishv1.ComputerSystem {
	// Extract CIM properties
	manufacturer, _ := cimData["Manufacturer"].(string)
	model, _ := cimData["Model"].(string)
	serialNumber, _ := cimData["SerialNumber"].(string)
	description, _ := cimData["Description"].(string)
	hostNameFromCIM, _ := cimData["DNSHostName"].(string)

	// Build Status from extracted health and state data
	status := r.buildStatusFromCIMData(cimData)

	// Build ComputerSystem using only CIM data
	system := &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         systemID, // Use systemID as default name
		Status:       status,
		PowerState:   powerState,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		SystemType:   redfishv1.SystemTypePhysical,
		ODataID:      "/redfish/v1/Systems/" + systemID,
		ODataType:    "#ComputerSystem.v1_26_0.ComputerSystem",
	}

	// Set optional properties only if we have actual CIM data
	if description != "" {
		system.Description = description
	}
	if hostNameFromCIM != "" {
		system.HostName = hostNameFromCIM
		system.Name = hostNameFromCIM // Use CIM hostname as the name
	}

	return system
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
	// Verify device exists first
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

	// Extract CIM data using the global configuration with static transformers
	cimData := r.extractCIMData(ctx, systemID, allCIMConfigs)

	// Build and return the complete ComputerSystem using only CIM data
	system := r.buildComputerSystemFromCIMData(systemID, redfishPowerState, cimData)
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
