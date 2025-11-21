package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

// Test constants
const (
	computerSystemODataType = "#ComputerSystem.v1_22_0.ComputerSystem"
	systemsEndpoint         = "/redfish/v1/Systems"
)

// Type declarations
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
	router.GET(systemsEndpoint, server.GetRedfishV1Systems)

	return router
}

// setupTestRouterForSystemByID sets up a test router with the system by ID endpoint
func setupTestRouterForSystemByID(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Register the system by ID endpoint with a wrapper function
	router.GET("/redfish/v1/Systems/:ComputerSystemId", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
	})

	return router
}

// ====================================================================================================
// MAIN TEST FUNCTIONS
// ====================================================================================================

// TestGetRedfishV1Systems provides minimal framework test for GetRedfishV1Systems endpoint
// Detailed testing is handled in systems_test.go
func TestGetRedfishV1Systems(t *testing.T) {
	t.Parallel()

	// Setup
	mockUseCase := new(MockComputerSystemUseCase)
	mockUseCase.On("GetAll", mock.Anything).Return([]string{"System1"}, nil)

	systemsHandler := NewSystemsHandler(mockUseCase, nil)
	server := &RedfishServer{
		SystemsHandler: systemsHandler,
	}

	router := setupTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", systemsEndpoint, http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert basic functionality
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockUseCase.AssertExpectations(t)
}

// TestGetRedfishV1SystemsComputerSystemId provides minimal framework test for GetRedfishV1SystemsComputerSystemId endpoint
// Detailed testing is handled in systems_test.go
func TestGetRedfishV1SystemsComputerSystemId(t *testing.T) {
	t.Parallel()

	// Setup
	mockUseCase := new(MockComputerSystemUseCase)
	system := createTestSystemData("System1", "Test System", "Test Manufacturer", "Test Model", "SN123456")
	mockUseCase.On("GetComputerSystem", mock.Anything, "System1").Return(system, nil)

	systemsHandler := NewSystemsHandler(mockUseCase, nil)
	server := &RedfishServer{
		SystemsHandler: systemsHandler,
	}

	router := setupTestRouterForSystemByID(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/redfish/v1/Systems/System1", http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert basic functionality
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	mockUseCase.AssertExpectations(t)
}
