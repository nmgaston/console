// Package v1 provides Redfish-compliant error handling utilities.
package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

const (
	// HTTP header constants for Redfish responses
	headerODataVersion = "OData-Version"
	headerContentType  = "Content-Type"
	headerLocation     = "Location"
	headerRetryAfter   = "Retry-After"

	// Header values
	contentTypeJSON = "application/json; charset=utf-8"
	contentTypeXML  = "application/xml"
	odataVersion    = "4.0"

	// Common error messages
	msgInternalServerError = "An internal server error occurred."
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
	c.Header(headerContentType, contentTypeJSON)
	c.Header(headerODataVersion, odataVersion)
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
func ConflictError(c *gin.Context, _, message string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceInUse")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Use custom error message if provided
	if message != "" {
		errorResponse.Error.Message = &message
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// PowerStateConflictError returns a Redfish-compliant 409 error for power state conflicts
func PowerStateConflictError(c *gin.Context, _ string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceInUse")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	c.JSON(http.StatusConflict, errorResponse)
}

// MethodNotAllowedError returns a Redfish-compliant 405 error
func MethodNotAllowedError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "MethodNotAllowed")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	c.JSON(http.StatusMethodNotAllowed, errorResponse)
}

// UnauthorizedError returns a Redfish-compliant 401 error
func UnauthorizedError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "InsufficientPrivilege")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with unauthorized-specific message
	errorMessage := "Unauthorized access"
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusUnauthorized, errorResponse)
}

// BadRequestError returns a Redfish-compliant 400 error
func BadRequestError(c *gin.Context, customMessage string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "GeneralError")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with custom message if provided
	if customMessage != "" {
		errorResponse.Error.Message = &customMessage
		(*errorResponse.Error.MessageExtendedInfo)[0].Message = &customMessage
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// ForbiddenError returns a Redfish-compliant 403 error for insufficient privileges
func ForbiddenError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "InsufficientPrivilege")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with forbidden-specific message
	errorMessage := "Insufficient privileges to perform operation"
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusForbidden, errorResponse)
}

// ServiceUnavailableError returns a Redfish-compliant 503 error
func ServiceUnavailableError(c *gin.Context, retryAfterSeconds int) {
	SetRedfishHeaders(c)

	if retryAfterSeconds > 0 {
		c.Header(headerRetryAfter, fmt.Sprintf("%d", retryAfterSeconds))
	}

	errorResponse, err := createErrorResponse("Base", "ServiceTemporarilyUnavailable", fmt.Sprintf("%d", retryAfterSeconds))
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	c.JSON(http.StatusServiceUnavailable, errorResponse)
}

// NotFoundError returns a Redfish-compliant 404 error
func NotFoundError(c *gin.Context, resource string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "ResourceMissing", resource, resource)
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with custom message
	errorMessage := resource + " not found"
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusNotFound, errorResponse)
}

// InternalServerError returns a Redfish-compliant 500 error
func InternalServerError(c *gin.Context, err error) {
	SetRedfishHeaders(c)

	errorResponse, regErr := createErrorResponse("Base", "InternalError")
	if regErr != nil {
		// Ultimate fallback - if even the registry lookup fails, return a minimal error
		errorMessage := msgInternalServerError
		errMsg := err.Error()
		c.JSON(http.StatusInternalServerError, generated.RedfishError{
			Error: struct {
				MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
				Code                *string                     `json:"code,omitempty"`
				Message             *string                     `json:"message,omitempty"`
			}{
				Code:    &[]string{"Base.1.22.0.InternalError"}[0],
				Message: &errorMessage,
				MessageExtendedInfo: &[]generated.MessageMessage{
					{
						MessageId: &[]string{"Base.1.22.0.InternalError"}[0],
						Message:   &errMsg,
						Severity:  &[]string{string(generated.Critical)}[0],
					},
				},
			},
		})

		return
	}

	// Override message with actual error
	errMsg := err.Error()
	errorMessage := msgInternalServerError
	errorResponse.Error.Message = &errorMessage
	(*errorResponse.Error.MessageExtendedInfo)[0].Message = &errMsg

	c.JSON(http.StatusInternalServerError, errorResponse)
}

// MalformedJSONError returns a Redfish-compliant 400 error for malformed JSON
func MalformedJSONError(c *gin.Context) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "MalformedJSON")
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// PropertyMissingError returns a Redfish-compliant 400 error for missing required property
func PropertyMissingError(c *gin.Context, propertyName string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "PropertyMissing", propertyName)
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with custom message
	errorMessage := "Missing or empty " + propertyName
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusBadRequest, errorResponse)
}

// PropertyValueNotInListError returns a Redfish-compliant 400 error for invalid property value
func PropertyValueNotInListError(c *gin.Context, propertyName string) {
	SetRedfishHeaders(c)

	errorResponse, err := createErrorResponse("Base", "PropertyValueNotInList", "invalid", propertyName)
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	// Override with custom message
	errorMessage := "Invalid " + propertyName
	errorResponse.Error.Message = &errorMessage

	c.JSON(http.StatusBadRequest, errorResponse)
}
