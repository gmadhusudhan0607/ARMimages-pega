/*
* Copyright (c) 2024 Pegasystems Inc.
* All rights reserved.
 */

package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHttpMetricsMiddleware(t *testing.T) {

	ctx := cntx.ServiceContext("metrics_middleware_test")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))
	models := engine.Group("/openai")
	{
		models.POST("/deployments/:modelId/*api", func(c *gin.Context) { c.Status(http.StatusOK) })
	}

	buddies := engine.Group("/v1/:isolationId/buddies")
	{
		buddies.POST("/:buddyId/question", func(c *gin.Context) { c.Status(http.StatusOK) })
	}

	type arguments struct {
		path    string
		method  string
		envGuid string
	}

	var requestData = []arguments{
		{"/openai/deployments/gpt-35-turbo/chat/completions?api-version=2023-05-15", "POST", ""},
		{"/v1/555bd0b8-1bkk-45a8-83ea-9f7840e11235/buddies/buddy-id-001/question?api-version=2023-05-15", "POST", ""},
		{"/openai/deployments/gpt-4o-mini/chat/completions?api-version=2023-05-15", "POST", ""},
		{"/not/valid/model", "POST", ""},
	}

	for _, request := range requestData {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, nil))
	}

	assert.Equal(t, float64(1), testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/openai/deployments/gpt-35-turbo/chat/completions", "POST", "200", "gpt-35-turbo", "2023-05-15", "")))
	assert.Equal(t, float64(1), testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/v1/555bd0b8-1bkk-45a8-83ea-9f7840e11235/buddies/buddy-id-001/question", "POST", "200", "buddy-id-001", "2023-05-15", "")))
	assert.Equal(t, float64(1), testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/openai/deployments/gpt-4o-mini/chat/completions", "POST", "200", "gpt-4o-mini", "2023-05-15", "")))
	assert.Equal(t, float64(0), testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/not/valid/model", "POST", "200", "", "", "")))
	assert.Equal(t, float64(1), testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/not/valid/model", "POST", "404", "", "", "")))
}

func Test_getGuidFromToken(t *testing.T) {
	tests := []struct {
		name         string
		bearerToken  string
		expectedGuid string
	}{
		{
			name:         "Valid token",
			bearerToken:  "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJndWlkIjoiZ3VpZC12YWx1ZSJ9.signature",
			expectedGuid: "guid-value",
		},
		{
			name:         "Token without Bearer prefix",
			bearerToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJndWlkIjoiZ3VpZC12YWx1ZSJ9.signature",
			expectedGuid: "",
		},
		{
			name:         "Invalid token format",
			bearerToken:  "Bearer invalid.token.format",
			expectedGuid: "",
		},
		{
			name:         "Missing guid in claims",
			bearerToken:  "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJub3RfZ3VpZCI6Im5vdC1ndWlkIn0.signature",
			expectedGuid: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock gin context with the authorization header
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tt.bearerToken)
			c.Request = req

			guid, _ := getGuidFromToken(c)
			if guid != tt.expectedGuid {
				t.Errorf("expected %v, got %v", tt.expectedGuid, guid)
			}
		})
	}
}

func TestCustomResponseWriter_StreamingTimingProblem(t *testing.T) {
	// This test demonstrates the real problem using a real HTTP server with streaming responses
	// and the middleware configured on the route

	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	// Create a test server with the metrics middleware
	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	// Add LLM middleware to the OpenAI route group
	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	delays := []time.Duration{30 * time.Millisecond, 200 * time.Millisecond, 50 * time.Millisecond, 30 * time.Millisecond, 20 * time.Millisecond}

	// Define a streaming endpoint that simulates AI model response with delays
	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)

		// Simulate streaming response with delays between chunks
		chunks := []struct {
			data  string
			delay time.Duration
		}{
			{`{"choices":[{"delta":{"content":"First"}}]}` + "\n", delays[0]},
			{`{"choices":[{"delta":{"content":" chunk"}}]}` + "\n", delays[1]},
			{`{"choices":[{"delta":{"content":" second"}}]}` + "\n", delays[2]},
			{`{"choices":[{"delta":{"content":" third"}}]}` + "\n", delays[3]},
			{`{"choices":[{"delta":{"content":" final"}}]}` + "\n", delays[4]},
		}

		for i, chunk := range chunks {
			if chunk.delay > 0 {
				time.Sleep(chunk.delay)
			}

			// Write chunk and flush to simulate real streaming
			_, err := c.Writer.Write([]byte(chunk.data))
			if err != nil {
				t.Errorf("Failed to write chunk %d: %v", i, err)
				return
			}

			// Flush to ensure chunk is sent immediately
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	// Create test server
	server := httptest.NewServer(engine)
	defer server.Close()

	// Make request to streaming endpoint
	resp, err := http.Post(server.URL+"/openai/deployments/gpt-4/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"test"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Read the response body (to ensure all chunks are received)
	_, err = io.ReadAll(resp.Body)

	// Calculate total elapsed time
	assert.NoError(t, err)

	// Check the trailer headers that the client actually received
	responseTimeHeader := resp.Trailer.Get("X-Genai-Gateway-Response-Time-Ms")
	modelIdHeader := resp.Trailer.Get("X-Genai-Gateway-Model-Id")
	regionHeader := resp.Trailer.Get("X-Genai-Gateway-Region")
	ttftHeader := resp.Trailer.Get("X-Genai-Gateway-Time-To-First-Token")

	assert.NotEmpty(t, responseTimeHeader, "Response time trailer header should be present")
	assert.Equal(t, "gpt-4", modelIdHeader, "Model ID should be correct")
	assert.Equal(t, "Standard", regionHeader, "Region should be Standard")
	assert.NotEmpty(t, ttftHeader, "Time To First Token trailer header should be present")

	// Verify trailer headers are present and working correctly
	expectedResponseTime := 0 * time.Millisecond
	for _, d := range delays {
		expectedResponseTime = expectedResponseTime + d
	}

	// Parse and analyze the timing
	reportedTime, err := time.ParseDuration(responseTimeHeader + "ms")
	assert.NoError(t, err, "Should be able to parse response time header")
	reportedMs := reportedTime.Milliseconds()

	assert.GreaterOrEqual(t, reportedMs, expectedResponseTime.Milliseconds(),
		"Response time trailer header should be  at least %dms (total delays), got %dms.", expectedResponseTime, reportedMs)

	expectedTimeToFirstToken := delays[0]
	// Parse and analyze the timing
	ttftTime, err := time.ParseDuration(ttftHeader + "ms")
	assert.NoError(t, err, "Should be able to parse time to first token header")
	reportedTtftMs := ttftTime.Milliseconds()

	assert.GreaterOrEqual(t, reportedTtftMs, expectedTimeToFirstToken.Milliseconds(),
		"Time to First Token trailer header should be  at least %dms (total delays), got %dms.", expectedTimeToFirstToken, reportedTtftMs)
}

// TestChatCompletionsStreamingWithUsageTrailers tests the full end-to-end streaming flow
// for Chat Completions (OpenAI/Vertex) with proper SSE format including usage metadata in trailers
func TestChatCompletionsStreamingWithUsageTrailers(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	// Create a test server with the metrics middleware
	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	// Add LLM middleware to the OpenAI route group
	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	// Define a streaming endpoint that simulates Azure OpenAI / Vertex Chat Completions SSE response
	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		// Simulate streaming SSE response with usage metadata in the final chunk
		// This is the format used by Azure OpenAI and Vertex AI (OpenAI-compatible)
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1694268190,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1694268190,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1694268190,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1694268190,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"!\"},\"finish_reason\":null}]}\n\n",
			// Final chunk with usage metadata - this is what we're testing to extract
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1694268190,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":25,\"completion_tokens\":75,\"total_tokens\":100}}\n\n",
			"data: [DONE]\n\n",
		}

		for i, chunk := range chunks {
			// Small delay to simulate real streaming
			time.Sleep(10 * time.Millisecond)

			// Write chunk and flush
			_, err := c.Writer.Write([]byte(chunk))
			if err != nil {
				t.Errorf("Failed to write chunk %d: %v", i, err)
				return
			}

			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	// Create test server
	server := httptest.NewServer(engine)
	defer server.Close()

	// Make streaming request
	resp, err := http.Post(server.URL+"/openai/deployments/gpt-4/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"Say hello"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Read the entire response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Verify the response body contains SSE data
	assert.Contains(t, string(body), "data: ", "Response should contain SSE data prefix")
	assert.Contains(t, string(body), "[DONE]", "Response should contain DONE marker")

	// Check trailer headers - these should contain the extracted usage metrics
	inputTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Input-Tokens")
	outputTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Output-Tokens")
	tokensPerSecondHeader := resp.Trailer.Get("X-Genai-Gateway-Tokens-Per-Second")
	responseTimeHeader := resp.Trailer.Get("X-Genai-Gateway-Response-Time-Ms")
	ttftHeader := resp.Trailer.Get("X-Genai-Gateway-Time-To-First-Token")
	modelIdHeader := resp.Trailer.Get("X-Genai-Gateway-Model-Id")

	// Verify all expected trailer headers are present
	assert.NotEmpty(t, responseTimeHeader, "Response time trailer should be present")
	assert.NotEmpty(t, ttftHeader, "Time to first token trailer should be present")
	assert.Equal(t, "gpt-4", modelIdHeader, "Model ID trailer should be correct")

	// Verify token metrics are extracted from SSE stream
	assert.Equal(t, "25", inputTokensHeader, "Input tokens should be extracted from SSE stream (25 prompt tokens)")
	assert.Equal(t, "75", outputTokensHeader, "Output tokens should be extracted from SSE stream (75 completion tokens)")
	assert.NotEmpty(t, tokensPerSecondHeader, "Tokens per second should be calculated")

	// Verify tokens per second is a valid number
	tps, err := strconv.Atoi(tokensPerSecondHeader)
	assert.NoError(t, err, "Tokens per second should be a valid integer")
	assert.GreaterOrEqual(t, tps, 0, "Tokens per second should be non-negative")
}

// TestChatCompletionsStreamingWithoutUsage tests graceful handling when usage is not provided
func TestChatCompletionsStreamingWithoutUsage(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	// Streaming response without usage metadata (some providers don't include it)
	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		chunks := []string{
			"data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n",
			"data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n",
			"data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n", // No usage!
			"data: [DONE]\n\n",
		}

		for _, chunk := range chunks {
			time.Sleep(5 * time.Millisecond)
			_, _ = c.Writer.Write([]byte(chunk))
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	resp, err := http.Post(server.URL+"/openai/deployments/gpt-4/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"test"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Per US requirements: graceful degradation when usage not provided
	// Trailers should still be present but with 0 values
	inputTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Input-Tokens")
	outputTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Output-Tokens")

	assert.Equal(t, "0", inputTokensHeader, "Input tokens should be 0 when not provided in stream")
	assert.Equal(t, "0", outputTokensHeader, "Output tokens should be 0 when not provided in stream")

	// Basic timing headers should still be present
	assert.NotEmpty(t, resp.Trailer.Get("X-Genai-Gateway-Response-Time-Ms"), "Response time should be present")
	assert.Equal(t, "gpt-4", resp.Trailer.Get("X-Genai-Gateway-Model-Id"), "Model ID should be present")
}

func TestChatCompletionsTemplate_GetTokensPerSecond(t *testing.T) {
	tests := []struct {
		name        string
		template    chatCompletionsTemplate
		expectedTPS float64
	}{
		{
			name: "Valid values",
			template: chatCompletionsTemplate{
				Usage: chatCompletionsUsage{
					CompletionTokens: 100,
					PromptTokens:     300,
					TotalTokens:      400,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 1000,
				},
			},
			expectedTPS: 100,
		},
		{
			name: "Zero total tokens",
			template: chatCompletionsTemplate{
				Usage: chatCompletionsUsage{
					CompletionTokens: 100,
					PromptTokens:     300,
					TotalTokens:      0,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 1000,
				},
			},
			expectedTPS: 0,
		},
		{
			name: "Zero latency",
			template: chatCompletionsTemplate{
				Usage: chatCompletionsUsage{
					CompletionTokens: 100,
					PromptTokens:     300,
					TotalTokens:      400,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 0,
				},
			},
			expectedTPS: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTPS := tt.template.getTokensPerSecond()
			assert.InDelta(t, tt.expectedTPS, actualTPS, 0.001, "getTokensPerSecond() result mismatch")
		})
	}
}

func TestConverseTemplate_GetTokensPerSecond(t *testing.T) {
	tests := []struct {
		name        string
		template    converseTemplate
		expectedTPS float64
	}{
		{
			name: "Valid values",
			template: converseTemplate{
				Usage: struct {
					InputTokens  int `json:"inputTokens"`
					OutputTokens int `json:"outputTokens"`
					TotalTokens  int `json:"totalTokens"`
				}{
					InputTokens:  300,
					OutputTokens: 100,
					TotalTokens:  400,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 1000,
				},
			},
			expectedTPS: 100,
		},
		{
			name: "Zero input tokens",
			template: converseTemplate{
				Usage: struct {
					InputTokens  int `json:"inputTokens"`
					OutputTokens int `json:"outputTokens"`
					TotalTokens  int `json:"totalTokens"`
				}{
					InputTokens:  0,
					OutputTokens: 100,
					TotalTokens:  400,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 1000,
				},
			},
			expectedTPS: 0,
		},
		{
			name: "Zero latency",
			template: converseTemplate{
				Usage: struct {
					InputTokens  int `json:"inputTokens"`
					OutputTokens int `json:"outputTokens"`
					TotalTokens  int `json:"totalTokens"`
				}{
					InputTokens:  300,
					OutputTokens: 100,
					TotalTokens:  400,
				},
				Metrics: struct {
					LatencyMs int `json:"latencyMs"`
				}{
					LatencyMs: 0,
				},
			},
			expectedTPS: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTPS := tt.template.getTokensPerSecond()
			assert.InDelta(t, tt.expectedTPS, actualTPS, 0.001, "getTokensPerSecond() result mismatch")
		})
	}
}

func TestGatewayHeadersFollowCanonicalFormat(t *testing.T) {
	for _, h := range AllGatewayHeaders {
		result := http.CanonicalHeaderKey(string(h))
		assert.Equal(t, result, string(h), "Header format mismatch")
	}
}

func TestExtractGenAIHeaders(t *testing.T) {
	tests := []struct {
		name                   string
		headers                map[string]string
		expectedRequestID      string
		expectedContextID      string
		expectedConversationID string
	}{
		{
			name: "All headers present",
			headers: map[string]string{
				string(GenAIServiceRequestID): "req-123",
				string(GenAIContextID):        "ctx-456",
				string(GenAIConversationID):   "conv-789",
			},
			expectedRequestID:      "req-123",
			expectedContextID:      "ctx-456",
			expectedConversationID: "conv-789",
		},
		{
			name: "Only request and context headers present",
			headers: map[string]string{
				string(GenAIServiceRequestID): "req-abc",
				string(GenAIContextID):        "ctx-def",
			},
			expectedRequestID:      "req-abc",
			expectedContextID:      "ctx-def",
			expectedConversationID: "", // Should be empty when missing
		},
		{
			name: "Only conversation header present",
			headers: map[string]string{
				string(GenAIConversationID): "conv-xyz",
			},
			expectedRequestID:      "", // Should be empty when missing
			expectedContextID:      "", // Should be empty when missing
			expectedConversationID: "conv-xyz",
		},
		{
			name:                   "No GenAI headers present",
			headers:                map[string]string{},
			expectedRequestID:      "", // Should be empty when missing
			expectedContextID:      "", // Should be empty when missing
			expectedConversationID: "", // Should be empty when missing
		},
		{
			name: "Headers with empty values",
			headers: map[string]string{
				string(GenAIServiceRequestID): "",
				string(GenAIContextID):        "",
				string(GenAIConversationID):   "",
			},
			expectedRequestID:      "", // Empty values should remain empty
			expectedContextID:      "",
			expectedConversationID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)

			// Set up the request with headers
			req := httptest.NewRequest("POST", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			ginCtx.Request = req

			// Extract headers
			requestID, contextID, conversationID := extractGenAIHeaders(ginCtx)

			// Verify results
			assert.Equal(t, tt.expectedRequestID, requestID, "RequestID should match expected value")
			assert.Equal(t, tt.expectedContextID, contextID, "ContextID should match expected value")
			assert.Equal(t, tt.expectedConversationID, conversationID, "ConversationID should match expected value")
		})
	}
}

func TestGenAIRequestHeaderConstants(t *testing.T) {
	// Verify that the GenAI header constants have the correct values
	assert.Equal(t, "pega-genai-service-request-id", string(GenAIServiceRequestID), "Service request ID header should match specification")
	assert.Equal(t, "pega-genai-context-id", string(GenAIContextID), "Context ID header should match specification")
	assert.Equal(t, "pega-genai-conversation-id", string(GenAIConversationID), "Conversation ID header should match specification")
}

func TestLLMMetricsMiddleware_GenAIHeaderLogging(t *testing.T) {
	tests := []struct {
		name                   string
		headers                map[string]string
		expectedRequestID      string
		expectedContextID      string
		expectedConversationID string
	}{
		{
			name: "All GenAI headers present",
			headers: map[string]string{
				string(GenAIServiceRequestID): "req-test-123",
				string(GenAIContextID):        "ctx-test-456",
				string(GenAIConversationID):   "conv-test-789",
			},
			expectedRequestID:      "req-test-123",
			expectedContextID:      "ctx-test-456",
			expectedConversationID: "conv-test-789",
		},
		{
			name: "Missing conversation ID",
			headers: map[string]string{
				string(GenAIServiceRequestID): "req-missing-conv",
				string(GenAIContextID):        "ctx-missing-conv",
			},
			expectedRequestID:      "req-missing-conv",
			expectedContextID:      "ctx-missing-conv",
			expectedConversationID: "", // Should be empty
		},
		{
			name:                   "No GenAI headers",
			headers:                map[string]string{},
			expectedRequestID:      "", // Should be empty
			expectedContextID:      "", // Should be empty
			expectedConversationID: "", // Should be empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := cntx.ServiceContext("genai_header_test")
			gin.SetMode(gin.TestMode)

			// Create engine with both middlewares
			engine := gin.New()
			engine.Use(HttpMetricsMiddleware(ctx))

			// Add LLM middleware to test route
			testGroup := engine.Group("/openai")
			testGroup.Use(LLMMetricsMiddleware(ctx))
			testGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
				// Simulate LLM response with metrics
				response := `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":1000}}`
				c.Header("Content-Type", "application/json")
				c.String(http.StatusOK, response)
			})

			// Create request with GenAI headers
			req := httptest.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions", nil)
			req.Header.Set("Content-Type", "application/json")
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Execute request
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			// Verify response status
			assert.Equal(t, http.StatusOK, recorder.Code, "Request should succeed")

			// Note: In a real test environment, we would need to capture the log output
			// to verify that the GenAI IDs are properly logged. For now, we verify
			// that the middleware processes the headers without errors.

			// The actual log verification would require setting up a test logger
			// and capturing its output, which is beyond the scope of this basic test.
			// The important part is that the middleware extracts and stores the headers
			// correctly, which we can verify by checking that no panics occur
			// and the request completes successfully.
		})
	}
}

