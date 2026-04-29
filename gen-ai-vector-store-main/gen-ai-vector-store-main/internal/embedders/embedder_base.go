/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	errorshelper "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/errors"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var injectTestHeaders = helpers.GetEnvOrDefault("INJECT_TEST_HEADERS", "false") == "true"

// EmbedderBase provides common functionality for all HTTP-based embedders
type EmbedderBase struct {
	uri          string
	client       http_client.HTTPClient
	httpHeaders  map[string]string
	modelName    string
	modelVersion string
	logPrefix    string
	logger       *zap.Logger
}

// BaseEmbedder is deprecated: use EmbedderBase instead
// This alias is provided for backward compatibility and will be removed in a future version
type BaseEmbedder = EmbedderBase

// EmbeddingProcessor defines the interface for processing embedding requests/responses
type EmbeddingProcessor interface {
	CreateRequest(ctx context.Context, chunk string) (*http.Request, error)
	ProcessResponse(resp *http.Response) ([]float32, error)
}

// NewEmbedderBase creates a new base embedder with common HTTP client setup
func NewEmbedderBase(
	uri string,
	httpHeaders map[string]string,
	cfg http_client.HTTPClientConfig,
	modelName, modelVersion,
	logPrefix string,
	logger *zap.Logger,
) (*EmbedderBase, error) {
	httpClient, err := http_client.NewHTTPClientWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init HTTP client for %s: %w", logPrefix, err)
	}

	// Wrap with tracing
	tracedClient, err := http_client.NewTracedHTTPClient(httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create traced client for %s: %w", logPrefix, err)
	}

	return &EmbedderBase{
		uri:          uri,
		client:       tracedClient,
		httpHeaders:  httpHeaders,
		modelName:    modelName,
		modelVersion: modelVersion,
		logPrefix:    logPrefix,
		logger:       logger,
	}, nil
}

// GetEmbeddingWithProcessor handles the common embedding flow using a processor for model-specific logic
func (b *BaseEmbedder) GetEmbeddingWithProcessor(ctx context.Context, chunk string, processor EmbeddingProcessor) ([]float32, int, error) {
	var modelCallHostName, modelCallPath, modelCallMethod, modelCallCode string
	var measurement = servicemetrics.FromContext(ctx).EmbeddingMetrics.NewMeasurement(b.modelName, b.modelVersion)

	// Start metrics tracking
	AddModelHttpMetrics(modelCallHostName, modelCallPath, modelCallMethod, modelCallCode, b.modelName, b.modelVersion, measurement.Duration().Seconds(), false, measurement.Retries())
	measurement.Start()

	defer func() {
		measurement.Stop()
		AddModelHttpMetrics(modelCallHostName, modelCallPath, modelCallMethod, modelCallCode, b.modelName, b.modelVersion, measurement.Duration().Seconds(), true, measurement.Retries())
	}()

	// Create request using processor
	req, err := processor.CreateRequest(ctx, chunk)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("cannot create request: %w", err)
	}

	req = req.WithContext(servicemetrics.WithEmbeddingMeasurement(req.Context(), &measurement))

	// Parse URL for metrics
	parsedUrl, err := url.Parse(req.URL.String())
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("non parsable url found: [%s], error: %w", req.URL.String(), err)
	}
	modelCallHostName = parsedUrl.Hostname()
	modelCallPath = parsedUrl.Path
	modelCallMethod = req.Method

	// Make HTTP call
	httpResp, err := b.call(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, http.StatusGatewayTimeout, fmt.Errorf("request timeout: %w", err)
		}

		// Check if it's an HTTPClientError to get status code
		var httpClientErr *http_client.HTTPClientError
		if errors.As(err, &httpClientErr) {
			// Check if it's a query embedding timeout
			if errorshelper.IsTimeout(err) {
				return nil, http.StatusGatewayTimeout, fmt.Errorf("query embedding timeout: %w", err)
			}
			if httpClientErr.LastStatusCode == http.StatusTooManyRequests {
				return nil, http.StatusTooManyRequests, fmt.Errorf("%s", httpClientErr.GetLastStatusText())
			}
			return nil, http.StatusInternalServerError, fmt.Errorf("embedding returned an error: %s", httpClientErr.GetLastStatusText())
		}

		// For other errors
		return nil, http.StatusInternalServerError, err
	}

	// Handle common HTTP error responses
	if httpResp.StatusCode == http.StatusForbidden {
		return nil, http.StatusForbidden, ConstructModelForbiddenError(httpResp.Body)
	}

	if httpResp.StatusCode == http.StatusNotFound {
		return nil, http.StatusNotFound, ConstructModelNotFoundError(httpResp.Body)
	}

	if httpResp.StatusCode > 399 {
		return nil, httpResp.StatusCode, fmt.Errorf("embedding returned status code %d without an error", httpResp.StatusCode)
	}

	if httpResp.StatusCode != 200 {
		b.logger.Debug("httpResponse statusCode", zap.Int("statusCode", httpResp.StatusCode))
	}

	// Process response using processor
	embedding, err := processor.ProcessResponse(httpResp)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to process response: %w", err)
	}

	return embedding, http.StatusOK, nil
}

