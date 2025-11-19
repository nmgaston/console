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
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// Test constants
const (
	computerSystemODataType              = "#ComputerSystem.v1_22_0.ComputerSystem"
	computerSystemCollectionODataType    = "#ComputerSystemCollection.ComputerSystemCollection"
	computerSystemCollectionODataContext = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	systemsEndpoint                      = "/redfish/v1/Systems"
	computerSystemCollectionName         = "Computer System Collection"
	jsonContentType                      = "application/json; charset=utf-8"
)

// Test error constants
var (
	errDatabaseConnection = errors.New("database connection failed")
)

// Type declarations
// MockComputerSystemRepository is a mock implementation of ComputerSystemRepository
type MockComputerSystemRepository struct {
	mock.Mock
}

// TestCase represents a generic test case structure
type TestCase[T any] struct {
	name           string
	setupMock      func(*MockComputerSystemRepository, T)
	httpMethod     string
	expectedStatus int
	validateFunc   func(*testing.T, *httptest.ResponseRecorder, T)
	params         T
}

// TestConfig holds configuration for running tests
type TestConfig[T any] struct {
	endpoint    string
	routerSetup func(*RedfishServer) *gin.Engine
	urlBuilder  func(T) string
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
	router.GET(systemsEndpoint, server.GetRedfishV1Systems)

	return router
}

// validateJSONContentType validates that the response has the expected JSON content type
func validateJSONContentType(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, jsonContentType, w.Header().Get("Content-Type"))
}

// unmarshalJSONResponse unmarshals JSON response and validates no error occurred
func unmarshalJSONResponse(t *testing.T, w *httptest.ResponseRecorder, response interface{}) {
	t.Helper()

	err := json.Unmarshal(w.Body.Bytes(), response)
	assert.NoError(t, err)
}

// validateCollectionResponse validates the basic structure of a collection response
func validateCollectionResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCount int64, expectedMembers []string) {
	t.Helper()
	validateJSONContentType(t, w)

	var response generated.ComputerSystemCollectionComputerSystemCollection
	unmarshalJSONResponse(t, w, &response)

	// Validate response structure
	assert.Equal(t, computerSystemCollectionODataContext, *response.OdataContext)
	assert.Equal(t, systemsEndpoint, *response.OdataId)
	assert.Equal(t, computerSystemCollectionODataType, *response.OdataType)
	assert.Equal(t, computerSystemCollectionName, response.Name)
	assert.NotNil(t, response.Description)
	assert.Equal(t, expectedCount, *response.MembersOdataCount)
	assert.NotNil(t, response.Members)
	assert.Len(t, *response.Members, int(expectedCount))

	// Validate members if any expected
	if expectedCount > 0 {
		members := *response.Members
		for i, expectedMember := range expectedMembers {
			assert.Equal(t, expectedMember, *members[i].OdataId)
		}
	}
}

// Validation functions for Systems collection responses (reusing shared function)
func validateMultipleSystemsResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateCollectionResponse(t, w, 2, []string{
		fmt.Sprintf("%s/System1", systemsEndpoint),
		fmt.Sprintf("%s/b4c3a390-468c-491f-8e1d-9ce04c2fcbc1", systemsEndpoint),
	})
}

func validateEmptyCollectionResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateCollectionResponse(t, w, 0, []string{})
}

func validateFilteredSystemsResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateCollectionResponse(t, w, 3, []string{
		fmt.Sprintf("%s/System1", systemsEndpoint),
		fmt.Sprintf("%s/abc-123", systemsEndpoint),
		fmt.Sprintf("%s/System2", systemsEndpoint),
	})
}

func validateSingleSystemResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateCollectionResponse(t, w, 1, []string{fmt.Sprintf("%s/System1", systemsEndpoint)})
}

func validateLargeCollectionResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()

	expectedMembers := make([]string, 10)
	for i := 0; i < 10; i++ {
		expectedMembers[i] = fmt.Sprintf("%s/System%d", systemsEndpoint, i+1)
	}

	validateCollectionResponse(t, w, 10, expectedMembers)
}

func validateJSONResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var jsonResponse map[string]interface{}
	unmarshalJSONResponse(t, w, &jsonResponse)
	assert.NotEmpty(t, jsonResponse)
}

