/*
* Copyright (c) 2024 Pegasystems Inc.
* All rights reserved.
 */

package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

// BufferingWriter is implemented by response writers that support deferred body buffering.
// It is used to detect whether an outer writer (e.g. RequestModificationResponseWriter) is
// still holding the response body, so that MetricsResponseWriter can defer its write-through
// until the outer writer has made its flush-or-retry decision.
type BufferingWriter interface {
	ShouldBuffer() bool
}

// MetricsResult holds extracted metrics from LLM responses
type MetricsResult struct {
	InputTokens     int
	OutputTokens    int
	ReasoningTokens int
	LatencyMs       int
	Message         string
}

type chatCompletionsUsage struct {
	CompletionTokens        int `json:"completion_tokens"`
	PromptTokens            int `json:"prompt_tokens"`
	TotalTokens             int `json:"total_tokens"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

type chatCompletionsTemplate struct {
	Usage   chatCompletionsUsage `json:"usage"`
	Metrics struct {
		LatencyMs int `json:"latencyMs"`
	} `json:"metrics"`
}

func (c *chatCompletionsTemplate) getTokensPerSecond() float64 {
	if c.Usage.TotalTokens == 0 || c.Metrics.LatencyMs == 0 {
		return 0
	}
	return calculateOutputTokensPerSecond(float64(c.Usage.CompletionTokens), float64(c.Metrics.LatencyMs))
}

// parseChatCompletionsMetrics extracts metrics from OpenAI/Google chat completions response
func parseChatCompletionsMetrics(responseData string, isStreaming bool) (*MetricsResult, error) {
	if isStreaming {
		return parseChatCompletionsStreamResponse(responseData)
	}

	var c chatCompletionsTemplate
	if err := json.Unmarshal([]byte(responseData), &c); err != nil {
		return nil, fmt.Errorf("error unmarshaling chat completions: %w", err)
	}
	return &MetricsResult{
		InputTokens:     c.Usage.PromptTokens,
		OutputTokens:    c.Usage.CompletionTokens,
		ReasoningTokens: c.Usage.CompletionTokensDetails.ReasoningTokens,
		LatencyMs:       c.Metrics.LatencyMs,
	}, nil
}

func calculateOutputTokensPerSecond(outputTokens, latencyMs float64) float64 {
	return outputTokens / (latencyMs / 1000)
}

func (c *converseTemplate) getTokensPerSecond() float64 {
	if c.Usage.InputTokens == 0 || c.Metrics.LatencyMs == 0 {
		return 0
	}
	return calculateOutputTokensPerSecond(float64(c.Usage.OutputTokens), float64(c.Metrics.LatencyMs))
}

type converseTemplate struct {
	Usage struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
		TotalTokens  int `json:"totalTokens"`
	} `json:"usage"`
	Metrics struct {
		LatencyMs int `json:"latencyMs"`
	} `json:"metrics"`
}

// parseConverseMetrics extracts metrics from Anthropic/Amazon converse response
func parseConverseMetrics(responseData string, isStreaming bool) (*MetricsResult, error) {
	var c converseTemplate

	if isStreaming {
		eventJSON, err := parseConverseStreamUsageMetadata([]byte(responseData))
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(eventJSON, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal streaming event: %w", err)
		}

		// Validate that we got token data
		if c.Usage.InputTokens == 0 && c.Usage.OutputTokens == 0 {
			return nil, fmt.Errorf("failed to parse streaming response data: no token information found")
		}
	} else {
		if err := json.Unmarshal([]byte(responseData), &c); err != nil {
			return nil, fmt.Errorf("error unmarshaling converse: %w", err)
		}
	}

	return &MetricsResult{
		InputTokens:  c.Usage.InputTokens,
		OutputTokens: c.Usage.OutputTokens,
		LatencyMs:    c.Metrics.LatencyMs,
	}, nil
}

// Label represents the possible labels for Prometheus metrics
type Label string

const (
	Path        Label = "path"
	Method      Label = "method"
	Code        Label = "code"
	ModelID     Label = "modelID"
	APIVersion  Label = "apiVersion"
	IsolationID Label = "isolationID"
)

// GatewayHeader defines the header keys for GenAI Gateway responses.
type GatewayHeader string

// All headeres must follow the Canonical Header Key format: capitalized words with hyphens.
const (
	GatewayResponseTimeMs       GatewayHeader = "X-Genai-Gateway-Response-Time-Ms"
	GatewayInputTokens          GatewayHeader = "X-Genai-Gateway-Input-Tokens"
	GatewayModelID              GatewayHeader = "X-Genai-Gateway-Model-Id"
	GatewayRegion               GatewayHeader = "X-Genai-Gateway-Region"
	GatewayOutputTokens         GatewayHeader = "X-Genai-Gateway-Output-Tokens"
	GatewayTokensPerSecond      GatewayHeader = "X-Genai-Gateway-Tokens-Per-Second"
	GatewayRetryCount           GatewayHeader = "X-Genai-Gateway-Retry-Count"
	GatewayTimeToFirstToken     GatewayHeader = "X-Genai-Gateway-Time-To-First-Token"
	GatewayProcessingDurationMs GatewayHeader = "X-Genai-Gateway-Processing-Duration-Ms"
	GatewayModelCallDurationMs  GatewayHeader = "X-Genai-Gateway-Model-Call-Duration-Ms"
	GatewayReasoningTokens      GatewayHeader = "X-Genai-Gateway-Reasoning-Tokens"
)

// GenAIRequestHeader defines the header keys for GenAI request correlation IDs.
type GenAIRequestHeader string

const (
	GenAIServiceRequestID GenAIRequestHeader = "pega-genai-service-request-id"
	GenAIContextID        GenAIRequestHeader = "pega-genai-context-id"
	GenAIConversationID   GenAIRequestHeader = "pega-genai-conversation-id"
)

// AllGatewayHeaders collects all headers with prefix "x-genai-gateway-"
// Note: GatewayReasoningTokens is intentionally excluded — it is a conditional header
// only set when reasoning_tokens > 0 (for reasoning models). It is still included in
// streaming trailer announcements (setupTrailers) and set directly in setTokenHeaders.
var AllGatewayHeaders = []GatewayHeader{
	GatewayResponseTimeMs,
	GatewayInputTokens,
	GatewayModelID,
	GatewayRegion,
	GatewayOutputTokens,
	GatewayTokensPerSecond,
	GatewayRetryCount,
	GatewayTimeToFirstToken,
	GatewayProcessingDurationMs,
	GatewayModelCallDurationMs,
}

var FailedInferenceHeaders = slices.DeleteFunc(slices.Clone(AllGatewayHeaders), func(header GatewayHeader) bool {
	return header == GatewayOutputTokens || header == GatewayTokensPerSecond || header == GatewayInputTokens || header == GatewayTimeToFirstToken
})

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{
			string(Path),
			string(Method),
			string(Code),
			string(ModelID),
			string(APIVersion),
			string(IsolationID),
		},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds",
			Help:    "A histogram of the HTTP request durations in milliseconds",
			Buckets: []float64{500, 1000, 3000, 5000, 7000, 10000, 30000, 60000, 600000},
		},
		[]string{
			string(Path),
			string(Method),
			string(Code),
			string(ModelID),
			string(APIVersion),
			string(IsolationID),
		},
	)

	requestDurationHighDefinition = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds_hd",
			Help:    "A histogram of the HTTP request durations in milliseconds",
			Buckets: []float64{500, 1000, 3000, 5000, 6000, 7000, 8000, 10000, 12000, 14000, 17000, 21000, 30000, 45000, 60000, 90000, 120000, 180000, 240000, 300000},
		},
		[]string{
			string(Path),
			string(Method),
			string(Code),
			string(ModelID),
			string(APIVersion),
			string(IsolationID),
		},
	)

	activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP requests",
		},
		[]string{
			string(Path),
			string(ModelID),
			string(APIVersion),
			string(IsolationID),
		},
	)

	inputTokensCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "input_tokens_per_request",
			Help: "Total number of input tokens used during a request",
		},
		[]string{
			string(ModelID),
			string(IsolationID),
		},
	)

	outputTokensCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "output_tokens_per_request",
			Help: "Total number of output tokens used during a request",
		},
		[]string{
			string(ModelID),
			string(IsolationID),
		},
	)

	tokensPerSecondCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tokens_per_second",
			Help: "token processing performance of genai models in tokens per second",
		},
		[]string{
			string(ModelID),
			string(IsolationID),
		},
	)

	reasoningTokensCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reasoning_tokens_per_request",
			Help: "Total number of reasoning tokens used by reasoning models during a request",
		},
		[]string{
			string(ModelID),
			string(IsolationID),
		},
	)

	latencyCollector = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "latency_ms_per_request",
			Help: "Time between call and response in milliseconds",
		},
		[]string{
			string(ModelID),
			string(IsolationID),
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(requestDurationHighDefinition)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(inputTokensCollector)
	prometheus.MustRegister(outputTokensCollector)
	prometheus.MustRegister(tokensPerSecondCollector)
	prometheus.MustRegister(reasoningTokensCollector)
	prometheus.MustRegister(latencyCollector)
}

func HttpMetricsMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(ginContext *gin.Context) {

		l := cntx.LoggerFromContext(ginContext).Sugar()

		// collect GUID from request JWT
		isolationId, err := getGuidFromToken(ginContext)
		if err != nil {
			l.Debugf("Failed to extract GUID from token: %v - using empty value", err)
			isolationId = "" // Provide a default value or handle the error as needed
		}

		path := ginContext.Request.URL.Path
		method := ginContext.Request.Method
		modelID, apiVersion := getModelRequestUrlValues(ginContext)

		activeConnections.WithLabelValues(path, modelID, apiVersion, isolationId).Inc()
		defer activeConnections.WithLabelValues(path, modelID, apiVersion, isolationId).Dec()

		// Create response writer with HTTP-only metrics collection
		mrw := NewMetricsResponseWriter(ginContext.Writer, l, path, isolationId)
		ginContext.Writer = mrw

		ginContext.Next()

		// Release the body buffer after request processing is complete.
		// For LLM endpoints, the buffer has already been released by writeHeaders();
		// for non-LLM endpoints (GET /models, /health, etc.) the buffer was never
		// needed but was still accumulating response data — release it now to avoid leaking.
		mrw.releaseBuffer()

		code := strconv.Itoa(ginContext.Writer.Status())
		elapsed := int(time.Since(mrw.start).Milliseconds())

		// Collect HTTP metrics for all endpoints (no headers for non-LLM endpoints)
		httpRequestsTotal.WithLabelValues(path, method, code, modelID, apiVersion, isolationId).Inc()
		requestDuration.WithLabelValues(path, method, code, modelID, apiVersion, isolationId).Observe(float64(elapsed))
		requestDurationHighDefinition.WithLabelValues(path, method, code, modelID, apiVersion, isolationId).Observe(float64(elapsed))

	}
}

// LLMMetricsMiddleware enhances existing MetricsResponseWriter to collect LLM-specific metrics and handle headers/trailers
func LLMMetricsMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		l := cntx.LoggerFromContext(ginContext).Sugar()

		var mrw *MetricsResponseWriter

		// Try to find MetricsResponseWriter even if wrapped
		if tacw, ok := ginContext.Writer.(interface{ GetMetricsWriter() *MetricsResponseWriter }); ok {
			// TrailerAwareCustomResponseWriter implements GetMetricsWriter()
			mrw = tacw.GetMetricsWriter()
			l.Debugf("Found MetricsResponseWriter via GetMetricsWriter interface")
		} else if mw, ok := ginContext.Writer.(*MetricsResponseWriter); ok {
			// Direct MetricsResponseWriter
			mrw = mw
			l.Debugf("Found MetricsResponseWriter directly")
		} else {
			l.Warnf("LLMMetricsMiddleware: Expected MetricsResponseWriter but got %T", ginContext.Writer)
		}

		if mrw != nil {
			// Initialize LLM metrics struct to enable LLM mode
			modelID, _ := getModelRequestUrlValues(ginContext)
			// Extract GenAI correlation headers
			requestID, contextID, conversationID := extractGenAIHeaders(ginContext)
			mrw.llmMetrics = &LLMMetrics{
				modelId:        modelID,
				requestID:      requestID,
				contextID:      contextID,
				conversationID: conversationID,
			}
			// Decide streaming vs non-streaming based on path/body
			mrw.llmMetrics.usesTrailers = isStreamingRequest(ginContext)
			l.Debugf("Enhanced response writer for LLM metrics collection")
		}

		ginContext.Next()

		// Find MetricsResponseWriter again after request processing
		if tacw, ok := ginContext.Writer.(interface{ GetMetricsWriter() *MetricsResponseWriter }); ok {
			mrw = tacw.GetMetricsWriter()
		} else if mw, ok := ginContext.Writer.(*MetricsResponseWriter); ok {
			mrw = mw
		} else {
			l.Warnf("LLMMetricsMiddleware: Expected MetricsResponseWriter after processing but got %T", ginContext.Writer)
			return
		}

		if mrw == nil {
			l.Warnf("LLMMetricsMiddleware: MetricsResponseWriter is nil")
			return
		}

		if mrw.llmMetrics == nil {
			l.Warnf("LLMMetricsMiddleware: llmMetrics is nil")
			return
		}

		mrw.setBaseHeaders()

		// Collect LLM metrics and handle headers/trailers based on mode
		shouldProcessMetrics := mrw.llmMetrics.usesTrailers || mrw.status < 400
		if shouldProcessMetrics {
			mrw.setTokenHeaders(ginContext)
		}

		isBufferedSuccess := !mrw.llmMetrics.usesTrailers && mrw.status < 400
		if isBufferedSuccess {
			mrw.writeHeaders(ginContext)
		}

		if mrw.llmMetrics != nil {
			// Collect LLM-specific Prometheus metrics
			latencyCollector.WithLabelValues(mrw.llmMetrics.modelId, mrw.isolationId).Set(float64(mrw.llmMetrics.elapsed))
			if mrw.status < 300 {
				// Ensure counters only receive non-negative values
				if mrw.llmMetrics.inputTokens > 0 {
					inputTokensCollector.WithLabelValues(mrw.llmMetrics.modelId, mrw.isolationId).Add(float64(mrw.llmMetrics.inputTokens))
				}
				if mrw.llmMetrics.outputTokens > 0 {
					outputTokensCollector.WithLabelValues(mrw.llmMetrics.modelId, mrw.isolationId).Add(float64(mrw.llmMetrics.outputTokens))
				}
				if mrw.llmMetrics.tokensPerSecond > 0 {
					tokensPerSecondCollector.WithLabelValues(mrw.llmMetrics.modelId, mrw.isolationId).Add(float64(mrw.llmMetrics.tokensPerSecond))
				}
				if mrw.llmMetrics.reasoningTokens > 0 {
					reasoningTokensCollector.WithLabelValues(mrw.llmMetrics.modelId, mrw.isolationId).Add(float64(mrw.llmMetrics.reasoningTokens))
				}
			}

			var contentLength int
			if mrw.body != nil { // a safety check to avoid calling on nil
				contentLength = mrw.body.Len()
			}
			l.Infof("LLM Request: infinity_isolation=%s, model=%s, response_code=%d, payload_size=%d, input_tokens=%d, output_tokens=%d, reasoning_tokens=%d, response_time_ms=%d, tokens_per_second=%d, requestID=%s, contextID=%s, conversationID=%s, model_call_duration_ms=%d, processing_duration_ms=%d, message=%s",
				mrw.isolationId,
				mrw.llmMetrics.modelId,
				mrw.Status(),
				contentLength,
				mrw.llmMetrics.inputTokens,
				mrw.llmMetrics.outputTokens,
				mrw.llmMetrics.reasoningTokens,
				mrw.llmMetrics.elapsed,
				mrw.llmMetrics.tokensPerSecond,
				mrw.llmMetrics.requestID,
				mrw.llmMetrics.contextID,
				mrw.llmMetrics.conversationID,
				mrw.llmMetrics.modelCallDuration,
				mrw.llmMetrics.processingDuration,
				mrw.llmMetrics.message,
			)
		}
	}
}

// LLMMetrics holds LLM-specific metrics and state
type LLMMetrics struct {
	elapsed            int
	modelId            string
	inputTokens        int
	outputTokens       int
	reasoningTokens    int
	tokensPerSecond    int
	timeToFirstToken   int
	trailersSet        bool
	usesTrailers       bool
	mu                 sync.Mutex // protects concurrent access to trailersSet
	modelCallDuration  int
	processingDuration int

	// GenAI correlation headers
	requestID      string
	contextID      string
	conversationID string
	message        string
}

// MetricsResponseWriter wraps the http.ResponseWriter to capture response data and metrics
type MetricsResponseWriter struct {
	gin.ResponseWriter
	*zap.SugaredLogger

	// Common fields for all endpoints
	body        *bytes.Buffer
	path        string
	start       time.Time
	status      int
	isolationId string

	// LLM-specific metrics (nil for HTTP-only endpoints, initialized for LLM endpoints)
	llmMetrics *LLMMetrics
}

func (l *LLMMetrics) updateLlmMetrics(m MetricsResult) {

	l.inputTokens = m.InputTokens
	l.outputTokens = m.OutputTokens
	l.reasoningTokens = m.ReasoningTokens
	l.message = m.Message

}

// NewMetricsResponseWriter creates a new MetricsResponseWriter with properly initialized fields
func NewMetricsResponseWriter(w gin.ResponseWriter, logger *zap.SugaredLogger, path string, isolationId string) *MetricsResponseWriter {
	return &MetricsResponseWriter{
		ResponseWriter: w,
		SugaredLogger:  logger,
		body:           bytes.NewBuffer(nil),
		path:           path,
		start:          time.Now(),
		isolationId:    isolationId,
		llmMetrics:     nil, // llmMetris initialized as nil. Will be populated only if all is a successful LLM call completes.
	}
}

// MetricsResponseWriter implementation
func (w *MetricsResponseWriter) Write(b []byte) (int, error) {
	timeFirstChunkReceived := int(time.Since(w.start).Milliseconds())
	if w.body != nil {
		w.body.Write(b)
	}

	// LLM endpoints: streaming vs non-streaming behavior
	if w.llmMetrics != nil {
		if w.llmMetrics.usesTrailers {
			if !w.llmMetrics.trailersSet {
				w.llmMetrics.mu.Lock()
				if !w.llmMetrics.trailersSet {
					w.setupTrailers()
					w.llmMetrics.trailersSet = true
					w.llmMetrics.timeToFirstToken = timeFirstChunkReceived
				}
				w.llmMetrics.mu.Unlock()
			}
			written, err := w.ResponseWriter.Write(b)
			w.ResponseWriter.Flush()
			return written, err
		}
		// Non-streaming LLM: buffer only for success responses, write through for errors
		if w.status >= 400 {
			written, err := w.ResponseWriter.Write(b)
			w.ResponseWriter.Flush()
			return written, err
		}
		return len(b), nil
	}

	written, err := w.ResponseWriter.Write(b)
	w.ResponseWriter.Flush()
	return written, err
}

// MetricsResponseWriter implementation
func (w *MetricsResponseWriter) WriteString(s string) (int, error) {
	if w.body != nil {
		w.body.WriteString(s)
	}

	timeFirstChunkReceived := int(time.Since(w.start).Milliseconds())
	// LLM endpoints: streaming vs non-streaming behavior
	if w.llmMetrics != nil {
		if w.llmMetrics.usesTrailers {
			if !w.llmMetrics.trailersSet {
				w.llmMetrics.mu.Lock()
				if !w.llmMetrics.trailersSet {
					w.setupTrailers()
					w.llmMetrics.trailersSet = true
					w.llmMetrics.timeToFirstToken = timeFirstChunkReceived
				}
				w.llmMetrics.mu.Unlock()
			}
			written, err := w.ResponseWriter.WriteString(s)
			w.ResponseWriter.Flush()
			return written, err
		}
		// Non-streaming LLM: buffer only for success responses, write through for errors
		if w.status >= 400 {
			written, err := w.ResponseWriter.WriteString(s)
			w.ResponseWriter.Flush()
			return written, err
		}
		return len(s), nil
	}

	written, err := w.ResponseWriter.WriteString(s)
	w.ResponseWriter.Flush()
	return written, err
}

func (w *MetricsResponseWriter) WriteHeader(statusCode int) {
	// record status
	w.status = statusCode
	// Defer sending status for non-streaming LLM responses so we can set headers at flush time
	// BUT write through immediately for error responses (no metrics parsing needed)
	if w.llmMetrics != nil && !w.llmMetrics.usesTrailers && statusCode < 400 {
		return
	}
	// For streaming LLM, non-LLM endpoints, and error responses, write through immediately
	w.ResponseWriter.WriteHeader(statusCode)
}

// Status returns the HTTP status code
func (w *MetricsResponseWriter) Status() int {
	if w.status != 0 {
		return w.status
	}
	return w.ResponseWriter.Status()
}

// setupTrailers declares which headers will be sent as trailers for streaming responses
func (w *MetricsResponseWriter) setupTrailers() {
	// Declare trailer headers that will be sent after the response body
	trailerHeaders := []string{
		string(GatewayResponseTimeMs),
		string(GatewayModelID),
		string(GatewayRegion),
		string(GatewayRetryCount),
		string(GatewayInputTokens),
		string(GatewayOutputTokens),
		string(GatewayTokensPerSecond),
		string(GatewayTimeToFirstToken),
		string(GatewayModelCallDurationMs),
		string(GatewayProcessingDurationMs),
		string(GatewayReasoningTokens),
	}

	// Remove Content-Length header if present - it's incompatible with chunked transfer encoding
	// RFC 2616 Section 4.4: "Messages MUST NOT include both a Content-Length header field and a
	// non-identity transfer-coding. If the message does include a non-identity transfer-coding,
	// the Content-Length MUST be ignored."
	w.Header().Del("Content-Length")
	w.Header().Set("Transfer-Encoding", "chunked")                // https://www.rfc-editor.org/rfc/rfc2616#section-14.41
	w.Header().Set("Trailer", strings.Join(trailerHeaders, ", ")) // https://www.rfc-editor.org/rfc/rfc2616#section-14.40
	w.SugaredLogger.Debugf("Set up trailer headers for streaming response")
}

// setTokenHeaders extracts metrics from response body and sets token headers for successful responses
func (w *MetricsResponseWriter) setTokenHeaders(ginContext *gin.Context) {
	// Get response body for parsing
	responseBody := w.getResponseBodyForParsing(ginContext)

	// Calculate final metrics: parse tokens from response
	metricsResult, err := extractTokenAndLatency(responseBody, w.path, w.llmMetrics.usesTrailers)
	if metricsResult != nil {
		w.llmMetrics.updateLlmMetrics(*metricsResult)
	}
	if err != nil {
		w.llmMetrics.message = err.Error()
	}

	// Set token headers for successful responses
	if w.status < 300 {
		oTps := math.Round(calculateOutputTokensPerSecond(float64(w.llmMetrics.outputTokens), float64(w.llmMetrics.elapsed)))
		w.llmMetrics.tokensPerSecond = int(oTps)
		headers := w.Header()
		headers.Set(string(GatewayInputTokens), strconv.Itoa(w.llmMetrics.inputTokens))
		headers.Set(string(GatewayOutputTokens), strconv.Itoa(w.llmMetrics.outputTokens))
		headers.Set(string(GatewayTokensPerSecond), strconv.Itoa(w.llmMetrics.tokensPerSecond))
		headers.Set(string(GatewayTimeToFirstToken), strconv.Itoa(w.llmMetrics.timeToFirstToken))
		if w.llmMetrics.reasoningTokens > 0 {
			headers.Set(string(GatewayReasoningTokens), strconv.Itoa(w.llmMetrics.reasoningTokens))
		}
	}
}

// writeHeaders writes gateway headers as regular HTTP headers for non-streaming responses.
// ginContext is used to check whether an outer buffering writer (RequestModificationResponseWriter)
// is still holding the body — in that case we must NOT commit headers/body to the wire here,
// because either FlushBufferedResponse or writeRetryResponse will do so later.
func (w *MetricsResponseWriter) writeHeaders(ginContext *gin.Context) {
	w.SugaredLogger.Debugf("Writing gateway headers for non-streaming response")

	// If the outermost writer is still buffering (i.e. RequestModificationResponseWriter
	// has not yet decided to flush or retry), we must not write through to net/http now.
	// The headers we just set via setBaseHeaders/setTokenHeaders are already in the shared
	// header map and will be sent when the outer writer eventually flushes.
	if outerWriter, ok := ginContext.Writer.(BufferingWriter); ok {
		if outerWriter.ShouldBuffer() {
			w.SugaredLogger.Debugf("writeHeaders: outer writer is still buffering — deferring write-through to flush/retry path")
			return
		}
	}

	// Set Content-Length from our own buffer
	w.Header().Set("Content-Length", strconv.Itoa(w.body.Len()))

	// Write status code and buffered body to the underlying writer
	w.ResponseWriter.WriteHeader(w.status)
	if _, err := w.ResponseWriter.Write(w.body.Bytes()); err != nil {
		w.SugaredLogger.Errorf("Failed to write response body: %v", err)
	}
	w.ResponseWriter.Flush()

	// Release the buffer to free memory — it is no longer needed after writing.
	w.releaseBuffer()
}

// setBaseHeaders calculates timing metrics and sets the base gateway headers
func (w *MetricsResponseWriter) setBaseHeaders() {
	// Compute elapsed from local clock
	actualElapsedMs := int(time.Since(w.start).Milliseconds())
	processingDuration := actualElapsedMs - w.llmMetrics.modelCallDuration

	w.llmMetrics.elapsed = actualElapsedMs
	w.llmMetrics.processingDuration = processingDuration

	// Set base gateway headers
	headers := w.Header()
	headers.Set(string(GatewayResponseTimeMs), strconv.Itoa(w.llmMetrics.elapsed))
	headers.Set(string(GatewayModelID), w.llmMetrics.modelId)
	headers.Set(string(GatewayRegion), "Standard")
	headers.Set(string(GatewayRetryCount), "0")
	headers.Set(string(GatewayProcessingDurationMs), strconv.Itoa(w.llmMetrics.processingDuration))
	headers.Set(string(GatewayModelCallDurationMs), strconv.Itoa(w.llmMetrics.modelCallDuration))
}

// ForceWriteThrough disables LLM buffering to allow immediate write-through to the underlying writer
// This is used for retry scenarios where we need to bypass the normal LLM buffering logic
func (w *MetricsResponseWriter) ForceWriteThrough() {
	if w.llmMetrics != nil {
		w.SugaredLogger.Debug("ForceWriteThrough: Disabling LLM buffering for write-through")
		// Clear the LLM metrics to disable LLM-specific buffering behavior
		// This makes Write() and WriteString() pass through to the underlying writer
		w.llmMetrics = nil
	}
}

// releaseBuffer nils the body buffer to free memory.
// After this call, any further Write/WriteString calls will still work —
// they write to w.body which will be nil, but the write path always falls through
// to the underlying writer, so the nil buffer is never dereferenced unsafely.
func (w *MetricsResponseWriter) releaseBuffer() {
	w.body = nil
}

// extractGenAIHeaders extracts GenAI correlation headers from the request
func extractGenAIHeaders(ginContext *gin.Context) (requestID, contextID, conversationID string) {
	requestID = ginContext.Request.Header.Get(string(GenAIServiceRequestID))
	contextID = ginContext.Request.Header.Get(string(GenAIContextID))
	conversationID = ginContext.Request.Header.Get(string(GenAIConversationID))
	return requestID, contextID, conversationID
}

/*
isStreamingRequest determines whether the request should be treated as streaming.

Rules:
- ConverseStream: streaming if path ends with "/converse-stream"
- OpenAI: streaming only for Chat Completions when path ends with "/chat/completions" AND request body has {"stream": true}
- All other cases: non-streaming
*/
func isStreamingRequest(ginContext *gin.Context) bool {
	path := ginContext.Request.URL.Path

	// ConverseStream streaming by suffix
	if strings.HasSuffix(path, "/converse-stream") {
		return true
	}

	// OpenAI Chat Completions streaming by suffix + minimal body flag
	if strings.HasSuffix(path, "/chat/completions") {
		raw, _ := io.ReadAll(ginContext.Request.Body)
		ginContext.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
		type chatCompletionStreaming struct {
			Stream bool `json:"stream"`
		}
		var req chatCompletionStreaming
		if err := json.Unmarshal(raw, &req); err == nil && req.Stream {
			return true
		}
		return false
	}

	return false
}

// Helper function to extract GUID from token
func getGuidFromToken(ctx *gin.Context) (environmentGuid string, err error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	bearerFromHeader := ctx.Request.Header.Get("Authorization")

	var trimmedToken string
	const bearerPrefix = "Bearer "

	// Check if the token starts with the "Bearer " prefix
	if strings.HasPrefix(bearerFromHeader, bearerPrefix) {
		// Remove the "Bearer " prefix
		trimmedToken = strings.TrimPrefix(bearerFromHeader, bearerPrefix)
	} else {
		// TODO: change this to warning after testing
		logger.Info("Token does not have the 'Bearer ' prefix")
	}

	// Split the token into its parts - done this way so I don't have to verify the signature
	parts := strings.Split(trimmedToken, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// Decode the payload part
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("error decoding payload: %w", err)
	}

	// Parse the JSON payload
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("error parsing payload: %w", err)
	}

	// Search for the GUID in the claims
	guid, exists := claims["guid"]
	if !exists {
		return "", fmt.Errorf("GUID not found in token claims")
	}

	guidString, ok := guid.(string)
	if !ok {
		return "", fmt.Errorf("invalid GUID format in token claims")
	}

	return guidString, nil
}

// Helper function to extract model and API version from the request
func getModelRequestUrlValues(ginContext *gin.Context) (modelID string, apiVersion string) {
	path := ginContext.Request.URL.Path
	modelID = ginContext.Param("modelId")

	if len(modelID) == 0 {
		modelID = ginContext.Param("buddyId")
	}

	// Split the path into segments
	providerCheck := strings.Split(path, "/")

	// Check if the second segment is "amazon" and modelID is empty
	if len(providerCheck) > 1 && providerCheck[1] == "amazon" && len(modelID) == 0 {
		// Hardcoded model id for amazon
		modelID = "amazon-titan-embed-text" // this is the hardcoded model id
	}

	apiVersion = ginContext.Query("api-version")
	return modelID, apiVersion
}

// parseChatCompletionsStreamResponse parses SSE (Server-Sent Events) format streaming response
// from OpenAI/Vertex Chat Completions API and extracts usage metadata.
// SSE format:
//
//	data: {"choices": [{"delta": {"content": "Hello"}}]}
//	data: {"choices": [{"finish_reason": "stop"}], "usage": {"prompt_tokens": 50, "completion_tokens": 100}}
//	data: [DONE]
func parseChatCompletionsStreamResponse(responseData string) (*MetricsResult, error) {
	scanner := bufio.NewScanner(strings.NewReader(responseData))

	var inputTokens, outputTokens int
	var metricsResult MetricsResult

	for scanner.Scan() {
		line := scanner.Text()

		// Skip non-data lines (empty lines, comments, etc.)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// End of stream marker
		if data == "[DONE]" {
			break
		}

		// Parse JSON chunk - use a minimal struct to extract only usage data
		var chunk struct {
			Usage struct {
				PromptTokens            int `json:"prompt_tokens"`
				CompletionTokens        int `json:"completion_tokens"`
				CompletionTokensDetails struct {
					ReasoningTokens int `json:"reasoning_tokens"`
				} `json:"completion_tokens_details"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip invalid JSON chunks (this is expected for some SSE events)
			continue
		}

		// Extract usage if present (typically in the final chunk)
		// Use last non-zero values as the final usage data
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
			metricsResult.ReasoningTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning chat completions streaming response: %w", err)
	}

	metricsResult.InputTokens = inputTokens
	metricsResult.OutputTokens = outputTokens
	metricsResult.LatencyMs = 0
	if inputTokens == 0 && outputTokens == 0 {
		metricsResult.Message = "metrics metadata not present in response payload"
	}

	return &metricsResult, nil
}

