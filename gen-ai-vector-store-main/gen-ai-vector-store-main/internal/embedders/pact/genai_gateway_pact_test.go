// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//
//go:build pact
// +build pact

package pact

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/assert"
)

var (
	pact          dsl.Pact
	pactServerURL string
)

const (
	consumerName = "GenaiVectorStore"
	providerName = "GenAIGateway"

	// GenAI Gateway embedding endpoint paths from gen-ai-hub-service API spec
	// See: gen-ai-hub-service/apidocs/spec.yaml
	adaEndpointPath    = "/openai/deployments/text-embedding-ada-002/embeddings"
	titanEndpointPath  = "/amazon/deployments/titan-embed-text/embeddings"
	googleEndpointPath = "/google/deployments/text-multilingual-embedding-002/embeddings"
)

func TestMain(m *testing.M) {
	setup()
	exitCode := m.Run()

	// Shutdown the Mock Service and Write pact files to disk
	if err := pact.WritePact(); err != nil {
		fmt.Printf("Failed to write pact: %v\n", err)
		os.Exit(1)
	}

	pact.Teardown()
	os.Exit(exitCode)
}

func setup() {
	pact = createPact()
	pact.Setup(true)

	pactServerURL = fmt.Sprintf("http://localhost:%d", pact.Server.Port)
}

func createPact() dsl.Pact {
	return dsl.Pact{
		Consumer:                 consumerName,
		Provider:                 providerName,
		LogDir:                   "pact",
		PactDir:                  "pact",
		LogLevel:                 "DEBUG",
		DisableToolValidityCheck: true, // Known issue: https://github.com/pact-foundation/pact-go/issues/85
		ClientTimeout:            30 * time.Second,
	}
}

// =============================================================================
// Ada (OpenAI text-embedding-ada-002) Embedding Tests
// =============================================================================

