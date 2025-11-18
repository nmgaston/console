package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// MockComputerSystemRepository is a mock implementation of ComputerSystemRepository
type MockComputerSystemRepository struct {
	mock.Mock
}

func (m *MockComputerSystemRepository) GetAll(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if result, ok := args.Get(0).([]string); ok {
		return result, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockComputerSystemRepository) GetByID(ctx context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	args := m.Called(ctx, systemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	if result, ok := args.Get(0).(*redfishv1.ComputerSystem); ok {
		return result, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockComputerSystemRepository) UpdatePowerState(ctx context.Context, systemID string, state redfishv1.PowerState) error {
	args := m.Called(ctx, systemID, state)

	return args.Error(0)
}

// setupTestRouter sets up a test router with the given server
func setupTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Register the systems endpoint
	router.GET("/redfish/v1/Systems", server.GetRedfishV1Systems)

	return router
}

// Helper functions for test validation
func validateMultipleSystemsResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response generated.ComputerSystemCollectionComputerSystemCollection

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Validate response structure
	assert.Equal(t, "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection", *response.OdataContext)
	assert.Equal(t, "/redfish/v1/Systems", *response.OdataId)
	assert.Equal(t, "#ComputerSystemCollection.ComputerSystemCollection", *response.OdataType)
	assert.Equal(t, "Computer System Collection", response.Name)
	assert.NotNil(t, response.Description)
	assert.Equal(t, int64(2), *response.MembersOdataCount)
	assert.NotNil(t, response.Members)
	assert.Len(t, *response.Members, 2)

	// Validate members
	members := *response.Members
	assert.Equal(t, "/redfish/v1/Systems/System1", *members[0].OdataId)
	assert.Equal(t, "/redfish/v1/Systems/b4c3a390-468c-491f-8e1d-9ce04c2fcbc1", *members[1].OdataId)
}

func validateEmptyCollectionResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var response generated.ComputerSystemCollectionComputerSystemCollection

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, int64(0), *response.MembersOdataCount)
	assert.NotNil(t, response.Members)
	assert.Len(t, *response.Members, 0)
}

func validateFilteredSystemsResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var response generated.ComputerSystemCollectionComputerSystemCollection

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Validate that empty strings are filtered out
	assert.Equal(t, int64(3), *response.MembersOdataCount)
	assert.Len(t, *response.Members, 3)

	members := *response.Members
	assert.Equal(t, "/redfish/v1/Systems/System1", *members[0].OdataId)
	assert.Equal(t, "/redfish/v1/Systems/abc-123", *members[1].OdataId)
	assert.Equal(t, "/redfish/v1/Systems/System2", *members[2].OdataId)
}

func validateSingleSystemResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var response generated.ComputerSystemCollectionComputerSystemCollection

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), *response.MembersOdataCount)
	assert.Len(t, *response.Members, 1)

	members := *response.Members
	assert.Equal(t, "/redfish/v1/Systems/System1", *members[0].OdataId)
}

func validateLargeCollectionResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var response generated.ComputerSystemCollectionComputerSystemCollection

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, int64(10), *response.MembersOdataCount)
	assert.Len(t, *response.Members, 10)

	members := *response.Members

	for i := 0; i < 10; i++ {
		expectedURI := fmt.Sprintf("/redfish/v1/Systems/System%d", i+1)
		assert.Equal(t, expectedURI, *members[i].OdataId)
	}
}

func validateJSONResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var jsonResponse map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonResponse)
}

func validateErrorResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var errorResponse map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)

	assert.Contains(t, errorResponse, "error")
	errorObj, ok := errorResponse["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Base.1.22.0.GeneralError", errorObj["code"])
	assert.Equal(t, "An internal server error occurred.", errorObj["message"])
}

// Mock setup functions
func setupMultipleSystemsMock(mockRepo *MockComputerSystemRepository) {
	systemIDs := []string{"System1", "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"}

	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

func setupEmptyCollectionMock(mockRepo *MockComputerSystemRepository) {
	systemIDs := []string{}
	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

func setupFilteredSystemsMock(mockRepo *MockComputerSystemRepository) {
	systemIDs := []string{"System1", "", "abc-123", "", "System2"}
	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

func setupSingleSystemMock(mockRepo *MockComputerSystemRepository) {
	systemIDs := []string{"System1"}
	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

func setupLargeCollectionMock(mockRepo *MockComputerSystemRepository) {
	systemIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		systemIDs[i] = fmt.Sprintf("System%d", i+1)
	}

	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

func setupErrorMock(mockRepo *MockComputerSystemRepository) {
	expectedError := errors.New("database connection failed")
	mockRepo.On("GetAll", mock.Anything).Return([]string(nil), expectedError)
}

func setupNoMock(_ *MockComputerSystemRepository) {
	// No mock setup needed for HTTP method tests
}

// TestGetRedfishV1Systems tests the GetRedfishV1Systems endpoint with various scenarios
func TestGetRedfishV1Systems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func(*MockComputerSystemRepository)
		httpMethod     string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{"Success - Multiple Systems", setupMultipleSystemsMock, "GET", http.StatusOK, validateMultipleSystemsResponse},
		{"Success - Empty Collection", setupEmptyCollectionMock, "GET", http.StatusOK, validateEmptyCollectionResponse},
		{"Success - Filter Empty System IDs", setupFilteredSystemsMock, "GET", http.StatusOK, validateFilteredSystemsResponse},
		{"Success - Single System", setupSingleSystemMock, "GET", http.StatusOK, validateSingleSystemResponse},
		{"Success - Large Members List", setupLargeCollectionMock, "GET", http.StatusOK, validateLargeCollectionResponse},
		{"Success - Response Headers", setupSingleSystemMock, "GET", http.StatusOK, validateJSONResponse},
		{"Error - Repository Error", setupErrorMock, "GET", http.StatusInternalServerError, validateErrorResponse},
		{"Error - HTTP Method POST Not Allowed", setupNoMock, "POST", http.StatusNotFound, nil},
		{"Error - HTTP Method PUT Not Allowed", setupNoMock, "PUT", http.StatusNotFound, nil},
		{"Error - HTTP Method DELETE Not Allowed", setupNoMock, "DELETE", http.StatusNotFound, nil},
		{"Error - HTTP Method PATCH Not Allowed", setupNoMock, "PATCH", http.StatusNotFound, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockRepo := new(MockComputerSystemRepository)
			tt.setupMock(mockRepo)

			useCase := &usecase.ComputerSystemUseCase{
				Repo: mockRepo,
			}
			server := &RedfishServer{
				ComputerSystemUC: useCase,
			}

			// Setup router and request
			router := setupTestRouter(server)
			req, _ := http.NewRequestWithContext(context.Background(), tt.httpMethod, "/redfish/v1/Systems", http.NoBody)
			w := httptest.NewRecorder()

			// Execute
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
