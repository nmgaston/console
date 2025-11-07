// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// Task state constants from Redfish Task.v1_8_0 specification
	taskStateCompleted = "Completed"

	// Error message constants for power state error handling
	errMsgSystemNotFound      = "system not found"
	errMsgNotSupported        = "Not Supported -  - : "
	errMsgConnectionRefused   = "connection refused"
	errMsgConnectionTimeout   = "connection timeout"
	errMsgServiceUnavailable  = "service unavailable"
	errMsgDeviceNotResponding = "device not responding"

	// Registry message IDs
	msgIDBaseSuccess      = "Base.1.22.0.Success"
	msgIDBaseGeneralError = "Base.1.22.0.GeneralError"
)

// RedfishServer implements the Redfish API handlers
// Add dependencies here if needed (e.g., usecase, presenter, etc.)
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
}

// Ensure RedfishServer implements generated.ServerInterface
var _ generated.ServerInterface = (*RedfishServer)(nil)

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
func ChassisTypePtr(ct generated.ChassisChassisType) *generated.ChassisChassisType {
	return &ct
}

// ManagerTypePtr creates a pointer to a ManagerManagerType value.
func ManagerTypePtr(mt generated.ManagerManagerType) *generated.ManagerManagerType {
	return &mt
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// GetRedfishV1 returns the service root
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	serviceRoot := generated.ServiceRootServiceRoot{
		OdataContext:   StringPtr("/redfish/v1/$metadata#ServiceRoot.ServiceRoot"),
		OdataId:        StringPtr("/redfish/v1"),
		OdataType:      StringPtr("#ServiceRoot.v1_19_0.ServiceRoot"),
		Id:             "RootService",
		Name:           "Root Service",
		RedfishVersion: StringPtr("1.19.0"),
		Systems: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Systems"),
		},
		Chassis: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Chassis"),
		},
		Managers: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Managers"),
		},
	}
	c.JSON(http.StatusOK, serviceRoot)
}

// GetRedfishV1Metadata returns the OData metadata
func (s *RedfishServer) GetRedfishV1Metadata(c *gin.Context) {
	metadata := ""

	c.Header(headerContentType, contentTypeXML)
	c.String(http.StatusOK, metadata)
}

// GetRedfishV1Systems returns the computer systems collection
func (s *RedfishServer) GetRedfishV1Systems(c *gin.Context) {
	// Get all systems from the repository
	systems, err := s.ComputerSystemUC.GetAll()
	if err != nil {
		InternalServerError(c, err)
		return
	}

	// Convert systems to members array
	members := make([]generated.OdataV4IdRef, 0, len(systems))
	for _, system := range systems {
		if system.ID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr("/redfish/v1/Systems/" + system.ID),
			})
		}
	}

	collection := generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataId:           StringPtr("/redfish/v1/Systems"),
		OdataType:         StringPtr("#ComputerSystemCollection.ComputerSystemCollection"),
		Name:              "Computer System Collection",
		Description:       nil,
		MembersOdataCount: Int64Ptr(int64(len(members))),
		Members:           &members,
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1SystemsComputerSystemId returns a specific computer system
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	// Get the computer system from the use case
	system, err := s.ComputerSystemUC.GetComputerSystem(computerSystemID)
	if err != nil {
		if errors.Is(err, usecase.ErrSystemNotFound) {
			NotFoundError(c, "System")
			return
		}
		InternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, system)
}

// GetRedfishV1Chassis returns the chassis collection
func (s *RedfishServer) GetRedfishV1Chassis(c *gin.Context) {
	collection := generated.ChassisCollectionChassisCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ChassisCollection.ChassisCollection"),
		OdataId:           StringPtr("/redfish/v1/Chassis"),
		OdataType:         StringPtr("#ChassisCollection.ChassisCollection"),
		Name:              "Chassis Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]generated.OdataV4IdRef{
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

	chassis := generated.ChassisChassis{
		OdataContext: StringPtr("/redfish/v1/$metadata#Chassis.Chassis"),
		OdataId:      StringPtr("/redfish/v1/Chassis/Chassis1"),
		OdataType:    StringPtr("#Chassis.v1_28_0.Chassis"),
		Id:           "Chassis1",
		Name:         "Computer System Chassis",
		SerialNumber: StringPtr("CH123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example Chassis"),
		ChassisType:  generated.RackMount,
	}
	c.JSON(http.StatusOK, chassis)
}

