// Package v1 provides Redfish-compliant error handling utilities.
package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/http/redfish/v1/redfishapi"
)

// SetRedfishHeaders sets Redfish-compliant headers
func SetRedfishHeaders(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("OData-Version", "4.0")
}

// ConflictError returns a Redfish-compliant 409 error
func ConflictError(c *gin.Context, resource, message string) {
	SetRedfishHeaders(c)

	messageStr := resource + " is in a conflicting state."
	messageID := "Base.1.11.0.ResourceInUse"
	resolution := "Change the resource state or try again later."
	severity := string(redfishapi.Critical)

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &message,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The requested power state transition is not allowed from the current state."
	messageID := "Base.1.11.0.ResourceCannotBeDeleted" // Using closest available message or use custom
	resolution := "Verify the current power state and request a valid state transition."
	severity := string(redfishapi.Critical)
	errorMessage := "Power state transition not allowed"

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The HTTP method used is not allowed for this resource."
	messageID := "Base.1.11.0.MethodNotAllowed"
	resolution := "Use a supported HTTP method for this endpoint."
	severity := string(redfishapi.Critical)
	errorMessage := "The HTTP method is not allowed for the requested resource."

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The request was denied due to insufficient privilege."
	messageID := "Base.1.11.0.InsufficientPrivilege"
	resolution := "Ensure the request includes valid authentication credentials."
	severity := string(redfishapi.Critical)
	errorMessage := "Unauthorized access"

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &message,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The account or credentials associated with the current session do not have sufficient privileges for the operation."
	messageID := "Base.1.11.0.InsufficientPrivilege"
	resolution := "Either abandon the operation or change the associated access rights and resubmit the request."
	severity := string(redfishapi.Critical)
	errorMessage := "Insufficient privileges to perform operation"

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The service is temporarily unavailable. Retry the request."
	messageID := "Base.1.11.0.ServiceTemporarilyUnavailable"
	resolution := "Wait for the indicated retry time and retry the operation."
	severity := string(redfishapi.Critical)
	errorMessage := "Service temporarily unavailable"

	if retryAfterSeconds > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	}

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The requested resource does not exist."
	messageID := "Base.1.11.0.ResourceMissing"
	resolution := "Verify the resource ID and try again."
	severity := string(redfishapi.Critical)
	errorMessage := resource + " not found"

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	errMsg := err.Error()
	messageID := "Base.1.11.0.InternalError"
	resolution := "Retry the operation or contact your administrator."
	severity := string(redfishapi.Critical)
	errorMessage := "An internal server error occurred."

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The request body contained malformed JSON."
	messageID := "Base.1.11.0.MalformedJSON"
	resolution := "Correct the request body to be valid JSON."
	severity := string(redfishapi.Critical)
	errorMessage := "Malformed JSON in request body"

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The property " + propertyName + " is missing or empty."
	messageID := "Base.1.11.0.PropertyMissing"
	resolution := "Supply a valid " + propertyName + " property in the request body."
	severity := string(redfishapi.Critical)
	errorMessage := "Missing or empty " + propertyName

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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

	messageStr := "The value provided for " + propertyName + " is not in the list of acceptable values."
	messageID := "Base.1.11.0.PropertyValueNotInList"
	resolution := "Supply a valid " + propertyName + " property value."
	severity := string(redfishapi.Critical)
	errorMessage := "Invalid " + propertyName

	errorResponse := redfishapi.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]redfishapi.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                      `json:"code,omitempty"`
			Message             *string                      `json:"message,omitempty"`
		}{
			Code:    &[]string{"Base.1.11.GeneralError"}[0],
			Message: &errorMessage,
			MessageExtendedInfo: &[]redfishapi.MessageMessage{
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