func TestLLMMetricsMiddleware_NonStreaming(t *testing.T) {
	// Test non-streaming responses - should send metrics as regular headers, not trailers
	// This covers both success and failure scenarios to validate different header expectations

	tests := []struct {
		name                 string
		description          string
		responseStatus       int
		responseBody         string
		requestBody          string
		expectedBasicHeaders map[GatewayHeader]string // Headers expected in all scenarios
		expectedTokenHeaders []GatewayHeader          // Headers only expected for successful requests
		excludedHeaders      []GatewayHeader          // Headers that should NOT be present
	}{
		{
			name:           "Success - All gateway headers as regular headers",
			description:    "Successful non-streaming request should include all gateway headers with token metrics",
			responseStatus: http.StatusOK,
			responseBody:   `{"usage":{"prompt_tokens":150,"completion_tokens":75,"total_tokens":225},"metrics":{"latencyMs":800}}`,
			requestBody:    `{"messages":[{"role":"user","content":"Hello, how are you?"}]}`,
			expectedBasicHeaders: map[GatewayHeader]string{
				GatewayModelID:    "gpt-4",
				GatewayRegion:     "Standard",
				GatewayRetryCount: "0",
			},
			expectedTokenHeaders: []GatewayHeader{
				GatewayResponseTimeMs,
				GatewayInputTokens,
				GatewayOutputTokens,
				GatewayTokensPerSecond,
				GatewayTimeToFirstToken,
			},
			excludedHeaders: []GatewayHeader{}, // No headers should be excluded for success
		},
		{
			name:           "Failure - Basic headers only, no token metrics",
			description:    "Failed non-streaming request should exclude token-related headers",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"error":{"message":"Invalid request format","type":"invalid_request_error"}}`,
			requestBody:    `{"invalid_field":"this should cause an error"}`,
			expectedBasicHeaders: map[GatewayHeader]string{
				GatewayModelID:        "gpt-4",
				GatewayRegion:         "Standard",
				GatewayRetryCount:     "0",
				GatewayResponseTimeMs: "", // Present but value will be verified separately
			},
			expectedTokenHeaders: []GatewayHeader{}, // No token headers for failures
			excludedHeaders: []GatewayHeader{
				GatewayInputTokens,
				GatewayOutputTokens,
				GatewayTokensPerSecond,
				GatewayTimeToFirstToken,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cntx.ServiceContext("test")
			gin.SetMode(gin.TestMode)

			// Create engine with both middlewares
			engine := gin.New()
			engine.Use(HttpMetricsMiddleware(ctx))

			// Add LLM middleware to OpenAI route group
			openaiGroup := engine.Group("/openai")
			openaiGroup.Use(LLMMetricsMiddleware(ctx))

			// Define endpoint that returns the configured response
			openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
				c.Header("Content-Type", "application/json")
				c.String(tt.responseStatus, tt.responseBody)
			})

			// Create non-streaming request (no "stream": true in body)
			req := httptest.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions", bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			// Verify response status matches expectation
			assert.Equal(t, tt.responseStatus, recorder.Code, tt.description)

			// Verify non-streaming transport characteristics
			responseHeaders := recorder.Header()
			assert.Empty(t, responseHeaders.Get("Trailer"), "Non-streaming response should not have Trailer header")
			assert.NotEqual(t, "chunked", responseHeaders.Get("Transfer-Encoding"), "Non-streaming response should not use chunked encoding")
			assert.Empty(t, recorder.Result().Trailer, "Non-streaming response should not have any trailer headers")

			// Verify expected basic headers are present with correct values
			for header, expectedValue := range tt.expectedBasicHeaders {
				actualValue := responseHeaders.Get(string(header))
				if expectedValue == "" {
					// Just verify presence for timing headers
					assert.NotEmpty(t, actualValue, "Header %s should be present", string(header))
				} else {
					assert.Equal(t, expectedValue, actualValue, "Header %s should have correct value", string(header))
				}
			}

			// Verify expected token headers are present (for success scenarios)
			for _, header := range tt.expectedTokenHeaders {
				actualValue := responseHeaders.Get(string(header))
				assert.NotEmpty(t, actualValue, "Token header %s should be present for successful requests", string(header))

				// Additional validation for specific headers
				switch header {
				case GatewayInputTokens:
					if tt.responseStatus == http.StatusOK {
						assert.Equal(t, "150", actualValue, "Input tokens should match response data")
					}
				case GatewayOutputTokens:
					if tt.responseStatus == http.StatusOK {
						assert.Equal(t, "75", actualValue, "Output tokens should match response data")
					}
				case GatewayResponseTimeMs:
					// Verify it's a valid integer >= 0
					timeMs, err := strconv.Atoi(actualValue)
					assert.NoError(t, err, "Response time should be a valid integer")
					assert.GreaterOrEqual(t, timeMs, 0, "Response time should be >= 0ms")
				case GatewayTokensPerSecond:
					// Verify it's a valid number (can be negative in fast unit tests due to 0ms timing)
					_, err := strconv.ParseFloat(actualValue, 64)
					assert.NoError(t, err, "Tokens per second should be a valid number")
				}
			}

			// Verify excluded headers are NOT present (for failure scenarios)
			for _, header := range tt.excludedHeaders {
				actualValue := responseHeaders.Get(string(header))
				assert.Empty(t, actualValue, "Header %s should NOT be present for failed requests", string(header))
			}
		})
	}
}

func TestCustomResponseWriter_HeaderDuplication(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		modelId         string
		writeOperations []struct {
			method string // "Write" or "WriteString"
			data   string
		}
		expectedHeaderCount int
		expectedModelId     string
	}{
		{
			name:    "Multiple Write calls should not duplicate headers",
			path:    "/openai/deployments/gpt-4/chat/completions",
			modelId: "gpt-4",
			writeOperations: []struct {
				method string
				data   string
			}{
				{"Write", `{"choices":[{"delta":{"content":"Hello"}}]}`},
				{"Write", `{"choices":[{"delta":{"content":" World"}}]}`},
				{"Write", `{"choices":[{"delta":{"content":"!"}}]}`},
			},
			expectedHeaderCount: 1,
			expectedModelId:     "gpt-4",
		},
		{
			name:    "Response time should reflect total request time, not partial chunk timing",
			path:    "/openai/deployments/gpt-4/chat/completions",
			modelId: "gpt-4",
			writeOperations: []struct {
				method string
				data   string
			}{
				{"Write", `{"choices":[{"delta":{"content":"Chunk1"}}]}`},
				{"Sleep", "200"}, // Biggest delay first - 200ms
				{"Write", `{"choices":[{"delta":{"content":" Chunk2"}}]}`},
				{"Sleep", "50"}, // Smaller delay - 50ms
				{"Write", `{"choices":[{"delta":{"content":" Chunk3"}}]}`},
				{"Sleep", "30"}, // Smaller delay - 30ms
				{"Write", `{"choices":[{"delta":{"content":" Chunk4"}}]}`},
				{"Sleep", "20"}, // Smaller delay - 20ms
				{"Write", `{"choices":[{"delta":{"content":" Chunk5"}}]}`},
			},
			expectedHeaderCount: 1,
			expectedModelId:     "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := cntx.ServiceContext("test")
			recorder := httptest.NewRecorder()

			// Create a gin context with the recorder
			gin.SetMode(gin.TestMode)
			ginCtx, _ := gin.CreateTestContext(recorder)

			crw := &MetricsResponseWriter{
				body:           bytes.NewBuffer(nil),
				path:           tt.path,
				start:          time.Now(),
				ResponseWriter: ginCtx.Writer,
				SugaredLogger:  cntx.LoggerFromContext(ctx).Sugar(),
				llmMetrics: &LLMMetrics{
					modelId:      tt.modelId,
					usesTrailers: true, // simulate streaming mode for these header duplication/timing tests
				},
			}

			// Record start time for timing verification
			testStart := time.Now()

			// Execute write operations
			for _, op := range tt.writeOperations {
				switch op.method {
				case "Write":
					_, err := crw.Write([]byte(op.data))
					assert.NoError(t, err, "Write operation should not fail")
				case "WriteString":
					_, err := crw.WriteString(op.data)
					assert.NoError(t, err, "WriteString operation should not fail")
				case "Sleep":
					// Parse sleep duration from data field
					if sleepMs := op.data; sleepMs != "" {
						if duration, err := time.ParseDuration(sleepMs + "ms"); err == nil {
							time.Sleep(duration)
						}
					}
				}
			}

			// Calculate actual elapsed time for comparison
			actualElapsed := int(time.Since(testStart).Milliseconds())

			// Since we now use trailers for streaming responses, we need to manually call setBaseHeaders
			// and setTokenHeaders to simulate the middleware completion for this unit test
			if crw.llmMetrics != nil && crw.llmMetrics.usesTrailers {
				crw.setBaseHeaders()
				crw.setTokenHeaders(ginCtx)
			}

			// Check trailer headers for streaming responses, regular headers for non-streaming
			var responseTimeHeaders, modelIdHeaders, regionHeaders, retryHeaders []string

			if crw.llmMetrics != nil && crw.llmMetrics.usesTrailers {
				// For streaming responses, check trailer headers
				responseTimeHeaders = recorder.Header().Values("X-Genai-Gateway-Response-Time-Ms")
				modelIdHeaders = recorder.Header().Values("X-Genai-Gateway-Model-Id")
				regionHeaders = recorder.Header().Values("X-Genai-Gateway-Region")
				retryHeaders = recorder.Header().Values("X-Genai-Gateway-Retry-Count")
			} else {
				// For non-streaming responses, check regular headers
				responseTimeHeaders = recorder.Header().Values("X-Genai-Gateway-Response-Time-Ms")
				modelIdHeaders = recorder.Header().Values("X-Genai-Gateway-Model-Id")
				regionHeaders = recorder.Header().Values("X-Genai-Gateway-Region")
				retryHeaders = recorder.Header().Values("X-Genai-Gateway-Retry-Count")
			}

			assert.Len(t, responseTimeHeaders, tt.expectedHeaderCount,
				"Response time header should appear exactly %d time(s)", tt.expectedHeaderCount)
			assert.Len(t, modelIdHeaders, tt.expectedHeaderCount,
				"Model ID header should appear exactly %d time(s)", tt.expectedHeaderCount)
			assert.Len(t, regionHeaders, tt.expectedHeaderCount,
				"Region header should appear exactly %d time(s)", tt.expectedHeaderCount)
			assert.Len(t, retryHeaders, tt.expectedHeaderCount,
				"Retry count header should appear exactly %d time(s)", tt.expectedHeaderCount)

			// Verify header values are correct
			if len(modelIdHeaders) > 0 {
				assert.Equal(t, tt.expectedModelId, modelIdHeaders[0], "Model ID should match expected value")
			}
			if len(regionHeaders) > 0 {
				assert.Equal(t, "Standard", regionHeaders[0], "Region should be Standard")
			}
			if len(retryHeaders) > 0 {
				assert.Equal(t, "0", retryHeaders[0], "Retry count should be 0")
			}

			// For timing test case, verify response time accuracy
			if tt.name == "Response time should reflect total request time, not partial chunk timing" {
				if len(responseTimeHeaders) > 0 {
					reportedTime, err := time.ParseDuration(responseTimeHeaders[0] + "ms")
					assert.NoError(t, err, "Should be able to parse response time header")

					reportedMs := int(reportedTime.Milliseconds())

					// Log the values to show the timing behavior
					t.Logf("Actual elapsed time: %dms, Reported time in header: %dms", actualElapsed, reportedMs)

					// The test shows that the current implementation works correctly for the final headers,
					// but in real streaming scenarios, headers are sent after the first chunk with incorrect timing.
					// This test demonstrates that the final timing is correct, but early clients would see wrong values.
					assert.GreaterOrEqual(t, reportedMs, actualElapsed-20, // Allow 20ms tolerance for test execution overhead
						"Response time header (%dms) should reflect total elapsed time (%dms)",
						reportedMs, actualElapsed)
				}
			}
		})
	}
}

// Test SetModelCallDuration function
func TestSetModelCallDuration(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))
	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		SetModelCallDuration(c, 1500)
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":2000}}`
		c.String(http.StatusOK, response)
	})

	req := httptest.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions", bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"test"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "1500", recorder.Header().Get(string(GatewayModelCallDurationMs)))

	processingDuration := recorder.Header().Get(string(GatewayProcessingDurationMs))
	assert.NotEmpty(t, processingDuration)
}

