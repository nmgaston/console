package presenters

import (
	"github.com/device-management-toolkit/console/internal/entity/redfish/v1"
	"github.com/gin-gonic/gin"
)

// ComputerSystemPresenter formats ComputerSystem data for HTTP responses.
type ComputerSystemPresenter struct{}

// PresentComputerSystem converts a ComputerSystem entity to a gin.H map for JSON response.
func (p *ComputerSystemPresenter) PresentComputerSystem(system *redfish.ComputerSystem) gin.H {
	return gin.H{
		"@odata.id":    system.ODataID,
		"@odata.type":  system.ODataType,
		"Id":           system.ID,
		"Name":         system.Name,
		"SystemType":   string(system.SystemType),
		"Manufacturer": system.Manufacturer,
		"Model":        system.Model,
		"SerialNumber": system.SerialNumber,
		"PowerState":   string(system.PowerState),
	}
}
