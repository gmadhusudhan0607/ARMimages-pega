/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	emulationLogger = log.GetNamedLogger("emulation-middleware")

	// Initialize random seed
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Constants for common strings
	defaultEmbeddingProfile = "openai-text-embedding-ada-002"
	fakeDocumentID          = "fake-doc-1"
	fakeCollectionPrefix    = "fake-collection-"
)

// HTTP method constants
const (
	methodGET    = "GET"
	methodPOST   = "POST"
	methodPUT    = "PUT"
	methodPATCH  = "PATCH"
	methodDELETE = "DELETE"
)

// Path segment constants
const (
	pathV1                   = "/v1/"
	pathV2                   = "/v2/"
	pathCollections          = "/collections"
	pathCollectionsSlash     = "/collections/"
	pathDocuments            = "/documents"
	pathChunks               = "/chunks"
	pathQuery                = "/query"
	pathQueryChunks          = "/query/chunks"
	pathQueryDocuments       = "/query/documents"
	pathFindDocuments        = "/find-documents"
	pathAttributes           = "/attributes"
	pathSmartAttributesGroup = "/smart-attributes-group"
	pathDeleteByID           = "/delete-by-id"
	pathFile                 = "/file"
)

// EndpointMatcher represents a rule for matching endpoints
type EndpointMatcher struct {
	Method      string
	PathPattern *regexp.Regexp
	Handler     ResponseGenerator
	StatusCode  func(consistencyLevel string) int
}

// ResponseGenerator generates fake responses for matched endpoints
type ResponseGenerator func(path string) interface{}

// EmulationConfig holds configuration for the emulation middleware
type EmulationConfig struct {
	MinDelayMs int64
	MaxDelayMs int64
}

// EmulationMiddleware returns a middleware that provides fake responses with configurable delays
// when emulation mode is enabled via environment variables
func EmulationMiddleware(logger *zap.Logger) gin.HandlerFunc {
	config := &EmulationConfig{
		MinDelayMs: helpers.GetEmulationMinTime(),
		MaxDelayMs: helpers.GetEmulationMaxTime(),
	}

	return func(c *gin.Context) {
		if !isEmulationModeActive(c) {
			c.Next()
			return
		}

		delay := calculateDelay(config)

		logger.Info("Emulation mode active - returning fake response",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Duration("delay", delay),
		)

		// Sleep for the calculated delay
		time.Sleep(delay)

		consistencyLevel := c.DefaultQuery("consistencyLevel", "eventual")

		// Generate response using improved endpoint matching
		fakeResponse, statusCode := generateResponseForEndpoint(c.Request.Method, c.Request.URL.Path, consistencyLevel)

		// Set fake response headers for emulation
		setEmulationHeaders(c, c.Request.Method, c.Request.URL.Path, delay)

		c.JSON(statusCode, fakeResponse)
		c.Abort()
	}
}

// calculateDelay calculates a random delay within the configured range
func calculateDelay(config *EmulationConfig) time.Duration {
	minTime, maxTime := config.MinDelayMs, config.MaxDelayMs

	// Ensure min <= max
	if minTime > maxTime {
		minTime, maxTime = maxTime, minTime
	}

	var delayMs int64
	if minTime == maxTime {
		delayMs = minTime
	} else {
		delayMs = rng.Int63n(maxTime-minTime) + minTime
	}

	return time.Duration(delayMs) * time.Millisecond
}

// isEmulationModeActive determines if emulation mode is active based on environment variables and runtime configuration
func isEmulationModeActive(c *gin.Context) bool {
	// 1. Check EMULATION environment variable
	if helpers.IsEmulationEnabled() {
		emulationLogger.Debug("Emulation mode activated from environment variable")
		return true
	}

	// 2. Check runtime configuration from context
	if runtimeConfig := config.GetRuntimeConfigFromContext(c.Request.Context()); runtimeConfig != nil {
		if runtimeConfig.ServiceMode == config.ServiceModeEmulation {
			emulationLogger.Debug("Emulation mode activated from runtime configuration",
				zap.String("client_ip", c.ClientIP()),
			)
			return true
		}
	}

	// 3. Fallback: Check header directly if runtime configuration via headers is not enabled
	if !helpers.IsRuntimeConfigurationViaHeadersEnabled() {
		if serviceModeHeader := c.GetHeader(headers.ServiceMode); serviceModeHeader != "" {
			if serviceMode := config.ParseServiceMode(serviceModeHeader); serviceMode == config.ServiceModeEmulation {
				emulationLogger.Debug("Emulation mode activated from direct header check",
					zap.String("client_ip", c.ClientIP()),
				)
				return true
			}
		}
	}

	return false
}

