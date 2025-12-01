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
	systemODataType                   = "#ComputerSystem.v1_26_0.ComputerSystem"
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

func (r *TestSystemsComputerSystemRepository) UpdatePowerState(_ context.Context, systemID string, state redfishv1.PowerState) error {
	if system, exists := r.systems[systemID]; exists {
		system.PowerState = state

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

// setupIntelOEMTestRouter sets up router for testing Intel OEM endpoints
func setupIntelOEMTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// Intel OEM PowerState endpoint
	router.GET("/redfish/v1/Systems/:ComputerSystemId/Oem/Intel/PowerState", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemIdOemIntelPowerState(c, computerSystemID)
	})
	// Intel OEM PowerCapabilities endpoint
	router.GET("/redfish/v1/Systems/:ComputerSystemId/Oem/Intel/PowerCapabilities", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemIdOemIntelPowerCapabilities(c, computerSystemID)
	})
	// Main system endpoint for reference validation
	router.GET("/redfish/v1/Systems/:ComputerSystemId", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
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

// createTestSystemEntityDataWithAllProperties creates a test system entity with Description, UUID, HostName, BiosVersion, and Status populated
func createTestSystemEntityDataWithAllProperties(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		Description:  "Computer System managed by Intel AMT",
		UUID:         "12345678-1234-1234-1234-" + systemID + "000",
		HostName:     "amt-" + systemID + ".example.com",
		BiosVersion:  "BIOS.Version.2.1.0",
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   redfishv1.PowerStateOn,
		Status: &redfishv1.Status{
			State:  "Enabled",
			Health: "OK",
		},
	}
}

// createTestSystemEntityDataMinimal creates a test system entity with minimal properties to test CIM enrichment
func createTestSystemEntityDataMinimal(systemID, name string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		Manufacturer: "Intel Corporation",
		Model:        "Test Model",
		SerialNumber: "MIN001",
		PowerState:   redfishv1.PowerStateOff,
		// Missing: Description, UUID, HostName, BiosVersion, Status - should be enriched by CIM data
	}
}

// Repository setup functions for system by ID tests
func setupExistingSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityData(systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
	repo.AddSystem(systemID, system)
}

// setupSystemWithAllPropertiesMockTest sets up a system with Description, UUID, HostName, BiosVersion, and Status populated
func setupSystemWithAllPropertiesMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithAllProperties(systemID, "Enhanced Test System", "Intel Corporation", "vPro Test Model", "ENH123456")
	repo.AddSystem(systemID, system)
}

// setupMinimalSystemMockTest sets up a system with minimal properties to test CIM enrichment
func setupMinimalSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataMinimal(systemID, "Minimal Test System")
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

