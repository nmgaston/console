// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"embed"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// ServiceRoot OData metadata constants
const (
	odataContextServiceRoot = "/redfish/v1/$metadata#ServiceRoot.ServiceRoot"
	odataIDServiceRoot      = "/redfish/v1"
	odataTypeServiceRoot    = "#ServiceRoot.v1_19_0.ServiceRoot"
	serviceRootID           = "RootService"
	serviceRootName         = "Root Service"
	redfishVersion          = "1.19.0"
)

// Metadata embedding and caching
//
//go:embed metadata.xml
var metadataFS embed.FS

var (
	metadataXML    string
	metadataLoaded bool
	metadataMutex  sync.Mutex
)

// loadMetadata loads embedded metadata.xml with XML validation.
// This function is thread-safe and ensures metadata is loaded only once.
// The metadata file is embedded at build time using go:embed directive
// from the current package.
func loadMetadata() {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	// Double-check locking pattern
	if metadataLoaded {
		return
	}

	data, err := metadataFS.ReadFile("metadata.xml")
	if err != nil {
		log.Warnf("Could not load embedded metadata.xml: %v", err)

		return
	}

	metadataXML = string(data)

	// Validate XML
	if err := validateMetadataXML(metadataXML); err != nil {
		log.Warnf("Invalid metadata.xml: %v", err)

		return
	}

	log.Infof("Embedded metadata.xml loaded and validation passed")

	metadataLoaded = true
}

// validateMetadataXML checks if the provided XML string is well-formed.
func validateMetadataXML(xmlData string) error {
	if xmlData == "" {
		return nil // Empty is acceptable
	}

	var doc interface{}

	if err := xml.Unmarshal([]byte(xmlData), &doc); err != nil {
		return fmt.Errorf("XML parsing failed: %w", err)
	}

	return nil
}

const uuidFileName = "service_uuid"

// getUUIDStoragePath returns the OS-agnostic path for storing the service UUID.
// Works across Linux, Windows, and macOS using the user config directory.
func getUUIDStoragePath(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	dir := filepath.Join(base, appName)

	const dirPermissions = 0o755

	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(dir, uuidFileName), nil
}

// loadOrCreateUUID loads an existing UUID from file or creates a new one.
// This ensures the UUID persists across service restarts.
func loadOrCreateUUID(appName string) (string, error) {
	file, err := getUUIDStoragePath(appName)
	if err != nil {
		return "", err
	}

	// Try to read existing UUID
	if data, err := os.ReadFile(file); err == nil {
		storedUUID := string(data)
		// Validate it's a proper UUID
		if _, err := uuid.Parse(storedUUID); err == nil {
			return storedUUID, nil
		}

		log.Warnf("Invalid UUID in storage file, generating new one")
	}

	// Create new UUID
	newUUID := uuid.New().String()

	const filePermissions = 0o600

	// Save to file
	if err := os.WriteFile(file, []byte(newUUID), filePermissions); err != nil {
		log.Warnf("Failed to save UUID to file: %v", err)
		// Continue with the generated UUID even if save fails
	}

	return newUUID, nil
}

// generateServiceUUID generates or retrieves the service instance UUID.
// Per Redfish specification, this UUID should be consistent across service restarts.
// Priority order:
// 1. REDFISH_UUID environment variable (allows admin override)
// 2. Persisted UUID from config file
// 3. Newly generated UUID (saved to config file for future use)
func generateServiceUUID() string {
	// 1. Check environment variable override
	if envUUID := os.Getenv("REDFISH_UUID"); envUUID != "" {
		if _, parseErr := uuid.Parse(envUUID); parseErr == nil {
			return envUUID
		}

		log.Warnf("Invalid REDFISH_UUID environment variable, ignoring")
	}

	// 2. Load or create persistent UUID
	serviceUUID, err := loadOrCreateUUID("dmt-redfish-service")
	if err != nil {
		log.Warnf("Failed to load/create persistent UUID: %v, generating temporary UUID", err)
		// Fallback to temporary UUID for this session
		return uuid.New().String()
	}

	return serviceUUID
}

// GetRedfishV1 returns the service root
// Path: GET /redfish/v1
// Spec: Redfish ServiceRoot.v1_19_0
// This is the entry point for the Redfish API, providing links to all available resources.
// Per Redfish specification, this endpoint must be accessible without authentication.
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	// Set Redfish-compliant headers
	SetRedfishHeaders(c)

	serviceRoot := generated.ServiceRootServiceRoot{
		OdataContext:   StringPtr(odataContextServiceRoot),
		OdataId:        StringPtr(odataIDServiceRoot),
		OdataType:      StringPtr(odataTypeServiceRoot),
		Id:             serviceRootID,
		Name:           serviceRootName,
		RedfishVersion: StringPtr(redfishVersion),
		UUID:           StringPtr(generateServiceUUID()),
		Product:        StringPtr("Device Management Toolkit - Redfish Service"),
		Vendor:         StringPtr("Device Management Toolkit"),
		Links: generated.ServiceRootLinks{
			Sessions: generated.OdataV4IdRef{
				OdataId: StringPtr("/redfish/v1/SessionService/Sessions"),
			},
		},
		Systems: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Systems"),
		},
		Registries: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Registries"),
		},
	}

	c.JSON(http.StatusOK, serviceRoot)
}

// GetRedfishV1Metadata returns the OData CSDL metadata document describing the service's data model.
// Path: GET /redfish/v1/$metadata
// Spec: OData CSDL v4.0 - Redfish specification mandates this endpoint is accessible without authentication.
func (s *RedfishServer) GetRedfishV1Metadata(c *gin.Context) {
	// Ensure metadata is loaded
	loadMetadata()

	// Read metadata safely
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	metadata := metadataXML

	// Set Redfish-compliant headers
	c.Header(headerContentType, contentTypeXML)
	c.Header(headerODataVersion, odataVersion)
	c.String(http.StatusOK, metadata)
}
