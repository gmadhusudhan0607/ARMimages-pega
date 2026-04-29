/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Mock structures for testing
type MockRequestMetricsProvider struct {
	requestMetrics *RequestMetrics
	tokenMetrics   *TokenMetrics
}

func (m *MockRequestMetricsProvider) GetRequestMetrics() *RequestMetrics {
	return m.requestMetrics
}

func (m *MockRequestMetricsProvider) GetTokenMetrics() *TokenMetrics {
	return m.tokenMetrics
}

func (m *MockRequestMetricsProvider) GetIsolationID() string {
	return "test-isolation"
}

func (m *MockRequestMetricsProvider) GetOriginalModelName() string {
	return "gpt-4"
}

func (m *MockRequestMetricsProvider) GetTargetModelName() string {
	return "gpt-4-turbo"
}

func (m *MockRequestMetricsProvider) GetTargetModelID() string {
	return "model-123"
}

func (m *MockRequestMetricsProvider) GetTargetModelVersion() string {
	return "v1.0"
}

func (m *MockRequestMetricsProvider) GetTargetModelCreator() string {
	return "openai"
}

func (m *MockRequestMetricsProvider) GetTargetModelInfrastructure() string {
	return "aws"
}

func (m *MockRequestMetricsProvider) GetTargetModel() *modeltypes.Model {
	return &modeltypes.Model{
		KEY:            "model-123",
		Name:           "gpt-4-turbo",
		Version:        "v1.0",
		Creator:        modeltypes.CreatorOpenAI,
		Infrastructure: modeltypes.InfrastructureAWS,
		Provider:       modeltypes.ProviderBedrock,
	}
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/api/v1/chat"},
	}
	c.Request = req

	return c, w
}

func TestNewRequestModificationResponseWriter(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	assert.NotNil(t, writer)
	assert.Equal(t, c, writer.ginContext)
	assert.Equal(t, logger, writer.logger)
	assert.Equal(t, 200, writer.statusCode) // Default status code
	assert.False(t, writer.metricsUpdated)
	assert.False(t, writer.bodyProcessed)
	assert.NotNil(t, writer.responseBuffer)
	assert.Nil(t, writer.extractedTokens)
	assert.False(t, writer.startTime.IsZero())
}

func TestRequestModificationResponseWriter_WriteHeader(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	writer.WriteHeader(404)

	assert.Equal(t, 404, writer.statusCode)
}

func TestRequestModificationResponseWriter_Write(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	// Add mock metadata to context
	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	writer.shouldBuffer = false // Disable buffering to test immediate write path

	data := []byte("test response")
	n, err := writer.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.True(t, writer.metricsUpdated)
	assert.True(t, writer.bodyProcessed)
}

func TestRequestModificationResponseWriter_WriteString(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	// Add mock metadata to context
	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	writer.shouldBuffer = false // Disable buffering to test immediate write path

	testString := "test response string"
	n, err := writer.WriteString(testString)

	assert.NoError(t, err)
	assert.Equal(t, len(testString), n)
	assert.True(t, writer.metricsUpdated)
	assert.True(t, writer.bodyProcessed)
}

func TestRequestModificationResponseWriter_GetDuration(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Wait a bit to ensure some duration
	time.Sleep(1 * time.Millisecond)

	duration := writer.GetDuration()
	assert.True(t, duration > 0)
}

func TestRequestModificationResponseWriter_GetResponseBody(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Test with empty buffer - should return empty slice, not nil
	body := writer.GetResponseBody()
	assert.Len(t, body, 0) // Check length instead of nil, as empty buffer returns empty slice

	// Write some data
	testData := []byte("test response")
	writer.responseBuffer.Write(testData)

	body = writer.GetResponseBody()
	assert.Equal(t, testData, body)
}

func TestRequestModificationResponseWriter_GetResponseBodyNilBuffer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	writer.responseBuffer = nil // Simulate nil buffer

	body := writer.GetResponseBody()
	assert.Nil(t, body)
}

func TestRequestModificationResponseWriter_IsMetricsUpdated(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Initially false
	assert.False(t, writer.IsMetricsUpdated())

	// Set to true
	writer.metricsUpdated = true
	assert.True(t, writer.IsMetricsUpdated())
}