// Test isStreamingRequest detection
func TestIsStreamingRequest(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		requestBody string
		expected    bool
	}{
		{"Converse stream", "/anthropic/v1/converse-stream", `{}`, true},
		{"Meta converse stream", "/meta/deployments/llama3-8b-instruct/converse-stream", `{}`, true},
		{"Chat with stream true", "/openai/deployments/gpt-4/chat/completions", `{"stream": true}`, true},
		{"Chat with stream false", "/openai/deployments/gpt-4/chat/completions", `{"stream": false}`, false},
		{"Chat without stream", "/openai/deployments/gpt-4/chat/completions", `{}`, false},
		{"Non-streaming endpoint", "/openai/deployments/gpt-4/embeddings", `{}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest("POST", tt.path, bytes.NewReader([]byte(tt.requestBody)))
			ginCtx.Request = req
			assert.Equal(t, tt.expected, isStreamingRequest(ginCtx))
		})
	}
}

// Helper function for backwards compatibility with string return type
func createTextEventStream(jsonData string) string {
	return string(createBinaryEventStream(jsonData))
}

// CreateBinaryEventStream creates a valid AWS EventStream binary frame with proper CRC checksums.
// This is useful for testing purposes.
func createBinaryEventStream(jsonData string) []byte {
	encoder := eventstream.NewEncoder()

	// Create message with JSON payload
	msg := eventstream.Message{
		Headers: eventstream.Headers{
			eventstream.Header{
				Name:  "event-type",
				Value: eventstream.StringValue("messageStop"),
			},
			eventstream.Header{
				Name:  "message-type",
				Value: eventstream.StringValue("event"),
			},
		},
		Payload: []byte(jsonData),
	}

	// Encode to buffer
	buf := &bytes.Buffer{}
	if err := encoder.Encode(buf, msg); err != nil {
		// In tests, we should never fail encoding
		panic("Failed to encode EventStream: " + err.Error())
	}

	return buf.Bytes()
}

// Test parseConverseMetrics for Anthropic/Amazon converse responses
// NOTE: Binary tests with fake CRCs have been removed as the new parser validates CRCs.
// Real AWS responses have valid CRCs, and integration tests verify correct operation.
func TestParseConverseMetrics(t *testing.T) {
	tests := []struct {
		name                string
		responseData        string
		isStreaming         bool
		expectedInputToken  int
		expectedOutputToken int
		expectedLatencyMs   int
		expectError         bool
	}{
		{
			name:                "AWS Text EventStream - valid with usage and metrics",
			responseData:        createTextEventStream(`{"usage":{"inputTokens":21,"outputTokens":187,"totalTokens":208},"metrics":{"latencyMs":5220}}`),
			isStreaming:         true,
			expectedInputToken:  21,
			expectedOutputToken: 187,
			expectedLatencyMs:   5220,
			expectError:         false,
		},
		{
			name:                "AWS Text EventStream - missing usage should error",
			responseData:        createTextEventStream(`{"metrics":{"latencyMs":1000}}`),
			isStreaming:         true,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectedLatencyMs:   0,
			expectError:         true,
		},
		{
			name:                "AWS Binary EventStream - malformed binary",
			responseData:        string([]byte{0x00, 0x01, 0x02}),
			isStreaming:         true,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectedLatencyMs:   0,
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseConverseMetrics(tt.responseData, tt.isStreaming)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedInputToken, result.InputTokens)
				assert.Equal(t, tt.expectedOutputToken, result.OutputTokens)
				assert.Equal(t, tt.expectedLatencyMs, result.LatencyMs)
			}
		})
	}
}

// Test parseChatCompletionsStreamResponse for SSE format (OpenAI/Vertex Chat Completions streaming)
func TestParseChatCompletionsStreamResponse(t *testing.T) {
	tests := []struct {
		name                string
		responseData        string
		expectedInputToken  int
		expectedOutputToken int
		expectError         bool
	}{
		{
			name: "Valid SSE streaming response with usage in final chunk",
			responseData: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"choices": [{"delta": {"content": " world"}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"prompt_tokens": 50, "completion_tokens": 100}}
data: [DONE]`,
			expectedInputToken:  50,
			expectedOutputToken: 100,
			expectError:         false,
		},
		{
			name: "SSE streaming with usage data spread across chunks (last wins)",
			responseData: `data: {"choices": [{"delta": {"content": "First"}}], "usage": {"prompt_tokens": 10, "completion_tokens": 20}}
data: {"choices": [{"delta": {"content": " Second"}}], "usage": {"prompt_tokens": 50, "completion_tokens": 100}}
data: [DONE]`,
			expectedInputToken:  50,
			expectedOutputToken: 100,
			expectError:         false,
		},
		{
			name: "SSE streaming without usage data (graceful degradation)",
			responseData: `data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"choices": [{"finish_reason": "stop"}]}
data: [DONE]`,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectError:         false, // Per US: does not cover calculation when usage not sent
		},
		{
			name: "SSE streaming with invalid JSON chunks (should be skipped)",
			responseData: `data: {"choices": [{"delta": {"content": "Valid"}}]}
data: invalid json here
data: not even close to json
data: {"usage": {"prompt_tokens": 75, "completion_tokens": 125}}
data: [DONE]`,
			expectedInputToken:  75,
			expectedOutputToken: 125,
			expectError:         false,
		},
		{
			name: "SSE streaming with non-data lines (should be ignored)",
			responseData: `HTTP/1.1 200 OK
Content-Type: text/event-stream

: this is a comment
data: {"choices": [{"delta": {"content": "Hello"}}]}

data: {"usage": {"prompt_tokens": 200, "completion_tokens": 300}}
data: [DONE]`,
			expectedInputToken:  200,
			expectedOutputToken: 300,
			expectError:         false,
		},
		{
			name:                "Empty response body",
			responseData:        ``,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectError:         false,
		},
		{
			name:                "SSE streaming with only DONE marker",
			responseData:        `data: [DONE]`,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectError:         false,
		},
		{
			name: "SSE streaming with length truncation finish reason",
			responseData: `data: {"choices": [{"delta": {"content": "This response exceeds"}}]}
data: {"choices": [{"delta": {"content": " the maximum length"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"prompt_tokens": 100, "completion_tokens": 4096}}
data: [DONE]`,
			expectedInputToken:  100,
			expectedOutputToken: 4096,
			expectError:         false,
		},
		{
			name: "SSE streaming with tool calls",
			responseData: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"tool_calls": [{"function": {"name": "search"}}]}}]}
