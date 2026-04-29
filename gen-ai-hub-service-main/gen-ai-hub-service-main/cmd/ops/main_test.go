/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

type testType struct {
	name       string
	method     string
	uri        string
	code       int
	errMsgText string
	reqBody    string
}

func TestEndpoints(t *testing.T) {
	tests := []testType{
		{
			name:       "GET metrics models",
			method:     http.MethodGet,
			uri:        "/v1/isolations/1234/metrics",
			code:       http.StatusOK,
			errMsgText: "",
		},
		{
			name:       "GET metrics models with query params",
			method:     http.MethodGet,
			uri:        "/v1/isolations/1234/metrics?from=1618119364&to=1728929364",
			code:       http.StatusOK,
			errMsgText: "",
		},
		{
			name:       "GET metrics models no isolation",
			method:     http.MethodGet,
			uri:        "/v1/isolations//metrics",
			code:       http.StatusBadRequest,
			errMsgText: "",
		},
		{
			name:       "GET metrics models bad query param",
			method:     http.MethodGet,
			uri:        "/v1/isolations/12345/metrics?from=abc",
			code:       http.StatusBadRequest,
			errMsgText: "",
		},
		{
			name:       "POST event",
			method:     http.MethodPost,
			uri:        "/v1/events",
			reqBody:    `{"isolation":"abcd", "timestamp":12345}`,
			code:       http.StatusOK,
			errMsgText: "",
		},
		{
			name:       "GET mappings",
			method:     http.MethodGet,
			uri:        "/v1/mappings",
			code:       http.StatusOK,
			errMsgText: "",
		},
	}
	os.Setenv("LLM_MODELS_REGION", "us-east-1")
	defer os.Unsetenv("LLM_MODELS_REGION")
	os.Setenv("LLM_ACCOUNT_ID", "045666071234")
	defer os.Unsetenv("LLM_ACCOUNT_ID")
	os.Setenv("STAGE_NAME", "integration")
	defer os.Unsetenv("STAGE_NAME")
	os.Setenv("SAX_CELL", "us")
	defer os.Unsetenv("SAX_CELL")
	os.Setenv("SAX_CONFIG_PATH", "/sax_config.json")
	defer os.Unsetenv("SAX_CONFIG_PATH")

	runTests(t, tests)
}

func runTests(t *testing.T, tests []testType) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			contextName := "unit-test"
			ctx := cntx.ServiceContext(contextName)
			d := &mapping.SyncMappingStore{}
			r := mapping.NewAwsCredentialProvider()
			d.Write([]infra.ModelConfig{})
			opsEngine := setupOpsServer(ctx, d, r)
			req, err := http.NewRequest(tt.method, tt.uri, bytes.NewBuffer([]byte(tt.reqBody)))
			assert.NoError(t, err)
			opsEngine.ServeHTTP(recorder, req)
			assert.Equal(t, tt.code, recorder.Result().StatusCode)
			if tt.code != http.StatusOK {
				responseBody, _ := io.ReadAll(recorder.Result().Body)
				assert.Contains(t, string(responseBody), tt.errMsgText)
			}
		})
	}
}

func TestSanity(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	ctx := context.Background()
	cancel, cancelFunc := context.WithCancel(ctx)

	os.Setenv("CONTEXT_NAME", "unit-test")
	defer os.Unsetenv("CONTEXT_NAME")
	os.Setenv("SAX_CONFIG_PATH", "testdata/sax_config.json")
	defer os.Unsetenv("SAX_CONFIG_PATH")
	os.Setenv("GENAI_INFRA_MAPPING_REFRESH_INTERVAL", "1s")
	defer os.Unsetenv("GENAI_INFRA_MAPPING_REFRESH_INTERVAL")
	os.Setenv("LLM_ACCOUNT_ID", "123456789012")
	defer os.Unsetenv("LLM_ACCOUNT_ID")
	os.Setenv("LLM_MODELS_REGION", "us-east-1")
	defer os.Unsetenv("LLM_MODELS_REGION")
	os.Setenv("STAGE_NAME", "integration")
	defer os.Unsetenv("STAGE_NAME")
	os.Setenv("SAX_CELL", "us")
	defer os.Unsetenv("SAX_CELL")
	os.Setenv("USE_GENAI_INFRA", "true")
	defer os.Unsetenv("USE_GENAI_INFRA")
	os.Setenv("USE_AUTO_MAPPING", "true")
	defer os.Unsetenv("USE_AUTO_MAPPING")

	// Use random available ports to avoid conflicts
	os.Setenv("OPS_PORT", "0")
	defer os.Unsetenv("OPS_PORT")
	os.Setenv("SERVICE_HEALTHCHECK_PORT", "0")
	defer os.Unsetenv("SERVICE_HEALTHCHECK_PORT")

	sc := saxtypes.SaxAuthClientConfig{
		ClientId:      "",
		PrivateKey:    "",
		Scopes:        "",
		TokenEndpoint: "",
	}
	jsn, _ := json.Marshal(sc)

	fs := &testutils.FileSystemMock{}
	fs.With("testdata/sax_config.json", string(jsn))
	helperSuite.FileReader = fs.FileReader()
	helperSuite.FileExists = fs.FileExists()
	helperSuite.CreateServiceContext = func(string) context.Context { return cancel }
	defer helperSuite.Reset()

	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NotPanics(t, main, "main should not panic")
	}()

	// Simulate cancellation after a short delay
	time.Sleep(3 * time.Second)
	cancelFunc()

	select {
	case <-done:
		// Test completed successfully
	case <-time.After(6 * time.Second):
		t.Fatal("TestMainPanic: main function did not exit after context cancellation")
	}
}

func Test_setupAWSMappingSynchronizer(t *testing.T) {
	type args struct {
		ctx      context.Context
		interval string
		task     func() error
	}
	tests := []struct {
		name     string
		args     args
		duration time.Duration
	}{
		{
			name: "Test setupAWSMappingSynchronizer_success",
			args: args{
				ctx:      context.Background(),
				interval: "1h",
				task:     func() error { return nil },
			},
			duration: time.Hour,
		},
		{
			name: "Test setupAWSMappingSynchronizer_defaultInterval",
			args: args{
				ctx:      context.Background(),
				interval: "",
				task:     func() error { return nil },
			},
			duration: time.Minute * 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GENAI_INFRA_MAPPING_REFRESH_INTERVAL", tt.args.interval)
			defer os.Unsetenv("GENAI_INFRA_MAPPING_REFRESH_INTERVAL")
			assert.Equalf(t, tt.duration, setupAWSMappingSynchronizer(tt.args.ctx, tt.args.task).interval, "setupAWSMappingSynchronizer(%v, %v)", tt.args.ctx, tt.args.task)
		})
	}
}
