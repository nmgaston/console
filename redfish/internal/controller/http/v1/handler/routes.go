// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// Task state constants from Redfish Task.v1_8_0 specification
	taskStateCompleted = "Completed"

	// Registry message IDs
	msgIDBaseSuccess      = "Base.1.22.0.Success"
	msgIDBaseGeneralError = "Base.1.22.0.GeneralError"
)

// RedfishServer implements the Redfish API handlers
// Add dependencies here if needed (e.g., usecase, presenter, etc.)
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
	Config           *dmtconfig.Config
	Logger           logger.Interface
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

// CreateDescription creates a Description from a string using ResourceDescription.
func CreateDescription(desc string) *generated.ComputerSystemCollectionComputerSystemCollection_Description {
	description := &generated.ComputerSystemCollectionComputerSystemCollection_Description{}
	if err := description.FromResourceDescription(desc); err != nil {
		return nil
	}

	return description
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
	// Get all system IDs from the repository
	systemIDs, err := s.ComputerSystemUC.GetAll(c.Request.Context())
	if err != nil {
		InternalServerError(c, err)

		return
	}

	// Convert system IDs to members array
	members := make([]generated.OdataV4IdRef, 0, len(systemIDs))
	for _, systemID := range systemIDs {
		if systemID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr("/redfish/v1/Systems/" + systemID),
			})
		}
	}

	collection := generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataId:           StringPtr("/redfish/v1/Systems"),
		OdataType:         StringPtr("#ComputerSystemCollection.ComputerSystemCollection"),
		Name:              "Computer System Collection",
		Description:       CreateDescription("Collection of Computer Systems"),
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
	system, err := s.ComputerSystemUC.GetComputerSystem(c.Request.Context(), computerSystemID)
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

	err := s.ComputerSystemUC.SetPowerState(c.Request.Context(), computerSystemID, *req.ResetType)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, computerSystemID)
		case errors.Is(err, usecase.ErrInvalidResetType):
			BadRequestError(c, fmt.Sprintf("Invalid reset type: %s", string(*req.ResetType)))
		case errors.Is(err, usecase.ErrPowerStateConflict):
			PowerStateConflictError(c, string(*req.ResetType))
		case errors.Is(err, usecase.ErrUnsupportedPowerState):
			BadRequestError(c, fmt.Sprintf("Unsupported power state: %s", string(*req.ResetType)))
		default:
			InternalServerError(c, err)
		}

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
