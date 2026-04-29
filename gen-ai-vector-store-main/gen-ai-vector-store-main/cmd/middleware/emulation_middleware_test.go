/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func enableEmulationMode() {
	os.Setenv("EMULATION_MODE", "true")
	os.Setenv("EMULATION_MIN_TIME", "100")
	os.Setenv("EMULATION_MAX_TIME", "300")
}

func disableEmulationMode() {
	os.Unsetenv("EMULATION_MODE")
	os.Unsetenv("EMULATION_MIN_TIME")
	os.Unsetenv("EMULATION_MAX_TIME")
}

func TestEmulationMiddleware_Disabled(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Ensure emulation is disabled
	os.Setenv("EMULATION_MODE", "false")
	defer os.Unsetenv("EMULATION_MODE")

	// Create test router
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "real response")
	assert.Less(t, duration, 100*time.Millisecond) // Should be fast since no emulation
}

func TestEmulationMiddleware_Enabled(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation with specific timing
	enableEmulationMode()
	defer disableEmulationMode()

	// Create test router with a valid collections endpoint
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/v2/test-isolation/collections", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/v2/test-isolation/collections", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "collections")
	assert.NotContains(t, w.Body.String(), "real response")
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond) // Should take at least min time
	assert.LessOrEqual(t, duration, 300*time.Millisecond)    // Should not take much more than max time
}

func TestEmulationMiddleware_CollectionsEndpoint(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Create test router
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/v2/test-isolation/collections", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real collections"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/v2/test-isolation/collections", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "collections")
	assert.Contains(t, w.Body.String(), "fake-collection-1")
	assert.Contains(t, w.Body.String(), "isolationID")
	assert.Contains(t, w.Body.String(), "test-isolation")
}

func TestEmulationMiddleware_DocumentsEndpoint(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Create test router - Use documents list endpoint instead of single document
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/v1/test-isolation/collections/test-collection/documents", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real documents"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/v1/test-isolation/collections/test-collection/documents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "documents")
	assert.Contains(t, w.Body.String(), "fake-doc-1")
	assert.Contains(t, w.Body.String(), "fake document content")
}

func TestEmulationMiddleware_PostRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Create test router
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.POST("/v2/test-isolation/collections", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "real creation"})
	})

	// Test request
	req := httptest.NewRequest("POST", "/v2/test-isolation/collections", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, 202, w.Code) // Collection creation returns 202
	assert.Contains(t, w.Body.String(), "collectionID")
	assert.Contains(t, w.Body.String(), "defaultEmbeddingProfile")
	assert.Contains(t, w.Body.String(), "documentsTotal")
	assert.Contains(t, w.Body.String(), "col-") // Collection IDs start with "col-"
}

func TestGenerateFakeID(t *testing.T) {
	id1 := generateFakeID()
	id2 := generateFakeID()

	// Should generate different IDs
	assert.NotEqual(t, id1, id2)

	// Should start with "fake-"
	assert.Contains(t, id1, "fake-")
	assert.Contains(t, id2, "fake-")

	// Should be reasonable length
	assert.Greater(t, len(id1), 5)
	assert.Greater(t, len(id2), 5)
}

func TestEmulationMiddleware_BasicHeaders(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Create test router - use unmatched endpoint to test basic error handling
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions for API-specification-compliant headers
	assert.Equal(t, http.StatusBadRequest, w.Code) // Unmatched endpoints return 400

	// Check that basic API headers are always set
	requestDuration := w.Header().Get(headers.RequestDurationMs)
	assert.NotEmpty(t, requestDuration)

	dbQueryTime := w.Header().Get(headers.DbQueryTimeMs)
	assert.NotEmpty(t, dbQueryTime)

	// Embedding headers should NOT be set for non-embedding endpoints
	assert.Empty(t, w.Header().Get(headers.ModelId))
	assert.Empty(t, w.Header().Get(headers.ModelVersion))
	assert.Empty(t, w.Header().Get(headers.EmbeddingTimeMs))
	assert.Empty(t, w.Header().Get(headers.EmbeddingCallsCount))

	// Response count header should NOT be set for single-item endpoints
	assert.Empty(t, w.Header().Get(headers.ResponseReturnedItemsCount))
}