// validateSystemResponseWithAllPropertiesTest validates system response with Description, UUID, HostName, BiosVersion, Status, and Intel OEM properties
func validateSystemResponseWithAllPropertiesTest(t *testing.T, w *httptest.ResponseRecorder, systemID, expectedName, expectedDescription, expectedUUID, expectedHostName, expectedBiosVersion string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate basic properties
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, expectedName, response.Name)

	// Validate Description property
	if expectedDescription != "" {
		assert.NotNil(t, response.Description)
		description, err := response.Description.AsResourceDescription()
		assert.NoError(t, err)
		assert.Equal(t, expectedDescription, description)
	}

	// Validate UUID property
	if expectedUUID != "" {
		assert.NotNil(t, response.UUID)
		uuid, err := response.UUID.AsResourceUUID()
		assert.NoError(t, err)
		assert.Equal(t, expectedUUID, uuid)
	}

	// Validate HostName property
	if expectedHostName != "" {
		assert.NotNil(t, response.HostName)
		assert.Equal(t, expectedHostName, *response.HostName)
	}

	// Validate BiosVersion property
	if expectedBiosVersion != "" {
		assert.NotNil(t, response.BiosVersion)
		assert.Equal(t, expectedBiosVersion, *response.BiosVersion)
	}

	// Validate Status if present
	if response.Status != nil {
		assert.NotNil(t, response.Status.State)
		assert.NotNil(t, response.Status.Health)

		state, err := response.Status.State.AsResourceStatusState1()
		assert.NoError(t, err)
		assert.Equal(t, "Enabled", state)

		health, err := response.Status.Health.AsResourceStatusHealth1()
		assert.NoError(t, err)
		assert.Equal(t, "OK", health)
	}

	// Validate Intel OEM extension
	assert.NotNil(t, response.Oem)
	oem := *response.Oem
	assert.Contains(t, oem, "Intel")

	intelOem, ok := oem["Intel"].(map[string]interface{})
	assert.True(t, ok)

	// Validate PowerState OEM reference
	assert.Contains(t, intelOem, "PowerState")
	powerStateRef, ok := intelOem["PowerState"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerState", systemID), powerStateRef["@odata.id"])

	// Validate PowerCapabilities OEM reference
	assert.Contains(t, intelOem, "PowerCapabilities")
	powerCapabilitiesRef, ok := intelOem["PowerCapabilities"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerCapabilities", systemID), powerCapabilitiesRef["@odata.id"])

	// Validate OData fields
	assert.NotNil(t, response.OdataContext)
	assert.Equal(t, "/redfish/v1/$metadata#ComputerSystem.ComputerSystem", *response.OdataContext)
	assert.NotNil(t, response.OdataId)
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s", systemID), *response.OdataId)
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

// validateSystemWithAllPropertiesResponseTest validates system response with Description, UUID, HostName, BiosVersion, Status, and Intel OEM properties
func validateSystemWithAllPropertiesResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseWithAllPropertiesTest(t, w, systemID, "Enhanced Test System",
		"Computer System managed by Intel AMT",
		"12345678-1234-1234-1234-"+systemID+"000",
		"amt-"+systemID+".example.com",
		"BIOS.Version.2.1.0")
}

// validateMinimalSystemResponseTest validates system response with CIM-enriched properties
func validateMinimalSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate that CIM enrichment occurred
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, "Minimal Test System", response.Name)

	// Validate CIM-enriched Description
	assert.NotNil(t, response.Description)
	description, err := response.Description.AsResourceDescription()
	assert.NoError(t, err)
	assert.Equal(t, "Computer System managed by Intel AMT", description)

	// Validate UUID - should be nil if not available from CIM data (no longer use systemID fallback)
	assert.Nil(t, response.UUID)

	// Validate CIM-enriched HostName
	assert.NotNil(t, response.HostName)
	assert.Contains(t, *response.HostName, "amt-system-")

	// Validate CIM-enriched BiosVersion
	assert.NotNil(t, response.BiosVersion)
	assert.Equal(t, "BIOS.Version.1.0.0", *response.BiosVersion)

	// Validate Intel OEM extension is present
	assert.NotNil(t, response.Oem)
	oem := *response.Oem
	assert.Contains(t, oem, "Intel")
}

// validateIntelOEMPowerStateResponseTest validates Intel OEM PowerState response
func validateIntelOEMPowerStateResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response map[string]interface{}
	unmarshalJSONResponseTest(t, w, &response)

	// Validate OData fields
	assert.Equal(t, "#Intel.v1_0_0.PowerState", response["@odata.type"])
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerState", systemID), response["@odata.id"])
	assert.Equal(t, "PowerState", response["Id"])
	assert.Equal(t, "Intel Power State", response["Name"])
	assert.Equal(t, "Intel-specific Power State Information", response["Description"])

	// Validate PowerState field exists
	assert.Contains(t, response, "PowerState")
}

// validateIntelOEMPowerCapabilitiesResponseTest validates Intel OEM PowerCapabilities response
func validateIntelOEMPowerCapabilitiesResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response map[string]interface{}
	unmarshalJSONResponseTest(t, w, &response)

	// Validate OData fields
	assert.Equal(t, "#Intel.v1_0_0.PowerCapabilities", response["@odata.type"])
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerCapabilities", systemID), response["@odata.id"])
	assert.Equal(t, "PowerCapabilities", response["Id"])
	assert.Equal(t, "Intel Power Capabilities", response["Name"])
	assert.Equal(t, "Intel-specific Power Management Capabilities", response["Description"])

	// Validate power capability fields
	assert.Contains(t, response, "MaxPowerConsumptionWatts")
	assert.Contains(t, response, "MinPowerConsumptionWatts")

	// Validate power values are reasonable
	maxPower, ok := response["MaxPowerConsumptionWatts"].(float64)
	assert.True(t, ok)
	assert.True(t, maxPower > 0)

	minPower, ok := response["MinPowerConsumptionWatts"].(float64)
	assert.True(t, ok)
	assert.True(t, minPower >= 0)
	assert.True(t, maxPower >= minPower)
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
		{"Success - System with All Properties", setupSystemWithAllPropertiesMockTest, "GET", http.StatusOK, validateSystemWithAllPropertiesResponseTest, "enhanced-system-1"},
		{"Success - Minimal System with CIM Enrichment", setupMinimalSystemMockTest, "GET", http.StatusOK, validateMinimalSystemResponseTest, "minimal-system-1"},
		{"Success - Long System ID for UUID/HostName Truncation", setupMinimalSystemMockTest, "GET", http.StatusOK, validateMinimalSystemResponseTest, "very-long-system-identifier-that-exceeds-character-limits"},
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