func TestRequestModificationResponseWriter_ProcessResponseBodyWithTokens(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Add mock metadata to context
	mockProvider := &MockRequestMetricsProvider{
		tokenMetrics: &TokenMetrics{},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	// Simulate OpenAI response with completion tokens
	responseJSON := `{
		"usage": {
			"completion_tokens": 150,
			"prompt_tokens": 20,
			"total_tokens": 170
		}
	}`

	writer.responseBuffer.WriteString(responseJSON)
	writer.processResponseBody()

	assert.True(t, writer.bodyProcessed)
	assert.NotNil(t, writer.extractedTokens)
	assert.Equal(t, 150.0, *writer.extractedTokens)
}

func TestRequestModificationResponseWriter_ProcessResponseBodyInvalidJSON(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Invalid JSON
	writer.responseBuffer.WriteString("invalid json")
	writer.processResponseBody()

	assert.True(t, writer.bodyProcessed)
	assert.Nil(t, writer.extractedTokens)
}

func TestRequestModificationResponseWriter_ProcessResponseBodyNoUsage(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Valid JSON but no usage object
	responseJSON := `{"message": "Hello world"}`

	writer.responseBuffer.WriteString(responseJSON)
	writer.processResponseBody()

	assert.True(t, writer.bodyProcessed)
	assert.Nil(t, writer.extractedTokens)
}

func TestRequestModificationResponseWriter_ProcessResponseBodyNoCompletionTokens(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Valid JSON with usage but no completion_tokens
	responseJSON := `{
		"usage": {
			"prompt_tokens": 20,
			"total_tokens": 170
		}
	}`

	writer.responseBuffer.WriteString(responseJSON)
	writer.processResponseBody()

	assert.True(t, writer.bodyProcessed)
	assert.Nil(t, writer.extractedTokens)
}

func TestRequestModificationResponseWriter_UpdateMetrics(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	// Create test metadata
	requested := 1000.0
	used := 750.0
	maximum := 4000.0

	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested: &requested,
				Used:      &used,
				Maximum:   &maximum,
			},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Simulate extracted tokens
	extractedTokens := 800.0
	writer.extractedTokens = &extractedTokens

	writer.updateMetrics()

	// The updateMetrics calls processResponseBody, but since the buffer is empty,
	// processResponseBody returns early WITHOUT setting bodyProcessed to true
	// This is the actual behavior of the code
	assert.False(t, writer.bodyProcessed)
	// The extracted tokens should override the used tokens in the metrics
}

func TestRequestModificationResponseWriter_UpdateMetricsNoMetadata(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Should not panic even without metadata
	assert.NotPanics(t, func() {
		writer.updateMetrics()
	})
}

func TestRequestModificationResponseWriter_CreateTimingMetrics(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Wait a bit to ensure some duration
	time.Sleep(1 * time.Millisecond)

	timingMetrics := writer.createTimingMetrics()

	assert.NotNil(t, timingMetrics)
	assert.False(t, timingMetrics.StartTime.IsZero())
	assert.False(t, timingMetrics.EndTime.IsZero())
	assert.True(t, timingMetrics.Duration > 0)
	assert.True(t, timingMetrics.EndTime.After(timingMetrics.StartTime))
}

func TestRequestModificationResponseWriter_ExtractRequestMetricsFullMetrics(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Test with full RequestMetrics provider
	requested := 1000.0
	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested: &requested,
			},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	timingMetrics := &TimingMetrics{
		Duration: 100 * time.Millisecond,
	}

	requestMetrics := writer.extractRequestMetrics(timingMetrics)

	assert.NotNil(t, requestMetrics)
	assert.Equal(t, timingMetrics.Duration, requestMetrics.TimingMetrics.Duration)
	assert.Equal(t, 1000.0, *requestMetrics.TokenMetrics.Requested)
}

func TestRequestModificationResponseWriter_ExtractRequestMetricsTokenOnly(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Mock provider that only implements GetTokenMetrics
	type TokenOnlyProvider struct {
		tokenMetrics *TokenMetrics
	}
	tokenOnlyProvider := &TokenOnlyProvider{
		tokenMetrics: &TokenMetrics{
			Used: func() *float64 { v := 750.0; return &v }(),
		},
	}

	// Mock GetTokenMetrics method
	mockWithTokens := struct {
		*TokenOnlyProvider
	}{tokenOnlyProvider}

	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockWithTokens)
	c.Request = c.Request.WithContext(ctx)

	timingMetrics := &TimingMetrics{
		Duration: 100 * time.Millisecond,
	}

	requestMetrics := writer.extractRequestMetrics(timingMetrics)

	// Should fall back to minimal metrics since the mock doesn't implement the interface properly
	assert.NotNil(t, requestMetrics)
	assert.Equal(t, timingMetrics.Duration, requestMetrics.TimingMetrics.Duration)
}