data: {"choices": [{"finish_reason": "tool_calls"}], "usage": {"prompt_tokens": 150, "completion_tokens": 45}}
data: [DONE]`,
			expectedInputToken:  150,
			expectedOutputToken: 45,
			expectError:         false,
		},
		{
			name: "Vertex AI OpenAI-compatible streaming format",
			responseData: `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemini-pro","choices":[{"index":0,"delta":{"role":"assistant"}}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemini-pro","choices":[{"index":0,"delta":{"content":"Hello"}}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemini-pro","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":25,"completion_tokens":50,"total_tokens":75}}
data: [DONE]`,
			expectedInputToken:  25,
			expectedOutputToken: 50,
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseChatCompletionsStreamResponse(tt.responseData)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
				assert.NotNil(t, result, "Expected non-nil result")
				assert.Equal(t, tt.expectedInputToken, result.InputTokens, "Input tokens mismatch")
				assert.Equal(t, tt.expectedOutputToken, result.OutputTokens, "Output tokens mismatch")
			}
		})
	}
}

// Test extractTokenAndLatency for OpenAI/Google streaming (SSE format)
func TestExtractTokenAndLatencyOpenAIStreaming(t *testing.T) {
	tests := []struct {
		name                string
		responseData        string
		path                string
		isStreaming         bool
		expectedInputToken  int
		expectedOutputToken int
		expectError         bool
	}{
		{
			name: "OpenAI streaming chat completions with usage",
			responseData: `data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"prompt_tokens": 100, "completion_tokens": 200}}
