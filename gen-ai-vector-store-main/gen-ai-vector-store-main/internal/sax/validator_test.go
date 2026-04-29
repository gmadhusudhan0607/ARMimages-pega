/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// --- Helpers ---

// setupTestTracer creates a tracer that records spans to memory for verification.
func setupTestTracer(t *testing.T) (trace.Tracer, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	return provider.Tracer("test-tracer"), recorder
}

// setupJWKSServer creates a local HTTP server acting as the Identity Provider.
func setupJWKSServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey, string) {
	t.Helper()

	// 1. Generate real RSA keys
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 2. Create the JWKS structure
	jwk := jose.JSONWebKey{
		Key:       &priv.PublicKey,
		KeyID:     "test-key-id",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	// 3. Serve it locally
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))

	return server, priv, server.URL
}

// generateJWT creates a signed token string using the private key.
func generateJWT(t *testing.T, priv *rsa.PrivateKey, issuer, audience, scope string) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key-id"),
	)
	require.NoError(t, err)

	builder := jwt.Signed(signer).Claims(jwt.Claims{
		Issuer:   issuer,
		Audience: jwt.Audience{audience},
		Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	})

	if scope != "" {
		builder = builder.Claims(map[string]interface{}{
			"scp": strings.Split(scope, " "),
		})
	}

	raw, err := builder.CompactSerialize()
	require.NoError(t, err)
	return raw
}

// --- Tests ---

func TestNew_Configuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid Configuration",
			config: Config{
				Audience:     "aud",
				Issuer:       "iss",
				JWKSEndpoint: "http://jwks",
			},
			expectError: false,
		},
		{
			name: "Missing Audience",
			config: Config{
				Issuer:       "iss",
				JWKSEndpoint: "http://jwks",
			},
			expectError: true,
		},
		{
			name: "Missing Issuer",
			config: Config{
				Audience:     "aud",
				JWKSEndpoint: "http://jwks",
			},
			expectError: true,
		},
		{
			name: "Missing JWKS Endpoint",
			config: Config{
				Audience: "aud",
				Issuer:   "iss",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := New(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, val)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, val)
			}
		})
	}
}

func TestValidateRequest_Authentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwksServer, privKey, jwksURL := setupJWKSServer(t)
	defer jwksServer.Close()

	tracer, recorder := setupTestTracer(t)

	const (
		testIssuer   = "https://test-issuer.com"
		testAudience = "test-app"
	)

	v, err := New(Config{
		Issuer:       testIssuer,
		Audience:     testAudience,
		JWKSEndpoint: jwksURL,
	})
	require.NoError(t, err)

	// Inject test tracer
	val := v.(*validator)
	val.tracer = tracer

	tests := []struct {
		name           string
		token          string
		requiredScopes []string
		expectedStatus int
		expectSpanSucc bool
	}{
		{
			name:           "ValidToken_NoScopesRequired",
			token:          generateJWT(t, privKey, testIssuer, testAudience, ""),
			requiredScopes: nil,
			expectedStatus: http.StatusOK,
			expectSpanSucc: true,
		},
		{
			name: "InvalidToken_WrongKey",
			token: func() string {
				wrongKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				return generateJWT(t, wrongKey, testIssuer, testAudience, "")
			}(),
			expectedStatus: http.StatusUnauthorized,
			expectSpanSucc: false,
		},
		{
			name:           "InvalidToken_WrongIssuer",
			token:          generateJWT(t, privKey, "https://wrong-issuer.com", testAudience, ""),
			expectedStatus: http.StatusUnauthorized,
			expectSpanSucc: false,
		},
		{
			name:           "MissingToken",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectSpanSucc: false,
		},
		{
			name:           "ValidToken_WithRequiredScope",
			token:          generateJWT(t, privKey, testIssuer, testAudience, "read:data"),
			requiredScopes: []string{"read:data"},
			expectedStatus: http.StatusOK,
			expectSpanSucc: true,
		},
		{
			name:           "ValidToken_SupersetOfScopes",
			token:          generateJWT(t, privKey, testIssuer, testAudience, "read:data write:data"),
			requiredScopes: []string{"read:data"},
			expectedStatus: http.StatusOK,
			expectSpanSucc: true,
		},
		{
			name:           "InsufficientScope",
			token:          generateJWT(t, privKey, testIssuer, testAudience, "read:data"),
			requiredScopes: []string{"write:data"},
			expectedStatus: http.StatusForbidden,
			expectSpanSucc: false,
		},
		{
			name:           "MissingScopesInToken",
			token:          generateJWT(t, privKey, testIssuer, testAudience, ""),
			requiredScopes: []string{"read:data"},
			expectedStatus: http.StatusForbidden,
			expectSpanSucc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/protected", val.ValidateRequest(tt.requiredScopes...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			spans := recorder.Ended()
			require.NotEmpty(t, spans)
			lastSpan := spans[len(spans)-1]

			found := false
			for _, attr := range lastSpan.Attributes() {
				if attr.Key == attrValidationSuccess {
					assert.Equal(t, tt.expectSpanSucc, attr.Value.AsBool(), "validation_success attribute mismatch")
					found = true
					break
				}
			}
			assert.True(t, found, "Span missing validation_success attribute")
		})
	}
}
