/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"net/http/httptest"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/gin-gonic/gin"
)

// Test helper functions

func createTestGinContext(method, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	c.Request = req
	return c
}

func createTestMapping() *api.Mapping {
	return &api.Mapping{
		Models: []api.Model{
			{
				Name:           "gpt-4o",
				Infrastructure: "azure",
				Provider:       "azure",
				Creator:        "openai",
				ModelId:        "gpt-4o-2024-11-20",
				RedirectURL:    "https://azure-openai.openai.azure.com",
				Active:         true,
			},
			{
				Name:           "claude-3-5-sonnet",
				Infrastructure: "aws",
				Provider:       "bedrock",
				Creator:        "anthropic",
				ModelId:        "anthropic.claude-3-5-sonnet-20241022-v2:0",
				RedirectURL:    "https://bedrock-runtime.us-east-1.amazonaws.com",
				Active:         true,
			},
			{
				Name:           "gemini-1.5-pro",
				Infrastructure: "gcp",
				Provider:       "vertex",
				Creator:        "google",
				ModelId:        "gemini-1.5-pro-002",
				RedirectURL:    "https://us-central1-aiplatform.googleapis.com",
				Active:         true,
			},
		},
		Buddies: []api.Buddy{
			{
				Name:        "selfstudybuddy",
				RedirectURL: "https://buddy-service.example.com/api/v1",
			},
		},
	}
}