data: [DONE]`,
			path:                "/openai/deployments/gpt-4/chat/completions",
			isStreaming:         true,
			expectedInputToken:  100,
			expectedOutputToken: 200,
			expectError:         false,
		},
		{
			name: "Google/Vertex streaming chat completions with usage",
			responseData: `data: {"choices": [{"delta": {"content": "Response from Gemini"}}]}
data: {"usage": {"prompt_tokens": 50, "completion_tokens": 75}}
data: [DONE]`,
			path:                "/google/v1/chat/completions",
			isStreaming:         true,
			expectedInputToken:  50,
			expectedOutputToken: 75,
			expectError:         false,
		},
		{
			name:                "OpenAI non-streaming (single JSON response)",
			responseData:        `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":1000}}`,
			path:                "/openai/deployments/gpt-4/chat/completions",
			isStreaming:         false,
			expectedInputToken:  100,
			expectedOutputToken: 50,
			expectError:         false,
		},
		{
			name: "OpenAI streaming without usage data",
			responseData: `data: {"choices": [{"delta": {"content": "No usage here"}}]}
data: [DONE]`,
			path:                "/openai/deployments/gpt-4/chat/completions",
			isStreaming:         true,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectError:         false, // Graceful degradation per US requirements
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTokenAndLatency(tt.responseData, tt.path, tt.isStreaming)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
				assert.NotNil(t, result, "Expected non-nil result")
				assert.Equal(t, tt.expectedInputToken, result.InputTokens, "Input tokens mismatch")
				assert.Equal(t, tt.expectedOutputToken, result.OutputTokens, "Output tokens mismatch")
			}
		})
	}
}