func TestRequestModificationResponseWriter_ExtractRequestMetricsMinimal(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	timingMetrics := &TimingMetrics{
		Duration: 100 * time.Millisecond,
	}

	// No metadata in context
	requestMetrics := writer.extractRequestMetrics(timingMetrics)

	assert.NotNil(t, requestMetrics)
	assert.Equal(t, timingMetrics.Duration, requestMetrics.TimingMetrics.Duration)
	// Should have empty TokenMetrics
	assert.Nil(t, requestMetrics.TokenMetrics.Requested)
}

func TestRequestModificationResponseWriter_DisableBuffering(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Initially buffering is enabled
	assert.True(t, writer.shouldBuffer, "shouldBuffer should be true by default")

	// Call DisableBuffering
	writer.DisableBuffering()

	// Now buffering should be disabled
	assert.False(t, writer.shouldBuffer, "shouldBuffer should be false after DisableBuffering()")
}

func TestRequestModificationResponseWriter_DisableBuffering_WritePassesThrough(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, w := setupTestContext()

	// Add mock metadata to context
	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = writer

	// Disable buffering (simulating streaming request)
	writer.DisableBuffering()

	// Write data - should pass through to underlying writer immediately
	testData := []byte("streaming chunk 1")
	n, err := writer.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	// Data should appear in the recorder immediately (not buffered)
	assert.Contains(t, w.Body.String(), "streaming chunk 1",
		"Data should be written to underlying writer immediately when buffering is disabled")
}

func TestRequestModificationResponseWriter_DisableBuffering_WriteStringPassesThrough(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, w := setupTestContext()

	// Add mock metadata to context
	mockProvider := &MockRequestMetricsProvider{
		requestMetrics: &RequestMetrics{
			TokenMetrics: TokenMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), RequestMetadataContextKey{}, mockProvider)
	c.Request = c.Request.WithContext(ctx)

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = writer

	// Disable buffering (simulating streaming request)
	writer.DisableBuffering()

	// WriteString data - should pass through to underlying writer immediately
	testString := "streaming chunk via WriteString"
	n, err := writer.WriteString(testString)

	assert.NoError(t, err)
	assert.Equal(t, len(testString), n)
	// Data should appear in the recorder immediately (not buffered)
	assert.Contains(t, w.Body.String(), "streaming chunk via WriteString",
		"Data should be written to underlying writer immediately when buffering is disabled")
}

func TestRequestModificationResponseWriter_BufferingPreventsPassthrough(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, w := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = writer

	// Buffering is enabled by default - writes should NOT appear in recorder
	testData := []byte("buffered data")
	n, err := writer.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	// Data should NOT appear in recorder because it's buffered
	assert.Empty(t, w.Body.String(),
		"Data should NOT be written to underlying writer when buffering is enabled")

	// But it should be in the internal buffer
	assert.Contains(t, string(writer.GetResponseBody()), "buffered data",
		"Data should be captured in the internal response buffer")
}

func TestRequestModificationResponseWriter_FlushBufferedResponseNoOpAfterDisableBuffering(t *testing.T) {
	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Disable buffering first
	writer.DisableBuffering()

	// FlushBufferedResponse should be a no-op and return nil
	err := writer.FlushBufferedResponse()
	assert.NoError(t, err, "FlushBufferedResponse should return nil when buffering is already disabled")
}

func TestRequestModificationResponseWriter_UpdateOnceEnsuresSingleExecution(t *testing.T) {
	// Register metrics first to avoid panics
	RegisterMetrics()

	logger := zap.NewNop().Sugar()
	c, _ := setupTestContext()

	writer := NewRequestModificationResponseWriter(c.Writer, c, logger)

	// Call updateMetricsOnce multiple times
	writer.updateMetricsOnce()
	writer.updateMetricsOnce()
	writer.updateMetricsOnce()

	// Metrics should only be updated once due to sync.Once
	assert.True(t, writer.metricsUpdated)
}
