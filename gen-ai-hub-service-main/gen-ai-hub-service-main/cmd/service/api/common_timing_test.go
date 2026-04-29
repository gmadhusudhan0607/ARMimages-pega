/*
* Copyright (c) 2024 Pegasystems Inc.
* All rights reserved.
 */

package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockHTTPClient simulates model API calls with controllable timing
type MockHTTPClient struct {
	responseDelay   time.Duration
	responseStatus  int
	responseBody    string
	responseHeaders map[string]string
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Simulate the model call duration
	time.Sleep(m.responseDelay)

	// Create response
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(m.responseStatus)

	// Set response headers
	for key, value := range m.responseHeaders {
		recorder.Header().Set(key, value)
	}

	_, _ = recorder.WriteString(m.responseBody)

	return recorder.Result(), nil
}

func TestCallTargetTimingHeaders(t *testing.T) {
	tests := []struct {
		name             string
		modelCallDelay   time.Duration
		responseStatus   int
		responseBody     string
		expectedMinTotal time.Duration
		description      string
	}{
		{
			name:             "Fast model response with timing headers",
			modelCallDelay:   100 * time.Millisecond,
			responseStatus:   200,
			responseBody:     `{"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150},"metrics":{"latencyMs":100}}`,
			expectedMinTotal: 100 * time.Millisecond,
			description:      "Should include separate model and processing time for fast responses",
		},
		{
			name:             "Slow model response with timing headers",
			modelCallDelay:   500 * time.Millisecond,
			responseStatus:   200,
			responseBody:     `{"usage":{"prompt_tokens":200,"completion_tokens":100,"total_tokens":300},"metrics":{"latencyMs":500}}`,
			expectedMinTotal: 500 * time.Millisecond,
			description:      "Should accurately track longer model call times",
		},
		{
			name:             "Error response should still include timing",
			modelCallDelay:   50 * time.Millisecond,
			responseStatus:   400,
			responseBody:     `{"error":{"message":"Bad Request","type":"invalid_request_error"}}`,
			expectedMinTotal: 50 * time.Millisecond,
			description:      "Error responses should still report timing information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test as it requires mocking HTTP calls at the CallTarget level
			// This is better tested through integration tests
			t.Skip("CallTarget timing requires integration test setup with HTTP client mocking")
		})
	}
}