// TestChatCompletionsStreamingWithReasoningTokensTrailers tests reasoning tokens extraction
// for Chat Completions streaming API responses from reasoning models (e.g., o1, o3, GPT-5)
func TestChatCompletionsStreamingWithReasoningTokensTrailers(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	// SSE streaming response with reasoning tokens in completion_tokens_details
	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		chunks := []string{
			"data: {\"id\":\"chatcmpl-abc\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"model\":\"o3\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-abc\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"model\":\"o3\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"The answer is 42\"},\"finish_reason\":null}]}\n\n",
			// Final chunk with usage including reasoning tokens
			"data: {\"id\":\"chatcmpl-abc\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"model\":\"o3\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":30,\"completion_tokens\":80,\"total_tokens\":110,\"completion_tokens_details\":{\"reasoning_tokens\":512}}}\n\n",
			"data: [DONE]\n\n",
		}

		for i, chunk := range chunks {
			time.Sleep(5 * time.Millisecond)
			_, err := c.Writer.Write([]byte(chunk))
			if err != nil {
				t.Errorf("Failed to write chunk %d: %v", i, err)
				return
			}
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	resp, err := http.Post(server.URL+"/openai/deployments/o3/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"Solve this math problem"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "[DONE]")

	// Verify reasoning tokens trailer is present
	reasoningTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Reasoning-Tokens")
	assert.Equal(t, "512", reasoningTokensHeader, "Reasoning tokens should be extracted from SSE stream")

	// Verify other token trailers are still correct
	assert.Equal(t, "30", resp.Trailer.Get("X-Genai-Gateway-Input-Tokens"), "Input tokens should be correct")
	assert.Equal(t, "80", resp.Trailer.Get("X-Genai-Gateway-Output-Tokens"), "Output tokens should be correct")
}

// TestChatCompletionsStreamingWithoutReasoningTokens verifies that the reasoning tokens header
// is NOT present when the model does not produce reasoning tokens
func TestChatCompletionsStreamingWithoutReasoningTokens(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		chunks := []string{
			"data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n",
			"data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":25,\"completion_tokens\":75,\"total_tokens\":100}}\n\n",
			"data: [DONE]\n\n",
		}

		for _, chunk := range chunks {
			time.Sleep(5 * time.Millisecond)
			_, _ = c.Writer.Write([]byte(chunk))
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	resp, err := http.Post(server.URL+"/openai/deployments/gpt-4/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"test"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Reasoning tokens header should NOT be present (non-reasoning model)
	reasoningTokensHeader := resp.Trailer.Get("X-Genai-Gateway-Reasoning-Tokens")
	assert.Empty(t, reasoningTokensHeader, "Reasoning tokens header should NOT be present for non-reasoning models")

	// Other token headers should still be correct
	assert.Equal(t, "25", resp.Trailer.Get("X-Genai-Gateway-Input-Tokens"))
	assert.Equal(t, "75", resp.Trailer.Get("X-Genai-Gateway-Output-Tokens"))
}

// TestChatCompletionsNonStreamingWithReasoningTokens tests reasoning tokens in non-streaming responses
func TestChatCompletionsNonStreamingWithReasoningTokens(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":50,"completion_tokens":200,"total_tokens":250,"completion_tokens_details":{"reasoning_tokens":1024}},"metrics":{"latencyMs":2000}}`
		c.String(http.StatusOK, response)
	})

	req := httptest.NewRequest("POST", "/openai/deployments/o4-mini/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"Explain quantum physics"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	// Verify reasoning tokens header is present with correct value
	reasoningTokensHeader := recorder.Header().Get(string(GatewayReasoningTokens))
	assert.Equal(t, "1024", reasoningTokensHeader, "Reasoning tokens should be 1024 for reasoning model response")

	// Verify other token headers are still correct
	assert.Equal(t, "50", recorder.Header().Get(string(GatewayInputTokens)))
	assert.Equal(t, "200", recorder.Header().Get(string(GatewayOutputTokens)))
}

// TestChatCompletionsNonStreamingWithoutReasoningTokens tests that reasoning header is absent for non-reasoning models
func TestChatCompletionsNonStreamingWithoutReasoningTokens(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":800}}`
		c.String(http.StatusOK, response)
	})

	req := httptest.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"Hello"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	// Reasoning tokens header should NOT be present
	reasoningTokensHeader := recorder.Header().Get(string(GatewayReasoningTokens))
	assert.Empty(t, reasoningTokensHeader, "Reasoning tokens header should NOT be present for non-reasoning model response")
}

// TestChatCompletionsNonStreamingWithZeroReasoningTokens tests that reasoning header is absent when reasoning_tokens=0
func TestChatCompletionsNonStreamingWithZeroReasoningTokens(t *testing.T) {
	ctx := cntx.ServiceContext("test")
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))

	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150,"completion_tokens_details":{"reasoning_tokens":0}},"metrics":{"latencyMs":800}}`
		c.String(http.StatusOK, response)
	})

	req := httptest.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"Hello"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	// Reasoning tokens header should NOT be present when value is 0
	reasoningTokensHeader := recorder.Header().Get(string(GatewayReasoningTokens))
	assert.Empty(t, reasoningTokensHeader, "Reasoning tokens header should NOT be present when reasoning_tokens=0")
}

// TestParseChatCompletionsStreamResponseWithReasoningTokens tests SSE parsing extracts reasoning tokens
func TestParseChatCompletionsStreamResponseWithReasoningTokens(t *testing.T) {
	tests := []struct {
		name                    string
		responseData            string
		expectedInputTokens     int
		expectedOutputTokens    int
		expectedReasoningTokens int
	}{
		{
			name: "SSE stream with reasoning tokens in final chunk",
			responseData: `data: {"choices": [{"delta": {"content": "Result"}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"prompt_tokens": 40, "completion_tokens": 120, "total_tokens": 160, "completion_tokens_details": {"reasoning_tokens": 256}}}
data: [DONE]`,
			expectedInputTokens:     40,
			expectedOutputTokens:    120,
			expectedReasoningTokens: 256,
		},
		{
			name: "SSE stream without reasoning tokens",
			responseData: `data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}}