// TestSystemsHandler_IntelOEMPowerState tests Intel OEM PowerState endpoint
func TestSystemsHandler_IntelOEMPowerState(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[string]{
		endpoint:    "/redfish/v1/Systems/{id}/Oem/Intel/PowerState",
		routerSetup: setupIntelOEMTestRouter,
		urlBuilder: func(systemID string) string {
			return fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerState", systemID)
		},
	}

	tests := []SystemsTestCase[string]{
		{"Success - Intel OEM PowerState", setupSystemWithAllPropertiesMockTest, "GET", http.StatusOK, validateIntelOEMPowerStateResponseTest, "intel-system-1"},
		{"Success - Intel OEM PowerState UUID System", setupUUIDSystemMockTest, "GET", http.StatusOK, validateIntelOEMPowerStateResponseTest, "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"},
		{"Error - Empty System ID", setupNoSystemMockTest, "GET", http.StatusBadRequest, nil, ""},
		{"Error - System Not Found", setupSystemNotFoundMockTest, "GET", http.StatusNotFound, nil, "NonExistentSystem"},
		{"Error - HTTP Method POST Not Allowed", setupNoSystemMockTest, "POST", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method PUT Not Allowed", setupNoSystemMockTest, "PUT", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method DELETE Not Allowed", setupNoSystemMockTest, "DELETE", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method PATCH Not Allowed", setupNoSystemMockTest, "PATCH", http.StatusNotFound, nil, "intel-system-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestSystemsHandler_IntelOEMPowerCapabilities tests Intel OEM PowerCapabilities endpoint
func TestSystemsHandler_IntelOEMPowerCapabilities(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[string]{
		endpoint:    "/redfish/v1/Systems/{id}/Oem/Intel/PowerCapabilities",
		routerSetup: setupIntelOEMTestRouter,
		urlBuilder: func(systemID string) string {
			return fmt.Sprintf("/redfish/v1/Systems/%s/Oem/Intel/PowerCapabilities", systemID)
		},
	}

	tests := []SystemsTestCase[string]{
		{"Success - Intel OEM PowerCapabilities", setupSystemWithAllPropertiesMockTest, "GET", http.StatusOK, validateIntelOEMPowerCapabilitiesResponseTest, "intel-system-1"},
		{"Success - Intel OEM PowerCapabilities UUID System", setupUUIDSystemMockTest, "GET", http.StatusOK, validateIntelOEMPowerCapabilitiesResponseTest, "b4c3a390-468c-491f-8e1d-9ce04c2fcbc1"},
		{"Error - Empty System ID", setupNoSystemMockTest, "GET", http.StatusBadRequest, nil, ""},
		{"Error - System Not Found", setupSystemNotFoundMockTest, "GET", http.StatusNotFound, nil, "NonExistentSystem"},
		{"Error - HTTP Method POST Not Allowed", setupNoSystemMockTest, "POST", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method PUT Not Allowed", setupNoSystemMockTest, "PUT", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method DELETE Not Allowed", setupNoSystemMockTest, "DELETE", http.StatusNotFound, nil, "intel-system-1"},
		{"Error - HTTP Method PATCH Not Allowed", setupNoSystemMockTest, "PATCH", http.StatusNotFound, nil, "intel-system-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestGetPowerStateString tests the getPowerStateString function edge cases
func TestGetPowerStateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		powerState     *generated.ComputerSystemComputerSystem_PowerState
		expectedResult string
	}{
		{
			name:           "Nil power state",
			powerState:     nil,
			expectedResult: "Unknown",
		},
		{
			name: "Valid power state - On",
			powerState: func() *generated.ComputerSystemComputerSystem_PowerState {
				ps := &generated.ComputerSystemComputerSystem_PowerState{}
				_ = ps.FromResourcePowerState("On")

				return ps
			}(),
			expectedResult: "On",
		},
		{
			name: "Valid power state - Off",
			powerState: func() *generated.ComputerSystemComputerSystem_PowerState {
				ps := &generated.ComputerSystemComputerSystem_PowerState{}
				_ = ps.FromResourcePowerState("Off")

				return ps
			}(),
			expectedResult: "Off",
		},
		{
			name: "Invalid power state - fallback to Unknown",
			powerState: func() *generated.ComputerSystemComputerSystem_PowerState {
				// Create a power state that cannot be converted to ResourcePowerState
				ps := &generated.ComputerSystemComputerSystem_PowerState{}
				// This will create an empty union that fails AsResourcePowerState()
				return ps
			}(),
			expectedResult: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getPowerStateString(tt.powerState)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
