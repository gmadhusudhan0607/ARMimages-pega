/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
)

func TestGetOpsReadiness(t *testing.T) {

	type args struct {
		envUseGenAIInfra  string
		envUseAutoMapping string
		mapping           *mapping.SyncMappingStore
	}
	emptyStore := mapping.NewSyncMappingStore()
	tests := []struct {
		name         string
		args         args
		expectStatus int
	}{
		{
			name: "Service Available hen USE_GENAI_INFRA and USE_AUTO_MAPPING set to false",
			args: args{
				envUseGenAIInfra:  "false",
				envUseAutoMapping: "false",
				mapping:           emptyStore,
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Service Unavailable Test with USE_GENAI_INFRA and USE_AUTO_MAPPING set to true but no mappings",
			args: args{
				envUseGenAIInfra:  "true",
				envUseAutoMapping: "true",
				mapping:           emptyStore,
			},
			expectStatus: http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			os.Setenv("USE_GENAI_INFRA", tt.args.envUseGenAIInfra)
			os.Setenv("USE_AUTO_MAPPING", tt.args.envUseAutoMapping)
			ctx := cntx.ServiceContext("healthtest")

			router := gin.Default()
			router.POST("/health", GetOpsReadiness(ctx, tt.args.mapping))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/health", nil)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("GetLiveness() = %v, want %v", w.Code, tt.expectStatus)
			}

			os.Unsetenv("USE_GENAI_INFRA")
			os.Unsetenv("USE_AUTO_MAPPING")
		})
	}
}

func TestGetReadinessNoGenAIInfra(t *testing.T) {
	w := httptest.NewRecorder()
	router := gin.Default()
	router.POST("/health", GetReadiness)

	req, _ := http.NewRequest("POST", "/health", nil)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GetReadiness() = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestGetReadinessWithGenAIInfraAutoMapping(t *testing.T) {
	tests := []struct {
		name         string
		mockLoader   infra.ConfigLoader
		expectStatus int
		expectBody   bool
	}{
		{
			name: "ConfigLoader returns error",
			mockLoader: func(ctx context.Context) ([]infra.ModelConfig, error) {
				return nil, errors.New("failed to load config")
			},
			expectStatus: http.StatusServiceUnavailable,
			expectBody:   true,
		},
		{
			name: "ConfigLoader returns empty array",
			mockLoader: func(ctx context.Context) ([]infra.ModelConfig, error) {
				return []infra.ModelConfig{}, nil
			},
			expectStatus: http.StatusServiceUnavailable,
			expectBody:   true,
		},
		{
			name: "ConfigLoader returns array with 1 item",
			mockLoader: func(ctx context.Context) ([]infra.ModelConfig, error) {
				return []infra.ModelConfig{
					{
						ModelMapping: "test-mapping",
						ModelId:      "test-model",
						Region:       "us-east-1",
					},
				}, nil
			},
			expectStatus: http.StatusOK,
			expectBody:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cntx.ServiceContext("healthtest")

			router := gin.Default()
			router.POST("/health", GetReadinessDependingOnMappings(ctx, tt.mockLoader))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/health", nil)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("GetReadinessDependingOnMappings() = %v, want %v", w.Code, tt.expectStatus)
			}

			if tt.expectBody && len(w.Body.String()) == 0 {
				t.Errorf("Expected response body but got empty")
			}

			if !tt.expectBody && len(w.Body.String()) > 0 {
				t.Errorf("Expected empty response body but got: %s", w.Body.String())
			}
		})
	}
}

func TestGetLiveness(t *testing.T) {
	w := httptest.NewRecorder()
	router := gin.Default()
	router.POST("/liveness", GetLiveness)

	req, _ := http.NewRequest("POST", "/liveness", nil)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GetLiveness() = %v, want %v", w.Code, http.StatusOK)
	}
}
