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
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// Test constants specific to systems handler
const (
	systemsOdataContextCollectionTest = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	systemsOdataIDCollectionTest      = "/redfish/v1/Systems"
	systemsOdataTypeCollectionTest    = "#ComputerSystemCollection.ComputerSystemCollection"
	systemsCollectionTitleTest        = "Computer System Collection"
	systemsEndpointTest               = "/redfish/v1/Systems"
	jsonContentTypeTest               = "application/json; charset=utf-8"
	systemODataType                   = "#ComputerSystem.v1_22_0.ComputerSystem"
)

// Test error constants specific to systems
var (
	errSystemRepoFailure = errors.New("system repository operation failed")
)

// MockComputerSystemUseCase mocks the ComputerSystemUseCase interface for systems handler
type MockComputerSystemUseCase struct {
	mock.Mock
}

func (m *MockComputerSystemUseCase) GetAll(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	result, ok := args.Get(0).([]string)
	if !ok {
		return nil, args.Error(1)
	}

	return result, args.Error(1)
}

func (m *MockComputerSystemUseCase) GetComputerSystem(ctx context.Context, systemID string) (*generated.ComputerSystemComputerSystem, error) {
	args := m.Called(ctx, systemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	result, ok := args.Get(0).(*generated.ComputerSystemComputerSystem)
	if !ok {
		return nil, args.Error(1)
	}

	return result, args.Error(1)
}

// TestCase represents a generic test case structure
type SystemsTestCase[T any] struct {
	name           string
	setupMock      func(*MockComputerSystemUseCase, T)
	httpMethod     string
	expectedStatus int
	validateFunc   func(*testing.T, *httptest.ResponseRecorder, T)
	params         T
}

// TestConfig holds configuration for running tests
type SystemsTestConfig[T any] struct {
	endpoint    string
	routerSetup func(*SystemsHandler) *gin.Engine
	urlBuilder  func(T) string
}

// validateJSONContentTypeTest validates that the response has the expected JSON content type
func validateJSONContentTypeTest(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, jsonContentTypeTest, w.Header().Get("Content-Type"))
}

// unmarshalJSONResponseTest unmarshals JSON response and validates no error occurred
func unmarshalJSONResponseTest(t *testing.T, w *httptest.ResponseRecorder, response interface{}) {
	t.Helper()

	err := json.Unmarshal(w.Body.Bytes(), response)
	assert.NoError(t, err)
}

// validateSystemsCollectionResponse validates the basic structure of a systems collection response
func validateSystemsCollectionResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCount int64, expectedMembers []string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemCollectionComputerSystemCollection
	unmarshalJSONResponseTest(t, w, &response)

	// Validate response structure
	assert.Equal(t, systemsOdataContextCollectionTest, *response.OdataContext)
	assert.Equal(t, systemsEndpointTest, *response.OdataId)
	assert.Equal(t, systemsOdataTypeCollectionTest, *response.OdataType)
	assert.Equal(t, systemsCollectionTitleTest, response.Name)
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

// Router setup for testing

func setupSystemsTestRouter(handler *SystemsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(systemsEndpointTest, handler.GetSystemsCollection)

	return router
}

func setupSystemByIDTestRouter(handler *SystemsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/redfish/v1/Systems/:ComputerSystemId", handler.GetSystemByID)
	// Add route for empty system ID case (trailing slash)
	router.GET("/redfish/v1/Systems/", handler.GetSystemByID)

	return router
}

// Validation functions for Systems collection responses (reusing shared function)
func validateMultipleSystemsResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 2, []string{
		fmt.Sprintf("%s/System1", systemsEndpointTest),
		fmt.Sprintf("%s/System2", systemsEndpointTest),
	})
}

func validateEmptyCollectionResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 0, []string{})
}

func validateFilteredSystemsResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 3, []string{
		fmt.Sprintf("%s/System1", systemsEndpointTest),
		fmt.Sprintf("%s/abc-123", systemsEndpointTest),
		fmt.Sprintf("%s/System2", systemsEndpointTest),
	})
}

func validateSingleSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 1, []string{fmt.Sprintf("%s/System1", systemsEndpointTest)})
}

func validateLargeCollectionResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()

	expectedMembers := make([]string, 10)
	for i := 0; i < 10; i++ {
		expectedMembers[i] = fmt.Sprintf("%s/System%d", systemsEndpointTest, i+1)
	}

	validateSystemsCollectionResponse(t, w, 10, expectedMembers)
}

func validateJSONResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var jsonResponse map[string]interface{}
	unmarshalJSONResponseTest(t, w, &jsonResponse)
	assert.NotEmpty(t, jsonResponse)
}

// validateErrorResponseDataTest validates error response with expected code and message
//
//nolint:unparam // expectedCode varies in different test contexts
func validateErrorResponseDataTest(t *testing.T, w *httptest.ResponseRecorder, expectedCode, expectedMessage string) map[string]interface{} {
	t.Helper()

	var errorResponse map[string]interface{}
	unmarshalJSONResponseTest(t, w, &errorResponse)

	assert.Contains(t, errorResponse, "error")
	errorObj, ok := errorResponse["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, expectedCode, errorObj["code"])
	assert.Equal(t, expectedMessage, errorObj["message"])

	return errorObj
}

// validateExtendedInfoTest validates the extended info section of an error response
func validateExtendedInfoTest(t *testing.T, errorObj map[string]interface{}, expectedMessageID, expectedMessageContains, expectedSeverity string) {
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

func validateErrorResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

// setupCollectionMockWithIDsTest sets up mock to return specific system IDs
func setupCollectionMockWithIDsTest(mockUseCase *MockComputerSystemUseCase, systemIDs []string) {
	mockUseCase.On("GetAll", mock.Anything).Return(systemIDs, nil)
}

// setupCollectionMockWithErrorTest sets up mock to return an error
func setupCollectionMockWithErrorTest(mockUseCase *MockComputerSystemUseCase, err error) {
	mockUseCase.On("GetAll", mock.Anything).Return([]string(nil), err)
}

// Mock setup functions for Systems collection (reusing shared functions)
func setupMultipleSystemsMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	setupCollectionMockWithIDsTest(mockUseCase, []string{"System1", "System2"})
}

func setupEmptyCollectionMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	setupCollectionMockWithIDsTest(mockUseCase, []string{})
}

func setupFilteredSystemsMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	setupCollectionMockWithIDsTest(mockUseCase, []string{"System1", "", "abc-123", "", "System2"})
}

func setupSingleSystemMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	setupCollectionMockWithIDsTest(mockUseCase, []string{"System1"})
}

func setupLargeCollectionMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	systemIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		systemIDs[i] = fmt.Sprintf("System%d", i+1)
	}

	setupCollectionMockWithIDsTest(mockUseCase, systemIDs)
}

func setupErrorMockTest(mockUseCase *MockComputerSystemUseCase, _ struct{}) {
	setupCollectionMockWithErrorTest(mockUseCase, errSystemRepoFailure)
}

func setupNoMockTest(_ *MockComputerSystemUseCase, _ struct{}) {
	// No mock setup needed for HTTP method tests
}

// Generic Test Framework for Systems endpoints

// runGenericSystemsTest executes a generic test case
func runGenericSystemsTest[T any](t *testing.T, testCase SystemsTestCase[T], config SystemsTestConfig[T]) {
	t.Helper()

	// Setup
	mockUseCase := new(MockComputerSystemUseCase)
	testCase.setupMock(mockUseCase, testCase.params)

	// Create SystemsHandler
	systemsHandler := NewSystemsHandler(mockUseCase, nil) // Using nil logger for tests

	// Setup router and request
	router := config.routerSetup(systemsHandler)
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

	mockUseCase.AssertExpectations(t)
}

// createTestSystemData creates a test system with specified properties
func createTestSystemData(systemID, name, manufacturer, model, serialNumber string) *generated.ComputerSystemComputerSystem {
	powerState := &generated.ComputerSystemComputerSystem_PowerState{}
	_ = powerState.FromResourcePowerState(generated.On)

	return &generated.ComputerSystemComputerSystem{
		OdataContext: StringPtr("/redfish/v1/$metadata#ComputerSystem.ComputerSystem"),
		OdataId:      StringPtr(fmt.Sprintf("%s/%s", systemsEndpointTest, systemID)),
		OdataType:    StringPtr(systemODataType),
		Id:           systemID,
		Name:         name,
		Manufacturer: StringPtr(manufacturer),
		Model:        StringPtr(model),
		SerialNumber: StringPtr(serialNumber),
		PowerState:   powerState,
	}
}

