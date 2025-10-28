package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RedfishServer implements the ServerInterface for Redfish v1 API
type RedfishServer struct {
	// Add dependencies here (database, logger, etc.)
}

// NewRedfishServer creates a new instance of RedfishServer
func NewRedfishServer() *RedfishServer {
	return &RedfishServer{}
}

// GetServiceRoot implements ServerInterface.GetServiceRoot
func (s *RedfishServer) GetServiceRoot(c *gin.Context) {
	odataID := ODataID("/redfish/v1")
	odataType := ODataType("#ServiceRoot.v1_15_0.ServiceRoot")
	odataContext := ODataContext("/redfish/v1/$metadata#ServiceRoot.ServiceRoot")

	systemsOdataID := ODataID("/redfish/v1/Systems")
	chassisOdataID := ODataID("/redfish/v1/Chassis")
	managersOdataID := ODataID("/redfish/v1/Managers")

	uuidVal := uuid.New()

	// Create structs for Systems, Chassis, and Managers references
	systemsRef := &struct {
		OdataID *ODataID `json:"@odata.id,omitempty"`
	}{
		OdataID: &systemsOdataID,
	}

	chassisRef := &struct {
		OdataID *ODataID `json:"@odata.id,omitempty"`
	}{
		OdataID: &chassisOdataID,
	}

	managersRef := &struct {
		OdataID *ODataID `json:"@odata.id,omitempty"`
	}{
		OdataID: &managersOdataID,
	}

	serviceRoot := ServiceRoot{
		OdataType:      odataType,
		OdataID:        odataID,
		OdataContext:   &odataContext,
		Id:             "RootService",
		Name:           "Root Service",
		Description:    stringPtr("The root of the Redfish service"),
		RedfishVersion: stringPtr("1.15.0"),
		UUID:           &uuidVal,
		Systems:        systemsRef,
		Chassis:        chassisRef,
		Managers:       managersRef,
	}

	c.JSON(http.StatusOK, serviceRoot)
}

// GetSystemsCollection implements ServerInterface.GetSystemsCollection
func (s *RedfishServer) GetSystemsCollection(c *gin.Context) {
	odataID := ODataID("/redfish/v1/Systems")
	odataType := ODataType("#ComputerSystemCollection.ComputerSystemCollection")
	odataContext := ODataContext("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection")

	systemOdataID := ODataID("/redfish/v1/Systems/1")

	collection := ComputerSystemCollection{
		OdataType:    odataType,
		OdataID:      odataID,
		OdataContext: &odataContext,
		Name:         "Computer System Collection",
		Description:  stringPtr("Collection of Computer Systems"),
		Members: []struct {
			OdataID *ODataID `json:"@odata.id,omitempty"`
		}{
			{OdataID: &systemOdataID},
		},
		MembersOdataCount: 1,
	}

	c.JSON(http.StatusOK, collection)
}

// GetSystem implements ServerInterface.GetSystem
func (s *RedfishServer) GetSystem(c *gin.Context, systemID string) {
	if systemID != "1" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "Base.1.11.ResourceNotFound",
				"message": "The requested resource was not found",
			},
		})

		return
	}

	odataID := ODataID("/redfish/v1/Systems/1")
	odataType := ODataType("#ComputerSystem.v1_20_0.ComputerSystem")
	odataContext := ODataContext("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")

	system := ComputerSystem{
		OdataType:    odataType,
		OdataID:      odataID,
		OdataContext: &odataContext,
		Id:           "1",
		Name:         "System",
		Description:  stringPtr("Computer System"),
		SystemType:   (*ComputerSystemSystemType)(stringPtr("Physical")),
		Model:        stringPtr("Example System Model"),
		Manufacturer: stringPtr("Example Manufacturer"),
		SerialNumber: stringPtr("12345678"),
		PowerState:   (*ComputerSystemPowerState)(stringPtr("On")),
		Status: &Status{
			State:  (*StatusState)(stringPtr("Enabled")),
			Health: (*StatusHealth)(stringPtr("OK")),
		},
	}

	c.JSON(http.StatusOK, system)
}

