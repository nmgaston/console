// Package redfish provides a WSMAN-backed implementation of ComputerSystemRepository.
package redfish

import (
	"context"
	"errors"
	"fmt"

	"github.com/device-management-toolkit/console/internal/entity/redfish/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

const (
	// ErrMsgDeviceNotFound is the error message returned by devices use case when device is not found.
	ErrMsgDeviceNotFound = "DevicesUseCase -  - : "
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

func NewWsmanComputerSystemRepo(uc *devices.UseCase) *WsmanComputerSystemRepo {
	return &WsmanComputerSystemRepo{usecase: uc}
}

func (r *WsmanComputerSystemRepo) GetByID(systemID string) (*redfish.ComputerSystem, error) {
	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(context.Background(), systemID)
	if err != nil {
		if err.Error() == ErrMsgDeviceNotFound {
			return nil, ErrSystemNotFound
		}

		return nil, err
	}

	// Map the integer power state to Redfish PowerState
	var redfishPowerState redfish.PowerState
	switch powerState.PowerState {
	case redfish.CIMPowerStateOn:
		redfishPowerState = redfish.PowerStateOn
	case redfish.CIMPowerStateOffSoft:
		redfishPowerState = redfish.PowerStateOff
	case redfish.CIMPowerStateOffHard:
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
	//nolint:godox // TODO comment is intentional - feature not yet implemented
	// TODO: Implement WSMAN query for all ComputerSystems
	return nil, ErrGetAllNotImplemented
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
		return fmt.Errorf("%w: %s", ErrUnsupportedPowerState, state)
	}

	_, err := r.usecase.SendPowerAction(context.Background(), systemID, action)
	if err != nil && err.Error() == ErrMsgDeviceNotFound {
		return ErrSystemNotFound
	}

	return err
}