// setupSystemMockWithDataTest sets up mock to return a specific system
func setupSystemMockWithDataTest(mockUseCase *MockComputerSystemUseCase, systemID string, system *generated.ComputerSystemComputerSystem) {
	mockUseCase.On("GetComputerSystem", mock.Anything, systemID).Return(system, nil)
}

// setupSystemMockWithErrorTest sets up mock to return an error
func setupSystemMockWithErrorTest(mockUseCase *MockComputerSystemUseCase, systemID string, err error) {
	mockUseCase.On("GetComputerSystem", mock.Anything, systemID).Return(nil, err)
}

// Mock setup functions for system by ID tests (reusing shared functions)
func setupExistingSystemMockTest(mockUseCase *MockComputerSystemUseCase, systemID string) {
	system := createTestSystemData(systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
	setupSystemMockWithDataTest(mockUseCase, systemID, system)
}

func setupUUIDSystemMockTest(mockUseCase *MockComputerSystemUseCase, systemID string) {
	system := createTestSystemData(systemID, "UUID System", "UUID Manufacturer", "UUID Model", "UUID-SN789")
	setupSystemMockWithDataTest(mockUseCase, systemID, system)
}

func setupSystemNotFoundMockTest(mockUseCase *MockComputerSystemUseCase, systemID string) {
	setupSystemMockWithErrorTest(mockUseCase, systemID, usecase.ErrSystemNotFound)
}

func setupSystemRepositoryErrorMockTest(mockUseCase *MockComputerSystemUseCase, systemID string) {
	setupSystemMockWithErrorTest(mockUseCase, systemID, errSystemRepoFailure)
}

func setupNoSystemMockTest(_ *MockComputerSystemUseCase, _ string) {
	// No mock setup needed for HTTP method tests
}

// validateSystemResponseDataTest validates system response with expected data
func validateSystemResponseDataTest(t *testing.T, w *httptest.ResponseRecorder, systemID, expectedName, expectedManufacturer, expectedModel, expectedSerial string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

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
	assert.Equal(t, fmt.Sprintf("%s/%s", systemsEndpointTest, systemID), *response.OdataId)
	assert.NotNil(t, response.OdataType)
	assert.Equal(t, systemODataType, *response.OdataType)
}

// Validation functions for system by ID tests (reusing shared validation)
func validateSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseDataTest(t, w, systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
}

func validateUUIDSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseDataTest(t, w, systemID, "UUID System", "UUID Manufacturer", "UUID Model", "UUID-SN789")
}

func validateSystemNotFoundResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	errorObj := validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "System not found")
	validateExtendedInfoTest(t, errorObj, "Base.1.22.0.ResourceMissing", fmt.Sprintf("The requested resource of type System named %s was not found.", systemID), "Critical")
}

func validateSystemErrorResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

func validateBadRequestResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "Computer system ID is required")
}

