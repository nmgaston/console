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
	if err := s.applyBootSettings(c, computerSystemID, &req); err != nil {
		return
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

// applyBootSettings processes boot override parameters from reset request and applies them to the system.
// Returns error if boot settings cannot be applied, nil otherwise.
//
//nolint:gocognit // Complexity is inherent to boot parameter conversion logic
func (s *RedfishServer) applyBootSettings(
	c *gin.Context,
	computerSystemID string,
	req *generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody,
) error {
	if req.BootSourceOverrideTarget == nil && req.BootSourceOverrideEnabled == nil && req.BootSourceOverrideMode == nil {
		return nil
	}

	boot := &generated.ComputerSystemBoot{}

	if err := s.convertBootTarget(req, boot); err != nil {
		return err
	}

	if err := s.convertBootEnabled(req, boot); err != nil {
		return err
	}

	if err := s.convertBootMode(req, boot); err != nil {
		return err
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

		return err
	}

	log.Infof("Boot settings updated for ComputerSystem %s", computerSystemID)

	return nil
}

func (s *RedfishServer) convertBootTarget(
	req *generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody,
	boot *generated.ComputerSystemBoot,
) error {
	if req.BootSourceOverrideTarget == nil {
		return nil
	}

	targetVal, err := req.BootSourceOverrideTarget.AsComputerSystemBootSourceOverrideTarget()
	if err != nil {
		log.Warnf("Failed to convert boot target: %v", err)
		return nil
	}

	bootTarget := &generated.ComputerSystemBoot_BootSourceOverrideTarget{}
	if err := bootTarget.FromComputerSystemBootSourceOverrideTarget(targetVal); err != nil {
		log.Warnf("Failed to set boot target: %v", err)
		return nil
	}

	boot.BootSourceOverrideTarget = bootTarget

	return nil
}

func (s *RedfishServer) convertBootEnabled(
	req *generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody,
	boot *generated.ComputerSystemBoot,
) error {
	if req.BootSourceOverrideEnabled == nil {
		return nil
	}

	enabledVal, err := req.BootSourceOverrideEnabled.AsComputerSystemBootSourceOverrideEnabled()
	if err != nil {
		log.Warnf("Failed to convert boot enabled: %v", err)
		return nil
	}

	bootEnabled := &generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
	if err := bootEnabled.FromComputerSystemBootSourceOverrideEnabled(enabledVal); err != nil {
		log.Warnf("Failed to set boot enabled: %v", err)
		return nil
	}

	boot.BootSourceOverrideEnabled = bootEnabled

	return nil
}

func (s *RedfishServer) convertBootMode(
	req *generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody,
	boot *generated.ComputerSystemBoot,
) error {
	if req.BootSourceOverrideMode == nil {
		return nil
	}

	modeVal, err := req.BootSourceOverrideMode.AsComputerSystemBootSourceOverrideMode()
	if err != nil {
		log.Warnf("Failed to convert boot mode: %v", err)
		return nil
	}

	bootMode := &generated.ComputerSystemBoot_BootSourceOverrideMode{}
	if err := bootMode.FromComputerSystemBootSourceOverrideMode(modeVal); err != nil {
		log.Warnf("Failed to set boot mode: %v", err)
		return nil
	}

	boot.BootSourceOverrideMode = bootMode

	return nil
}