// generateResponseForEndpoint generates appropriate fake responses and status codes based on the endpoint
func generateResponseForEndpoint(method, path, consistencyLevel string) (interface{}, int) {
	// Log the incoming request for debugging
	emulationLogger.Info("Processing emulation request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("consistencyLevel", consistencyLevel),
	)

	// Define endpoint matchers with their patterns and handlers
	// Order matters! More specific patterns should come first
	matchers := []EndpointMatcher{
		// Document chunks - must be before general documents pattern
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections/[^/]+/documents/[^/]+/chunks$`),
			Handler:     generateDocumentChunksResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Find documents - must be before general collection patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections/[^/]+/find-documents$`),
			Handler:     generateFindDocumentsResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// File upload operations - must be before general document patterns
		{
			Method:      methodPUT,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/file$`),
			Handler:     generateFileUploadResponse,
			StatusCode:  func(string) int { return http.StatusCreated },
		},
		// File upload text operations - must be before general document patterns
		{
			Method:      methodPUT,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/file/text$`),
			Handler:     generateFileUploadResponse,
			StatusCode:  func(string) int { return http.StatusCreated },
		},
		// Document DELETE by ID (POST method) - must be before general document patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/document/delete-by-id$`),
			Handler:     generateDocumentDeleteResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Single document GET - must be before general documents pattern
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents/[^/]+$`),
			Handler:     generateSingleDocumentResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Document PATCH operations - must be before general document patterns
		{
			Method:      methodPATCH,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents/[^/]+$`),
			Handler:     generateDocumentPatchResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Document DELETE by ID (DELETE method) - must be before general document patterns
		{
			Method:      methodDELETE,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents/[^/]+$`),
			Handler:     generateDocumentDeleteResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Document PUT operations - must be before general document patterns
		{
			Method:      methodPUT,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			Handler:     generateDocumentPutResponse,
			StatusCode:  defineStatusCodePut,
		},
		// Document DELETE by attributes - must be before general document patterns
		{
			Method:      methodDELETE,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			Handler:     generateDocumentDeleteResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Documents GET (list) - must be before general document patterns
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			Handler:     generateDocumentsGetResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Query chunks - must be before general document patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/query/chunks$`),
			Handler:     generateQueryChunksResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Query documents - must be before general document patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/query/documents$`),
			Handler:     generateQueryDocumentsResponse,
			StatusCode:  func(string) int { return http.StatusAccepted },
		},
		// Attributes listing - must be before general document patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/attributes$`),
			Handler:     generateAttributesListResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Document listing (POST) - must be before general document patterns
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/collections/[^/]+/documents$`),
			Handler:     generateDocumentsListResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Smart attributes group by ID - must be before smart attributes group list
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/smart-attributes-group/[^/]+/?$`),
			Handler:     generateSmartAttributesGroupResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Smart attributes group list - handle both with and without trailing slash
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/smart-attributes-group/?$`),
			Handler:     generateSmartAttributesGroupListResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Smart attributes group creation - handle both with and without trailing slash
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/smart-attributes-group/?$`),
			Handler:     generateSmartAttributesGroupCreateResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Smart attributes group deletion
		{
			Method:      methodDELETE,
			PathPattern: regexp.MustCompile(`^/v1/[^/]+/smart-attributes-group/[^/]+/?$`),
			Handler:     generateEmptySuccessResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Single collection GET - must be before collections list
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections/[^/]+$`),
			Handler:     generateSingleCollectionResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Collection deletion - must be before collections list
		{
			Method:      methodDELETE,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections/[^/]+$`),
			Handler:     generateEmptySuccessResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
		// Collection creation
		{
			Method:      methodPOST,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections$`),
			Handler:     generateCollectionCreationResponse,
			StatusCode:  func(string) int { return 202 },
		},
		// Collections list
		{
			Method:      methodGET,
			PathPattern: regexp.MustCompile(`^/v2/[^/]+/collections$`),
			Handler:     generateCollectionsListResponse,
			StatusCode:  func(string) int { return http.StatusOK },
		},
	}

	// Try to match against defined patterns
	for i, matcher := range matchers {
		emulationLogger.Info("Trying pattern match",
			zap.Int("pattern_index", i),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("expected_method", matcher.Method),
			zap.String("pattern", matcher.PathPattern.String()),
		)
		if matcher.Method == method && matcher.PathPattern.MatchString(path) {
			emulationLogger.Info("Pattern matched successfully",
				zap.Int("pattern_index", i),
				zap.String("method", method),
				zap.String("path", path),
				zap.String("pattern", matcher.PathPattern.String()),
			)
			response := matcher.Handler(path)
			statusCode := matcher.StatusCode(consistencyLevel)
			return response, statusCode
		}
	}

	emulationLogger.Warn("No pattern matched - returning fallback response",
		zap.String("method", method),
		zap.String("path", path),
	)

	return gin.H{
		"message":   "Bad request. Invalid input or emulation do not support it.",
		"timestamp": time.Now().Format(time.RFC3339),
	}, http.StatusBadRequest
}

// Response generators for specific endpoints
func generateDocumentChunksResponse(path string) interface{} {
	return gin.H{
		"documentID": fakeDocumentID,
		"chunks": []gin.H{
			{
				"id":      fakeDocumentID + "-EMB-" + fmt.Sprintf("%d", rng.Intn(10)+1),
				"content": "This is fake chunk content for emulation testing",
				"attributes": []gin.H{
					{
						"name":  "version",
						"value": []string{"8.8"},
						"type":  "string",
					},
				},
			},
		},
	}
}

func generateFindDocumentsResponse(path string) interface{} {
	return gin.H{
		"documents": []gin.H{
			{
				"documentID":      fakeDocumentID,
				"ingestionStatus": "Completed",
				"ingestionTime":   time.Now().Format(time.RFC3339),
				"updateTime":      time.Now().Format(time.RFC3339),
				"errorMessage":    "",
				"chunkStatus":     gin.H{"COMPLETED": 10},
				"documentAttributes": []gin.H{
					{
						"name":   "version",
						"values": []string{"8.8"},
						"type":   "string",
					},
				},
			},
		},
		"pagination": gin.H{
			"limit":      500,
			"itemsTotal": 1,
		},
	}
}

func generateCollectionCreationResponse(path string) interface{} {
	return gin.H{
		"collectionID":            generateFakeCollectionID(),
		"defaultEmbeddingProfile": defaultEmbeddingProfile,
		"documentsTotal":          generateDocumentsCount(),
	}
}

func generateSingleCollectionResponse(path string) interface{} {
	return gin.H{
		"collectionID":            fakeCollectionPrefix + "1",
		"defaultEmbeddingProfile": defaultEmbeddingProfile,
		"documentsTotal":          generateDocumentsCount(),
	}
}

func generateCollectionsListResponse(path string) interface{} {
	isolationID := extractIsolationID(path)
	return gin.H{
		"isolationID": isolationID,
		"collections": []gin.H{
			{
				"collectionID":            fakeCollectionPrefix + "1",
				"defaultEmbeddingProfile": defaultEmbeddingProfile,
				"documentsTotal":          generateDocumentsCount(),
			},
			{
				"collectionID":            fakeCollectionPrefix + "2",
				"defaultEmbeddingProfile": defaultEmbeddingProfile,
				"documentsTotal":          generateDocumentsCount(),
			},
		},
	}
}

func generateQueryChunksResponse(path string) interface{} {
	return []gin.H{
		{
			"content":    "Fake query result for emulation",
			"documentID": fakeDocumentID,
			"distance":   rng.Float64(),
			"attributes": []gin.H{
				{
					"name":  "version",
					"value": []string{"8.8"},
					"type":  "string",
				},
			},
		},
	}
}

func generateQueryDocumentsResponse(path string) interface{} {
	return []gin.H{
		{
			"id":     fakeDocumentID,
			"status": "COMPLETED",
		},
	}
}

func generateDocumentsListResponse(path string) interface{} {
	return []gin.H{
		{
			"id":     fakeDocumentID,
			"status": "COMPLETED",
			"error":  "",
		},
		{
			"id":     "fake-doc-2",
			"status": "IN_PROGRESS",
			"error":  "",
		},
		{
			"id":     "fake-doc-3",
			"status": "ERROR",
			"error":  "Sample error message for testing",
		},
	}
}

func generateSingleDocumentResponse(path string) interface{} {
	return gin.H{
		"id":        generateFakeID(),
		"status":    "success",
		"message":   "Fake response - operation completed successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func generateDocumentsGetResponse(path string) interface{} {
	return gin.H{
		"documents": []gin.H{
			{
				"id":      fakeDocumentID,
				"content": "This is fake document content for emulation testing",
				"metadata": gin.H{
					"title":  "Fake Document 1",
					"author": "Emulation System",
				},
				"created_at": time.Now().Format(time.RFC3339),
			},
		},
		"total": 1,
	}
}

func generateEmptySuccessResponse(path string) interface{} {
	return gin.H{}
}

func generateDocumentPutResponse(path string) interface{} {
	return gin.H{
		"status":    "success",
		"message":   "Fake response - document created/updated successfully",
		"id":        generateFakeID(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func generateDocumentPatchResponse(path string) interface{} {
	return gin.H{
		"status":    "success",
		"message":   "Fake response - document updated successfully",
		"id":        generateFakeID(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func generateDocumentDeleteResponse(path string) interface{} {
	return gin.H{
		"status":    "success",
		"message":   "Fake response - document deleted successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func generateFileUploadResponse(path string) interface{} {
	return gin.H{
		"status":       "success",
		"message":      "Fake response - file uploaded successfully",
		"documentID":   generateFakeID(),
		"uploadStatus": "completed",
		"timestamp":    time.Now().Format(time.RFC3339),
	}
}

func generateSmartAttributesGroupListResponse(path string) interface{} {
	return []gin.H{
		{
			"groupID":     "test-group-1",
			"description": "Fake attributes group for emulation testing",
		},
		{
			"groupID":     "test-group-2",
			"description": "Another fake attributes group",
		},
	}
}

func generateSmartAttributesGroupResponse(path string) interface{} {
	return gin.H{
		"groupID":     "test-group-1",
		"description": "Fake attributes group for emulation testing",
		"attributes": gin.H{
			"attributes": []string{"version", "category", "source"},
		},
	}
}

func generateSmartAttributesGroupCreateResponse(path string) interface{} {
	return gin.H{
		"groupID":     generateFakeID(),
		"description": "Fake attributes group for emulation testing",
		"attributes":  []string{"version", "category"},
	}
}

func generateAttributesListResponse(path string) interface{} {
	return gin.H{
		"attributes": []gin.H{
			{
				"name":   "version",
				"type":   "string",
				"values": []string{"8.8", "8.7", "8.6"},
			},
			{
				"name":   "category",
				"type":   "string",
				"values": []string{"test", "production", "development"},
			},
			{
				"name":   "status",
				"type":   "string",
				"values": []string{"active", "inactive"},
			},
		},
		"total": 3,
	}
}

func defineStatusCodePut(consistencyLevel string) int {
	if consistencyLevel == indexer.ConsistencyLevelStrong {
		return http.StatusCreated
	} else {
		return http.StatusAccepted
	}
}

// Utility functions with improved efficiency
func generateFakeID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const idLength = 8

	b := make([]byte, idLength)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return "fake-" + string(b)
}

func generateFakeCollectionID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const idLength = 5

	b := make([]byte, idLength)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return "col-" + string(b)
}

// extractIsolationID extracts the isolationID from v2 API paths
func extractIsolationID(path string) string {
	if idx := strings.Index(path, pathV2); idx != -1 {
		start := idx + len(pathV2)
		if end := strings.Index(path[start:], "/"); end != -1 {
			return path[start : start+end]
		}
		return path[start:]
	}
	return "isolation-1"
}

// Header management functions
func setEmulationHeaders(c *gin.Context, method, path string, delay time.Duration) {
	// Always set basic headers
	c.Header(headers.RequestDurationMs, fmt.Sprintf("%d", generateRequestDurationMs(delay)))
	c.Header(headers.DbQueryTimeMs, fmt.Sprintf("%d", generateDbQueryTimeMs()))

	// Set specialized headers based on endpoint type
	if isCollectionEndpoint(method, path) {
		c.Header(headers.DocumentsCount, fmt.Sprintf("%d", generateDocumentsCount()))
		c.Header(headers.VectorsCount, fmt.Sprintf("%d", generateVectorsCount()))
	}

	if isEmbeddingEndpoint(method, path) {
		c.Header(headers.ModelId, generateFakeModelId())
		c.Header(headers.ModelVersion, generateFakeModelVersion())
		c.Header(headers.EmbeddingTimeMs, fmt.Sprintf("%d", generateEmbeddingTimeMs()))
		c.Header(headers.EmbeddingCallsCount, fmt.Sprintf("%d", generateEmbeddingCallsCount()))
	}

	if isMultiItemEndpoint(method, path) {
		c.Header(headers.ResponseReturnedItemsCount, fmt.Sprintf("%d", generateReturnedItemsCount(path)))
	}
}

// Endpoint classification functions using efficient string operations
func isEmbeddingEndpoint(method, path string) bool {
	if (method == methodPUT || method == methodPOST) && strings.Contains(path, pathDocuments) {
		return true
	}
	if method == methodPUT && strings.Contains(path, pathFile) {
		return true
	}
	if method == methodPOST && (strings.Contains(path, pathQueryChunks) || strings.Contains(path, pathQueryDocuments)) {
		return true
	}
	return false
}

func isCollectionEndpoint(method, path string) bool {
	return strings.Contains(path, pathCollectionsSlash) && strings.Contains(path, pathDocuments) ||
		method == methodPOST && (strings.Contains(path, pathQueryChunks) || strings.Contains(path, pathQueryDocuments)) ||
		strings.Contains(path, pathCollections) ||
		strings.Contains(path, pathFile) ||
		strings.Contains(path, pathChunks) ||
		strings.Contains(path, pathFindDocuments)
}

func isMultiItemEndpoint(method, path string) bool {
	if method == methodGET && strings.Contains(path, pathCollections) && !strings.Contains(path, pathCollectionsSlash) {
		return true
	}
	if method == methodPOST && strings.Contains(path, pathDocuments) && !strings.Contains(path, pathDeleteByID) {
		return true
	}
	if method == methodPOST && (strings.Contains(path, pathQueryChunks) || strings.Contains(path, pathQueryDocuments)) {
		return true
	}
	if method == methodGET && strings.Contains(path, pathChunks) {
		return true
	}
	if method == methodPOST && strings.Contains(path, pathAttributes) && !strings.Contains(path, pathSmartAttributesGroup) {
		return true
	}
	if method == methodGET && strings.Contains(path, pathSmartAttributesGroup) && !containsGroupID(path) {
		return true
	}
	if method == methodPOST && strings.Contains(path, pathFindDocuments) {
		return true
	}
	return false
}

func containsGroupID(path string) bool {
	if idx := strings.Index(path, pathSmartAttributesGroup+"/"); idx != -1 {
		remaining := path[idx+len(pathSmartAttributesGroup)+1:]
		return len(remaining) > 0
	}
	return false
}

// Header value generators with improved efficiency
func generateRequestDurationMs(delay time.Duration) int64 {
	totalMs := delay.Milliseconds() + int64(rng.Intn(50)+10)
	return totalMs
}

func generateDbQueryTimeMs() int {
	queryTime := rng.Intn(195) + 5
	return queryTime
}

func generateFakeModelId() string {
	models := []string{
		"openai-text-embedding-ada-002",
		"openai-text-embedding-3-small",
		"openai-text-embedding-3-large",
		"text-embedding-ada-002",
	}
	return models[rng.Intn(len(models))]
}

func generateFakeModelVersion() string {
	versions := []string{"1", "2", "3"}
	return versions[rng.Intn(len(versions))]
}

func generateEmbeddingTimeMs() int {
	embeddingTime := rng.Intn(450) + 50
	return embeddingTime
}

func generateEmbeddingCallsCount() int {
	callsCount := rng.Intn(5) + 1
	return callsCount
}

func generateReturnedItemsCount(path string) int {
	var count int
	switch {
	case strings.Contains(path, pathCollections):
		count = rng.Intn(10) + 1
	case strings.Contains(path, pathQuery):
		count = rng.Intn(20) + 1
	case strings.Contains(path, pathChunks):
		count = rng.Intn(50) + 1
	case strings.Contains(path, pathDocuments):
		count = rng.Intn(100) + 1
	default:
		count = rng.Intn(10) + 1
	}
	return count
}

func generateDocumentsCount() int {
	documentsCount := rng.Intn(990) + 10
	return documentsCount
}

func generateVectorsCount() int {
	vectorsCount := rng.Intn(4950) + 50
	return vectorsCount
}
