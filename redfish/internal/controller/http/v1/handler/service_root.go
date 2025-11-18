// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"embed"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
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
		log.Fatalf("Could not load embedded metadata.xml: %v", err)
	}

	metadataXML = string(data)

	// Validate XML
	if err := validateMetadataXML(metadataXML); err != nil {
		log.Fatalf("Invalid metadata.xml: %v", err)
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

// GetRedfishV1 returns the service root
// Path: GET /redfish/v1
// Spec: Redfish ServiceRoot.v1_19_0
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