// extractTokenAndLatency extracts token counts and latency from provider-specific response formats
func extractTokenAndLatency(bufferedResponseBody string, path string, isStreaming bool) (*MetricsResult, error) {
	providerCheck := strings.Split(path, "/")

	switch providerCheck[1] {
	case "anthropic", "amazon", "meta":
		return parseConverseMetrics(bufferedResponseBody, isStreaming)
	case "openai", "google":
		return parseChatCompletionsMetrics(bufferedResponseBody, isStreaming)
	default:
		return nil, fmt.Errorf("call %s does not go to hosted model", path)
	}
}

// SetModelCallDuration sets the model call duration in the MetricsResponseWriter
func SetModelCallDuration(ginContext *gin.Context, durationMs int) {
	var mrw *MetricsResponseWriter
	l := cntx.LoggerFromContext(ginContext).Sugar()
	// Try to find MetricsResponseWriter even if wrapped
	if tacw, ok := ginContext.Writer.(interface{ GetMetricsWriter() *MetricsResponseWriter }); ok {
		// RequestModificationResponseWriter or other wrappers implement GetMetricsWriter()
		mrw = tacw.GetMetricsWriter()
	} else if mw, ok := ginContext.Writer.(*MetricsResponseWriter); ok {
		// Direct MetricsResponseWriter
		mrw = mw
	} else {
		l.Warnf("LLMMetricsMiddleware: Expected MetricsResponseWriter but got %T", ginContext.Writer)
	}

	if mrw != nil && mrw.llmMetrics != nil {
		mrw.llmMetrics.modelCallDuration = durationMs
	}
}

