// Package redfish provides entity definitions for Redfish computer systems.
package redfish

// ComputerSystem represents a Redfish Computer System entity.
type ComputerSystem struct {
	ID           string
	Name         string
	SystemType   SystemType
	Manufacturer string
	Model        string
	SerialNumber string
	PowerState   PowerState
	ODataID      string
	ODataType    string
}

// SystemType represents the type of computer system.
type SystemType string

const (
	// SystemTypePhysical indicates a physical computer system.
	SystemTypePhysical SystemType = "Physical"
	// SystemTypeVirtual indicates a virtual computer system.
	SystemTypeVirtual SystemType = "Virtual"
)

// PowerState represents the power state of a computer system.
type PowerState string

const (
	// PowerStateOn indicates that the system is powered on.
	PowerStateOn PowerState = "On"
	// PowerStateOff indicates that the system is powered off.
	PowerStateOff PowerState = "Off"
	// ResetTypeOn indicates a power on reset.
	ResetTypeOn PowerState = "On"
	// ResetTypeForceOff indicates a forced power off.
	ResetTypeForceOff PowerState = "ForceOff"
	// ResetTypeForceRestart indicates a forced restart.
	ResetTypeForceRestart PowerState = "ForceRestart"
	// ResetTypePowerCycle indicates a power cycle.
	ResetTypePowerCycle PowerState = "PowerCycle"
)