// Mock logger for testing logger code paths
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(message interface{}, args ...interface{}) {
	m.Called(message, args)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Error(message interface{}, args ...interface{}) {
	m.Called(message, args)
}

func (m *MockLogger) Fatal(message interface{}, args ...interface{}) {
	m.Called(message, args)
}

// ====================================================================================================
// MAIN TEST FUNCTIONS
// ====================================================================================================

// TestSystemsHandler_GetSystemsCollection tests the GetSystemsCollection endpoint with various scenarios
func TestSystemsHandler_GetSystemsCollection(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[struct{}]{
		endpoint:    systemsEndpointTest,
		routerSetup: setupSystemsTestRouter,
		urlBuilder: func(_ struct{}) string {
			return systemsEndpointTest
		},
	}

	tests := []SystemsTestCase[struct{}]{
		{"Success - Multiple Systems", setupMultipleSystemsMockTest, "GET", http.StatusOK, validateMultipleSystemsResponseTest, struct{}{}},
		{"Success - Empty Collection", setupEmptyCollectionMockTest, "GET", http.StatusOK, validateEmptyCollectionResponseTest, struct{}{}},
		{"Success - Filter Empty System IDs", setupFilteredSystemsMockTest, "GET", http.StatusOK, validateFilteredSystemsResponseTest, struct{}{}},
		{"Success - Single System", setupSingleSystemMockTest, "GET", http.StatusOK, validateSingleSystemResponseTest, struct{}{}},
		{"Success - Large Members List", setupLargeCollectionMockTest, "GET", http.StatusOK, validateLargeCollectionResponseTest, struct{}{}},
		{"Success - Response Headers", setupSingleSystemMockTest, "GET", http.StatusOK, validateJSONResponseTest, struct{}{}},
		{"Error - Repository Error", setupErrorMockTest, "GET", http.StatusInternalServerError, validateErrorResponseTest, struct{}{}},
		{"Error - HTTP Method POST Not Allowed", setupNoMockTest, "POST", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PUT Not Allowed", setupNoMockTest, "PUT", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method DELETE Not Allowed", setupNoMockTest, "DELETE", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PATCH Not Allowed", setupNoMockTest, "PATCH", http.StatusNotFound, nil, struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestSystemsHandler_GetSystemByID tests the GetSystemByID endpoint with various scenarios
func TestSystemsHandler_GetSystemByID(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[string]{
		endpoint:    "/redfish/v1/Systems/{id}",
		routerSetup: setupSystemByIDTestRouter,
		urlBuilder: func(systemID string) string {
			if systemID == "" {
				return fmt.Sprintf("%s/", systemsEndpointTest)
			}

			return fmt.Sprintf("%s/%s", systemsEndpointTest, systemID)
		},
	}

	tests := []SystemsTestCase[string]{
		{"Success - Existing System", setupExistingSystemMockTest, "GET", http.StatusOK, validateSystemResponseTest, "System1"},
		{"Success - UUID System", setupUUIDSystemMockTest, "GET", http.StatusOK, validateUUIDSystemResponseTest, "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"},
		{"Error - Empty System ID", setupNoSystemMockTest, "GET", http.StatusBadRequest, validateBadRequestResponseTest, ""},
		{"Error - System Not Found", setupSystemNotFoundMockTest, "GET", http.StatusNotFound, validateSystemNotFoundResponseTest, "NonExistentSystem"},
		{"Error - Repository Error", setupSystemRepositoryErrorMockTest, "GET", http.StatusInternalServerError, validateSystemErrorResponseTest, "System1"},
		{"Error - HTTP Method POST Not Allowed", setupNoSystemMockTest, "POST", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method PUT Not Allowed", setupNoSystemMockTest, "PUT", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method DELETE Not Allowed", setupNoSystemMockTest, "DELETE", http.StatusNotFound, nil, "System1"},
		{"Error - HTTP Method PATCH Not Allowed", setupNoSystemMockTest, "PATCH", http.StatusNotFound, nil, "System1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestSystemsHandler_GetSystemsCollection_WithLogger tests logging paths
func TestSystemsHandler_GetSystemsCollection_WithLogger(t *testing.T) {
	t.Parallel()

	// Test error logging path
	mockUseCase := new(MockComputerSystemUseCase)
	mockLogger := new(MockLogger)

	mockUseCase.On("GetAll", mock.Anything).Return([]string(nil), errSystemRepoFailure)
	mockLogger.On("Error", "Failed to retrieve computer systems collection", []interface{}{"error", errSystemRepoFailure}).Return()

	handler := NewSystemsHandler(mockUseCase, mockLogger)
	router := setupSystemsTestRouter(handler)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", systemsEndpointTest, http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestSystemsHandler_GetSystemByID_WithLogger tests logging paths
func TestSystemsHandler_GetSystemByID_WithLogger(t *testing.T) {
	t.Parallel()

	// Test error logging path
	mockUseCase := new(MockComputerSystemUseCase)
	mockLogger := new(MockLogger)

	mockUseCase.On("GetComputerSystem", mock.Anything, "System1").Return(nil, errSystemRepoFailure)
	mockLogger.On("Error", "Failed to retrieve computer system", []interface{}{"systemID", "System1", "error", errSystemRepoFailure}).Return()

	handler := NewSystemsHandler(mockUseCase, mockLogger)
	router := setupSystemByIDTestRouter(handler)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/redfish/v1/Systems/System1", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}
