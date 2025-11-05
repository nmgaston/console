// Package v1 provides Redfish-compliant error handling utilities.
package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

const (
	// Registry message IDs.
	msgIDResourceInUse         = "Base.1.22.0.ResourceInUse"
	msgIDInsufficientPrivilege = "Base.1.22.0.InsufficientPrivilege"
	msgIDGeneralError          = "Base.1.22.0.GeneralError"
	msgIDInternalError         = "Base.1.22.0.InternalError"

	// Common messages.
	msgInsufficientPrivileges     = "There are insufficient privileges for the account or credentials associated with the current session to perform the requested operation."
	msgUnauthorizedAccess         = "Unauthorized access"
	msgInsufficientPrivsOperation = "Insufficient privileges to perform operation"
	msgInternalServerError        = "An internal server error occurred."

	// Common resolutions.
	resolutionRemoveCondition    = "Remove the condition and resubmit the request if the operation failed."
	resolutionChangeAccessRights = "Either abandon the operation or change the associated access rights and resubmit the request if the operation failed."
)

// registryMgr is the global registry manager instance
var registryMgr = GetRegistryManager()

// mapSeverityToResourceHealth converts registry severity strings to generated.ResourceHealth enum
func mapSeverityToResourceHealth(severity string) string {
	switch severity {
	case "Critical":
		return string(generated.Critical)
	case "Warning":
		return string(generated.Warning)
	case "OK":
		return string(generated.OK)
	default:
		// Default to Warning for unknown severity levels
		return string(generated.Warning)
	}
}

// SetRedfishHeaders sets Redfish-compliant headers
func SetRedfishHeaders(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("OData-Version", "4.0")
}

// createErrorResponse creates a Redfish error response using registry lookup.
// Note: registryName currently always receives "Base", but the parameter is kept for
// future extensibility when additional registries (Task, Update, ResourceEvent) are added.
//
//nolint:unparam // registryName is kept for future registry extensibility
func createErrorResponse(registryName, messageKey string, args ...interface{}) (*generated.RedfishError, error) {
	regMsg, err := registryMgr.LookupMessage(registryName, messageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup message %s.%s: %w", registryName, messageKey, err)
	}

	messageStr := regMsg.FormatMessage(args...)
	messageID := regMsg.MessageID
	resolution := regMsg.Resolution
	severity := mapSeverityToResourceHealth(regMsg.Severity)

	errorResponse := &generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{regMsg.RegistryPrefix + "." + regMsg.RegistryVersion + ".GeneralError"}[0],
			Message: &messageStr,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	return errorResponse, nil
}

// ConflictError returns a Redfish-compliant 409 error
func ConflictError(c *gin.Context, resource, message string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceInUse")
	if err != nil {
		// Fallback to hardcoded error if registry lookup fails
		fallbackConflictError(c, resource, message)

		return
	}

	// Use custom error message if provided
	if message != "" {
		errorResponse.Error.Message = &message
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// fallbackConflictError is the fallback when registry lookup fails
func fallbackConflictError(c *gin.Context, resource, message string) {
	messageStr := resource + " is in a conflicting state."
	messageID := msgIDResourceInUse
	resolution := resolutionRemoveCondition
	severity := string(generated.Warning)

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &message,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// PowerStateConflictError returns a Redfish-compliant 409 error for power state conflicts
func PowerStateConflictError(c *gin.Context, _ string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceInUse")
	if err != nil {
		// Fallback to hardcoded error if registry lookup fails
		fallbackPowerStateConflictError(c)

		return
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// fallbackPowerStateConflictError is the fallback when registry lookup fails
func fallbackPowerStateConflictError(c *gin.Context) {
	messageStr := "The change to the requested resource failed because the resource is in use or in transition."
	messageID := msgIDResourceInUse
	resolution := resolutionRemoveCondition
	severity := string(generated.Warning)
	errorMessage := "Power state transition not allowed"

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// MethodNotAllowedError returns a Redfish-compliant 405 error
func MethodNotAllowedError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "MethodNotAllowed")
	if err != nil {
		// Fallback to hardcoded error
		fallbackMethodNotAllowedError(c)

		return
	}

	c.JSON(http.StatusMethodNotAllowed, errorResponse)
}

// fallbackMethodNotAllowedError is the fallback when registry lookup fails
func fallbackMethodNotAllowedError(c *gin.Context) {
	messageStr := "The HTTP method is not allowed on this resource."
	messageID := "Base.1.22.0.MethodNotAllowed"
	resolution := "None"
	severity := string(generated.Critical)
	errorMessage := "The HTTP method is not allowed for the requested resource."

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusMethodNotAllowed, errorResponse)
}

// UnauthorizedError returns a Redfish-compliant 401 error
func UnauthorizedError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "InsufficientPrivilege")
	if err != nil {
		// Fallback to hardcoded error
		fallbackUnauthorizedError(c)

		return
	}

	// Override with unauthorized-specific message
	errorMessage := msgUnauthorizedAccess
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusUnauthorized, errorResponse)
}

