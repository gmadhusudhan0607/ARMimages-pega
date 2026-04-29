/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

// Package cntxtest provides utilities for testing code that uses cntx.
// It follows the Go standard library pattern (e.g., net/http/httptest).
package cntxtest

import (
	"context"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

// NewContext creates a test context with default values for testing.
// This allows tests to run in parallel without environment variable conflicts.
//
// Example:
//
//	ctx := cntxtest.NewContext("test")
//	ctx = cntxtest.WithSaxConfigPath(ctx, "/custom/path")
func NewContext(name string) context.Context {
	return cntx.NewTestContext(name)
}

// WithSaxConfigPath sets the SAX config path in a test context.
func WithSaxConfigPath(ctx context.Context, path string) context.Context {
	return cntx.WithSaxConfigPath(ctx, path)
}

// WithUseGenAIInfra sets the useGenAIInfra flag in a test context.
func WithUseGenAIInfra(ctx context.Context, useGenAiInfra bool) context.Context {
	return cntx.WithUseGenAIInfra(ctx, useGenAiInfra)
}

// WithAzureGenAIURL sets the Azure GenAI URL in a test context.
func WithAzureGenAIURL(ctx context.Context, url string) context.Context {
	return cntx.WithAzureGenAIURL(ctx, url)
}

// WithLogger replaces the logger stored in a test context.
// Useful for capturing log output with zaptest/observer.
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return cntx.ContextWithLogger(ctx, l)
}
