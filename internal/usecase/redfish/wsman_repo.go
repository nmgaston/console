// Package redfish provides a WSMAN-backed implementation of ComputerSystemRepository.
package redfish

import (
	"context"
	"fmt"

	"github.com/device-management-toolkit/console/internal/entity/redfish/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

var ErrSystemNotFound = fmt.Errorf("system not found")

// WsmanComputerSystemRepo implements ComputerSystemRepository using WSMAN backend.
type WsmanComputerSystemRepo struct {
	usecase *devices.UseCase
}

func NewWsmanComputerSystemRepo(uc *devices.UseCase) *WsmanComputerSystemRepo {
	return &WsmanComputerSystemRepo{usecase: uc}
}

func (r *WsmanComputerSystemRepo) GetByID(systemID string) (*redfish.ComputerSystem, error) {
	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(context.Background(), systemID)
	if err != nil {
		if err.Error() == "DevicesUseCase -  - : " {
			return nil, ErrSystemNotFound
		}
		return nil, err
	}

	// Map the integer power state to Redfish PowerState
	var redfishPowerState redfish.PowerState
	switch powerState.PowerState {
	case 2: // Power On
		redfishPowerState = redfish.PowerStateOn
	case 8: // Power Off (Soft)
		redfishPowerState = redfish.PowerStateOff
	case 6: // Power Off (Hard)
		redfishPowerState = redfish.PowerStateOff
	default:
		redfishPowerState = redfish.PowerStateOff // Default to Off for unknown states
	}

	// Return a minimal ComputerSystem with power state
	return &redfish.ComputerSystem{
		ID:         systemID,
		PowerState: redfishPowerState,
	}, nil
}

func (r *WsmanComputerSystemRepo) GetAll() ([]*redfish.ComputerSystem, error) {
	// TODO: Implement WSMAN query for all ComputerSystems
	return nil, fmt.Errorf("GetAll not implemented")
}

func (r *WsmanComputerSystemRepo) UpdatePowerState(systemID string, state redfish.PowerState) error {
	var action int
	switch state {
	case redfish.PowerStateOn:
		action = devices.CIMPMSPowerOn
	case redfish.PowerStateOff, redfish.ResetTypeForceOff:
		action = devices.PowerDown
	case redfish.ResetTypePowerCycle:
		action = devices.PowerCycle
	case redfish.ResetTypeForceRestart:
		action = devices.Reset
	default:
		return fmt.Errorf("unsupported power state: %s", state)
	}
	_, err := r.usecase.SendPowerAction(context.Background(), systemID, action)
	if err != nil && err.Error() == "DevicesUseCase -  - : " {
		return ErrSystemNotFound
	}
	return err
}
