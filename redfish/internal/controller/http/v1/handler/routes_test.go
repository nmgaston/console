package v1

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRedfishV1Metadata(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	t.Run("returns OData metadata XML with proper headers", func(t *testing.T) {
		t.Parallel()

		// Setup
		router := gin.New()
		server := &RedfishServer{}
		router.GET("/redfish/v1/$metadata", server.GetRedfishV1Metadata)

		// Execute request
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status and headers
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
		assert.Equal(t, "4.0", w.Header().Get("OData-Version"))

		body := w.Body.String()

		// Verify valid XML by parsing it
		assert.NoError(t, xml.Unmarshal([]byte(body), new(interface{})), "response should be valid XML")

		// Assert XML declaration and root element
		assert.True(t, strings.HasPrefix(body, `<?xml version="1.0" encoding="UTF-8"?>`))
		assert.Contains(t, body, `<edmx:Edmx`)
		assert.Contains(t, body, `xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"`)
		assert.Contains(t, body, `Version="4.0"`)

		// Assert required DMTF Redfish schema references
		requiredSchemas := []string{
			"ServiceRoot.v1_19_0",
			"ComputerSystemCollection",
			"ComputerSystem.v1_22_0",
			"Task.v1_6_0",
			"Message",
			"RedfishExtensions.v1_0_0",
		}
		for _, schema := range requiredSchemas {
			assert.Contains(t, body, schema, "metadata should reference %s schema", schema)
		}

		// Assert EntityContainer structure
		assert.Contains(t, body, `<edmx:DataServices>`)
		assert.Contains(t, body, `<Schema`)
		assert.Contains(t, body, `<EntityContainer Name="Service"`)
		assert.Contains(t, body, `Extends="ServiceRoot.v1_19_0.ServiceContainer"`)

		// Verify substantial content
		assert.Greater(t, len(body), 1000, "metadata should contain substantial content")
	})

	t.Run("metadata content is embedded and valid", func(t *testing.T) {
		t.Parallel()

		// Verify the embedded metadata is not empty
		require.NotEmpty(t, metadataXML, "embedded metadata should not be empty")

		// Verify it's valid XML
		assert.NoError(t, xml.Unmarshal([]byte(metadataXML), new(interface{})), "embedded metadata should be valid XML")

		// Verify it contains OData EDMX structure
		assert.Contains(t, metadataXML, "edmx:Edmx")
		assert.Contains(t, metadataXML, "http://docs.oasis-open.org/odata/ns/edmx")
	})

	t.Run("handles concurrent requests", func(t *testing.T) {
		t.Parallel()

		router := gin.New()
		server := &RedfishServer{}
		router.GET("/redfish/v1/$metadata", server.GetRedfishV1Metadata)

		// Execute multiple concurrent requests
		const numRequests = 10

		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				results <- w.Code
			}()
		}

		// Verify all requests succeeded
		for i := 0; i < numRequests; i++ {
			assert.Equal(t, http.StatusOK, <-results)
		}
	})
}