func parseConverseStreamUsageMetadata(bufferedResponse []byte) (json.RawMessage, error) {
	reader := bytes.NewReader(bufferedResponse)
	decoder := eventstream.NewDecoder()

	var usageEventJSON json.RawMessage
	eventCount := 0

	for {
		// Track reader position before decode to detect if reader is stuck
		prevPos := reader.Size() - int64(reader.Len())

		msg, err := decoder.Decode(reader, nil)
		if err != nil {
			if err == io.EOF {
				break
			}
			// Check if reader position didn't advance - this prevents infinite loop
			// when decoder encounters unrecoverable corrupted data
			currPos := reader.Size() - int64(reader.Len())
			if currPos == prevPos {
				break // Reader stuck on same position, exit to prevent infinite loop
			}
			// Reader advanced, skip corrupted frame and continue
			continue
		}

		eventCount++

		// Check if this event contains usage data using a minimal struct
		var checkEvent struct {
			Usage interface{} `json:"usage"`
		}
		if err := json.Unmarshal(msg.Payload, &checkEvent); err == nil && checkEvent.Usage != nil {
			usageEventJSON = msg.Payload
		}
	}

	if eventCount == 0 {
		return nil, fmt.Errorf("no valid JSON events found in EventStream binary response")
	}

	if usageEventJSON == nil {
		return nil, fmt.Errorf("no event with usage information found in EventStream response (parsed %d valid events)", eventCount)
	}

	return usageEventJSON, nil
}

// getResponseBodyForParsing gets the response body for metrics parsing.
// It first checks the MetricsResponseWriter's own buffer, and if empty,
// tries to get it from RequestModificationResponseWriter's buffer via the gin.Context.
func (w *MetricsResponseWriter) getResponseBodyForParsing(ginContext *gin.Context) string {
	// First try our own buffer
	var responseBody string
	if w.body != nil {
		responseBody = w.body.String()
	}

	// If our buffer is empty, try to get from RequestModificationResponseWriter via gin.Context
	if responseBody == "" {
		w.SugaredLogger.Debugf("MetricsResponseWriter body is empty, checking RequestModificationResponseWriter")
		// Get the current writer from gin.Context (which should be RequestModificationResponseWriter)
		if rmrw, ok := ginContext.Writer.(interface{ GetResponseBody() []byte }); ok {
			if bufferedBody := rmrw.GetResponseBody(); len(bufferedBody) > 0 {
				responseBody = string(bufferedBody)
				w.SugaredLogger.Debugf("Found response body in RequestModificationResponseWriter: %d bytes", len(bufferedBody))
			}
		}
	} else {
		w.SugaredLogger.Debugf("Using MetricsResponseWriter body: %d bytes", len(responseBody))
	}

	return responseBody
}
