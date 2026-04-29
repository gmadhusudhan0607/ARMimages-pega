//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package copyrights

import (
	"net/url"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
)

// CreateWireMockCopyrightsExpectation creates expectation for copyright protection validation
func CreateWireMockCopyrightsExpectation(wiremockURL, isolationID, urlPath, expectedBody string, model *functions.Model) (*functions.WireMockExpectation, error) {
	// Extract path from RedirectUrl similar to CreateModelMockExpectation
	redirectUrlPath := functions.ParseRegexParameters(model.RedirectUrl, `https?://(.*):(\d+)(?P<path>.*)`)["path"]

	// Remove parameters from URL path if any
	u, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	cleanPath := u.Path

	fullPath := redirectUrlPath + cleanPath

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "POST",
			"urlPath": fullPath,
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": isolationID,
				},
			},
			"bodyPatterns": []map[string]interface{}{
				{
					"equalToJson":         expectedBody,
					"ignoreExtraElements": true,
				},
			},
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": map[string]interface{}{
				"id":      "chatcmpl-test",
				"object":  "chat.completion",
				"created": 1234567890,
				"model":   model.Name,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]string{
							"role":    "assistant",
							"content": "Test response for copyright protection",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": 50,
					"total_tokens":      60,
				},
			},
		},
	}

	return functions.CreateWireMockExpectation(wiremockURL, mapping)
}

// VerifyWireMockCopyrightsExpectation verifies the copyright expectation was matched
func VerifyWireMockCopyrightsExpectation(wiremockURL string, expectedCount int, model *functions.Model, urlPath string) error {
	// Extract path from RedirectUrl similar to CreateModelMockExpectation
	redirectUrlPath := functions.ParseRegexParameters(model.RedirectUrl, `https?://(.*):(\d+)(?P<path>.*)`)["path"]

	// Remove parameters from URL path if any
	u, err := url.Parse(urlPath)
	if err != nil {
		return err
	}
	cleanPath := u.Path

	fullPath := redirectUrlPath + cleanPath

	criteria := map[string]interface{}{
		"method":  "POST",
		"urlPath": fullPath,
	}

	return functions.VerifyWireMockRequest(wiremockURL, criteria, expectedCount)
}
