/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/client"
)

func Test_ValidateBedrockConverseRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	tests := []struct {
		name         string
		path         string
		requestBody  string
		expectedCode int
	}{
		{
			name:         "Unrecognized ModelId",
			path:         "/validate//test",
			requestBody:  `{"modelId": "unrecognized-id"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "ModelId Not Set",
			path:         "/validate/modelId/test",
			requestBody:  `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Bad Request Body",
			path:         "/validate/modelId/test",
			requestBody:  `{"modelId": "someID"`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.Default()
			router.POST("/validate/:modelId/test", ValidateBedrockConverseRequest(ctx))

			req, _ := http.NewRequest("POST", tt.path, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, but got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func Test_SelectModelMapping_ModelNotFound(t *testing.T) {
	ctx := context.Background()
	mapping := &Mapping{}
	router := gin.Default()
	router.GET("/test", SelectModelMapping(ctx, mapping))

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_DoBedrockConverseCall_ModelConfigNotFound(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.GET("/test", DoBedrockConverseCall(ctx))

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_modelIdPresentInRequestBody_MissingModelIdField(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", strings.NewReader(`{"otherField": "value"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	modelId, err := modelIdPresentInRequestbody(c)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if modelId != "" {
		t.Errorf("Expected empty modelId, but got %s", modelId)
	}
}

func Test_modelIdPresentInRequestBody_InvalidJSON(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", strings.NewReader("invalid-json"))
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := modelIdPresentInRequestbody(c)
	if err == nil {
		t.Errorf("Expected error")
	}
}

func Test_writePayloadField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		requestBody   string
		fieldName     string
		newValue      interface{}
		expectError   bool
		expectedValue interface{}
	}{
		{
			name:          "Success",
			requestBody:   `{"modelId": "originalValue"}`,
			fieldName:     "modelId",
			newValue:      "newValue",
			expectError:   false,
			expectedValue: "newValue",
		},
		{
			name:        "Invalid JSON Payload",
			requestBody: `"modelId": "originalValue"`,
			fieldName:   "modelId",
			newValue:    "newValue",
			expectError: true,
		},
		{
			name:        "Invalid JSON Field",
			requestBody: `{"modelId": "originalValue"}`,
			fieldName:   "modelId",
			newValue:    func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request, _ = http.NewRequest("POST", "/", strings.NewReader(tt.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			err := writePayloadField(c, tt.fieldName, tt.newValue)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				body, _ := io.ReadAll(c.Request.Body)
				var payload map[string]interface{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Errorf("Expected no error, but got %v", err)
				}
				assert.Equal(t, tt.expectedValue, payload[tt.fieldName])
			}
		})
	}
}

func Test_DoBedrockRedirectCall_ModelConfigNotFound(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.GET("/test", DoBedrockRedirectCall(ctx))

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}
func TestHandleBedrockConverseSdkCall(t *testing.T) {

	ctx := cntx.ServiceContext("bedrock_handler_test")

	infraModels := []infra.ModelConfig{
		{
			ModelMapping: "findMe",
		},
	}

	type args struct {
		ctx          context.Context
		configLoader infra.ConfigLoader
		invoker      client.ConverseSdkInvoke
		payload      string
	}
	tests := []struct {
		name   string
		args   args
		status int
	}{
		{
			"BadRequest_FailToParseRequest",
			args{
				ctx,
				mockLoader(infraModels),
				mockInvoker(),
				"{",
			},
			400,
		},
		{
			"InternalError_FailToLoadModels",
			args{
				ctx,
				mockLoaderWithError(),
				mockInvoker(),
				"{}",
			},
			500,
		},
		{
			"BadRequest_ModelNotFound",
			args{
				ctx,
				mockLoader([]infra.ModelConfig{}),
				mockInvoker(),
				"{}",
			},
			404,
		},
		{
			"Unauthorized_CredentialAcquisitionFailed",
			args{
				ctx,
				mockLoader(infraModels),
				mockInvokerWithCredentialError(),
				`{"ModelMapping":"findMe"}`,
			},
			401,
		},
		{
			"InternalError_SdkInvokationFail",
			args{
				ctx,
				mockLoader(infraModels),
				mockInvokerWithError(),
				`{"ModelMapping":"findMe"}`,
			},
			500,
		},
		{
			"OK_Success",
			args{
				ctx,
				mockLoader(infraModels),
				mockInvoker(),
				`{"ModelMapping":"findMe"}`,
			},
			200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.Default()
			router.POST("/:modelId/model", HandleBedrockModelCall(ctx, tt.args.configLoader, tt.args.invoker))

			req, _ := http.NewRequest("POST", "/findMe/model", strings.NewReader(tt.args.payload))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code, "HandleBedrockModelCall(%v, %v, %v)", tt.args.ctx, tt.args.configLoader, tt.args.invoker)
		})
	}
}

func mockLoader(l []infra.ModelConfig) infra.ConfigLoader {
	return func(context.Context) ([]infra.ModelConfig, error) {
		return l, nil
	}
}

func mockLoaderWithError() infra.ConfigLoader {
	return func(context.Context) ([]infra.ModelConfig, error) {
		return nil, errors.New("fail")
	}
}

func mockInvoker() client.ConverseSdkInvoke {
	return func(inference *client.ConverseModelInference, awsProxy client.AwsProvider) error {
		return nil
	}
}

func mockInvokerWithCredentialError() client.ConverseSdkInvoke {
	return func(inference *client.ConverseModelInference, awsProxy client.AwsProvider) error {
		return &client.ErrCredentialsAcquisition{Cause: errors.New("failed to assume role with web identity: invalid token")}
	}
}

func mockInvokerWithError() client.ConverseSdkInvoke {
	return func(inference *client.ConverseModelInference, awsProxy client.AwsProvider) error {
		return errors.New("fail")
	}
}

func Test_calculateTargetModelAndApiFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		modelId   string
		targetApi string
		want      string
		want1     string
	}{
		{
			name:      "chat completions returns converse",
			modelId:   "model-id",
			targetApi: "/chat/completions",
			want:      "model-id",
			want1:     "converse",
		},
		{
			name:      "embeddings returns invoke",
			modelId:   "model-id",
			targetApi: "/embeddings",
			want:      "model-id",
			want1:     "invoke",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a Gin router with the route pattern
			router := gin.Default()
			var got, got1 string
			router.GET("/provider/deployments/:modelId/*targetApi", func(c *gin.Context) {
				got, got1 = calculateTargetModelAndApiFromRequest(c)
				c.Status(200)
			})
			// Build the request URL
			url := "/provider/deployments/" + tt.modelId + tt.targetApi
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equalf(t, tt.want, got, "calculateTargetModelAndApiFromRequest(%v)", url)
			assert.Equalf(t, tt.want1, got1, "calculateTargetModelAndApiFromRequest(%v)", url)
		})
	}
}
