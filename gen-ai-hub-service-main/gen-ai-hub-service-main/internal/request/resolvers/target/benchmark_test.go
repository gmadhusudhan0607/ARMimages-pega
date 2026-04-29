/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/stretchr/testify/require"
)

// Benchmark Tests

func BenchmarkTargetResolver_Resolve(b *testing.B) {
	mapping := createTestMapping()
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(b, err)
	resolver.staticMapping = mapping

	c := createTestGinContext("POST", "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-15")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(ctx, c)
	}
}

func BenchmarkExtractBasicInfo(b *testing.B) {
	resolver := &TargetResolver{}
	c := createTestGinContext("POST", "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-15")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &ResolutionRequest{
			GinContext: c,
			Target:     &ResolvedTarget{},
			Metadata:   make(map[string]interface{}),
		}
		_ = resolver.extractBasicInfo(ctx, req)
	}
}

func BenchmarkDetermineTargetType(b *testing.B) {
	resolver := &TargetResolver{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &ResolutionRequest{
			GinContext: createTestGinContext("GET", "/test"),
			Target:     &ResolvedTarget{},
			Metadata: map[string]interface{}{
				"routePattern": "openai",
			},
		}
		_ = resolver.determineTargetType(ctx, req)
	}
}

func BenchmarkExtractVersion(b *testing.B) {
	modelID := "anthropic.claude-3-5-sonnet-20241022-v2:0"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractVersion(modelID)
	}
}

func BenchmarkMappingClient_GetModels_Cached(b *testing.B) {
	models := []infra.ModelConfig{
		{
			ModelId:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
			ModelMapping: "claude-3-5-sonnet",
			Endpoint:     "https://bedrock-runtime.us-east-1.amazonaws.com",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	}))
	defer server.Close()

	client := NewMappingClient(server.URL)
	ctx := context.Background()

	// Prime the cache
	_, _ = client.GetModels(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetModels(ctx)
	}
}
