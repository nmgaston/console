// Package v1 provides HTTP handlers for Redfish Computer Systems endpoints.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// Systems-specific OData metadata constants
	systemsOdataContextCollection = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	systemsOdataIDCollection      = "/redfish/v1/Systems"
	systemsOdataTypeCollection    = "#ComputerSystemCollection.ComputerSystemCollection"
	systemsCollectionTitle        = "Computer System Collection"
	systemsCollectionDescription  = "Collection of Computer Systems"

	// Systems path patterns
	systemsBasePath = "/redfish/v1/Systems/"

	// SystemID validation limits
	maxSystemIDLength = 50
	minSystemIDLength = 1
)

var (
	// validSystemIDPattern matches alphanumeric characters, hyphens, and underscores only.
	validSystemIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// SystemID validation errors
	errSystemIDEmpty             = errors.New("system ID cannot be empty")
	errSystemIDTooShort          = errors.New("system ID must be at least 1 character long")
	errSystemIDTooLong           = errors.New("system ID exceeds maximum length of 50 characters")
	errSystemIDInvalidUTF8       = errors.New("system ID contains invalid UTF-8 characters")
	errSystemIDNullByte          = errors.New("system ID contains null byte")
	errSystemIDPathCharacters    = errors.New("system ID contains invalid path characters")
	errSystemIDSpecialCharacters = errors.New("system ID contains invalid special characters")
	errSystemIDInvalidCharacters = errors.New("system ID contains invalid characters (only alphanumeric, hyphen, and underscore allowed)")
)

// validateSystemID validates system ID parameter for security and format compliance.
// Performs layered security validation: basic checks (empty, length, UTF-8) followed by
// security checks (null bytes, path traversal, special chars) and pattern enforcement.
// Length limit: 1-50 characters (accommodates GUIDs which are typically 36 characters).
//
// Example:
//
//	validateSystemID("test-system-1")           // returns nil (valid)
//	validateSystemID("<script>alert</script>")  // returns errSystemIDSpecialCharacters
//	validateSystemID("../etc/passwd")          // returns errSystemIDPathCharacters
func validateSystemID(systemID string) error {
	// Check for empty or whitespace-only ID
	if strings.TrimSpace(systemID) == "" {
		return errSystemIDEmpty
	}

	// Check minimum length
	if len(systemID) < minSystemIDLength {
		return errSystemIDTooShort
	}

	// Check maximum length
	if len(systemID) > maxSystemIDLength {
		return errSystemIDTooLong
	}

	// Check for valid UTF-8 encoding
	if !utf8.ValidString(systemID) {
		return errSystemIDInvalidUTF8
	}

	// Check for null bytes
	if strings.Contains(systemID, "\x00") {
		return errSystemIDNullByte
	}

	// Check for path traversal attempts
	if strings.Contains(systemID, "..") || strings.Contains(systemID, "/") || strings.Contains(systemID, "\\") {
		return errSystemIDPathCharacters
	}

	// Check for special characters that might indicate injection attempts
	if strings.ContainsAny(systemID, "<>\"';&|`$(){}[]") {
		return errSystemIDSpecialCharacters
	}

	// Check against valid pattern (alphanumeric, hyphen, underscore only)
	if !validSystemIDPattern.MatchString(systemID) {
		return errSystemIDInvalidCharacters
	}

	return nil
}

// CreateDescription creates a Description from a string using ResourceDescription.
// If an error occurs during description creation, it logs the error and returns nil.
// This allows the calling code to continue with a nil description while ensuring
// the error is captured for debugging purposes.
func CreateDescription(desc string, lgr logger.Interface) *generated.ComputerSystemCollectionComputerSystemCollection_Description {
	description := &generated.ComputerSystemCollectionComputerSystemCollection_Description{}
	if err := description.FromResourceDescription(desc); err != nil {
		if lgr != nil {
			lgr.Error("Failed to create description from resource description: %v, input: %s", err, desc)
		}

		return nil
	}

	return description
}

// transformToMembers converts system IDs to OData member references.
func (s *RedfishServer) transformToMembers(systemIDs []string) []generated.OdataV4IdRef {
	members := make([]generated.OdataV4IdRef, 0, len(systemIDs))
	for _, systemID := range systemIDs {
		if systemID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr(systemsBasePath + systemID),
			})
		}
	}

	return members
}

// buildSystemsCollectionResponse constructs the systems collection response.
func (s *RedfishServer) buildSystemsCollectionResponse(members []generated.OdataV4IdRef) generated.ComputerSystemCollectionComputerSystemCollection {
	return generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr(systemsOdataContextCollection),
		OdataId:           StringPtr(systemsOdataIDCollection),
		OdataType:         StringPtr(systemsOdataTypeCollection),
		Name:              systemsCollectionTitle,
		Description:       CreateDescription(systemsCollectionDescription, s.Logger),
		MembersOdataCount: Int64Ptr(int64(len(members))),
		Members:           &members,
	}
}

// handleGetSystemError handles errors from GetComputerSystem operations.
func (s *RedfishServer) handleGetSystemError(c *gin.Context, err error, systemID string) {
	switch {
	case errors.Is(err, usecase.ErrSystemNotFound):
		NotFoundError(c, "System", systemID)
	default:
		if s.Logger != nil {
			s.Logger.Error("Failed to retrieve computer system",
				"systemID", systemID,
				"error", err)
		}

		InternalServerError(c, err)
	}
}

// GetRedfishV1Systems handles GET requests for the systems collection
func (s *RedfishServer) GetRedfishV1Systems(c *gin.Context) {
	ctx := c.Request.Context()

	systemIDs, err := s.ComputerSystemUC.GetAll(ctx)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("Failed to retrieve computer systems collection", "error", err)
		}

		InternalServerError(c, err)

		return
	}

	members := s.transformToMembers(systemIDs)
	collection := s.buildSystemsCollectionResponse(members)

	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1SystemsComputerSystemId handles GET requests for individual computer systems.
// Validates system ID parameter before retrieval to prevent injection attacks.
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	ctx := c.Request.Context()

	// Validate system ID to prevent injection attacks
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	system, err := s.ComputerSystemUC.GetComputerSystem(ctx, computerSystemID)
	if err != nil {
		s.handleGetSystemError(c, err, computerSystemID)

		return
	}

	c.JSON(http.StatusOK, system)
}