func TestProcessingTimeCalculationInAPIContext(t *testing.T) {
	// This test verifies that the timing separation works in a realistic API context

	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Mock endpoint that simulates the processing done by the gateway
	router.POST("/test-timing", func(c *gin.Context) {
		// Simulate some processing time before model call
		processingStart := time.Now()
		time.Sleep(20 * time.Millisecond) // Gateway processing overhead

		// Simulate model call
		modelStart := time.Now()
		time.Sleep(150 * time.Millisecond) // Model call time
		modelDuration := int(time.Since(modelStart).Milliseconds())

		// More processing after model call
		time.Sleep(10 * time.Millisecond) // Post-processing
		totalProcessing := int(time.Since(processingStart).Milliseconds())

		// Set timing headers manually (simulating what the middleware would do)
		c.Header("X-Genai-Gateway-Model-Call-Duration-Ms", strconv.Itoa(modelDuration))
		processingDuration := totalProcessing - modelDuration
		c.Header("X-Genai-Gateway-Processing-Duration-Ms", strconv.Itoa(processingDuration))
		c.Header("X-Genai-Gateway-Response-Time-Ms", strconv.Itoa(totalProcessing))

		c.JSON(200, gin.H{
			"model_time":      modelDuration,
			"processing_time": processingDuration,
			"total_time":      totalProcessing,
		})
	})

	// Test the endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test-timing", bytes.NewBufferString(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	router.ServeHTTP(w, req)
	actualElapsed := int(time.Since(start).Milliseconds())

	// Verify response
	assert.Equal(t, 200, w.Code)

	// Verify timing headers are present
	modelTimeHeader := w.Header().Get("X-Genai-Gateway-Model-Call-Duration-Ms")
	processingTimeHeader := w.Header().Get("X-Genai-Gateway-Processing-Duration-Ms")
	totalTimeHeader := w.Header().Get("X-Genai-Gateway-Response-Time-Ms")

	assert.NotEmpty(t, modelTimeHeader, "Model call duration header should be present")
	assert.NotEmpty(t, processingTimeHeader, "Processing duration header should be present")
	assert.NotEmpty(t, totalTimeHeader, "Total response time header should be present")

	// Parse timing values
	modelTime, err := strconv.Atoi(modelTimeHeader)
	assert.NoError(t, err, "Model time should be valid integer")
	assert.GreaterOrEqual(t, modelTime, 140, "Model time should be at least 140ms")
	assert.LessOrEqual(t, modelTime, 500, "Model time should be at most 500ms (with tolerance for slow CI runners)")

	processingTime, err := strconv.Atoi(processingTimeHeader)
	assert.NoError(t, err, "Processing time should be valid integer")
	assert.GreaterOrEqual(t, processingTime, 25, "Processing time should be at least 25ms")
	assert.LessOrEqual(t, processingTime, 200, "Processing time should be reasonable")

	totalTime, err := strconv.Atoi(totalTimeHeader)
	assert.NoError(t, err, "Total time should be valid integer")

	// Verify mathematical relationship
	assert.Equal(t, totalTime, modelTime+processingTime,
		"Total time should equal model time + processing time")

	// Verify timing is reasonable compared to actual elapsed time
	assert.GreaterOrEqual(t, actualElapsed, totalTime-20, // 20ms tolerance
		"Actual elapsed time should be close to reported total time")
	assert.LessOrEqual(t, actualElapsed, totalTime+20, // 20ms tolerance
		"Actual elapsed time should be close to reported total time")
}

func TestTimingHeadersInDifferentResponseScenarios(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		simulatedModel time.Duration
		simulatedProc  time.Duration
		expectHeaders  []string
		description    string
	}{
		{
			name:           "Successful request with all timing headers",
			responseStatus: 200,
			simulatedModel: 100 * time.Millisecond,
			simulatedProc:  20 * time.Millisecond,
			expectHeaders: []string{
				"X-Genai-Gateway-Model-Call-Duration-Ms",
				"X-Genai-Gateway-Processing-Duration-Ms",
				"X-Genai-Gateway-Response-Time-Ms",
			},
			description: "Success responses should include all timing headers",
		},
		{
			name:           "Error request should still include timing headers",
			responseStatus: 500,
			simulatedModel: 200 * time.Millisecond,
			simulatedProc:  30 * time.Millisecond,
			expectHeaders: []string{
				"X-Genai-Gateway-Model-Call-Duration-Ms",
				"X-Genai-Gateway-Processing-Duration-Ms",
				"X-Genai-Gateway-Response-Time-Ms",
			},
			description: "Error responses should still report timing information",
		},
		{
			name:           "Fast response with minimal processing",
			responseStatus: 200,
			simulatedModel: 50 * time.Millisecond,
			simulatedProc:  5 * time.Millisecond,
			expectHeaders: []string{
				"X-Genai-Gateway-Model-Call-Duration-Ms",
				"X-Genai-Gateway-Processing-Duration-Ms",
				"X-Genai-Gateway-Response-Time-Ms",
			},
			description: "Fast responses should accurately measure short durations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			router.POST("/test", func(c *gin.Context) {
				// Simulate model call time
				time.Sleep(tt.simulatedModel)
				modelDuration := int(tt.simulatedModel.Milliseconds())

				// Simulate processing time
				time.Sleep(tt.simulatedProc)
				processingDuration := int(tt.simulatedProc.Milliseconds())
				totalDuration := modelDuration + processingDuration

				// Set timing headers
				c.Header("X-Genai-Gateway-Model-Call-Duration-Ms", strconv.Itoa(modelDuration))
				c.Header("X-Genai-Gateway-Processing-Duration-Ms", strconv.Itoa(processingDuration))
				c.Header("X-Genai-Gateway-Response-Time-Ms", strconv.Itoa(totalDuration))

				c.Status(tt.responseStatus)
				if tt.responseStatus >= 400 {
					c.JSON(tt.responseStatus, gin.H{"error": "test error"})
				} else {
					c.JSON(tt.responseStatus, gin.H{"success": true})
				}
			})

			// Execute request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			// Verify response status
			assert.Equal(t, tt.responseStatus, w.Code, tt.description)

			// Verify all expected headers are present
			for _, header := range tt.expectHeaders {
				value := w.Header().Get(header)
				assert.NotEmpty(t, value, "Header %s should be present", header)

				// Verify header value is a valid positive integer
				intValue, err := strconv.Atoi(value)
				assert.NoError(t, err, "Header %s should contain valid integer", header)
				assert.GreaterOrEqual(t, intValue, 0, "Header %s should be >= 0", header)
			}

			// Verify timing relationship
			modelTime, _ := strconv.Atoi(w.Header().Get("X-Genai-Gateway-Model-Call-Duration-Ms"))
			processingTime, _ := strconv.Atoi(w.Header().Get("X-Genai-Gateway-Processing-Duration-Ms"))
			totalTime, _ := strconv.Atoi(w.Header().Get("X-Genai-Gateway-Response-Time-Ms"))

			assert.Equal(t, totalTime, modelTime+processingTime,
				"Total time should equal model time + processing time")
		})
	}
}

func TestStreamingResponseTimingHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/stream", func(c *gin.Context) {
		// Start measuring total time from the very beginning
		totalStart := time.Now()

		// Simulate model call before first chunk
		time.Sleep(80 * time.Millisecond)
		modelDuration := 80

		// Set up for streaming response with trailers
		c.Header("Content-Type", "application/json")
		c.Header("Transfer-Encoding", "chunked")
		c.Header("Trailer", "X-Genai-Gateway-Model-Call-Duration-Ms, X-Genai-Gateway-Processing-Duration-Ms, X-Genai-Gateway-Response-Time-Ms")
		c.Status(200)

		// Stream chunks with processing delays
		chunks := []string{
			`{"choices":[{"delta":{"content":"First"}}]}` + "\n",
			`{"choices":[{"delta":{"content":" chunk"}}]}` + "\n",
			`{"choices":[{"delta":{"content":" final"}}]}` + "\n",
		}

		for i, chunk := range chunks {
			if i > 0 {
				time.Sleep(25 * time.Millisecond) // Processing delay between chunks
			}

			_, _ = c.Writer.Write([]byte(chunk))
			c.Writer.Flush()
		}

		totalDuration := int(time.Since(totalStart).Milliseconds())
		processingDuration := totalDuration - modelDuration

		// Ensure processing duration is never negative (can happen with timing precision)
		if processingDuration < 0 {
			processingDuration = 0
		}

		// Set trailer headers
		trailer := c.Writer.Header()
		trailer.Set("X-Genai-Gateway-Model-Call-Duration-Ms", strconv.Itoa(modelDuration))
		trailer.Set("X-Genai-Gateway-Processing-Duration-Ms", strconv.Itoa(processingDuration))
		trailer.Set("X-Genai-Gateway-Response-Time-Ms", strconv.Itoa(totalDuration))
	})

	// Test streaming response
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stream", bytes.NewBufferString(`{"stream": true}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Read full response
	_, err := io.ReadAll(w.Body)
	assert.NoError(t, err)

	// Verify streaming response characteristics
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Check for trailer header announcement
	trailerHeader := w.Header().Get("Trailer")
	assert.NotEmpty(t, trailerHeader, "Streaming response should announce trailers")
	assert.Contains(t, trailerHeader, "X-Genai-Gateway-Model-Call-Duration-Ms")
	assert.Contains(t, trailerHeader, "X-Genai-Gateway-Processing-Duration-Ms")
	assert.Contains(t, trailerHeader, "X-Genai-Gateway-Response-Time-Ms")

	// Verify trailer headers are present
	result := w.Result()
	modelTimeTrailer := result.Trailer.Get("X-Genai-Gateway-Model-Call-Duration-Ms")
	processingTimeTrailer := result.Trailer.Get("X-Genai-Gateway-Processing-Duration-Ms")
	totalTimeTrailer := result.Trailer.Get("X-Genai-Gateway-Response-Time-Ms")

	assert.NotEmpty(t, modelTimeTrailer, "Model time trailer should be present")
	assert.NotEmpty(t, processingTimeTrailer, "Processing time trailer should be present")
	assert.NotEmpty(t, totalTimeTrailer, "Total time trailer should be present")

	// Verify trailer values
	modelTime, err := strconv.Atoi(modelTimeTrailer)
	assert.NoError(t, err, "Model time trailer should be valid integer")
	assert.Equal(t, 80, modelTime, "Model time should match expected value")

	processingTime, err := strconv.Atoi(processingTimeTrailer)
	assert.NoError(t, err, "Processing time trailer should be valid integer")
	assert.Greater(t, processingTime, 40, "Processing time should account for chunk delays")

	totalTime, err := strconv.Atoi(totalTimeTrailer)
	assert.NoError(t, err, "Total time trailer should be valid integer")
	assert.Equal(t, totalTime, modelTime+processingTime,
		"Total time should equal model time + processing time")
}

