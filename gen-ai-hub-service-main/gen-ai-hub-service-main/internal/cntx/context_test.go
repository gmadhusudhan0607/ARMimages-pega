/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package cntx

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func Test_IsUseGenAiInfraModels(t *testing.T) {
	os.Setenv("USE_GENAI_INFRA", "true") //nolint:errcheck
	ctx := ServiceContext("test_genai_infra")
	os.Unsetenv("USE_GENAI_INFRA") //nolint:errcheck
	result := IsUseGenAiInfraModels(ctx)
	assert.True(t, result)
}

func Test_DefaultContext(t *testing.T) {
	ctx := ServiceContext("test_genai_infra")
	useGenAIInfra := IsUseGenAiInfraModels(ctx)
	assert.False(t, useGenAIInfra)
	assert.False(t, IsUseSax(ctx))
	assert.True(t, IsInfinityPlatform(ctx))
	assert.False(t, IsLaunchpadPlatform(ctx))
}

func Test_IsUseAzureGenAIURL(t *testing.T) {
	// Test with Azure GenAI URL enabled
	ctx := NewTestContext("test_azure_genai")
	ctx = context.WithValue(ctx, useAzureGenAIURLKey, true)
	assert.True(t, IsUseAzureGenAIURL(ctx))

	// Test with Azure GenAI URL disabled
	ctx = context.WithValue(ctx, useAzureGenAIURLKey, false)
	assert.False(t, IsUseAzureGenAIURL(ctx))

	// Test with no Azure GenAI URL value in context
	ctx = NewTestContext("test_azure_genai")
	assert.False(t, IsUseAzureGenAIURL(ctx))
}

func Test_AzureGenAIURL(t *testing.T) {
	// Test with Azure GenAI URL set
	expectedURL := "https://test-azure-endpoint.com"
	ctx := NewTestContext("test_azure_genai")
	ctx = context.WithValue(ctx, azureGenAIKey, expectedURL)
	assert.Equal(t, expectedURL, AzureGenAIURL(ctx))

	// Test with empty Azure GenAI URL
	ctx = context.WithValue(ctx, azureGenAIKey, "")
	assert.Equal(t, "", AzureGenAIURL(ctx))

	// Test with no Azure GenAI URL value in context
	ctx = NewTestContext("test_azure_genai")
	assert.Equal(t, "", AzureGenAIURL(ctx))
}

func Test_IsUseGCPVertex(t *testing.T) {
	// Test with GCP Vertex URL enabled
	ctx := NewTestContext("test_gcp_vertex")
	ctx = context.WithValue(ctx, useGCPVertexKey, true)
	assert.True(t, IsUseGCPVertex(ctx))

	// Test with GCP Vertex URL disabled
	ctx = context.WithValue(ctx, useGCPVertexKey, false)
	assert.False(t, IsUseGCPVertex(ctx))

	// Test with no GCP Vertex value in context
	ctx = NewTestContext("test_gcp_vertex")
	assert.False(t, IsUseGCPVertex(ctx))
}
func TestContextWithGinContext(t *testing.T) {
	t.Run("Store and retrieve gin context", func(t *testing.T) {
		// Create base context
		baseCtx := context.Background()

		// Create test gin context
		w := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(w)

		// Store gin context
		ctx := ContextWithGinContext(baseCtx, ginCtx)

		// Verify context is not nil
		assert.NotNil(t, ctx)

		// Retrieve and verify gin context
		retrievedGinCtx := ctx.Value(ginContextKey)
		assert.NotNil(t, retrievedGinCtx)
		assert.Equal(t, ginCtx, retrievedGinCtx)
	})

	t.Run("Store nil gin context", func(t *testing.T) {
		// Create base context
		baseCtx := context.Background()

		// Store nil gin context
		ctx := ContextWithGinContext(baseCtx, nil)

		// Verify context is not nil
		assert.NotNil(t, ctx)

		// Retrieve and verify gin context is nil
		retrievedGinCtx := ctx.Value(ginContextKey)
		assert.Nil(t, retrievedGinCtx)
	})
}

func TestGetGinContext(t *testing.T) {
	t.Run("Get existing gin context", func(t *testing.T) {
		// Create base context
		baseCtx := context.Background()

		// Create test gin context
		w := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(w)

		// Store gin context
		ctx := ContextWithGinContext(baseCtx, ginCtx)

		// Retrieve and verify gin context
		retrievedGinCtx := GetGinContext(ctx)
		assert.NotNil(t, retrievedGinCtx)
		assert.Equal(t, ginCtx, retrievedGinCtx)
	})

	t.Run("Get non-existing gin context", func(t *testing.T) {
		// Create base context without gin context
		ctx := context.Background()

		// Retrieve and verify gin context is nil
		retrievedGinCtx := GetGinContext(ctx)
		assert.Nil(t, retrievedGinCtx)
	})
}
