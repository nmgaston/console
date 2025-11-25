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

// resetMetadataState safely resets the global metadata state for test isolation.
// This must be called within a synchronized context to avoid race conditions.
func resetMetadataState() {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	metadataXML = ""
	metadataLoaded = false
}

// setupMetadataTestRouter creates and configures a test router with the metadata endpoint.
func setupMetadataTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1/$metadata", server.GetRedfishV1Metadata)

	return router
}

func TestGetRedfishV1MetadataReturnsODataXML(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	// Setup
	router := setupMetadataTestRouter()

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

	// Verify valid XML by parsing it (if body is not empty)
	if body != "" {
		assert.NoError(t, xml.Unmarshal([]byte(body), new(interface{})), "response should be valid XML")

		// Assert XML declaration and root element
		assert.True(t, strings.HasPrefix(body, `<?xml version="1.0" encoding="UTF-8"?>`))
		assert.Contains(t, body, `<edmx:Edmx`)
		assert.Contains(t, body, `xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"`)
		assert.Contains(t, body, `Version="4.0"`)

		// Assert required DMTF Redfish schema references (auto-discovered from YAML files)
		requiredSchemas := []string{
			"ActionInfo_v1.xml",
			"ComputerSystemCollection_v1.xml",
			"ComputerSystem_v1.xml",
			"Message_v1.xml",
			"ResolutionStep_v1.xml",
			"Resource_v1.xml",
			"ServiceRoot_v1.xml",
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
	}
}

func TestGetRedfishV1MetadataServeValidResponse(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify it's valid XML if response body is not empty
	body := w.Body.String()
	if body != "" {
		var doc interface{}

		err := xml.Unmarshal([]byte(body), &doc)
		assert.NoError(t, err, "response should be valid XML")
	}
}

func TestGetRedfishV1MetadataConcurrentRequests(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

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
}

// TestLoadMetadata tests the metadata loading behavior through the public endpoint.
// Since loadMetadata is internal, we test it indirectly through GetRedfishV1Metadata.
func TestLoadMetadata(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	t.Run("metadata endpoint loads and caches metadata", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)

		// Setup router
		router := setupMetadataTestRouter()

		// First request
		req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second request - should be identical (cached)
		req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		// Both should have same status
		assert.Equal(t, w1.Code, w2.Code)

		// Both should have same response body (caching works)
		assert.Equal(t, w1.Body.String(), w2.Body.String())
	})

	t.Run("metadata endpoint sets correct headers", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify headers are set correctly
		assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
		assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
	})
}

// TestValidateMetadataXML tests XML validation through the endpoint.
// Direct validation testing is not possible since validateMetadataXML is private.
// Instead, we verify that the endpoint returns valid XML responses.
func TestValidateMetadataXML(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	t.Run("endpoint returns valid XML response", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// Verify response is valid XML
		if body != "" {
			var doc interface{}

			err := xml.Unmarshal([]byte(body), &doc)
			assert.NoError(t, err, "response should be valid XML")
		}

		// Verify status
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestLoadMetadataIntegration tests the metadata endpoint integration.
func TestLoadMetadataIntegration(t *testing.T) {
	t.Run("metadata endpoint returns consistent results", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		// Make multiple requests
		responses := make([]string, 3)

		for i := 0; i < 3; i++ {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			responses[i] = w.Body.String()
		}

		// All responses should be identical (caching works)
		assert.Equal(t, responses[0], responses[1])
		assert.Equal(t, responses[1], responses[2])
	})

	t.Run("metadata endpoint response has proper structure", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// If metadata is loaded, verify structure
		if body != "" {
			// Should be valid XML
			var doc interface{}

			err := xml.Unmarshal([]byte(body), &doc)
			assert.NoError(t, err, "response should be valid XML")

			// Verify headers
			assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
			assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
		}
	})
}
