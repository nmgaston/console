package wsman

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "github.com/device-management-toolkit/console/internal/entity/redfish/v1"
)

type WSMANComputerSystemRepository struct {
    AMTHost  string
    Username string
    Password string
}

func (r *WSMANComputerSystemRepository) UpdatePowerState(id string, state redfish.PowerState) error {
    var powerAction string
    switch state {
    case redfish.PowerStateOn:
        powerAction = "2" // AMT: 2 = On
    case redfish.PowerStateOff:
        powerAction = "8" // AMT: 8 = Off (soft)
    default:
        return fmt.Errorf("unsupported power state: %s", state)
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
    req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(soap))
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

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    // Check for a success marker in the response
    if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("ReturnValue>0<")) {
        return fmt.Errorf("AMT power action failed: %s", string(body))
    }

    return nil
}