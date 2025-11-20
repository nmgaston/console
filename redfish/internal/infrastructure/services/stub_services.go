// Package services provides infrastructure-layer service implementations.
// This implements cross-cutting concerns like caching, auditing, and metrics.
package services

import (
	"context"
	"time"

	"github.com/device-management-toolkit/console/pkg/logger"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

// NoOpAccessPolicyService provides a no-operation implementation of AccessPolicyService.
// In production, this should be replaced with proper RBAC implementation.
type NoOpAccessPolicyService struct {
	logger logger.Interface
}

// CreateAccessPolicyService creates a new no-op access policy service.
func CreateAccessPolicyService(log logger.Interface) *NoOpAccessPolicyService {
	return &NoOpAccessPolicyService{logger: log}
}

// CanAccessSystem always allows access (no-op implementation).
func (s *NoOpAccessPolicyService) CanAccessSystem(_ context.Context, userID, systemID, operation string) error {
	s.logger.Debug("Access granted (no-op policy)",
		"userID", userID,
		"systemID", systemID,
		"operation", operation)

	return nil
}

// NoOpCacheService provides a no-operation implementation of CacheService.
type NoOpCacheService struct {
	logger logger.Interface
}

// CreateCacheService creates a new no-op cache service.
func CreateCacheService(log logger.Interface) *NoOpCacheService {
	return &NoOpCacheService{logger: log}
}

// Get always returns cache miss (no-op implementation).
func (s *NoOpCacheService) Get(_ context.Context, key string) (*redfishv1.ComputerSystem, error) {
	s.logger.Debug("Cache miss (no-op cache)", "key", key)

	return nil, nil // Cache miss
}

// Set does nothing (no-op implementation).
func (s *NoOpCacheService) Set(_ context.Context, key string, _ *redfishv1.ComputerSystem, ttl time.Duration) error {
	s.logger.Debug("Cache set ignored (no-op cache)",
		"key", key,
		"ttl", ttl)

	return nil
}

// Invalidate does nothing (no-op implementation).
func (s *NoOpCacheService) Invalidate(_ context.Context, key string) error {
	s.logger.Debug("Cache invalidate ignored (no-op cache)", "key", key)

	return nil
}

// NoOpAuditService provides a no-operation implementation of AuditService.
type NoOpAuditService struct {
	logger logger.Interface
}

// CreateAuditService creates a new no-op audit service.
func CreateAuditService(log logger.Interface) *NoOpAuditService {
	return &NoOpAuditService{logger: log}
}

// LogAccess logs access attempts (implementation via logger).
func (s *NoOpAuditService) LogAccess(_ context.Context, userID, systemID, operation string, success bool, details string) {
	if success {
		s.logger.Info("Access audit log",
			"userID", userID,
			"systemID", systemID,
			"operation", operation,
			"success", success)
	} else {
		s.logger.Warn("Access audit log - FAILED",
			"userID", userID,
			"systemID", systemID,
			"operation", operation,
			"success", success,
			"error", details)
	}
}

// NoOpMetricsService provides a no-operation implementation of MetricsService.
type NoOpMetricsService struct {
	logger logger.Interface
}

// CreateMetricsService creates a new no-op metrics service.
func CreateMetricsService(log logger.Interface) *NoOpMetricsService {
	return &NoOpMetricsService{logger: log}
}

// IncrementCounter logs metric increment (no-op implementation).
func (s *NoOpMetricsService) IncrementCounter(name string, tags map[string]string) {
	s.logger.Debug("Metric counter increment (no-op)",
		"metric", name,
		"tags", tags)
}

// RecordDuration logs duration metric (no-op implementation).
func (s *NoOpMetricsService) RecordDuration(name string, duration time.Duration, tags map[string]string) {
	s.logger.Debug("Metric duration recorded (no-op)",
		"metric", name,
		"duration", duration,
		"tags", tags)
}