// fallbackUnauthorizedError is the fallback when registry lookup fails
func fallbackUnauthorizedError(c *gin.Context) {
	messageStr := msgInsufficientPrivileges
	messageID := msgIDInsufficientPrivilege
	resolution := resolutionChangeAccessRights
	severity := string(generated.Critical)
	errorMessage := msgUnauthorizedAccess

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusUnauthorized, errorResponse)
}

// BadRequestError returns a Redfish-compliant 400 error
func BadRequestError(c *gin.Context, message, messageID, resolution, severity string) {
	SetRedfishHeaders(c)

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &message,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &message,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// ForbiddenError returns a Redfish-compliant 403 error for insufficient privileges
func ForbiddenError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "InsufficientPrivilege")
	if err != nil {
		// Fallback to hardcoded error
		fallbackForbiddenError(c)

		return
	}

	// Override with forbidden-specific message
	errorMessage := msgInsufficientPrivsOperation
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusForbidden, errorResponse)
}

// fallbackForbiddenError is the fallback when registry lookup fails
func fallbackForbiddenError(c *gin.Context) {
	messageStr := msgInsufficientPrivileges
	messageID := msgIDInsufficientPrivilege
	resolution := resolutionChangeAccessRights
	severity := string(generated.Critical)
	errorMessage := msgInsufficientPrivsOperation

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusForbidden, errorResponse)
}

// ServiceUnavailableError returns a Redfish-compliant 503 error
func ServiceUnavailableError(c *gin.Context, retryAfterSeconds int) {
	SetRedfishHeaders(c)

	if retryAfterSeconds > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	}

	errorResponse, err := createErrorResponse("Base", "ServiceTemporarilyUnavailable", fmt.Sprintf("%d", retryAfterSeconds))
	if err != nil {
		// Fallback to hardcoded error
		fallbackServiceUnavailableError(c, retryAfterSeconds)

		return
	}

	c.JSON(http.StatusServiceUnavailable, errorResponse)
}

// fallbackServiceUnavailableError is the fallback when registry lookup fails
func fallbackServiceUnavailableError(c *gin.Context, retryAfterSeconds int) {
	messageStr := fmt.Sprintf("The service is temporarily unavailable.  Retry in %d seconds.", retryAfterSeconds)
	messageID := "Base.1.22.0.ServiceTemporarilyUnavailable"
	resolution := "Wait for the indicated retry duration and retry the operation."
	severity := string(generated.Critical)
	errorMessage := "Service temporarily unavailable"

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusServiceUnavailable, errorResponse)
}

// NotFoundError returns a Redfish-compliant 404 error
func NotFoundError(c *gin.Context, resource string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceMissing", resource, resource)
	if err != nil {
		// Fallback to hardcoded error
		fallbackNotFoundError(c, resource)

		return
	}

	// Override with custom message
	errorMessage := resource + " not found"
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusNotFound, errorResponse)
}