func TestEmulationMiddleware_EmbeddingHeaders(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Test embedding endpoint (PUT document)
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.PUT("/v1/test-isolation/collections/test-collection/documents", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("PUT", "/v1/test-isolation/collections/test-collection/documents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions for embedding headers
	assert.Equal(t, 202, w.Code) // PUT documents returns 202 (Accepted) for eventual consistency

	// Basic headers should always be present
	assert.NotEmpty(t, w.Header().Get(headers.RequestDurationMs))
	assert.NotEmpty(t, w.Header().Get(headers.DbQueryTimeMs))

	// Embedding headers should be present for embedding endpoints
	modelId := w.Header().Get(headers.ModelId)
	assert.NotEmpty(t, modelId)
	assert.Contains(t, []string{"openai-text-embedding-ada-002", "openai-text-embedding-3-small", "openai-text-embedding-3-large", "text-embedding-ada-002"}, modelId)

	modelVersion := w.Header().Get(headers.ModelVersion)
	assert.NotEmpty(t, modelVersion)
	assert.Contains(t, []string{"1", "2", "3"}, modelVersion)

	assert.NotEmpty(t, w.Header().Get(headers.EmbeddingTimeMs))
	assert.NotEmpty(t, w.Header().Get(headers.EmbeddingCallsCount))
}

func TestEmulationMiddleware_MultiItemHeaders(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Test multi-item endpoint (GET collections)
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.GET("/v2/test-isolation/collections", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("GET", "/v2/test-isolation/collections", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions for multi-item headers
	assert.Equal(t, http.StatusOK, w.Code)

	// Basic headers should always be present
	assert.NotEmpty(t, w.Header().Get(headers.RequestDurationMs))
	assert.NotEmpty(t, w.Header().Get(headers.DbQueryTimeMs))

	// Response count header should be present for multi-item endpoints
	returnedCount := w.Header().Get(headers.ResponseReturnedItemsCount)
	assert.NotEmpty(t, returnedCount)

	// Embedding headers should NOT be present for non-embedding endpoints
	assert.Empty(t, w.Header().Get(headers.ModelId))
	assert.Empty(t, w.Header().Get(headers.ModelVersion))
	assert.Empty(t, w.Header().Get(headers.EmbeddingTimeMs))
	assert.Empty(t, w.Header().Get(headers.EmbeddingCallsCount))
}

func TestEmulationMiddleware_QueryEndpointHeaders(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Enable emulation
	enableEmulationMode()
	defer disableEmulationMode()

	// Test query endpoint (POST query/chunks) - this is both embedding and multi-item
	router := gin.New()
	router.Use(EmulationMiddleware(logger))
	router.POST("/v1/test-isolation/collections/test-collection/query/chunks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "real response"})
	})

	// Test request
	req := httptest.NewRequest("POST", "/v1/test-isolation/collections/test-collection/query/chunks", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions - should have both embedding and multi-item headers
	assert.Equal(t, http.StatusOK, w.Code) // Query chunks returns 200, not 202

	// Basic headers
	assert.NotEmpty(t, w.Header().Get(headers.RequestDurationMs))
	assert.NotEmpty(t, w.Header().Get(headers.DbQueryTimeMs))

	// Embedding headers (query involves embedding)
	assert.NotEmpty(t, w.Header().Get(headers.ModelId))
	assert.NotEmpty(t, w.Header().Get(headers.ModelVersion))
	assert.NotEmpty(t, w.Header().Get(headers.EmbeddingTimeMs))
	assert.NotEmpty(t, w.Header().Get(headers.EmbeddingCallsCount))

	// Multi-item header (query returns multiple results)
	assert.NotEmpty(t, w.Header().Get(headers.ResponseReturnedItemsCount))
}

func TestIsEmbeddingEndpoint(t *testing.T) {
	// Test cases for embedding endpoints
	testCases := []struct {
		method   string
		path     string
		expected bool
		name     string
	}{
		{"PUT", "/v1/isolation/collections/test/documents", true, "PUT documents"},
		{"POST", "/v1/isolation/collections/test/documents", true, "POST documents"},
		{"PUT", "/v1/isolation/collections/test/file", true, "PUT file upload"},
		{"POST", "/v1/isolation/collections/test/query/chunks", true, "POST query chunks"},
		{"POST", "/v1/isolation/collections/test/query/documents", true, "POST query documents"},
		{"GET", "/v1/isolation/collections/test/documents", false, "GET documents"},
		{"DELETE", "/v1/isolation/collections/test/documents", false, "DELETE documents"},
		{"GET", "/v2/isolation/collections", false, "GET collections"},
		{"POST", "/v2/isolation/collections", false, "POST collections"},
		{"PUT", "/v1/isolation/collections/test/chunks", false, "PUT chunks (not embedding)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isEmbeddingEndpoint(tc.method, tc.path)
			assert.Equal(t, tc.expected, result, "Expected %s %s to be embedding endpoint: %v", tc.method, tc.path, tc.expected)
		})
	}
}

func TestIsMultiItemEndpoint(t *testing.T) {
	// Test cases for multi-item endpoints
	testCases := []struct {
		method   string
		path     string
		expected bool
		name     string
	}{
		{"GET", "/v2/isolation/collections", true, "GET collections list"},
		{"POST", "/v1/isolation/collections/test/documents", true, "POST documents status"},
		{"POST", "/v1/isolation/collections/test/query/chunks", true, "POST query chunks"},
		{"POST", "/v1/isolation/collections/test/query/documents", true, "POST query documents"},
		{"GET", "/v2/isolation/collections/test/documents/123/chunks", true, "GET document chunks"},
		{"POST", "/v1/isolation/collections/test/attributes", true, "POST attributes list"},
		{"GET", "/v1/isolation/smart-attributes-group", true, "GET attributes groups"},
		{"POST", "/v2/isolation/collections/test/find-documents", true, "POST find documents"},
		{"GET", "/v2/isolation/collections/specific-id", false, "GET specific collection"},
		{"POST", "/v1/isolation/collections/test/document/delete-by-id", false, "POST delete by ID"},
		{"GET", "/v1/isolation/smart-attributes-group/group-123", false, "GET specific attributes group"},
		{"PUT", "/v1/isolation/collections/test/documents", false, "PUT documents (single operation)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isMultiItemEndpoint(tc.method, tc.path)
			assert.Equal(t, tc.expected, result, "Expected %s %s to be multi-item endpoint: %v", tc.method, tc.path, tc.expected)
		})
	}
}

func TestContainsGroupID(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
		name     string
	}{
		{"/v1/isolation/smart-attributes-group", false, "attributes group endpoint only"},
		{"/v1/isolation/smart-attributes-group/", false, "attributes group endpoint with trailing slash"},
		{"/v1/isolation/smart-attributes-group/group-123", true, "attributes group with ID"},
		{"/v1/isolation/smart-attributes-group/82ed8336-67ad-4449-9f30-772359c90dc8", true, "attributes group with UUID"},
		{"/other/path", false, "non-attributes-group path"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsGroupID(tc.path)
			assert.Equal(t, tc.expected, result, "Expected %s to contain group ID: %v", tc.path, tc.expected)
		})
	}
}

