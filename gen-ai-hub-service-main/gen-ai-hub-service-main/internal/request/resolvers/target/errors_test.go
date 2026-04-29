/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Error Handling Tests

func TestResolutionError(t *testing.T) {
	err := NewResolutionError("testStage", "test reason", "test details")
	assert.Equal(t, "testStage", err.Stage)
	assert.Equal(t, "test reason", err.Reason)
	assert.Equal(t, "test details", err.Details)
	assert.Contains(t, err.Error(), "testStage")
	assert.Contains(t, err.Error(), "test reason")
	assert.Contains(t, err.Error(), "test details")

	// Test without details
	err2 := NewResolutionError("testStage", "test reason", "")
	assert.NotContains(t, err2.Error(), "()")
}

func TestResolve_ModelNotFound(t *testing.T) {
	mapping := createTestMapping()
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	resolver.staticMapping = mapping

	c := createTestGinContext("POST", "/openai/deployments/nonexistent-model/chat/completions")
	_, err = resolver.Resolve(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}

func TestResolve_BuddyNotFound(t *testing.T) {
	mapping := createTestMapping()
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	resolver.staticMapping = mapping

	c := createTestGinContext("POST", "/v1/tenant123/buddies/nonexistent-buddy/question")
	_, err = resolver.Resolve(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "buddy not found")
}

func TestResolve_NoStaticMapping(t *testing.T) {
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)

	c := createTestGinContext("POST", "/openai/deployments/gpt-4o/chat/completions")
	_, err = resolver.Resolve(context.Background(), c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "static mapping not loaded")
}

// Edge Case Tests

func TestExtractBasicInfo_EmptyPath(t *testing.T) {
	resolver := &TargetResolver{}
	c := createTestGinContext("GET", "/")
	req := &ResolutionRequest{
		GinContext: c,
		Target:     &ResolvedTarget{},
		Metadata:   make(map[string]interface{}),
	}

	err := resolver.extractBasicInfo(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "", req.Metadata["routePattern"])
}

func TestExtractBasicInfo_WithComplexQueryParameters(t *testing.T) {
	resolver := &TargetResolver{}
	c := createTestGinContext("POST", "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-15&stream=true&temperature=0.7")
	req := &ResolutionRequest{
		GinContext: c,
		Target:     &ResolvedTarget{},
		Metadata:   make(map[string]interface{}),
	}

	err := resolver.extractBasicInfo(context.Background(), req)
	require.NoError(t, err)

	rawQuery := req.Metadata["rawQuery"].(string)
	assert.Contains(t, rawQuery, "api-version=2024-02-15")
	assert.Contains(t, rawQuery, "stream=true")
	assert.Contains(t, rawQuery, "temperature=0.7")
}
