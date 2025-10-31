// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/labstack/gommon/log"

	redfishv1 "github.com/device-management-toolkit/console/internal/entity/redfish/v1"
	"github.com/device-management-toolkit/console/internal/usecase/redfish"
	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/pkg/api"
)

// RedfishServer implements the Redfish API handlers
// Add dependencies here if needed (e.g., usecase, presenter, etc.)
type RedfishServer struct {
	ComputerSystemUC *redfish.ComputerSystemUseCase
}

/*
Comment: function not used/invoked
SetupRedfishV1Routes sets up the Redfish v1 routes on the main router
func SetupRedfishV1Routes(router *gin.Engine, devicesUC *devices.UseCase) {
	// Enable HandleMethodNotAllowed to properly distinguish between 404 and 405 errors
	router.HandleMethodNotAllowed = true

	repo := redfish.NewWsmanComputerSystemRepo(devicesUC)
	computerSystemUC := &redfish.ComputerSystemUseCase{Repo: repo}
	redfishServer := &RedfishServer{ComputerSystemUC: computerSystemUC}

	v1Group := router.Group("/redfish/v1")

	// Register the handlers with options
	redfishapi.RegisterHandlersWithOptions(v1Group, redfishServer, redfishapi.GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			switch statusCode {
			case http.StatusUnauthorized:
				UnauthorizedError(c)
			case http.StatusForbidden:
				ForbiddenError(c)
			case http.StatusMethodNotAllowed:
				MethodNotAllowedError(c)
			case http.StatusBadRequest:
				BadRequestError(c, err.Error(), "Base.1.11.GeneralError", "Check your request body and parameters.", "Critical")
			case http.StatusNotFound:
				NotFoundError(c, "Resource")
			case http.StatusConflict:
				ConflictError(c, "Resource", err.Error())
			case http.StatusServiceUnavailable:
				ServiceUnavailableError(c, 60)
			default:
				InternalServerError(c, err)
			}
		},
	})

	// Add Redfish-compliant handler for 405 Method Not Allowed
	router.NoMethod(func(c *gin.Context) {
		MethodNotAllowedError(c)
	})

	// Add Redfish-compliant NoRoute handler for /redfish/v1
	router.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 11 && c.Request.URL.Path[:11] == "/redfish/v1" {
			NotFoundError(c, "Resource")
		} else {
			c.Next() // fallback to Gin's default
		}
	})

}
*/

// Ensure RedfishServer implements api.ServerInterface
var _ api.ServerInterface = (*RedfishServer)(nil)

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr creates a pointer to an int value.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr creates a pointer to an int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// ChassisTypePtr creates a pointer to a ChassisChassisType value.
func ChassisTypePtr(ct api.ChassisChassisType) *api.ChassisChassisType {
	return &ct
}

// ManagerTypePtr creates a pointer to a ManagerManagerType value.
func ManagerTypePtr(mt api.ManagerManagerType) *api.ManagerManagerType {
	return &mt
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st api.ComputerSystemSystemType) *api.ComputerSystemSystemType {
	return &st
}

// GetRedfishV1 returns the service root
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	serviceRoot := api.ServiceRootServiceRoot{
		OdataContext:   StringPtr("/redfish/v1/$metadata#ServiceRoot.ServiceRoot"),
		OdataId:        StringPtr("/redfish/v1"),
		OdataType:      StringPtr("#ServiceRoot.v1_19_0.ServiceRoot"),
		Id:             "RootService",
		Name:           "Root Service",
		RedfishVersion: StringPtr("1.19.0"),
		Systems: &api.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Systems"),
		},
		Chassis: &api.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Chassis"),
		},
		Managers: &api.OdataV4IdRef{
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
	collection := api.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataId:           StringPtr("/redfish/v1/Systems"),
		OdataType:         StringPtr("#ComputerSystemCollection.ComputerSystemCollection"),
		Name:              "Computer System Collection",
		Description:       nil,
		MembersOdataCount: Int64Ptr(1),
		Members: &[]api.OdataV4IdRef{
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
		NotFoundError(c, "System")
		return
	}

	system := api.ComputerSystemComputerSystem{
		OdataContext: StringPtr("/redfish/v1/$metadata#ComputerSystem.ComputerSystem"),
		OdataId:      StringPtr("/redfish/v1/Systems/System1"),
		OdataType:    StringPtr("#ComputerSystem.v1_26_0.ComputerSystem"),
		Id:           "System1",
		Name:         "Computer System",
		SerialNumber: StringPtr("SN123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example System"),
		SystemType:   SystemTypePtr(api.Physical),
	}
	c.JSON(http.StatusOK, system)
}

// GetRedfishV1Chassis returns the chassis collection
func (s *RedfishServer) GetRedfishV1Chassis(c *gin.Context) {
	collection := api.ChassisCollectionChassisCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ChassisCollection.ChassisCollection"),
		OdataId:           StringPtr("/redfish/v1/Chassis"),
		OdataType:         StringPtr("#ChassisCollection.ChassisCollection"),
		Name:              "Chassis Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]api.OdataV4IdRef{
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
		NotFoundError(c, "Chassis")
		return
	}

	chassis := api.ChassisChassis{
		OdataContext: StringPtr("/redfish/v1/$metadata#Chassis.Chassis"),
		OdataId:      StringPtr("/redfish/v1/Chassis/Chassis1"),
		OdataType:    StringPtr("#Chassis.v1_28_0.Chassis"),
		Id:           "Chassis1",
		Name:         "Computer System Chassis",
		SerialNumber: StringPtr("CH123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example Chassis"),
		ChassisType:  api.RackMount,
	}
	c.JSON(http.StatusOK, chassis)
}

