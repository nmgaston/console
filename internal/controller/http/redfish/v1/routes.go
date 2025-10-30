// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/http/redfish/v1/redfishapi"
)

// SetupRedfishV1Routes sets up the Redfish v1 routes on the main router
func SetupRedfishV1Routes(router *gin.Engine) {
	// Create a new Redfish server instance
	redfishServer := &RedfishServer{}

	// Create a route group for Redfish v1 API
	v1Group := router.Group("/redfish/v1")

	// Register the handlers with options
	redfishapi.RegisterHandlersWithOptions(v1Group, redfishServer, redfishapi.GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"code":    "Base.1.11.GeneralError",
					"message": err.Error(),
				},
			})
		},
	})
}

// RedfishServer implements the generated ServerInterface from redfishapi package
type RedfishServer struct{}

// Ensure RedfishServer implements redfishapi.ServerInterface
var _ redfishapi.ServerInterface = (*RedfishServer)(nil)

// Helper function to create string pointers
func StringPtr(s string) *string {
	return &s
}

// Helper function to create int pointers
func IntPtr(i int) *int {
	return &i
}

// Helper function to create int64 pointers
func Int64Ptr(i int64) *int64 {
	return &i
}

// Helper function to create ChassisType pointers
func ChassisTypePtr(ct redfishapi.ChassisChassisType) *redfishapi.ChassisChassisType {
	return &ct
}

// Helper function to create ManagerType pointers
func ManagerTypePtr(mt redfishapi.ManagerManagerType) *redfishapi.ManagerManagerType {
	return &mt
}

// Helper function to create SystemType pointers
func SystemTypePtr(st redfishapi.ComputerSystemSystemType) *redfishapi.ComputerSystemSystemType {
	return &st
}

// GetRedfishV1 returns the service root
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	serviceRoot := redfishapi.ServiceRootServiceRoot{
		OdataContext:   StringPtr("/redfish/v1/$metadata#ServiceRoot.ServiceRoot"),
		OdataId:        StringPtr("/redfish/v1"),
		OdataType:      StringPtr("#ServiceRoot.v1_19_0.ServiceRoot"),
		Id:             "RootService",
		Name:           "Root Service",
		RedfishVersion: StringPtr("1.19.0"),
		Systems: &redfishapi.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Systems"),
		},
		Chassis: &redfishapi.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Chassis"),
		},
		Managers: &redfishapi.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Managers"),
		},
	}
	c.JSON(http.StatusOK, serviceRoot)
}

// GetRedfishV1Metadata returns the OData metadata
func (s *RedfishServer) GetRedfishV1Metadata(c *gin.Context) {
	metadata := ""

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, metadata)
}

// GetRedfishV1Systems returns the computer systems collection
func (s *RedfishServer) GetRedfishV1Systems(c *gin.Context) {
	collection := redfishapi.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataId:           StringPtr("/redfish/v1/Systems"),
		OdataType:         StringPtr("#ComputerSystemCollection.ComputerSystemCollection"),
		Name:              "Computer System Collection",
		Description:       nil,
		MembersOdataCount: Int64Ptr(1),
		Members: &[]redfishapi.OdataV4IdRef{
			{OdataId: StringPtr("/redfish/v1/Systems/System1")},
		},
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1SystemsComputerSystemId returns a specific computer system
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	if computerSystemID != "System1" {
		c.JSON(http.StatusNotFound, gin.H{"error": "System not found"})

		return
	}

	system := redfishapi.ComputerSystemComputerSystem{
		OdataContext: StringPtr("/redfish/v1/$metadata#ComputerSystem.ComputerSystem"),
		OdataId:      StringPtr("/redfish/v1/Systems/System1"),
		OdataType:    StringPtr("#ComputerSystem.v1_26_0.ComputerSystem"),
		Id:           "System1",
		Name:         "Computer System",
		SerialNumber: StringPtr("SN123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example System"),
		SystemType:   SystemTypePtr(redfishapi.Physical),
	}
	c.JSON(http.StatusOK, system)
}

// GetRedfishV1Chassis returns the chassis collection
func (s *RedfishServer) GetRedfishV1Chassis(c *gin.Context) {
	collection := redfishapi.ChassisCollectionChassisCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ChassisCollection.ChassisCollection"),
		OdataId:           StringPtr("/redfish/v1/Chassis"),
		OdataType:         StringPtr("#ChassisCollection.ChassisCollection"),
		Name:              "Chassis Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]redfishapi.OdataV4IdRef{
			{OdataId: StringPtr("/redfish/v1/Chassis/Chassis1")},
		},
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1ChassisChassisId returns a specific chassis
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1ChassisChassisId(c *gin.Context, chassisID string) {
	if chassisID != "Chassis1" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chassis not found"})

		return
	}

	chassis := redfishapi.ChassisChassis{
		OdataContext: StringPtr("/redfish/v1/$metadata#Chassis.Chassis"),
		OdataId:      StringPtr("/redfish/v1/Chassis/Chassis1"),
		OdataType:    StringPtr("#Chassis.v1_28_0.Chassis"),
		Id:           "Chassis1",
		Name:         "Computer System Chassis",
		SerialNumber: StringPtr("CH123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example Chassis"),
		ChassisType:  redfishapi.RackMount,
	}
	c.JSON(http.StatusOK, chassis)
}

// GetRedfishV1Managers returns the managers collection
func (s *RedfishServer) GetRedfishV1Managers(c *gin.Context) {
	collection := redfishapi.ManagerCollectionManagerCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ManagerCollection.ManagerCollection"),
		OdataId:           StringPtr("/redfish/v1/Managers"),
		OdataType:         StringPtr("#ManagerCollection.ManagerCollection"),
		Name:              "Manager Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]redfishapi.OdataV4IdRef{
			{OdataId: StringPtr("/redfish/v1/Managers/Manager1")},
		},
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1ManagersManagerId returns a specific manager
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1ManagersManagerId(c *gin.Context, managerID string) {
	if managerID != "Manager1" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manager not found"})

		return
	}

	manager := redfishapi.ManagerManager{
		OdataContext: StringPtr("/redfish/v1/$metadata#Manager.Manager"),
		OdataId:      StringPtr("/redfish/v1/Managers/Manager1"),
		OdataType:    StringPtr("#Manager.v1_21_0.Manager"),
		Id:           "Manager1",
		Name:         "System Manager",
		Model:        StringPtr("Example Manager"),
		ManagerType:  ManagerTypePtr(redfishapi.BMC),
	}
	c.JSON(http.StatusOK, manager)
}

// SetupRedfishV1RoutesProtected sets up the Redfish v1 routes with JWT protection at /redfish/v1
func SetupRedfishV1RoutesProtected(router *gin.Engine, jwtMiddleware gin.HandlerFunc) {
	// Create a new Redfish server instance
	redfishServer := &RedfishServer{}

	// Create a route group for Redfish v1 API with JWT middleware -
	// Note: /redfish/v1 is not mention as the spec by default uses this route
	v1Group := router.Group("")
	if jwtMiddleware != nil {
		v1Group.Use(jwtMiddleware)
	}

	// Register the handlers with options
	redfishapi.RegisterHandlersWithOptions(v1Group, redfishServer, redfishapi.GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"code":    "Base.1.11.GeneralError",
					"message": err.Error(),
				},
			})
		},
	})
}
