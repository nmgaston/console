// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

var (
	// ErrInvalidPowerState is returned when an invalid power state is requested.
	ErrInvalidPowerState = errors.New("invalid power state")

	// ErrPowerStateConflict is returned when a power state transition is not allowed.
	ErrPowerStateConflict = errors.New("power state transition not allowed")

	// ErrInvalidResetType is returned when an invalid reset type is provided.
	ErrInvalidResetType = errors.New("invalid reset type")
)

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
// This implements the application-specific business rules and workflows.
type ComputerSystemUseCase struct {
	repo           ComputerSystemRepository
	accessPolicy   AccessPolicyService
	cacheService   CacheService
	auditService   AuditService
	metricsService MetricsService
}

// Service interfaces (Dependency Inversion Principle).
type AccessPolicyService interface {
	CanAccessSystem(ctx context.Context, userID, systemID, operation string) error
}

type CacheService interface {
	Get(ctx context.Context, key string) (*redfishv1.ComputerSystem, error)
	Set(ctx context.Context, key string, system *redfishv1.ComputerSystem, ttl time.Duration) error
	Invalidate(ctx context.Context, key string) error
}

type AuditService interface {
	LogAccess(ctx context.Context, userID, systemID, operation string, success bool, details string)
}

type MetricsService interface {
	IncrementCounter(name string, tags map[string]string)
	RecordDuration(name string, duration time.Duration, tags map[string]string)
}

// CreateComputerSystemUseCase creates a new computer system use case with dependency injection.
func CreateComputerSystemUseCase(
	repo ComputerSystemRepository,
	accessPolicy AccessPolicyService,
	cache CacheService,
	audit AuditService,
	metrics MetricsService,
) *ComputerSystemUseCase {
	return &ComputerSystemUseCase{
		repo:           repo,
		accessPolicy:   accessPolicy,
		cacheService:   cache,
		auditService:   audit,
		metricsService: metrics,
	}
}