// GetRedfishV1Managers returns the managers collection
func (s *RedfishServer) GetRedfishV1Managers(c *gin.Context) {
	collection := api.ManagerCollectionManagerCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ManagerCollection.ManagerCollection"),
		OdataId:           StringPtr("/redfish/v1/Managers"),
		OdataType:         StringPtr("#ManagerCollection.ManagerCollection"),
		Name:              "Manager Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]api.OdataV4IdRef{
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
		NotFoundError(c, "Manager")
		return
	}

	manager := api.ManagerManager{
		OdataContext: StringPtr("/redfish/v1/$metadata#Manager.Manager"),
		OdataId:      StringPtr("/redfish/v1/Managers/Manager1"),
		OdataType:    StringPtr("#Manager.v1_21_0.Manager"),
		Id:           "Manager1",
		Name:         "System Manager",
		Model:        StringPtr("Example Manager"),
		ManagerType:  ManagerTypePtr(api.BMC),
	}
	c.JSON(http.StatusOK, manager)
}

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset handles the reset action for a computer system
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, computerSystemID string) {
	var req api.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)
		return
	}
	if req.ResetType == nil || *req.ResetType == "" {
		PropertyMissingError(c, "ResetType")
		return
	}

	log.Infof("Received reset request for ComputerSystem %s with ResetType %s", computerSystemID, *req.ResetType)
	// TODO: Add authorization check for 403 Forbidden
	// Example implementation (requires JWT middleware to store user claims in context):
	//
	// userRole, exists := c.Get("user_role")
	// if !exists || userRole == "read-only" {
	//     ForbiddenError(c)
	//     return
	// }
	//
	// More sophisticated example with permission-based checks:
	// permissions, exists := c.Get("user_permissions")
	// if !exists {
	//     ForbiddenError(c)
	//     return
	// }
	// permList := permissions.([]string)
	// if !contains(permList, "system:reset") {
	//     ForbiddenError(c)
	//     return
	// }

	err := s.ComputerSystemUC.SetPowerState(computerSystemID, redfishv1.PowerState(*req.ResetType))
	if err != nil {
		if err == redfish.ErrInvalidPowerState {
			PropertyValueNotInListError(c, "ResetType")
			return
		}
		if err == redfish.ErrPowerStateConflict {
			PowerStateConflictError(c, string(*req.ResetType))
			return
		}
		// Robust backend not found error
		if err.Error() == "system not found" {
			NotFoundError(c, "System")
			return
		}
		// Check for service unavailability errors (503)
		// This catches cases where the backend service (WSMAN/AMT) is unreachable
		errMsg := err.Error()
		if errMsg == "connection refused" || errMsg == "connection timeout" ||
			errMsg == "service unavailable" || errMsg == "device not responding" {
			ServiceUnavailableError(c, 60)
			return
		}
		InternalServerError(c, err)
		return
	}

	// Generate dynamic Task response
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	task := map[string]interface{}{
		"@odata.context": "/redfish/v1/$metadata#Task.Task",
		"@odata.id":      "/redfish/v1/TaskService/Tasks/" + taskID,
		"@odata.type":    "#Task.v1_6_0.Task",
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   "The request completed successfully.",
				"MessageId": "Base.1.11.0.Success",
				"Severity":  "OK",
			},
		},
		"Name":       "System Reset Task",
		"StartTime":  now,
		"TaskState":  "Completed",
		"TaskStatus": "OK",
	}
	c.Header("Location", "/redfish/v1/TaskService/Tasks/"+taskID)
	c.JSON(http.StatusAccepted, task)
}

// SetupRedfishV1RoutesProtected sets up the Redfish v1 routes with JWT protection at /redfish/v1
func SetupRedfishV1RoutesProtected(router *gin.Engine, jwtMiddleware gin.HandlerFunc, devicesUC *devices.UseCase) {
	// Enable HandleMethodNotAllowed to properly distinguish between 404 and 405 errors
	router.HandleMethodNotAllowed = true

	repo := redfish.NewWsmanComputerSystemRepo(devicesUC)
	computerSystemUC := &redfish.ComputerSystemUseCase{Repo: repo}
	redfishServer := &RedfishServer{ComputerSystemUC: computerSystemUC}

	v1Group := router.Group("")
	/* Ignore authentication for now until we implement http basic auth
	if jwtMiddleware != nil {
		v1Group.Use(jwtMiddleware)
	}
	*/

	api.RegisterHandlersWithOptions(v1Group, redfishServer, api.GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			switch statusCode {
			case http.StatusUnauthorized:
				UnauthorizedError(c)
			case http.StatusForbidden:
				ForbiddenError(c)
			case http.StatusMethodNotAllowed:
				MethodNotAllowedError(c)
			case http.StatusBadRequest:
				BadRequestError(c, err.Error(), "Base.1.11.GeneralError", "Check your request body and parameters.", "Critical")
			case http.StatusNotFound:
				NotFoundError(c, "Resource")
			case http.StatusConflict:
				ConflictError(c, "Resource", err.Error())
			case http.StatusServiceUnavailable:
				ServiceUnavailableError(c, 60)
			default:
				InternalServerError(c, err)
			}
		},
	})

	// Add Redfish-compliant handler for 405 Method Not Allowed
	router.NoMethod(func(c *gin.Context) {
		MethodNotAllowedError(c)
	})
	log.Info("Redfish v1 routes protected setup complete")
}
