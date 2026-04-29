/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"context"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// runtimeConfigKey is the context key for storing RuntimeConfig
	runtimeConfigKey contextKey = "runtimeConfig"
)

// ServiceMode represents the runtime service mode
type ServiceMode string

const (
	ServiceModeNormal    ServiceMode = "NORMAL"
	ServiceModeReadOnly  ServiceMode = "READONLY"
	ServiceModeEmulation ServiceMode = "EMULATION"
)

// RuntimeConfig holds configuration that can be modified at runtime via headers
type RuntimeConfig struct {
	ForceFreshDbMetrics bool        `json:"force_fresh_db_metrics"`
	ServiceMode         ServiceMode `json:"service_mode"`
	// Future configurations can be added here
}

// NewRuntimeConfig creates a new RuntimeConfig with default values
func NewRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		ForceFreshDbMetrics: false,
		ServiceMode:         ServiceModeNormal,
	}
}

// ParseServiceMode parses a string into a ServiceMode, returning ServiceModeNormal for invalid values
func ParseServiceMode(mode string) ServiceMode {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "READONLY":
		return ServiceModeReadOnly
	case "EMULATION":
		return ServiceModeEmulation
	case "NORMAL":
		return ServiceModeNormal
	default:
		return ServiceModeNormal
	}
}

// String returns the string representation of ServiceMode
func (sm ServiceMode) String() string {
	return string(sm)
}

// IsValid checks if the ServiceMode is valid
func (sm ServiceMode) IsValid() bool {
	switch sm {
	case ServiceModeNormal, ServiceModeReadOnly, ServiceModeEmulation:
		return true
	default:
		return false
	}
}

// GetRuntimeConfigFromContext retrieves RuntimeConfig from context
func GetRuntimeConfigFromContext(ctx context.Context) *RuntimeConfig {
	if ctx == nil {
		return nil
	}

	if config, ok := ctx.Value(runtimeConfigKey).(*RuntimeConfig); ok {
		return config
	}

	return nil
}

// WithRuntimeConfig adds RuntimeConfig to context
func WithRuntimeConfig(ctx context.Context, config *RuntimeConfig) context.Context {
	return context.WithValue(ctx, runtimeConfigKey, config)
}
