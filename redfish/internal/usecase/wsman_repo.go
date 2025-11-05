// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
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

	// Return a minimal ComputerSystem with power state
	return &redfishv1.ComputerSystem{
		ID:         systemID,
		PowerState: redfishPowerState,
	}, nil
}

func (r *WsmanComputerSystemRepo) GetAll() ([]*redfishv1.ComputerSystem, error) {
	//nolint:godox // TODO comment is intentional - feature not yet implemented
	// TODO: Implement WSMAN query for all ComputerSystems
	return nil, ErrGetAllNotImplemented
}

func (r *WsmanComputerSystemRepo) UpdatePowerState(_ string, _ redfishv1.PowerState) error {
	/* Comment the code for now - to be revisited later

	  var action int


		switch state {
		case redfishv1.PowerStateOn:
			action = devices.CIMPMSPowerOn
		case redfishv1.PowerStateOff, redfishv1.ResetTypeForceOff:
			action = devices.PowerDown
		case redfishv1.ResetTypePowerCycle:
			action = devices.PowerCycle
		case redfishv1.ResetTypeForceRestart:
			action = devices.Reset
		default:
			return fmt.Errorf("%w: %s", ErrUnsupportedPowerState, state)
		}

		_, err := r.usecase.SendPowerAction(context.Background(), systemID, action)
		if err != nil && err.Error() == ErrMsgDeviceNotFound {
			return ErrSystemNotFound
		}

		return err
	*/
	return devices.ErrNotSupportedUseCase
}