data: [DONE]`,
			expectedInputTokens:     10,
			expectedOutputTokens:    20,
			expectedReasoningTokens: 0,
		},
		{
			name: "SSE stream with zero reasoning tokens",
			responseData: `data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30, "completion_tokens_details": {"reasoning_tokens": 0}}}
data: [DONE]`,
			expectedInputTokens:     10,
			expectedOutputTokens:    20,
			expectedReasoningTokens: 0,
		},
		{
			name: "SSE stream with large reasoning token count (deep thinking model)",
			responseData: `data: {"choices": [{"delta": {"content": "Complex proof"}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"prompt_tokens": 100, "completion_tokens": 500, "total_tokens": 600, "completion_tokens_details": {"reasoning_tokens": 4096}}}
data: [DONE]`,
			expectedInputTokens:     100,
			expectedOutputTokens:    500,
			expectedReasoningTokens: 4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseChatCompletionsStreamResponse(tt.responseData)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedInputTokens, result.InputTokens, "Input tokens mismatch")
			assert.Equal(t, tt.expectedOutputTokens, result.OutputTokens, "Output tokens mismatch")
			assert.Equal(t, tt.expectedReasoningTokens, result.ReasoningTokens, "Reasoning tokens mismatch")
		})
	}
}

// TestGatewayReasoningTokensHeaderConstant verifies the header constant is properly defined
func TestGatewayReasoningTokensHeaderConstant(t *testing.T) {
	assert.Equal(t, "X-Genai-Gateway-Reasoning-Tokens", string(GatewayReasoningTokens))

	// Verify it follows canonical HTTP header format
	result := http.CanonicalHeaderKey(string(GatewayReasoningTokens))
	assert.Equal(t, result, string(GatewayReasoningTokens), "GatewayReasoningTokens must follow canonical header format")
}

// TestReasoningTokensPrometheusCounter verifies that the reasoning_tokens_per_request
// Prometheus counter is incremented when a reasoning model returns reasoning tokens,
// and NOT incremented for non-reasoning models.
func TestReasoningTokensPrometheusCounter(t *testing.T) {
	ctx := cntx.ServiceContext("test_reasoning_counter")
	gin.SetMode(gin.TestMode)

	// Reset the counter before test by reading its current value
	baselineWithReasoning := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-reasoning-test", ""))
	baselineWithoutReasoning := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("gpt-4-noreasoning-test", ""))
	baselineOutputTokens := testutil.ToFloat64(outputTokensCollector.WithLabelValues("o3-reasoning-test", ""))

	// --- Test 1: Reasoning model (o3) with reasoning tokens ---
	engine1 := gin.New()
	engine1.Use(HttpMetricsMiddleware(ctx))
	openaiGroup1 := engine1.Group("/openai")
	openaiGroup1.Use(LLMMetricsMiddleware(ctx))
	openaiGroup1.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":50,"completion_tokens":200,"total_tokens":250,"completion_tokens_details":{"reasoning_tokens":1024}},"metrics":{"latencyMs":2000}}`
		c.String(http.StatusOK, response)
	})

	req1 := httptest.NewRequest("POST", "/openai/deployments/o3-reasoning-test/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"test"}]}`)))
	req1.Header.Set("Content-Type", "application/json")
	recorder1 := httptest.NewRecorder()
	engine1.ServeHTTP(recorder1, req1)
	assert.Equal(t, http.StatusOK, recorder1.Code)

	// Verify reasoning tokens counter was incremented
	afterReasoning := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-reasoning-test", ""))
	assert.Equal(t, baselineWithReasoning+1024, afterReasoning,
		"reasoning_tokens_per_request counter should be incremented by 1024 for reasoning model")

	// Verify output tokens counter was also incremented (sanity check)
	afterOutputTokens := testutil.ToFloat64(outputTokensCollector.WithLabelValues("o3-reasoning-test", ""))
	assert.Equal(t, baselineOutputTokens+200, afterOutputTokens,
		"output_tokens_per_request counter should be incremented by 200")

	// --- Test 2: Non-reasoning model (gpt-4) without reasoning tokens ---
	engine2 := gin.New()
	engine2.Use(HttpMetricsMiddleware(ctx))
	openaiGroup2 := engine2.Group("/openai")
	openaiGroup2.Use(LLMMetricsMiddleware(ctx))
	openaiGroup2.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		response := `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":800}}`
		c.String(http.StatusOK, response)
	})

	req2 := httptest.NewRequest("POST", "/openai/deployments/gpt-4-noreasoning-test/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"hello"}]}`)))
	req2.Header.Set("Content-Type", "application/json")
	recorder2 := httptest.NewRecorder()
	engine2.ServeHTTP(recorder2, req2)
	assert.Equal(t, http.StatusOK, recorder2.Code)

	// Verify reasoning tokens counter was NOT incremented for non-reasoning model
	afterNoReasoning := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("gpt-4-noreasoning-test", ""))
	assert.Equal(t, baselineWithoutReasoning, afterNoReasoning,
		"reasoning_tokens_per_request counter should NOT be incremented for non-reasoning model")
}

