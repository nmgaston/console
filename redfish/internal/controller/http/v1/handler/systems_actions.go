// Package v1 provides Redfish v1 API handlers for system actions.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset handles reset action for a computer system.
// Validates system ID and reset type before executing power state change.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, computerSystemID string) {
	// Validate system ID to prevent injection attacks
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

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

	// Handle boot settings if provided
	if req.BootSourceOverrideTarget != nil || req.BootSourceOverrideEnabled != nil || req.BootSourceOverrideMode != nil {
		// Create boot object by converting the union types
		boot := &generated.ComputerSystemBoot{}
		
		if req.BootSourceOverrideTarget != nil {
			targetVal, err := req.BootSourceOverrideTarget.AsComputerSystemBootSourceOverrideTarget()
			if err == nil {
				bootTarget := &generated.ComputerSystemBoot_BootSourceOverrideTarget{}
				bootTarget.FromComputerSystemBootSourceOverrideTarget(targetVal)
				boot.BootSourceOverrideTarget = bootTarget
			} else {
				log.Warnf("Failed to convert boot target: %v", err)
			}
		}
		
		if req.BootSourceOverrideEnabled != nil {
			enabledVal, err := req.BootSourceOverrideEnabled.AsComputerSystemBootSourceOverrideEnabled()
			if err == nil {
				bootEnabled := &generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
				bootEnabled.FromComputerSystemBootSourceOverrideEnabled(enabledVal)
				boot.BootSourceOverrideEnabled = bootEnabled
			} else {
				log.Warnf("Failed to convert boot enabled: %v", err)
			}
		}
		
		if req.BootSourceOverrideMode != nil {
			modeVal, err := req.BootSourceOverrideMode.AsComputerSystemBootSourceOverrideMode()
			if err == nil {
				bootMode := &generated.ComputerSystemBoot_BootSourceOverrideMode{}
				bootMode.FromComputerSystemBootSourceOverrideMode(modeVal)
				boot.BootSourceOverrideMode = bootMode
			} else {
				log.Warnf("Failed to convert boot mode: %v", err)
			}
		}

		if err := s.ComputerSystemUC.UpdateBootSettings(c.Request.Context(), computerSystemID, boot); err != nil {
			switch {
			case errors.Is(err, usecase.ErrSystemNotFound):
				NotFoundError(c, "System", computerSystemID)
			case errors.Is(err, usecase.ErrInvalidBootSettings):
				BadRequestError(c, fmt.Sprintf("Invalid boot settings: %s", err.Error()))
			default:
				InternalServerError(c, err)
			}

			return
		}

		log.Infof("Boot settings updated for ComputerSystem %s", computerSystemID)
	}

	if err := s.ComputerSystemUC.SetPowerState(c.Request.Context(), computerSystemID, *req.ResetType); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
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
		"@odata.context": odataContextTask,
		"@odata.id":      taskServiceTasks + taskID,
		"@odata.type":    odataTypeTask,
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   successMsg.Message,
				"MessageId": msgIDBaseSuccess,
				"Severity":  string(generated.OK),
			},
		},
		"Name":       taskName,
		"StartTime":  now,
		"TaskState":  taskStateCompleted,
		"TaskStatus": string(generated.OK),
	}
	c.Header(headerLocation, taskServiceTasks+taskID)
	c.JSON(http.StatusAccepted, task)
}
