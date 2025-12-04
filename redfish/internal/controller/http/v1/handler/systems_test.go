package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
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

// TestSystemsComputerSystemRepository is a test implementation for systems tests
type TestSystemsComputerSystemRepository struct {
	systems        map[string]*redfishv1.ComputerSystem
	errorOnGetAll  bool
	errorOnGetByID map[string]error
}

func NewTestSystemsComputerSystemRepository() *TestSystemsComputerSystemRepository {
	return &TestSystemsComputerSystemRepository{
		systems:        make(map[string]*redfishv1.ComputerSystem),
		errorOnGetByID: make(map[string]error),
	}
}

func (r *TestSystemsComputerSystemRepository) AddSystem(id string, system *redfishv1.ComputerSystem) {
	r.systems[id] = system
}

func (r *TestSystemsComputerSystemRepository) SetErrorOnGetAll(err bool) {
	r.errorOnGetAll = err
}

func (r *TestSystemsComputerSystemRepository) SetErrorOnGetByID(systemID string, err error) {
	r.errorOnGetByID[systemID] = err
}

func (r *TestSystemsComputerSystemRepository) GetAll(_ context.Context) ([]string, error) {
	if r.errorOnGetAll {
		return nil, errSystemRepoFailure
	}

	ids := make([]string, 0, len(r.systems))
	for id := range r.systems {
		if id != "" {
			ids = append(ids, id)
		}
	}
	// Sort to ensure consistent ordering for tests
	sort.Strings(ids)

	return ids, nil
}

func (r *TestSystemsComputerSystemRepository) GetByID(_ context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	if err, exists := r.errorOnGetByID[systemID]; exists {
		return nil, err
	}

	if system, exists := r.systems[systemID]; exists {
		return system, nil
	}

	return nil, usecase.ErrSystemNotFound
}

func (r *TestSystemsComputerSystemRepository) UpdatePowerState(_ context.Context, systemID string, resetType redfishv1.PowerState) error {
	if system, exists := r.systems[systemID]; exists {
		// Convert reset types to final power state for testing
		switch resetType {
		case redfishv1.ResetTypeOn: // Covers both ResetTypeOn and PowerStateOn (same value "On")
			system.PowerState = redfishv1.PowerStateOn
		case redfishv1.ResetTypeForceOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.PowerStateOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			// These reset types cycle power, for test purposes we set to Off
			// (in reality the system would restart)
			system.PowerState = redfishv1.PowerStateOff
		}

		return nil
	}

	return usecase.ErrSystemNotFound
}

// TestCase represents a generic test case structure
type SystemsTestCase[T any] struct {
	name           string
	setupRepo      func(*TestSystemsComputerSystemRepository, T)
	httpMethod     string
	expectedStatus int
	validateFunc   func(*testing.T, *httptest.ResponseRecorder, T)
	params         T
}

// TestConfig holds configuration for running tests
type SystemsTestConfig[T any] struct {
	endpoint    string
	routerSetup func(*RedfishServer) *gin.Engine
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

func setupSystemsTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(systemsEndpointTest, server.GetRedfishV1Systems)

	return router
}

func setupSystemByIDTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/redfish/v1/Systems/:ComputerSystemId", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
	})
	// Add route for empty system ID case (trailing slash)
	router.GET("/redfish/v1/Systems/", func(c *gin.Context) {
		server.GetRedfishV1SystemsComputerSystemId(c, "")
	})

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
		fmt.Sprintf("%s/System2", systemsEndpointTest),
		fmt.Sprintf("%s/abc-123", systemsEndpointTest),
	})
}

func validateSingleSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 1, []string{fmt.Sprintf("%s/System1", systemsEndpointTest)})
}

func validateLargeCollectionResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()

	// Create expected members in alphabetical order (as returned by sorted repository)
	expectedMembers := []string{
		fmt.Sprintf("%s/System1", systemsEndpointTest),
		fmt.Sprintf("%s/System10", systemsEndpointTest),
		fmt.Sprintf("%s/System2", systemsEndpointTest),
		fmt.Sprintf("%s/System3", systemsEndpointTest),
		fmt.Sprintf("%s/System4", systemsEndpointTest),
		fmt.Sprintf("%s/System5", systemsEndpointTest),
		fmt.Sprintf("%s/System6", systemsEndpointTest),
		fmt.Sprintf("%s/System7", systemsEndpointTest),
		fmt.Sprintf("%s/System8", systemsEndpointTest),
		fmt.Sprintf("%s/System9", systemsEndpointTest),
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

// Repository setup functions for Systems collection
func setupMultipleSystemsMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem("System1", &redfishv1.ComputerSystem{ID: "System1", Name: "System 1", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem("System2", &redfishv1.ComputerSystem{ID: "System2", Name: "System 2", PowerState: redfishv1.PowerStateOn})
}

func setupEmptyCollectionMockTest(_ *TestSystemsComputerSystemRepository, _ struct{}) {
	// No systems added - empty collection
}

func setupFilteredSystemsMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem("System1", &redfishv1.ComputerSystem{ID: "System1", Name: "System 1", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem("", &redfishv1.ComputerSystem{ID: "", Name: "Empty ID", PowerState: redfishv1.PowerStateOn}) // This will be filtered out
	repo.AddSystem("abc-123", &redfishv1.ComputerSystem{ID: "abc-123", Name: "ABC System", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem("System2", &redfishv1.ComputerSystem{ID: "System2", Name: "System 2", PowerState: redfishv1.PowerStateOn})
}

func setupSingleSystemMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem("System1", &redfishv1.ComputerSystem{ID: "System1", Name: "System 1", PowerState: redfishv1.PowerStateOn})
}

func setupLargeCollectionMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	for i := 1; i <= 10; i++ {
		systemID := fmt.Sprintf("System%d", i)
		repo.AddSystem(systemID, &redfishv1.ComputerSystem{ID: systemID, Name: fmt.Sprintf("System %d", i), PowerState: redfishv1.PowerStateOn})
	}
}

func setupErrorMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.SetErrorOnGetAll(true)
}

func setupNoMockTest(_ *TestSystemsComputerSystemRepository, _ struct{}) {
	// No setup needed for HTTP method tests
}

// Generic Test Framework for Systems endpoints

// runGenericSystemsTest executes a generic test case
func runGenericSystemsTest[T any](t *testing.T, testCase SystemsTestCase[T], config SystemsTestConfig[T]) {
	t.Helper()

	// Setup
	testRepo := NewTestSystemsComputerSystemRepository()
	testCase.setupRepo(testRepo, testCase.params)

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	// Create RedfishServer
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
}

// createTestSystemEntityData creates a test system entity for repository
func createTestSystemEntityData(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   redfishv1.PowerStateOn,
	}
}

// Repository setup functions for system by ID tests
func setupExistingSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityData(systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
	repo.AddSystem(systemID, system)
}

func setupUUIDSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityData(systemID, "UUID System", "UUID Manufacturer", "UUID Model", "UUID-SN789")
	repo.AddSystem(systemID, system)
}

func setupSystemNotFoundMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	repo.SetErrorOnGetByID(systemID, usecase.ErrSystemNotFound)
}

func setupSystemRepositoryErrorMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	repo.SetErrorOnGetByID(systemID, errSystemRepoFailure)
}

func setupNoSystemMockTest(_ *TestSystemsComputerSystemRepository, _ string) {
	// No setup needed for HTTP method tests
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

// TestLogger for testing logger code paths - captures log calls
type TestLogger struct {
	DebugCalls [][]interface{}
	InfoCalls  [][]interface{}
	WarnCalls  [][]interface{}
	ErrorCalls [][]interface{}
	FatalCalls [][]interface{}
}

func NewTestLogger() *TestLogger {
	return &TestLogger{
		DebugCalls: make([][]interface{}, 0),
		InfoCalls:  make([][]interface{}, 0),
		WarnCalls:  make([][]interface{}, 0),
		ErrorCalls: make([][]interface{}, 0),
		FatalCalls: make([][]interface{}, 0),
	}
}

func (l *TestLogger) Debug(message interface{}, args ...interface{}) {
	l.DebugCalls = append(l.DebugCalls, append([]interface{}{message}, args...))
}

func (l *TestLogger) Info(msg string, args ...interface{}) {
	l.InfoCalls = append(l.InfoCalls, append([]interface{}{msg}, args...))
}

func (l *TestLogger) Warn(msg string, args ...interface{}) {
	l.WarnCalls = append(l.WarnCalls, append([]interface{}{msg}, args...))
}

func (l *TestLogger) Error(message interface{}, args ...interface{}) {
	l.ErrorCalls = append(l.ErrorCalls, append([]interface{}{message}, args...))
}

func (l *TestLogger) Fatal(message interface{}, args ...interface{}) {
	l.FatalCalls = append(l.FatalCalls, append([]interface{}{message}, args...))
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
	testRepo := NewTestSystemsComputerSystemRepository()
	testRepo.SetErrorOnGetAll(true)

	testLogger := NewTestLogger()

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
		Logger:           testLogger,
	}
	router := setupSystemsTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", systemsEndpointTest, http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Verify logger was called
	assert.Len(t, testLogger.ErrorCalls, 1)
	assert.Equal(t, "Failed to retrieve computer systems collection", testLogger.ErrorCalls[0][0])
}

// TestSystemsHandler_GetSystemByID_WithLogger tests logging paths
func TestSystemsHandler_GetSystemByID_WithLogger(t *testing.T) {
	t.Parallel()

	// Test error logging path
	testRepo := NewTestSystemsComputerSystemRepository()
	testRepo.SetErrorOnGetByID("System1", errSystemRepoFailure)

	testLogger := NewTestLogger()

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
		Logger:           testLogger,
	}
	router := setupSystemByIDTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/redfish/v1/Systems/System1", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Verify logger was called
	assert.Len(t, testLogger.ErrorCalls, 1)
	assert.Equal(t, "Failed to retrieve computer system", testLogger.ErrorCalls[0][0])
}