// validateErrorResponseData validates error response with expected code and message
func validateErrorResponseData(t *testing.T, w *httptest.ResponseRecorder, expectedCode, expectedMessage string) map[string]interface{} {
	t.Helper()

	var errorResponse map[string]interface{}
	unmarshalJSONResponse(t, w, &errorResponse)

	assert.Contains(t, errorResponse, "error")
	errorObj, ok := errorResponse["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, expectedCode, errorObj["code"])
	assert.Equal(t, expectedMessage, errorObj["message"])

	return errorObj
}

// validateExtendedInfo validates the extended info section of an error response
func validateExtendedInfo(t *testing.T, errorObj map[string]interface{}, expectedMessageID, expectedMessageContains, expectedSeverity string) {
	t.Helper()

	extendedInfo, ok := errorObj["@Message.ExtendedInfo"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, extendedInfo, 1)

	messageInfo, ok := extendedInfo[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, expectedMessageID, messageInfo["MessageId"])
	assert.Contains(t, messageInfo["Message"], expectedMessageContains)
	assert.Equal(t, expectedSeverity, messageInfo["Severity"])
}

func validateErrorResponse(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	_ = validateErrorResponseData(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

// setupCollectionMockWithIDs sets up mock to return specific system IDs
func setupCollectionMockWithIDs(mockRepo *MockComputerSystemRepository, systemIDs []string) {
	mockRepo.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

// setupCollectionMockWithError sets up mock to return an error
func setupCollectionMockWithError(mockRepo *MockComputerSystemRepository, err error) {
	mockRepo.On("GetAll", mock.Anything).Return([]string(nil), err)
}

// Mock setup functions for Systems collection (reusing shared functions)
func setupMultipleSystemsMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	setupCollectionMockWithIDs(mockRepo, []string{"System1", "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"})
}

func setupEmptyCollectionMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	setupCollectionMockWithIDs(mockRepo, []string{})
}

func setupFilteredSystemsMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	setupCollectionMockWithIDs(mockRepo, []string{"System1", "", "abc-123", "", "System2"})
}

func setupSingleSystemMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	setupCollectionMockWithIDs(mockRepo, []string{"System1"})
}

func setupLargeCollectionMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	systemIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		systemIDs[i] = fmt.Sprintf("System%d", i+1)
	}

	setupCollectionMockWithIDs(mockRepo, systemIDs)
}

func setupErrorMock(mockRepo *MockComputerSystemRepository, _ struct{}) {
	setupCollectionMockWithError(mockRepo, errDatabaseConnection)
}

func setupNoMock(_ *MockComputerSystemRepository, _ struct{}) {
	// No mock setup needed for HTTP method tests
}

// Generic Test Framework for Redfish endpoints

// runGenericTest executes a generic test case
func runGenericTest[T any](t *testing.T, testCase TestCase[T], config TestConfig[T]) {
	t.Helper()

	// Setup
	mockRepo := new(MockComputerSystemRepository)
	testCase.setupMock(mockRepo, testCase.params)

	useCase := &usecase.ComputerSystemUseCase{
		Repo: mockRepo,
	}
	server := &RedfishServer{
		ComputerSystemUC: useCase,
	}

	// Setup router and request
	router := config.routerSetup(server)
	url := config.urlBuilder(testCase.params)
	req, _ := http.NewRequestWithContext(context.Background(), testCase.httpMethod, url, http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, testCase.expectedStatus, w.Code)

	// Run custom validation
	if testCase.validateFunc != nil {
		testCase.validateFunc(t, w, testCase.params)
	}

	mockRepo.AssertExpectations(t)
}

// setupTestRouterForSystemByID sets up a test router with the system by ID endpoint
func setupTestRouterForSystemByID(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Register the system by ID endpoint with a wrapper function
	router.GET("/redfish/v1/Systems/:computerSystemID", func(c *gin.Context) {
		computerSystemID := c.Param("computerSystemID")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
	})

	return router
}

// createTestSystem creates a test system with specified properties
func createTestSystem(systemID, name, manufacturer, model, serialNumber string, systemType redfishv1.SystemType, powerState redfishv1.PowerState) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		SystemType:   systemType,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		ODataID:      fmt.Sprintf("%s/%s", systemsEndpoint, systemID),
		ODataType:    computerSystemODataType,
	}
}

// setupSystemMockWithData sets up mock to return a specific system
func setupSystemMockWithData(mockRepo *MockComputerSystemRepository, systemID string, system *redfishv1.ComputerSystem) {
	mockRepo.On("GetByID", mock.Anything, systemID).Return(system, nil)
}

// setupSystemMockWithError sets up mock to return an error
func setupSystemMockWithError(mockRepo *MockComputerSystemRepository, systemID string, err error) {
	mockRepo.On("GetByID", mock.Anything, systemID).Return(nil, err)
}