// fallbackNotFoundError is the fallback when registry lookup fails
func fallbackNotFoundError(c *gin.Context, resource string) {
	messageStr := "The requested resource does not exist."
	messageID := "Base.1.22.0.ResourceMissing"
	resolution := "Provide a valid resource identifier and resubmit the request."
	severity := string(generated.Critical)
	errorMessage := resource + " not found"

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusNotFound, errorResponse)
}

// InternalServerError returns a Redfish-compliant 500 error
func InternalServerError(c *gin.Context, err error) {
	SetRedfishHeaders(c)

	errorResponse, regErr := createErrorResponse("Base", "InternalError")
	if regErr != nil {
		// Fallback to hardcoded error
		fallbackInternalServerError(c, err)

		return
	}

	// Override message with actual error
	errMsg := err.Error()
	errorMessage := msgInternalServerError
	errorResponse.Error.Message = &errorMessage
	(*errorResponse.Error.MessageExtendedInfo)[0].Message = &errMsg

	c.JSON(http.StatusInternalServerError, errorResponse)
}

// fallbackInternalServerError is the fallback when registry lookup fails
func fallbackInternalServerError(c *gin.Context, err error) {
	errMsg := err.Error()
	messageID := msgIDInternalError
	resolution := "Resubmit the request.  If the problem persists, consider resetting the service."
	severity := string(generated.Critical)
	errorMessage := msgInternalServerError

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &errMsg,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusInternalServerError, errorResponse)
}

// MalformedJSONError returns a Redfish-compliant 400 error for malformed JSON
func MalformedJSONError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "MalformedJSON")
	if err != nil {
		// Fallback to hardcoded error
		fallbackMalformedJSONError(c)

		return
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// fallbackMalformedJSONError is the fallback when registry lookup fails
func fallbackMalformedJSONError(c *gin.Context) {
	messageStr := "The request body submitted was malformed JSON and could not be parsed by the receiving service."
	messageID := "Base.1.22.0.MalformedJSON"
	resolution := "Ensure that the request body is valid JSON and resubmit the request."
	severity := string(generated.Critical)
	errorMessage := "Malformed JSON in request body"

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// PropertyMissingError returns a Redfish-compliant 400 error for missing required property
func PropertyMissingError(c *gin.Context, propertyName string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "PropertyMissing", propertyName)
	if err != nil {
		// Fallback to hardcoded error
		fallbackPropertyMissingError(c, propertyName)

		return
	}

	// Override with custom message
	errorMessage := "Missing or empty " + propertyName
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusBadRequest, errorResponse)
}

// fallbackPropertyMissingError is the fallback when registry lookup fails
func fallbackPropertyMissingError(c *gin.Context, propertyName string) {
	messageStr := "The property " + propertyName + " is a required property and must be included in the request."
	messageID := "Base.1.22.0.PropertyMissing"
	resolution := "Ensure that the property is in the request body and has a valid value and resubmit the request if the operation failed."
	severity := string(generated.Warning)
	errorMessage := "Missing or empty " + propertyName

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// PropertyValueNotInListError returns a Redfish-compliant 400 error for invalid property value
func PropertyValueNotInListError(c *gin.Context, propertyName string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "PropertyValueNotInList", "invalid", propertyName)
	if err != nil {
		// Fallback to hardcoded error
		fallbackPropertyValueNotInListError(c, propertyName)

		return
	}

	// Override with custom message
	errorMessage := "Invalid " + propertyName
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusBadRequest, errorResponse)
}

// fallbackPropertyValueNotInListError is the fallback when registry lookup fails
func fallbackPropertyValueNotInListError(c *gin.Context, propertyName string) {
	messageStr := "The value provided for " + propertyName + " is not in the list of acceptable values."
	messageID := "Base.1.22.0.PropertyValueNotInList"
	resolution := "Choose a value from the enumeration list that the implementation can support and resubmit the request if the operation failed."
	severity := string(generated.Warning)
	errorMessage := "Invalid " + propertyName

	errorResponse := generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.22.0.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}