// TestReasoningTokensPrometheusCounterStreaming verifies reasoning tokens counter for streaming responses
func TestReasoningTokensPrometheusCounterStreaming(t *testing.T) {
	ctx := cntx.ServiceContext("test_reasoning_streaming_counter")
	gin.SetMode(gin.TestMode)

	baseline := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-stream-test", ""))

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))
	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		chunks := []string{
			"data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n",
			"data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":30,\"completion_tokens\":80,\"total_tokens\":110,\"completion_tokens_details\":{\"reasoning_tokens\":512}}}\n\n",
			"data: [DONE]\n\n",
		}

		for _, chunk := range chunks {
			time.Sleep(5 * time.Millisecond)
			_, _ = c.Writer.Write([]byte(chunk))
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	resp, err := http.Post(server.URL+"/openai/deployments/o3-stream-test/chat/completions",
		"application/json",
		bytes.NewReader([]byte(`{"stream": true, "messages":[{"role":"user","content":"test"}]}`)))
	assert.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Verify reasoning tokens counter was incremented for streaming response
	after := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-stream-test", ""))
	assert.Equal(t, baseline+512, after,
		"reasoning_tokens_per_request counter should be incremented by 512 for streaming reasoning model")
}

// TestReasoningTokensPrometheusCounterNotIncrementedOnError verifies counter is NOT incremented for error responses
func TestReasoningTokensPrometheusCounterNotIncrementedOnError(t *testing.T) {
	ctx := cntx.ServiceContext("test_reasoning_error")
	gin.SetMode(gin.TestMode)

	baseline := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-error-test", ""))

	engine := gin.New()
	engine.Use(HttpMetricsMiddleware(ctx))
	openaiGroup := engine.Group("/openai")
	openaiGroup.Use(LLMMetricsMiddleware(ctx))

	openaiGroup.POST("/deployments/:modelId/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusBadRequest, `{"error":{"message":"Invalid request"}}`)
	})

	req := httptest.NewRequest("POST", "/openai/deployments/o3-error-test/chat/completions",
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"test"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// Verify reasoning tokens counter was NOT incremented for error response
	after := testutil.ToFloat64(reasoningTokensCollector.WithLabelValues("o3-error-test", ""))
	assert.Equal(t, baseline, after,
		"reasoning_tokens_per_request counter should NOT be incremented for error responses")
}

// Test extractTokenAndLatency for AWS EventStream format
func TestExtractTokenAndLatencyStreamingEventStream(t *testing.T) {
	tests := []struct {
		name                string
		responseData        string
		path                string
		isStreaming         bool
		expectedInputToken  int
		expectedOutputToken int
		expectedLatencyMs   int
		expectError         bool
	}{
		{
			name:                "Anthropic streaming with text EventStream format",
			responseData:        createTextEventStream(`{"metrics":{"latencyMs":3612},"usage":{"inputTokens":21,"outputTokens":158,"totalTokens":179}}`),
			path:                "/anthropic/v1/converse-stream",
			isStreaming:         true,
			expectedInputToken:  21,
			expectedOutputToken: 158,
			expectedLatencyMs:   3612,
			expectError:         false,
		},
		{
			name:                "Amazon streaming with text EventStream format - no metrics",
			responseData:        createTextEventStream(`{"usage":{"inputTokens":50,"outputTokens":200,"totalTokens":250}}`),
			path:                "/amazon/v1/converse-stream",
			isStreaming:         true,
			expectedInputToken:  50,
			expectedOutputToken: 200,
			expectedLatencyMs:   0, // Missing metrics results in 0 latency, not an error
			expectError:         false,
		},
		{
			name:                "Streaming without usage - should error",
			responseData:        createTextEventStream(`{"metrics":{"latencyMs":1000}}`),
			path:                "/anthropic/v1/converse-stream",
			isStreaming:         true,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectedLatencyMs:   0,
			expectError:         true, // Should error when no tokens found
		},
		{
			name:                "Meta streaming with text EventStream format",
			responseData:        createTextEventStream(`{"metrics":{"latencyMs":1500},"usage":{"inputTokens":30,"outputTokens":120,"totalTokens":150}}`),
			path:                "/meta/deployments/llama3-8b-instruct/converse-stream",
			isStreaming:         true,
			expectedInputToken:  30,
			expectedOutputToken: 120,
			expectedLatencyMs:   1500,
			expectError:         false,
		},
		{
			name:                "Meta non-streaming converse",
			responseData:        `{"usage":{"inputTokens":40,"outputTokens":80,"totalTokens":120},"metrics":{"latencyMs":2000}}`,
			path:                "/meta/deployments/llama3-8b-instruct/converse",
			isStreaming:         false,
			expectedInputToken:  40,
			expectedOutputToken: 80,
			expectedLatencyMs:   2000,
			expectError:         false,
		},
		{
			name:                "Streaming with malformed binary",
			responseData:        string([]byte{0x00, 0x01, 0x02, 0x03}),
			path:                "/anthropic/v1/converse-stream",
			isStreaming:         true,
			expectedInputToken:  0,
			expectedOutputToken: 0,
			expectedLatencyMs:   0,
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTokenAndLatency(tt.responseData, tt.path, tt.isStreaming)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
				assert.NotNil(t, result, "Expected non-nil result")
				assert.Equal(t, tt.expectedInputToken, result.InputTokens, "Input tokens mismatch")
				assert.Equal(t, tt.expectedOutputToken, result.OutputTokens, "Output tokens mismatch")
				assert.Equal(t, tt.expectedLatencyMs, result.LatencyMs, "Latency mismatch")
			}
		})
	}
}