func TestAdaEmbeddingPact_Success(t *testing.T) {
	t.Run("POST embedding - Ada model success", func(t *testing.T) {
		// Expected request body
		requestBody := dsl.MapMatcher{
			"input": dsl.Like("sample text to embed"),
		}

		// Expected response body matching internal/embedders/ada/types.go
		// Note: We use small arrays (3 elements) for examples - actual dimensions (1536) are an implementation detail
		responseBody := dsl.MapMatcher{
			"object": dsl.Like("list"),
			"data": dsl.EachLike(map[string]interface{}{
				"object":    dsl.Like("embedding"),
				"embedding": dsl.EachLike(0.123456, 3), // Array of floats (actual size varies by model)
				"index":     dsl.Like(0),
			}, 1),
			"model": dsl.Like("text-embedding-ada-002"),
			"usage": dsl.Like(map[string]interface{}{
				"prompt_tokens": dsl.Like(5),
				"total_tokens":  dsl.Like(5),
			}),
		}

		pact.
			AddInteraction().
			Given("Ada embedding model is available").
			UponReceiving("A request to generate Ada embeddings").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(adaEndpointPath),
				Query: dsl.MapMatcher{
					"api-version": dsl.String("2023-05-15"),
				},
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("test-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusOK,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+adaEndpointPath+"?api-version=2023-05-15",
				`{"input":"sample text to embed"}`,
				"test-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

func TestAdaEmbeddingPact_Forbidden(t *testing.T) {
	t.Run("POST embedding - Ada model forbidden", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"input": dsl.Like("sample text"),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Access denied"),
				"type":    dsl.Like("invalid_request_error"),
				"code":    dsl.Like("access_denied"),
			}),
		}

		pact.
			AddInteraction().
			Given("Ada embedding model access is forbidden").
			UponReceiving("A request to generate Ada embeddings with invalid credentials").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(adaEndpointPath),
				Query: dsl.MapMatcher{
					"api-version": dsl.String("2023-05-15"),
				},
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("forbidden-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusForbidden,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+adaEndpointPath+"?api-version=2023-05-15",
				`{"input":"sample text"}`,
				"forbidden-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

func TestAdaEmbeddingPact_RateLimited(t *testing.T) {
	t.Run("POST embedding - Ada model rate limited", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"input": dsl.Like("sample text"),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Rate limit exceeded"),
				"type":    dsl.Like("rate_limit_error"),
			}),
		}

		pact.
			AddInteraction().
			Given("Ada embedding model is rate limited").
			UponReceiving("A request to generate Ada embeddings when rate limited").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(adaEndpointPath),
				Query: dsl.MapMatcher{
					"api-version": dsl.String("2023-05-15"),
				},
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("rate-limited-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusTooManyRequests,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+adaEndpointPath+"?api-version=2023-05-15",
				`{"input":"sample text"}`,
				"rate-limited-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

// =============================================================================
// Titan (AWS Bedrock) Embedding Tests
// =============================================================================

func TestTitanEmbeddingPact_Success(t *testing.T) {
	t.Run("POST embedding - Titan model success", func(t *testing.T) {
		// Request body matching internal/embedders/titan/types.go
		requestBody := dsl.MapMatcher{
			"inputText":  dsl.Like("sample text to embed"),
			"dimensions": dsl.Like(1024),
		}

		// Response body matching internal/embedders/titan/types.go
		// Note: We use small arrays (3 elements) for examples - actual dimensions are an implementation detail
		responseBody := dsl.MapMatcher{
			"embedding":           dsl.EachLike(0.123456, 3), // Array of floats
			"inputTextTokenCount": dsl.Like(5),
			"embeddingsByType": dsl.Like(map[string]interface{}{
				"float": dsl.EachLike(0.123456, 3),
			}),
		}

		pact.
			AddInteraction().
			Given("Titan embedding model is available").
			UponReceiving("A request to generate Titan embeddings").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(titanEndpointPath),
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("test-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusOK,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+titanEndpointPath,
				`{"inputText":"sample text to embed","dimensions":1024}`,
				"test-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

func TestTitanEmbeddingPact_ModelNotFound(t *testing.T) {
	t.Run("POST embedding - Titan model not found", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"inputText":  dsl.Like("sample text"),
			"dimensions": dsl.Like(1024),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Model not found"),
				"type":    dsl.Like("not_found_error"),
			}),
		}

		pact.
			AddInteraction().
			Given("Titan embedding model does not exist").
			UponReceiving("A request to generate embeddings with non-existent Titan model").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(titanEndpointPath),
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("not-found-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusNotFound,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+titanEndpointPath,
				`{"inputText":"sample text","dimensions":1024}`,
				"not-found-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

func TestTitanEmbeddingPact_Forbidden(t *testing.T) {
	t.Run("POST embedding - Titan model forbidden", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"inputText":  dsl.Like("sample text"),
			"dimensions": dsl.Like(1024),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Access denied"),
				"type":    dsl.Like("invalid_request_error"),
			}),
		}

		pact.
			AddInteraction().
			Given("Titan embedding model access is forbidden").
			UponReceiving("A request to generate Titan embeddings with invalid credentials").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(titanEndpointPath),
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("forbidden-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusForbidden,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+titanEndpointPath,
				`{"inputText":"sample text","dimensions":1024}`,
				"forbidden-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

// =============================================================================
// Google Embedding Tests
// =============================================================================

func TestGoogleEmbeddingPact_Success(t *testing.T) {
	t.Run("POST embedding - Google model success", func(t *testing.T) {
		// Request body matching internal/embedders/google/types.go
		requestBody := dsl.MapMatcher{
			"model": dsl.Like("text-multilingual-embedding-002"),
			"texts": dsl.EachLike("sample text to embed", 1),
		}

		// Response body matching internal/embedders/google/types.go
		// Note: We use small arrays (3 elements) for examples - actual dimensions are an implementation detail
		responseBody := dsl.MapMatcher{
			"embedding": dsl.EachLike(map[string]interface{}{
				"values": dsl.EachLike(0.123456, 3), // Array of floats
				"statistics": dsl.Like(map[string]interface{}{
					"token_count": dsl.Like(5.0),
					"truncated":   dsl.Like(false),
				}),
			}, 1),
			"usage": dsl.Like(map[string]interface{}{
				"prompt_tokens": dsl.Like(5.0),
			}),
		}

		pact.
			AddInteraction().
			Given("Google embedding model is available").
			UponReceiving("A request to generate Google embeddings").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(googleEndpointPath),
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("test-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusOK,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+googleEndpointPath,
				`{"model":"text-multilingual-embedding-002","texts":["sample text to embed"]}`,
				"test-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

func TestGoogleEmbeddingPact_Forbidden(t *testing.T) {
	t.Run("POST embedding - Google model forbidden", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"model": dsl.Like("text-multilingual-embedding-002"),
			"texts": dsl.EachLike("sample text", 1),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Access denied"),
				"code":    dsl.Like(403),
			}),
		}

		pact.
			AddInteraction().
			Given("Google embedding model access is forbidden").
			UponReceiving("A request to generate Google embeddings with invalid credentials").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(googleEndpointPath),
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("forbidden-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusForbidden,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+googleEndpointPath,
				`{"model":"text-multilingual-embedding-002","texts":["sample text"]}`,
				"forbidden-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

// =============================================================================
// Service Unavailable Tests (applicable to all models)
// =============================================================================

func TestEmbeddingPact_ServiceUnavailable(t *testing.T) {
	t.Run("POST embedding - Service unavailable (Ada)", func(t *testing.T) {
		requestBody := dsl.MapMatcher{
			"input": dsl.Like("sample text"),
		}

		responseBody := dsl.MapMatcher{
			"error": dsl.Like(map[string]interface{}{
				"message": dsl.Like("Service temporarily unavailable"),
				"type":    dsl.Like("service_unavailable_error"),
			}),
		}

		pact.
			AddInteraction().
			Given("GenAI Gateway service is unavailable").
			UponReceiving("A request when service is unavailable").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String(adaEndpointPath),
				Query: dsl.MapMatcher{
					"api-version": dsl.String("2023-05-15"),
				},
				Headers: dsl.MapMatcher{
					"Content-Type":    dsl.String("application/json"),
					"vs-isolation-id": dsl.Like("unavailable-isolation-id"),
				},
				Body: requestBody,
			}).
			WillRespondWith(dsl.Response{
				Status: http.StatusServiceUnavailable,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: responseBody,
			})

		err := pact.Verify(func() error {
			return makeEmbeddingRequest(
				pactServerURL+adaEndpointPath+"?api-version=2023-05-15",
				`{"input":"sample text"}`,
				"unavailable-isolation-id",
			)
		})
		assert.NoError(t, err)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// makeEmbeddingRequest is a helper function to make HTTP requests to the pact mock server
func makeEmbeddingRequest(url, body, isolationID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("vs-isolation-id", isolationID)
	req.ContentLength = int64(len(body))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Consume the response body to properly complete the request
	if _, err := io.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// For pact verification, we just need to make the request
	// The actual response handling is tested in the embedder unit tests
	return nil
}