// GetRedfishV1Managers returns the managers collection
func (s *RedfishServer) GetRedfishV1Managers(c *gin.Context) {
	collection := generated.ManagerCollectionManagerCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ManagerCollection.ManagerCollection"),
		OdataId:           StringPtr("/redfish/v1/Managers"),
		OdataType:         StringPtr("#ManagerCollection.ManagerCollection"),
		Name:              "Manager Collection",
		MembersOdataCount: Int64Ptr(1),
		Members: &[]generated.OdataV4IdRef{
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

	manager := generated.ManagerManager{
		OdataContext: StringPtr("/redfish/v1/$metadata#Manager.Manager"),
		OdataId:      StringPtr("/redfish/v1/Managers/Manager1"),
		OdataType:    StringPtr("#Manager.v1_21_0.Manager"),
		Id:           "Manager1",
		Name:         "System Manager",
		Model:        StringPtr("Example Manager"),
		ManagerType:  ManagerTypePtr(generated.BMC),
	}
	c.JSON(http.StatusOK, manager)
}

// handlePowerStateError handles errors from power state operations
func handlePowerStateError(c *gin.Context, err error, resetType string) {
	if errors.Is(err, usecase.ErrInvalidPowerState) || errors.Is(err, usecase.ErrInvalidResetType) {
		PropertyValueNotInListError(c, "ResetType")

		return
	}

	if errors.Is(err, usecase.ErrPowerStateConflict) {
		PowerStateConflictError(c, resetType)

		return
	}

	// Robust backend not found error
	if err.Error() == errMsgSystemNotFound {
		NotFoundError(c, "System")

		return
	}

	// Check for "Not Supported" error from the devices use case
	// This occurs when the power action functionality is not yet implemented
	errMsg := err.Error()
	if errMsg == errMsgNotSupported {
		handleNotSupportedError(c)

		return
	}

	// Check for service unavailability errors (503)
	// This catches cases where the backend service (WSMAN/AMT) is unreachable
	if errMsg == errMsgConnectionRefused || errMsg == errMsgConnectionTimeout ||
		errMsg == errMsgServiceUnavailable || errMsg == errMsgDeviceNotResponding {
		ServiceUnavailableError(c, redfishv1.ServiceUnavailableRetryAfterSeconds)

		return
	}

	InternalServerError(c, err)
}

// handleNotSupportedError returns a 501 Not Implemented error
func handleNotSupportedError(c *gin.Context) {
	SetRedfishHeaders(c)

	messageStr := "The power action operation is not yet supported by this service."
	messageID := msgIDBaseGeneralError
	resolution := "This feature is under development and will be available in a future release."
	severity := string(generated.Critical)

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					Message:    &messageStr,
					MessageId:  &messageID,
					Resolution: &resolution,
					Severity:   &severity,
				},
			},
			Code:    &messageID,
			Message: &messageStr,
		},
	}
	c.JSON(http.StatusNotImplemented, errorResponse)
}

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset handles the reset action for a computer system
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, computerSystemID string) {
	var req generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)

		return
	}

	if req.ResetType == nil || *req.ResetType == "" {
		PropertyMissingError(c, "ResetType")

		return
	}

	log.Infof("Received reset request for ComputerSystem %s with ResetType %s", computerSystemID, *req.ResetType)

	//nolint:godox // TODO comment is intentional - provides implementation guidance
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

	err := s.ComputerSystemUC.SetPowerState(computerSystemID, *req.ResetType)
	if err != nil {
		handlePowerStateError(c, err, string(*req.ResetType))

		return
	}

	// Generate dynamic Task response
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	// Get success message from registry
	successMsg, err := registryMgr.LookupMessage("Base", "Success")
	if err != nil {
		// Fallback if registry lookup fails
		InternalServerError(c, err)

		return
	}

	task := map[string]interface{}{
		"@odata.context": "/redfish/v1/$metadata#Task.Task",
		"@odata.id":      "/redfish/v1/TaskService/Tasks/" + taskID,
		"@odata.type":    "#Task.v1_6_0.Task",
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   successMsg.Message,
				"MessageId": msgIDBaseSuccess,
				"Severity":  string(generated.OK),
			},
		},
		"Name":       "System Reset Task",
		"StartTime":  now,
		"TaskState":  taskStateCompleted,
		"TaskStatus": string(generated.OK),
	}
	c.Header(headerLocation, "/redfish/v1/TaskService/Tasks/"+taskID)
	c.JSON(http.StatusAccepted, task)
}