// GetChassisCollection implements ServerInterface.GetChassisCollection
func (s *RedfishServer) GetChassisCollection(c *gin.Context) {
	odataID := ODataID("/redfish/v1/Chassis")
	odataType := ODataType("#ChassisCollection.ChassisCollection")
	odataContext := ODataContext("/redfish/v1/$metadata#ChassisCollection.ChassisCollection")

	chassisOdataID := ODataID("/redfish/v1/Chassis/1")

	collection := ChassisCollection{
		OdataType:    odataType,
		OdataID:      odataID,
		OdataContext: &odataContext,
		Name:         "Chassis Collection",
		Description:  stringPtr("Collection of Chassis"),
		Members: []struct {
			OdataID *ODataID `json:"@odata.id,omitempty"`
		}{
			{OdataID: &chassisOdataID},
		},
		MembersOdataCount: 1,
	}

	c.JSON(http.StatusOK, collection)
}

// GetChassis implements ServerInterface.GetChassis
func (s *RedfishServer) GetChassis(c *gin.Context, chassisID string) {
	if chassisID != "1" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "Base.1.11.ResourceNotFound",
				"message": "The requested resource was not found",
			},
		})

		return
	}

	odataID := ODataID("/redfish/v1/Chassis/1")
	odataType := ODataType("#Chassis.v1_21_0.Chassis")
	odataContext := ODataContext("/redfish/v1/$metadata#Chassis.Chassis")

	chassis := Chassis{
		OdataType:    odataType,
		OdataID:      odataID,
		OdataContext: &odataContext,
		Id:           "1",
		Name:         "Chassis",
		Description:  stringPtr("Main Chassis"),
		ChassisType:  (*ChassisChassisType)(stringPtr("RackMount")),
		Model:        stringPtr("Example Chassis Model"),
		Manufacturer: stringPtr("Example Manufacturer"),
		SerialNumber: stringPtr("87654321"),
		Status: &Status{
			State:  (*StatusState)(stringPtr("Enabled")),
			Health: (*StatusHealth)(stringPtr("OK")),
		},
	}

	c.JSON(http.StatusOK, chassis)
}

// GetManagersCollection implements ServerInterface.GetManagersCollection
func (s *RedfishServer) GetManagersCollection(c *gin.Context) {
	odataID := ODataID("/redfish/v1/Managers")
	odataType := ODataType("#ManagerCollection.ManagerCollection")
	odataContext := ODataContext("/redfish/v1/$metadata#ManagerCollection.ManagerCollection")

	managerOdataID := ODataID("/redfish/v1/Managers/1")

	collection := ManagerCollection{
		OdataType:    odataType,
		OdataID:      odataID,
		OdataContext: &odataContext,
		Name:         "Manager Collection",
		Description:  stringPtr("Collection of Managers"),
		Members: []struct {
			OdataID *ODataID `json:"@odata.id,omitempty"`
		}{
			{OdataID: &managerOdataID},
		},
		MembersOdataCount: 1,
	}

	c.JSON(http.StatusOK, collection)
}

// GetManager implements ServerInterface.GetManager
func (s *RedfishServer) GetManager(c *gin.Context, managerID string) {
	if managerID != "1" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "Base.1.11.ResourceNotFound",
				"message": "The requested resource was not found",
			},
		})

		return
	}

	odataID := ODataID("/redfish/v1/Managers/1")
	odataType := ODataType("#Manager.v1_17_0.Manager")
	odataContext := ODataContext("/redfish/v1/$metadata#Manager.Manager")

	manager := Manager{
		OdataType:       odataType,
		OdataID:         odataID,
		OdataContext:    &odataContext,
		Id:              "1",
		Name:            "Manager",
		Description:     stringPtr("BMC Manager"),
		ManagerType:     (*ManagerManagerType)(stringPtr("BMC")),
		Model:           stringPtr("Example BMC Model"),
		FirmwareVersion: stringPtr("1.0.0"),
		Status: &Status{
			State:  (*StatusState)(stringPtr("Enabled")),
			Health: (*StatusHealth)(stringPtr("OK")),
		},
	}

	c.JSON(http.StatusOK, manager)
}

// stringPtr is a helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