func TestAttributesPatternMatching(t *testing.T) {
	// Test the exact path from the failing test
	testPath := "/v1/iso-rvovhwdkyq/collections/col-wsiez/attributes"
	testMethod := "POST"

	// Test the attributes pattern
	attributesPattern := regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/attributes$`)

	t.Logf("Testing path: %s", testPath)
	t.Logf("Testing method: %s", testMethod)
	t.Logf("Pattern: %s", attributesPattern.String())

	if attributesPattern.MatchString(testPath) {
		t.Logf("✓ Attributes pattern MATCHES")
	} else {
		t.Errorf("✗ Attributes pattern does NOT match")
	}

	// Test other patterns that might be interfering
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
		method  string
	}{
		{
			name:    "Document listing (POST)",
			pattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			method:  "POST",
		},
		{
			name:    "Document PUT operations",
			pattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			method:  "PUT",
		},
		{
			name:    "Document DELETE by attributes",
			pattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			method:  "DELETE",
		},
		{
			name:    "Documents GET (list)",
			pattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			method:  "GET",
		},
	}

	for _, p := range patterns {
		if p.method == testMethod && p.pattern.MatchString(testPath) {
			t.Errorf("✗ CONFLICTING PATTERN: %s pattern matches when it shouldn't", p.name)
		} else if p.pattern.MatchString(testPath) {
			t.Logf("- %s pattern matches (but different method: %s)", p.name, p.method)
		} else {
			t.Logf("- %s pattern does not match", p.name)
		}
	}
}

func TestSmartAttributesGroupPatterns(t *testing.T) {
	// Set emulation mode
	os.Setenv("EMULATION_MODE", "true")
	defer os.Unsetenv("EMULATION_MODE")

	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := zap.NewNop()
	router.Use(EmulationMiddleware(logger))

	// Add a catch-all handler that should never be reached in emulation mode
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		body           string
	}{
		{
			name:           "GET smart-attributes-group list (no slash)",
			method:         "GET",
			path:           "/v1/iso-cfplvfjgkg/smart-attributes-group",
			expectedStatus: 200,
		},
		{
			name:           "GET smart-attributes-group list (with slash)",
			method:         "GET",
			path:           "/v1/iso-cfplvfjgkg/smart-attributes-group/",
			expectedStatus: 200,
		},
		{
			name:           "GET smart-attributes-group by ID (no slash)",
			method:         "GET",
			path:           "/v1/iso-zlyrzvlflw/smart-attributes-group/test-group-1",
			expectedStatus: 200,
		},
		{
			name:           "GET smart-attributes-group by ID (with slash)",
			method:         "GET",
			path:           "/v1/iso-zlyrzvlflw/smart-attributes-group/test-group-1/",
			expectedStatus: 200,
		},
		{
			name:           "POST smart-attributes-group create (no slash)",
			method:         "POST",
			path:           "/v1/iso-rfuqdmwyix/smart-attributes-group",
			expectedStatus: 200,
			body:           `{"description":"test group","attributes":["version","category"]}`,
		},
		{
			name:           "POST smart-attributes-group create (with slash)",
			method:         "POST",
			path:           "/v1/iso-rfuqdmwyix/smart-attributes-group/",
			expectedStatus: 200,
			body:           `{"description":"test group","attributes":["version","category"]}`,
		},
		{
			name:           "DELETE smart-attributes-group (no slash)",
			method:         "DELETE",
			path:           "/v1/iso-hoqxooxxam/smart-attributes-group/test-group-1",
			expectedStatus: 200,
		},
		{
			name:           "DELETE smart-attributes-group (with slash)",
			method:         "DELETE",
			path:           "/v1/iso-hoqxooxxam/smart-attributes-group/test-group-1/",
			expectedStatus: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tc.body != "" {
				req, err = http.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			} else {
				req, err = http.NewRequest(tc.method, tc.path, nil)
			}
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(headers.ServiceMode, string(config.ServiceModeEmulation))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			// Check if response is valid JSON (except for DELETE which may be empty)
			if tc.method != "DELETE" {
				var response interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Response is not valid JSON: %v", err)
				}
			}
		})
	}
}
