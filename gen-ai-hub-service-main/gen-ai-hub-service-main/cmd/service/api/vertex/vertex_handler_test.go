/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package vertex

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/gin-gonic/gin"
)

func Test_verifyModelInParams(t *testing.T) {

	router := gin.Default()
	router.POST("/deployments/:modelId/images/generations", func(c *gin.Context) {
		var msg string
		var ok bool
		if ok, msg = verifyModelInParams(c); !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}
		c.String(http.StatusOK, "Hello, World!")
	})

	type args struct {
		path         string
		desired_code int
	}

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		// test case definitions
		{
			name: "modelId path parameter is present",
			args: args{
				path:         "/deployments/imagen-3/images/generations",
				desired_code: 200,
			},
		},
		{
			name: "modelId path parameter is empty",
			args: args{
				path:         "/deployments//images/generations",
				desired_code: 400,
			},
		},
		{
			name: "modelId path parameter is NOT Present i.e. invalid path",
			args: args{
				path:         "/deployments/images/generations",
				desired_code: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//body := `{}`
			req, _ := http.NewRequest("POST", tt.args.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.args.desired_code {
				t.Errorf("Expected status code %d, but got %d", tt.args.desired_code, w.Code)
			}
		})
	}
}

func Test_modelIdPresentInRequestbody(t *testing.T) {

	router := gin.Default()
	router.POST("/test", func(c *gin.Context) {

		requestModelId, err := modelIdPresentInRequestbody(c)
		if err != nil {
			msg := fmt.Sprintf("failed to parse request body to JSON with error %s", err.Error())
			fmt.Println(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		if len(requestModelId) == 0 {
			fmt.Println("missing mandatory field modelId for AWS Bedrock model")
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "missing mandatory field modelId for AWS Bedrock model",
			})
			return
		}

		c.String(http.StatusOK, "Hello, World!")
	})

	type args struct {
		body         string
		desired_code int
	}

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		// test case definitions
		{
			name: "modelId key is present in the request body",
			args: args{
				body:         `{"modelId": "originalValue"}`,
				desired_code: 200,
			},
		},
		{
			name: "modelId key is NOT present in the request empty",
			args: args{
				body:         `{"payload": "someValue"}`,
				desired_code: 400,
			},
		},
		{
			name: "Bad request body format",
			args: args{
				body:         "Invalid Json",
				desired_code: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//body := `{}`
			req, _ := http.NewRequest("POST", "/test", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.args.desired_code {
				t.Errorf("Expected status code %d, but got %d", tt.args.desired_code, w.Code)
			}
		})
	}
}

func Test_getModelWithModelId(t *testing.T) {

	type args struct {
		ModelMapping   *api.Mapping
		modelName      string
		requestModelId string
		errMessage     string
	}

	tests := []struct {
		name string
		args args
		//want bool
	}{
		{
			name: "Test1: Model config for the given Model Name and Model Id is available",
			args: args{
				ModelMapping: &api.Mapping{
					Models: []api.Model{
						{
							Name:    "imagen-3",
							ModelId: "imagen-3.0-generate-001",
						},
					},
				},
				modelName:      "imagen-3",
				requestModelId: "imagen-3.0-generate-001",
				errMessage:     "",
			},
		},
		{
			name: "Test2: Model config for the given Model Name and Model Id is NOT available",
			args: args{
				ModelMapping: &api.Mapping{
					Models: []api.Model{
						{
							Name:    "imagen-3",
							ModelId: "imagen-3.0-generate-001",
						},
					},
				},
				modelName:      "imagen-3.5",
				requestModelId: "imagen-3.0-generate-001",
				errMessage:     "unrecognized model with name: imagen-3.5 and modelId: imagen-3.0-generate-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var modelConfig interface{}

			modelConfig, err := getModelWithModelId(tt.args.ModelMapping, tt.args.modelName, tt.args.requestModelId)

			if err != nil {
				if err.Message != tt.args.errMessage {
					t.Errorf("Error: %v", err.Message)
				}
			}

			if _, ok := modelConfig.(*api.Model); !ok {
				t.Errorf("The Modelconfig is not of type *Model")
			}

		})
	}
}

func Test_CheckVertexImagenRequest_GoodRequest(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.POST("/deployments/:modelId/images/generations", CheckVertexImagenRequest(ctx))

	body := `{"modelId": "good-model"}`
	req, _ := http.NewRequest("POST", "/deployments/imagen-3/images/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("Testing: Req Status Code: %v\n", w.Code)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_CheckVertexImagenRequest_NoModelIdInPath(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.POST("/deployments/:modelId/images/generations", CheckVertexImagenRequest(ctx))

	body := `{"modelId": "good-model"}`
	req, _ := http.NewRequest("POST", "/deployments//images/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("Testing: Req Status Code: %v\n", w.Code)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_CheckVertexImagenRequest_NoModelIdInRequest(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.POST("/deployments/:modelId/images/generations", CheckVertexImagenRequest(ctx))

	body := `{"modelDD": "good-model"}`
	req, _ := http.NewRequest("POST", "/deployments/Imagen3/images/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("Testing: Req Status Code: %v\n", w.Code)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_CheckVertexImagenRequest_BadRequestBody(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.POST("/deployments/:modelId/images/generations", CheckVertexImagenRequest(ctx))

	body := "Request Body"
	req, _ := http.NewRequest("POST", "/deployments/Imagen3/images/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("Testing: Req Status Code: %v\n", w.Code)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_SelectImagenModelMapping_ModelNotFound(t *testing.T) {
	ctx := context.Background()
	mapping := &api.Mapping{}
	router := gin.Default()
	router.GET("/test", SelectImagenModelMapping(ctx, mapping))

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}

func Test_CallImagenApi_ModelConfigNotFound(t *testing.T) {
	ctx := context.Background()
	router := gin.Default()
	router.GET("/test", CallImagenApi(ctx))

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, but got %d", http.StatusBadRequest, w.Code)
	}
}
