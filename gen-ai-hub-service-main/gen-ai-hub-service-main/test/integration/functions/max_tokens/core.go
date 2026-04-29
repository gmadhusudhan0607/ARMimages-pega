//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/gomega"
)

// Expectation type alias
type Expectation = functions.Expectation

// GetMaxOutputTokensFromModelSpecs retrieves the maxOutputTokens.maximum value from model specifications
func GetMaxOutputTokensFromModelSpecs(originalModelName string) int {
	ctx := context.Background()
	registry, err := models.GetGlobalRegistry(ctx)
	Expect(err).To(BeNil(), "Failed to get global model registry")
	Expect(registry).NotTo(BeNil(), "Global model registry is nil")

	// Find the model in the registry
	// For gpt-35-turbo, we need to find the Azure OpenAI model
	models := registry.GetAllModels()
	var foundModel *types.Model

	for _, model := range models {
		if model.Name == originalModelName && model.Infrastructure == types.InfrastructureAzure && model.Provider == types.ProviderAzure {
			foundModel = model
			break
		}
	}

	Expect(foundModel).NotTo(BeNil(), fmt.Sprintf("Model '%s' not found in registry", originalModelName))

	// Get the maxOutputTokens parameter
	maxOutputTokensParam, exists := foundModel.Parameters["maxOutputTokens"]
	Expect(exists).To(BeTrue(), fmt.Sprintf("maxOutputTokens parameter not found for model '%s'", originalModelName))

	// Extract the maximum value
	maximum, ok := maxOutputTokensParam.Maximum.(int)
	if !ok {
		// Try to convert from float64 (common in JSON unmarshaling)
		if maxFloat, isFloat := maxOutputTokensParam.Maximum.(float64); isFloat {
			maximum = int(maxFloat)
		} else {
			Expect(false).To(BeTrue(), fmt.Sprintf("maxOutputTokens.maximum is not a valid integer for model '%s', got: %T %v", originalModelName, maxOutputTokensParam.Maximum, maxOutputTokensParam.Maximum))
		}
	}

	return maximum
}

// CreateAuthHeaderWithIsolationID creates a valid Authorization header with a JWT token
// containing the specified isolation ID. This is the reverse function for ExtractIsolationIDFromToken.
// The generated token can be parsed by ExtractIsolationIDFromToken to retrieve the isolation ID.
func CreateAuthHeaderWithIsolationID(isolationID string) string {
	// Create a minimal JWT header (standard Base64URL encoded)
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(headerJSON)

	// Create JWT payload with the isolation ID
	// Using "guid" field as it's checked first in ExtractIsolationIDFromToken
	payload := map[string]interface{}{
		"guid": isolationID,
		"iat":  1609459200, // dummy timestamp
		"exp":  9999999999, // dummy expiration
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(payloadJSON)

	// Create a dummy signature (since ExtractIsolationIDFromToken only validates structure, not signature)
	signature := "dummy-signature-for-testing"
	signatureB64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(signature))

	// Combine into JWT format
	jwt := fmt.Sprintf("%s.%s.%s", headerB64, payloadB64, signatureB64)

	// Return with Bearer prefix
	return fmt.Sprintf("Bearer %s", jwt)
}
