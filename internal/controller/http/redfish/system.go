package redfish

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Lint constants
const (
	maxSystemsList        = 100
	powerStateUnknown     = "Unknown"
	powerStateOn          = "On"
	powerStateOff         = "Off"
	resetTypeOn           = "On"
	resetTypeForceOff     = "ForceOff"
	resetTypeForceRestart = "ForceRestart"
	resetTypePowerCycle   = "PowerCycle"
	actionPowerUp         = 2
	actionPowerCycle      = 5
	actionPowerDown       = 8
	actionReset           = 10
	// CIM PowerState enum values (Device.PowerState)
	cimPowerOn      = 2
	cimPowerSleep   = 3
	cimPowerStandby = 4
	cimPowerSoftOff = 7
	cimPowerHardOff = 8

	// Redfish Base Message Registry v1.0.0 Message IDs
	baseSuccessMessageID = "Base.1.0.Success"
	baseErrorMessageID   = "Base.1.0.GeneralError"
)

// NewSystemsRoutes registers minimal Redfish ComputerSystem routes.
// It exposes:
// - GET /redfish/v1/Systems
// - GET /redfish/v1/Systems/:id
// - POST /redfish/v1/Systems/:id/Actions/ComputerSystem.Reset
// The :id is expected to be the device GUID and will be mapped directly to SendPowerAction.
func NewSystemsRoutes(r *gin.RouterGroup, d devices.Feature, l logger.Interface) {
	systems := r.Group("/Systems")
	systems.GET("", getSystemsCollectionHandler(d, l))
	systems.GET(":id", getSystemInstanceHandler(d, l))
	systems.POST(":id/Actions/ComputerSystem.Reset", postSystemResetHandler(d, l))
	l.Info("Registered Redfish Systems routes under %s", r.BasePath()+"/Systems")
}

func getSystemsCollectionHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := d.Get(c.Request.Context(), maxSystemsList, 0, "")
		if err != nil {
			l.Error(err, "http - redfish - Systems collection")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		members := make([]any, 0, len(items))
		for i := range items { // avoid value copy
			it := &items[i]
			if it.GUID == "" {
				continue
			}

			members = append(members, map[string]any{
				"@odata.id": "/redfish/v1/Systems/" + it.GUID,
			})
		}

		payload := map[string]any{
			"@odata.type":         "#ComputerSystemCollection.ComputerSystemCollection",
			"@odata.id":           "/redfish/v1/Systems",
			"Name":                "Computer System Collection",
			"Members@odata.count": len(members),
			"Members":             members,
		}
		c.JSON(http.StatusOK, payload)
	}
}

func getSystemInstanceHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		powerState := powerStateUnknown

		if ps, err := d.GetPowerState(c.Request.Context(), id); err != nil {
			l.Warn("redfish - Systems instance: failed to get power state for %s: %v", id, err)
		} else {
			switch ps.PowerState { // CIM PowerState values
			case actionPowerUp: // 2 (On)
				powerState = powerStateOn
			case cimPowerSleep, cimPowerStandby: // Sleep/Standby -> treat as On
				powerState = powerStateOn
			case cimPowerSoftOff, cimPowerHardOff: // Soft Off / Hard Off
				powerState = powerStateOff
			default:
				powerState = powerStateUnknown
			}
		}

		payload := map[string]any{
			"@odata.type": "#ComputerSystem.v1_0_0.ComputerSystem",
			"@odata.id":   "/redfish/v1/Systems/" + id,
			"Id":          id,
			"Name":        "Computer System " + id,
			"PowerState":  powerState,
			"Actions": map[string]any{
				"#ComputerSystem.Reset": map[string]any{
					"target":                            "/redfish/v1/Systems/" + id + "/Actions/ComputerSystem.Reset",
					"ResetType@Redfish.AllowableValues": []string{resetTypeOn, resetTypeForceOff, resetTypeForceRestart, resetTypePowerCycle},
				},
			},
		}
		c.JSON(http.StatusOK, payload)
	}
}

func postSystemResetHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var body struct {
			ResetType string `json:"ResetType"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			// Return Redfish-compliant error for malformed JSON
			errorResponse := map[string]any{
				"error": map[string]any{
					"@Message.ExtendedInfo": []map[string]any{
						{
							"MessageId":  "Base.1.0.MalformedJSON",
							"Message":    "The request body submitted was malformed JSON and could not be parsed by the receiving service.",
							"Severity":   "Critical",
							"Resolution": "Ensure that the request body is valid JSON and resubmit the request.",
						},
					},
					"code":    "Base.1.0.MalformedJSON",
					"message": "Malformed JSON in request body: " + err.Error(),
				},
			}
			// Set Redfish-compliant headers
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("OData-Version", "4.0")
			c.Header("Cache-Control", "no-cache")

			c.JSON(http.StatusBadRequest, errorResponse)

			return
		}

		var action int

		switch body.ResetType {
		case resetTypeOn:
			action = actionPowerUp
		case resetTypeForceOff:
			action = actionPowerDown
		case resetTypeForceRestart:
			action = actionReset
		case resetTypePowerCycle:
			action = actionPowerCycle
		default:
			// Return Redfish-compliant error for unsupported ResetType
			errorResponse := map[string]any{
				"error": map[string]any{
					"@Message.ExtendedInfo": []map[string]any{
						{
							"MessageId":  "Base.1.0.ActionParameterNotSupported",
							"Message":    "The action parameter " + body.ResetType + " is not supported on the target resource.",
							"Severity":   "Warning",
							"Resolution": "Remove the parameter from the request body and resubmit the request.",
						},
					},
					"code":    "Base.1.0.ActionParameterNotSupported",
					"message": "The parameter ResetType with value '" + body.ResetType + "' is not supported.",
				},
			}
			// Set Redfish-compliant headers
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("OData-Version", "4.0")
			c.Header("Cache-Control", "no-cache")

			c.JSON(http.StatusBadRequest, errorResponse)

			return
		}

		res, err := d.SendPowerAction(c.Request.Context(), id, action)
		if err != nil {
			l.Error(err, "http - redfish - ComputerSystem.Reset")

			// Check if this is a "not found" error
			var statusCode int
			var messageID, message, severity, resolution string

			if strings.Contains(strings.ToLower(err.Error()), "not found") ||
				strings.Contains(err.Error(), "DevicesUseCase") {
				// Device not found - return 404
				statusCode = http.StatusNotFound
				messageID = "Base.1.0.ResourceNotFound"
				message = "The requested resource of type ComputerSystem named '" + id + "' was not found."
				severity = "Critical"
				resolution = "Provide a valid resource identifier and resubmit the request."
			} else {
				// General error - return 500
				statusCode = http.StatusInternalServerError
				messageID = "Base.1.0.GeneralError"
				message = "A general error has occurred. See ExtendedInfo for more information."
				severity = "Critical"
				resolution = "None."
			}

			// Return Redfish-compliant error response
			errorResponse := map[string]any{
				"error": map[string]any{
					"@Message.ExtendedInfo": []map[string]any{
						{
							"MessageId":  messageID,
							"Message":    message,
							"Severity":   severity,
							"Resolution": resolution,
						},
					},
					"code":    messageID,
					"message": message,
				},
			}
			// Set Redfish-compliant headers
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("OData-Version", "4.0")
			c.Header("Cache-Control", "no-cache")

			c.JSON(statusCode, errorResponse)

			return
		}

		// Generate a task ID for this reset operation
		taskID := fmt.Sprintf("%d", rand.Intn(999999)+100000)

		// Determine task state based on the result
		taskState := "Completed"
		taskStatus := "OK"
		messageID := baseSuccessMessageID
		message := "The request completed successfully."

		// Check if the operation was successful based on ReturnValue
		if int(res.ReturnValue) != 0 {
			taskState = "Exception"
			taskStatus = "Critical"
			messageID = baseErrorMessageID
			message = "A general error has occurred."
		}

		// Return Redfish-compliant Task response
		taskResponse := map[string]any{
			"@odata.context": "/redfish/v1/$metadata#Task.Task",
			"@odata.id":      "/redfish/v1/TaskService/Tasks/" + taskID,
			"@odata.type":    "#Task.v1_6_0.Task",
			"Id":             taskID,
			"Name":           "System Reset Task",
			"TaskState":      taskState,
			"TaskStatus":     taskStatus,
			"StartTime":      time.Now().UTC().Format(time.RFC3339),
			"EndTime":        time.Now().UTC().Format(time.RFC3339),
			"Messages": []map[string]any{
				{
					"MessageId": messageID,
					"Message":   message,
					"Severity":  taskStatus,
				},
			},
		}

		// Set Redfish-compliant headers
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("OData-Version", "4.0")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.JSON(http.StatusOK, taskResponse)
	}
}