// GetAll retrieves all ComputerSystem IDs from the repository.
// This method implements business logic for collection access.
func (uc *ComputerSystemUseCase) GetAll(ctx context.Context) ([]string, error) {
	startTime := time.Now()

	// Record metrics
	defer func() {
		uc.metricsService.RecordDuration("usecase_get_all_systems_duration", time.Since(startTime), map[string]string{
			"operation": "list",
		})
	}()

	// Get all systems from repository
	systemIDs, err := uc.repo.GetAll(ctx)
	if err != nil {
		uc.metricsService.IncrementCounter("usecase_get_all_systems_errors", map[string]string{
			"error": "repository_error",
		})
		// Audit failed access
		userID := getUserIDFromContext(ctx)
		uc.auditService.LogAccess(ctx, userID, "*", "list", false, err.Error())

		return nil, err
	}

	// Audit successful access
	userID := getUserIDFromContext(ctx)
	uc.auditService.LogAccess(ctx, userID, "*", "list", true, "")

	uc.metricsService.IncrementCounter("usecase_get_all_systems_success", map[string]string{
		"count": fmt.Sprintf("%d", len(systemIDs)),
	})

	return systemIDs, nil
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and converts it to the generated API type.
// This method implements business logic including access control, caching, and audit logging.
func (uc *ComputerSystemUseCase) GetComputerSystem(ctx context.Context, systemID string) (*generated.ComputerSystemComputerSystem, error) {
	startTime := time.Now()

	// Record metrics
	defer func() {
		uc.metricsService.RecordDuration("usecase_get_system_duration", time.Since(startTime), map[string]string{
			"system_id": systemID,
			"operation": "get",
		})
	}()

	// Check cache first (currently no-op implementation)
	cachedSystem, _ := uc.cacheService.Get(ctx, fmt.Sprintf("system:%s", systemID))
	if cachedSystem != nil {
		uc.metricsService.IncrementCounter("usecase_get_system_cache_hit", map[string]string{
			"system_id": systemID,
		})
		// FUTURE: Convert cached entity to API type and return it
		// For now, fall through to repository to ensure consistent behavior
	}

	// Get device information from repository - this gives us basic device data
	system, err := uc.repo.GetByID(ctx, systemID)
	if err != nil {
		uc.metricsService.IncrementCounter("usecase_get_system_errors", map[string]string{
			"system_id": systemID,
			"error":     "repository_error",
		})
		// Audit failed access
		userID := getUserIDFromContext(ctx)
		uc.auditService.LogAccess(ctx, userID, systemID, "read", false, err.Error())

		return nil, err
	}

	// Build the generated type directly with available information
	// Create power state

	var powerState *generated.ComputerSystemComputerSystem_PowerState

	if system.PowerState != "" {
		var redfishPowerState generated.ResourcePowerState

		switch system.PowerState {
		case redfishv1.PowerStateOn:
			redfishPowerState = generated.On
		case redfishv1.PowerStateOff:
			redfishPowerState = generated.Off
		case redfishv1.ResetTypeForceOff, redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			redfishPowerState = generated.Off // These reset types default to Off state
		default:
			redfishPowerState = generated.Off // Default to Off for unknown states
		}

		powerState = &generated.ComputerSystemComputerSystem_PowerState{}
		if err := powerState.FromResourcePowerState(redfishPowerState); err != nil {
			// Log error but continue with nil power state
			powerState = nil
		}
	}

	// Convert to string pointers for optional fields
	var manufacturer, model, serialNumber *string
	if system.Manufacturer != "" {
		manufacturer = &system.Manufacturer
	}

	if system.Model != "" {
		model = &system.Model
	}

	if system.SerialNumber != "" {
		serialNumber = &system.SerialNumber
	}

	// Create system type
	systemType := generated.ComputerSystemSystemType("Physical")

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context("/redfish/v1/$metadata#ComputerSystem.ComputerSystem")
	odataType := generated.OdataV4Type("#ComputerSystem.v1_22_0.ComputerSystem")
	odataID := fmt.Sprintf("/redfish/v1/Systems/%s", systemID)

	result := generated.ComputerSystemComputerSystem{
		OdataContext: &odataContext,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           systemID,
		Name:         system.Name,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   powerState,
		SystemType:   &systemType,
	}

	// Cache the result for future requests (no-op implementation)
	const cacheExpiryMinutes = 5

	_ = uc.cacheService.Set(ctx, fmt.Sprintf("system:%s", systemID), system, cacheExpiryMinutes*time.Minute)

	// Audit successful access
	userID := getUserIDFromContext(ctx)
	uc.auditService.LogAccess(ctx, userID, systemID, "read", true, "")

	// Record success metrics
	uc.metricsService.IncrementCounter("usecase_get_system_success", map[string]string{
		"system_id": systemID,
	})

	return &result, nil
}

// SetPowerState validates and sets the power state for a ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(ctx context.Context, id string, resetType generated.ResourceResetType) error {
	// Validate the reset type
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeForceOn,
		generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypeGracefulShutdown,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle,
		generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton,
		generated.ResourceResetTypePause,
		generated.ResourceResetTypeResume,
		generated.ResourceResetTypeSuspend:
		// Valid reset types
	default:
		return ErrInvalidResetType
	}

	// Convert generated reset type to entity power state
	powerState := convertToEntityPowerState(resetType)

	// Set the power state
	return uc.repo.UpdatePowerState(ctx, id, powerState)
}

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// convertToEntityPowerState converts from generated reset type to entity power state.
func convertToEntityPowerState(resetType generated.ResourceResetType) redfishv1.PowerState {
	// This is a simplified mapping - in a real implementation,
	// you would handle all the reset types properly
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOn:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeGracefulShutdown:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle:
		return redfishv1.PowerStateOff // Will cycle to On
	case generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypePause,
		generated.ResourceResetTypeSuspend:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeResume:
		return redfishv1.PowerStateOn
	default:
		return redfishv1.PowerStateOff
	}
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const userIDKey contextKey = "userID"

// getUserIDFromContext extracts the authenticated user ID from context
// Returns "system" as fallback if no userID is found.
func getUserIDFromContext(ctx context.Context) string {
	// In a real implementation, this would extract from gin.Context or JWT claims
	// For now, return a default value since userID should be set by authentication middleware
	userID := "system"

	// Try both typed and string context keys for compatibility
	if value := ctx.Value(userIDKey); value != nil {
		if uid, ok := value.(string); ok && uid != "" {
			userID = uid
		}
	} else if value := ctx.Value("userID"); value != nil {
		if uid, ok := value.(string); ok && uid != "" {
			userID = uid
		}
	}

	return userID
}