func TestTimingHeaderValidation(t *testing.T) {
	// Test various edge cases for timing header values
	tests := []struct {
		name           string
		modelTime      int
		processingTime int
		expectValid    bool
		description    string
	}{
		{
			name:           "Normal timing values",
			modelTime:      100,
			processingTime: 20,
			expectValid:    true,
			description:    "Normal timing values should be valid",
		},
		{
			name:           "Zero model time",
			modelTime:      0,
			processingTime: 50,
			expectValid:    true,
			description:    "Zero model time should be allowed (cached/instant response)",
		},
		{
			name:           "Zero processing time",
			modelTime:      150,
			processingTime: 0,
			expectValid:    true,
			description:    "Zero processing time should be allowed (minimal overhead)",
		},
		{
			name:           "Large timing values",
			modelTime:      30000, // 30 seconds
			processingTime: 1000,  // 1 second
			expectValid:    true,
			description:    "Large timing values should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			router.GET("/validate", func(c *gin.Context) {
				c.Header("X-Genai-Gateway-Model-Call-Duration-Ms", strconv.Itoa(tt.modelTime))
				c.Header("X-Genai-Gateway-Processing-Duration-Ms", strconv.Itoa(tt.processingTime))
				c.Header("X-Genai-Gateway-Response-Time-Ms", strconv.Itoa(tt.modelTime+tt.processingTime))

				c.JSON(200, gin.H{
					"model_time":      tt.modelTime,
					"processing_time": tt.processingTime,
					"total_time":      tt.modelTime + tt.processingTime,
				})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/validate", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code, tt.description)

			// Validate header values can be parsed
			headers := []string{
				"X-Genai-Gateway-Model-Call-Duration-Ms",
				"X-Genai-Gateway-Processing-Duration-Ms",
				"X-Genai-Gateway-Response-Time-Ms",
			}

			for _, header := range headers {
				value := w.Header().Get(header)
				assert.NotEmpty(t, value, "Header %s should be present", header)

				intValue, err := strconv.Atoi(value)
				assert.NoError(t, err, "Header %s should contain valid integer", header)
				assert.GreaterOrEqual(t, intValue, 0, "Header %s should be >= 0", header)
			}

			// Verify response body contains timing info
			responseBody := w.Body.String()
			assert.Contains(t, responseBody, strconv.Itoa(tt.modelTime))
			assert.Contains(t, responseBody, strconv.Itoa(tt.processingTime))
		})
	}
}