// GetURL returns the embedder's URI
func (b *BaseEmbedder) GetURL() string {
	return b.uri
}

// call handles the common HTTP request logic with headers, logging, and error handling
func (b *BaseEmbedder) call(req *http.Request) (*http.Response, error) {
	// Inject test headers if enabled
	if injectTestHeaders {
		for k, v := range b.httpHeaders {
			if !helpers.IsValidHeaderName(k) {
				b.logger.Warn(fmt.Sprintf("%s: skipping invalid header name", b.logPrefix), zap.String("header", k))
				continue
			}
			sanitizedValue := helpers.SanitizeHeaderValue(v)
			if sanitizedValue != v {
				b.logger.Warn(fmt.Sprintf("%s: sanitized suspicious header value", b.logPrefix), zap.String("header", k))
			}
			req.Header.Set(k, sanitizedValue)
			b.logger.Debug(fmt.Sprintf("%s: added header", b.logPrefix), zap.String("key", k), zap.String("value", sanitizedValue))
		}
	}

	// Always inject isolation-id header (regardless of test mode)
	// TODO (vitua): Refactor to not extract isolationID from httpHeaders map. (Fully Refactor injectTestHeaders approach)
	if isolationID, ok := b.httpHeaders["vs-isolation-id"]; ok && isolationID != "" {
		sanitizedValue := helpers.SanitizeHeaderValue(isolationID)
		if sanitizedValue != isolationID {
			b.logger.Warn(fmt.Sprintf("%s: sanitized suspicious isolation-id value", b.logPrefix))
		}
		req.Header.Set(headers.IsolationId, sanitizedValue)
		b.logger.Debug(fmt.Sprintf("%s: added isolation-id header", b.logPrefix), zap.String("key", headers.IsolationId), zap.String("value", sanitizedValue))
	}

	// Log request details
	if b.logger.Core().Enabled(zapcore.DebugLevel) {
		dReq, err := httputil.DumpRequest(req, false)
		if err != nil {
			b.logger.Error("failed to dump request", zap.Any("request", req), zap.Error(err))
		}
		b.logger.Info("Embedding HTTP request", zap.String("method", req.Method), zap.String("url", req.URL.String()), zap.String("request", regexp.MustCompile(`\s+`).ReplaceAllString(string(dReq), " ")))
	} else {
		b.logger.Info("Embedding HTTP request", zap.String("method", req.Method), zap.String("url", req.URL.String()))
	}

	// Make the HTTP call
	resp, err := b.client.Do(req)
	if resp != nil && resp.StatusCode != http.StatusOK {
		b.logger.Info("received response", zap.Int("statusCode", resp.StatusCode), zap.String("status", resp.Status))
	}
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateJSONRequest creates an HTTP request with JSON body
func (b *BaseEmbedder) CreateJSONRequest(ctx context.Context, method string, body interface{}) (*http.Request, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, b.uri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// UnmarshalJSONResponse unmarshals JSON response into the provided interface
func (b *BaseEmbedder) UnmarshalJSONResponse(r *http.Response, v interface{}) error {
	defer r.Body.Close()
	if v == nil {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

// ValidateEmbeddingResponse validates common embedding response patterns
func (b *BaseEmbedder) ValidateEmbeddingResponse(embeddings [][]float32, statusCode int) error {
	if len(embeddings) == 0 {
		return fmt.Errorf("no embedding data returned with code %d", statusCode)
	}

	if len(embeddings) > 1 {
		return fmt.Errorf("embeddings count > 1, case not yet handled")
	}

	return nil
}

// SimpleEmbeddingProcessor provides a basic implementation for simple embedding APIs
type SimpleEmbeddingProcessor struct {
	base             *BaseEmbedder
	createBody       func(chunk string) interface{}
	extractEmbedding func(resp interface{}) ([]float32, error)
}

// NewSimpleEmbeddingProcessor creates a processor for simple embedding APIs
func NewSimpleEmbeddingProcessor(base *BaseEmbedder, createBody func(chunk string) interface{}, extractEmbedding func(resp interface{}) ([]float32, error)) *SimpleEmbeddingProcessor {
	return &SimpleEmbeddingProcessor{
		base:             base,
		createBody:       createBody,
		extractEmbedding: extractEmbedding,
	}
}

func (s *SimpleEmbeddingProcessor) CreateRequest(ctx context.Context, chunk string) (*http.Request, error) {
	body := s.createBody(chunk)
	return s.base.CreateJSONRequest(ctx, http.MethodPost, body)
}

func (s *SimpleEmbeddingProcessor) ProcessResponse(resp *http.Response) ([]float32, error) {
	var responseData interface{}
	if err := s.base.UnmarshalJSONResponse(resp, &responseData); err != nil {
		return nil, err
	}

	return s.extractEmbedding(responseData)
}
