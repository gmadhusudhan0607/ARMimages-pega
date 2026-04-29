/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
)

func TestFindFirstModelConfigForMapping(t *testing.T) {

	modelToFind := ModelConfig{ModelMapping: "findit"}
	modelToFindToo := ModelConfig{ModelMapping: "findit"}
	modelToFindWithTargetApi := ModelConfig{
		ModelMapping: "findit",
		ModelId:      "findit",
		TargetApi:    "findit",
	}
	modelToFindWithTargetApiAndRegionalInference := ModelConfig{
		ModelMapping:                "findit",
		ModelId:                     "findit",
		TargetApi:                   "findit",
		UseRegionalInferenceProfile: true,
	}
	modelToFindWithRegionalInference := ModelConfig{
		ModelMapping:                "findit",
		ModelId:                     "findit",
		UseRegionalInferenceProfile: true,
	}
	notThisOne := ModelConfig{ModelMapping: "notthisone"}
	neitherThis := ModelConfig{ModelMapping: "neitherthis"}

	type args struct {
		configs     []ModelConfig
		mappingName string
		targetApi   string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 *ModelConfig
	}{
		{
			name:  "empty list returns nil and no error",
			args:  args{[]ModelConfig{}, "notfound", ""},
			want:  false,
			want1: nil,
		},
		{
			name: "find model in list where it is the last one",
			args: args{
				configs:     []ModelConfig{notThisOne, neitherThis, modelToFind, modelToFindWithTargetApi},
				mappingName: "findit",
				targetApi:   "findit",
			},
			want:  true,
			want1: &modelToFindWithTargetApi,
		},
		{
			name: "find model with targetApi and UseRegionalInferenceProfile",
			args: args{
				configs:     []ModelConfig{notThisOne, neitherThis, modelToFind, modelToFindWithTargetApi, modelToFindWithRegionalInference, modelToFindWithTargetApiAndRegionalInference},
				mappingName: "findit",
				targetApi:   "findit",
			},
			want:  true,
			want1: &modelToFindWithTargetApiAndRegionalInference,
		},
		{
			name: "find model with targetApi and UseRegionalInferenceProfile",
			args: args{
				configs:     []ModelConfig{notThisOne, neitherThis, modelToFind, modelToFindWithTargetApi, modelToFindWithRegionalInference},
				mappingName: "findit",
				targetApi:   "findit",
			},
			want:  true,
			want1: &modelToFindWithRegionalInference,
		},
		{
			name: "return first of two bad options",
			args: args{
				configs:     []ModelConfig{notThisOne, neitherThis, modelToFind, modelToFindToo},
				mappingName: "findit",
				targetApi:   "findit",
			},
			want:  true,
			want1: &modelToFind,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FindBestMatch(context.Background(), tt.args.configs, tt.args.mappingName, tt.args.targetApi)
			if got != tt.want {
				t.Errorf("FindBestMatch() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("FindBestMatch() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestLoadInfraModelsForContext(t *testing.T) {

	type infraFile struct {
		content []byte
		name    string
		dir     string
	}

	config1, _ := json.Marshal(ModelConfig{
		ModelMapping: "nottheone",
		ModelId:      "nottheone",
	})
	notTheOne := infraFile{
		config1,
		"nottheone",
		"dir1",
	}

	config2, _ := json.Marshal(ModelConfig{
		ModelMapping: "findit",
		ModelId:      "findit",
	})
	findIt := infraFile{
		config2,
		"genai_findit",
		"dir1",
	}

	config3 := []byte(`{
	 	"ModelMapping": "badJson
	}`)
	badConfig := infraFile{
		config3,
		"genai_badone",
		"dir1",
	}

	config4, _ := json.Marshal(ModelConfig{
		ModelMapping: "findit",
		ModelId:      "findit",
	})
	findItAnotherSubDir := infraFile{
		config4,
		"genai_findit",
		"dir2",
	}

	dirname := helpers.RandStringRunes(8)
	if e := os.Mkdir(dirname, 0755); e != nil {
		panic(e)
	}
	defer os.RemoveAll(dirname) //nolint:errcheck

	files := []infraFile{notTheOne, findIt, badConfig, findItAnotherSubDir}
	for _, f := range files {
		subDir := dirname + "/" + f.dir
		if _, err := os.Stat(subDir); os.IsNotExist(err) {
			if e := os.Mkdir(subDir, 0755); e != nil {
				panic(e)
			}
		}

		path := fmt.Sprintf("%s/%s/%s", dirname, f.dir, f.name)
		_ = os.WriteFile(path, f.content, 0644)
		defer os.Remove(path) //nolint:errcheck
	}

	os.Setenv("GENAI_INFRA_MODELS_DIR", dirname)
	defer os.Unsetenv("GENAI_INFRA_MODELS_DIR")

	ctx := cntx.ServiceContext("infratest")
	c, e := LoadInfraModelsForContext(ctx)

	assert.Nil(t, e)
	assert.Len(t, c, 2)
}

func TestGetInfraModelsForContext(t *testing.T) {

	tests := []struct {
		name          string
		mockResponse  []ModelConfig
		mockStatus    int
		mockError     error
		expectedError bool
	}{
		{
			name: "Success",
			mockResponse: []ModelConfig{
				{
					ModelMapping: "model1",
					ModelId:      "id1",
					ModelArn:     "arn1",
					OIDCRole:     "role1",
					Region:       "us-west-2",
					Endpoint:     "endpoint1",
					Path:         "/path1",
				},
				{
					ModelMapping: "model2",
					ModelId:      "id2",
					ModelArn:     "arn2",
					OIDCRole:     "role2",
					Region:       "us-east-1",
					Endpoint:     "endpoint2",
					Path:         "/path2",
				},
			},
			mockStatus:    http.StatusOK,
			mockError:     nil,
			expectedError: false,
		},
		{
			name:          "Server Error",
			mockResponse:  nil,
			mockStatus:    http.StatusInternalServerError,
			mockError:     nil,
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creating a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				// set response body
				if tt.mockResponse != nil {
					responseBody, _ := json.Marshal(tt.mockResponse)
					_, _ = w.Write(responseBody)
				}
			}))
			defer server.Close()

			// Set environment variable to point to our mock server
			oldEnv := os.Getenv("MAPPING_ENDPOINT")
			os.Setenv("MAPPING_ENDPOINT", server.URL)
			defer os.Setenv("MAPPING_ENDPOINT", oldEnv)

			// call the function
			_, err := GetInfraModelsForContext(context.Background())

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
