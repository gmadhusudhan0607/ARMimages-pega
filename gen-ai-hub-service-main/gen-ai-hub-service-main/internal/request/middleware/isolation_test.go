/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractIsolationIDFromToken(t *testing.T) {
	// Helper function to create a JWT token with given claims
	createJWT := func(claims map[string]interface{}) string {
		header := map[string]interface{}{
			"alg": "HS256",
			"typ": "JWT",
		}
		headerBytes, _ := json.Marshal(header)
		headerB64 := base64.URLEncoding.EncodeToString(headerBytes)

		payloadBytes, _ := json.Marshal(claims)
		payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)

		signature := "fake-signature"
		signatureB64 := base64.URLEncoding.EncodeToString([]byte(signature))

		return headerB64 + "." + payloadB64 + "." + signatureB64
	}

	t.Run("valid token with guid claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"guid": "12345678-1234-1234-1234-123456789abc",
			"exp":  1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.NoError(t, err)
		assert.Equal(t, "12345678-1234-1234-1234-123456789abc", isolationID)
	})

	t.Run("valid token with isolationId claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"isolationId": "87654321-4321-4321-4321-cba987654321",
			"exp":         1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.NoError(t, err)
		assert.Equal(t, "87654321-4321-4321-4321-cba987654321", isolationID)
	})

	t.Run("token with both guid and isolationId prefers guid", func(t *testing.T) {
		claims := map[string]interface{}{
			"guid":        "guid-value",
			"isolationId": "isolation-value",
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.NoError(t, err)
		assert.Equal(t, "guid-value", isolationID)
	})

	t.Run("missing Bearer prefix", func(t *testing.T) {
		claims := map[string]interface{}{"guid": "test-guid"}
		token := createJWT(claims)

		isolationID, err := ExtractIsolationIDFromToken(token)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "authorization header missing Bearer prefix")
	})

	t.Run("empty token after Bearer prefix", func(t *testing.T) {
		isolationID, err := ExtractIsolationIDFromToken("Bearer ")
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "empty token after Bearer prefix")
	})

	t.Run("empty token after Bearer prefix with spaces", func(t *testing.T) {
		isolationID, err := ExtractIsolationIDFromToken("Bearer   ")
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "empty token after Bearer prefix")
	})

	t.Run("invalid JWT format - too few parts", func(t *testing.T) {
		isolationID, err := ExtractIsolationIDFromToken("Bearer header.payload")
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "invalid JWT format: expected 3 parts, got 2")
	})

	t.Run("invalid JWT format - too many parts", func(t *testing.T) {
		isolationID, err := ExtractIsolationIDFromToken("Bearer header.payload.signature.extra")
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "invalid JWT format: expected 3 parts, got 4")
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		isolationID, err := ExtractIsolationIDFromToken("Bearer header.invalid-base64!@#.signature")
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "failed to decode JWT payload")
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		invalidJSON := base64.URLEncoding.EncodeToString([]byte("{invalid json"))
		token := "header." + invalidJSON + ".signature"
		isolationID, err := ExtractIsolationIDFromToken("Bearer " + token)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "failed to parse JWT claims")
	})

	t.Run("no guid or isolationId claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"sub": "user123",
			"exp": 1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "no valid isolation ID found in JWT claims")
		assert.Contains(t, err.Error(), "available claims: [")
	})

	t.Run("empty guid claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"guid": "",
			"exp":  1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "no valid isolation ID found in JWT claims")
	})

	t.Run("empty isolationId claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"isolationId": "",
			"exp":         1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "no valid isolation ID found in JWT claims")
	})

	t.Run("non-string guid claim", func(t *testing.T) {
		claims := map[string]interface{}{
			"guid": 12345,
			"exp":  1234567890,
		}
		token := createJWT(claims)
		authHeader := "Bearer " + token

		isolationID, err := ExtractIsolationIDFromToken(authHeader)
		assert.Error(t, err)
		assert.Empty(t, isolationID)
		assert.Contains(t, err.Error(), "no valid isolation ID found in JWT claims")
	})

	t.Run("base64 padding test - case 2", func(t *testing.T) {
		// Create a payload that needs == padding (length % 4 == 2)
		claims := map[string]interface{}{"gu": "test"}
		payloadBytes, _ := json.Marshal(claims)

		// Manually encode to ensure we get a case where length % 4 == 2
		payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)
		// Remove padding to simulate the case
		payloadB64 = strings.TrimRight(payloadB64, "=")

		// Ensure we have the right case for testing
		for len(payloadB64)%4 != 2 {
			claims["pad"] = strings.Repeat("x", len(claims["pad"].(string))+1)
			payloadBytes, _ = json.Marshal(claims)
			payloadB64 = base64.URLEncoding.EncodeToString(payloadBytes)
			payloadB64 = strings.TrimRight(payloadB64, "=")
		}

		token := "header." + payloadB64 + ".signature"
		authHeader := "Bearer " + token

		// Should handle padding correctly
		_, err := ExtractIsolationIDFromToken(authHeader)
		// Even though we don't have guid/isolationId, we should get past the base64 decoding
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid isolation ID found")
	})

	t.Run("base64 padding test - case 3", func(t *testing.T) {
		// Create a payload that needs = padding (length % 4 == 3)
		claims := map[string]interface{}{"gui": "test"}
		payloadBytes, _ := json.Marshal(claims)
		payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)
		payloadB64 = strings.TrimRight(payloadB64, "=")

		// Ensure we have the right case for testing
		for len(payloadB64)%4 != 3 {
			claims["pad"] = strings.Repeat("y", len(claims["pad"].(string))+1)
			payloadBytes, _ = json.Marshal(claims)
			payloadB64 = base64.URLEncoding.EncodeToString(payloadBytes)
			payloadB64 = strings.TrimRight(payloadB64, "=")
		}

		token := "header." + payloadB64 + ".signature"
		authHeader := "Bearer " + token

		// Should handle padding correctly
		_, err := ExtractIsolationIDFromToken(authHeader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid isolation ID found")
	})
}
