package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// TestGetRedfishV1Odata tests the GetRedfishV1Odata handler with full coverage.
func TestGetRedfishV1Odata(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	server := &RedfishServer{ComputerSystemUC: nil}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	// Test
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Parse response.
	var response generated.OdataServiceOdataService

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.NotNil(t, response.OdataContext)
	assert.NotNil(t, response.OdataId)
	assert.NotNil(t, response.OdataType)
	assert.Equal(t, "/redfish/v1/odata", *response.OdataId)
	assert.Equal(t, "OdataService", response.Id)
	assert.Equal(t, "OData Service Root", response.Name)
}
