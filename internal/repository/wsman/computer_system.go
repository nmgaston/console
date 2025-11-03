package wsman

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/device-management-toolkit/console/internal/entity/redfish/v1"
)

var (
	// ErrUnsupportedPowerState indicates the requested power state is not supported by this implementation.
	ErrUnsupportedPowerState = errors.New("unsupported power state")
	// ErrAMTPowerActionFailed indicates the AMT device rejected the power action request.
	ErrAMTPowerActionFailed = errors.New("AMT power action failed")
)

// ComputerSystemRepository provides WSMAN-based operations for computer systems.
type ComputerSystemRepository struct {
	AMTHost  string
	Username string
	Password string
}

// UpdatePowerState sends a power state change request to the AMT device.
func (r *ComputerSystemRepository) UpdatePowerState(_ string, state redfish.PowerState) error {
	var powerAction string

	switch state {
	case redfish.PowerStateOn:
		powerAction = fmt.Sprintf("%d", redfish.CIMPowerStateOn)
	case redfish.PowerStateOff:
		powerAction = fmt.Sprintf("%d", redfish.CIMPowerStateOffSoft)
	case redfish.ResetTypeForceOff, redfish.ResetTypeForceRestart, redfish.ResetTypePowerCycle:
		// These reset types are not yet implemented for AMT
		return fmt.Errorf("%w: %s", ErrUnsupportedPowerState, state)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedPowerState, state)
	}

	// Build SOAP payload
	soap := fmt.Sprintf(`
        <s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
            <s:Body>
                <RequestPowerStateChange_INPUT xmlns="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_PowerManagementService">
                    <PowerState>%s</PowerState>
                    <ManagedElement>Intel(r) AMT</ManagedElement>
                </RequestPowerStateChange_INPUT>
            </s:Body>
        </s:Envelope>
    `, powerAction)

	endpoint := fmt.Sprintf("http://%s:16992/wsman", r.AMTHost)

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBufferString(soap))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.SetBasicAuth(r.Username, r.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check for a success marker in the response
	if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("ReturnValue>0<")) {
		return fmt.Errorf("%w: %s", ErrAMTPowerActionFailed, string(body))
	}

	return nil
}