// Mock setup functions for system by ID tests (reusing shared functions)
func setupExistingSystemMock(mockRepo *MockComputerSystemRepository, systemID string) {
	system := createTestSystem(systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456", redfishv1.SystemTypePhysical, redfishv1.PowerStateOn)
	setupSystemMockWithData(mockRepo, systemID, system)
}

func setupUUIDSystemMock(mockRepo *MockComputerSystemRepository, systemID string) {
	system := createTestSystem(systemID, "UUID System", "UUID Manufacturer", "UUID Model", "UUID-SN789", redfishv1.SystemTypeVirtual, redfishv1.PowerStateOff)
	setupSystemMockWithData(mockRepo, systemID, system)
}

func setupSystemNotFoundMock(mockRepo *MockComputerSystemRepository, systemID string) {
	setupSystemMockWithError(mockRepo, systemID, usecase.ErrSystemNotFound)
}

func setupSystemRepositoryErrorMock(mockRepo *MockComputerSystemRepository, systemID string) {
	setupSystemMockWithError(mockRepo, systemID, errDatabaseConnection)
}

func setupNoSystemMock(_ *MockComputerSystemRepository, _ string) {
	// No mock setup needed for HTTP method tests
}

// validateSystemResponseData validates system response with expected data
func validateSystemResponseData(t *testing.T, w *httptest.ResponseRecorder, systemID, expectedName, expectedManufacturer, expectedModel, expectedSerial string) {
	t.Helper()
	validateJSONContentType(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponse(t, w, &response)

	// Validate response structure
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, expectedName, response.Name)
	assert.NotNil(t, response.Manufacturer)
	assert.Equal(t, expectedManufacturer, *response.Manufacturer)
	assert.NotNil(t, response.Model)
	assert.Equal(t, expectedModel, *response.Model)
	assert.NotNil(t, response.SerialNumber)
	assert.Equal(t, expectedSerial, *response.SerialNumber)
	assert.NotNil(t, response.OdataId)
	assert.Equal(t, fmt.Sprintf("%s/%s", systemsEndpoint, systemID), *response.OdataId)
	assert.NotNil(t, response.OdataType)
	assert.Equal(t, computerSystemODataType, *response.OdataType)
}

// Validation functions for system by ID tests (reusing shared validation)
func validateSystemResponse(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseData(t, w, systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
}

func validateUUIDSystemResponse(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseData(t, w, systemID, "UUID System", "UUID Manufacturer", "UUID Model", "UUID-SN789")
}

func validateSystemNotFoundResponse(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	errorObj := validateErrorResponseData(t, w, "Base.1.22.0.GeneralError", "System not found")
	validateExtendedInfo(t, errorObj, "Base.1.22.0.ResourceMissing", fmt.Sprintf("The requested resource of type System named %s was not found.", systemID), "Critical")
}

func validateSystemErrorResponse(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseData(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

// ====================================================================================================
// MAIN TEST FUNCTIONS
// ====================================================================================================

// TestGetRedfishV1Systems tests the GetRedfishV1Systems endpoint with various scenarios
func TestGetRedfishV1Systems(t *testing.T) {
	t.Parallel()

	config := TestConfig[struct{}]{
		endpoint:    systemsEndpoint,
		routerSetup: setupTestRouter,
		urlBuilder: func(_ struct{}) string {
			return systemsEndpoint
		},
	}

	tests := []TestCase[struct{}]{
		{"Success - Multiple Systems", setupMultipleSystemsMock, "GET", http.StatusOK, validateMultipleSystemsResponse, struct{}{}},
		{"Success - Empty Collection", setupEmptyCollectionMock, "GET", http.StatusOK, validateEmptyCollectionResponse, struct{}{}},
		{"Success - Filter Empty System IDs", setupFilteredSystemsMock, "GET", http.StatusOK, validateFilteredSystemsResponse, struct{}{}},
		{"Success - Single System", setupSingleSystemMock, "GET", http.StatusOK, validateSingleSystemResponse, struct{}{}},
		{"Success - Large Members List", setupLargeCollectionMock, "GET", http.StatusOK, validateLargeCollectionResponse, struct{}{}},
		{"Success - Response Headers", setupSingleSystemMock, "GET", http.StatusOK, validateJSONResponse, struct{}{}},
		{"Error - Repository Error", setupErrorMock, "GET", http.StatusInternalServerError, validateErrorResponse, struct{}{}},
		{"Error - HTTP Method POST Not Allowed", setupNoMock, "POST", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PUT Not Allowed", setupNoMock, "PUT", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method DELETE Not Allowed", setupNoMock, "DELETE", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PATCH Not Allowed", setupNoMock, "PATCH", http.StatusNotFound, nil, struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericTest(t, tt, config)
		})
	}
}

// TestGetRedfishV1SystemsComputerSystemId tests the GetRedfishV1SystemsComputerSystemId endpoint with various scenarios
func TestGetRedfishV1SystemsComputerSystemId(t *testing.T) {
	t.Parallel()

	config := TestConfig[string]{
		endpoint:    "/redfish/v1/Systems/{id}",
		routerSetup: setupTestRouterForSystemByID,
		urlBuilder: func(systemID string) string {
			return fmt.Sprintf("%s/%s", systemsEndpoint, systemID)
		},
	}

	tests := []TestCase[string]{
		{"Success - Existing System", setupExistingSystemMock, "GET", http.StatusOK, validateSystemResponse, "System1"},
		{"Success - UUID System", setupUUIDSystemMock, "GET", http.StatusOK, validateUUIDSystemResponse, "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"},
		{"Error - System Not Found", setupSystemNotFoundMock, "GET", http.StatusNotFound, validateSystemNotFoundResponse, "NonExistentSystem"},
		{"Error - Repository Error", setupSystemRepositoryErrorMock, "GET", http.StatusInternalServerError, validateSystemErrorResponse, "System1"},
		{"Error - HTTP Method POST Not Allowed", setupNoSystemMock, "POST", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method PUT Not Allowed", setupNoSystemMock, "PUT", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method DELETE Not Allowed", setupNoSystemMock, "DELETE", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method PATCH Not Allowed", setupNoSystemMock, "PATCH", http.StatusNotFound, nil, "System1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericTest(t, tt, config)
		})
	}
}
