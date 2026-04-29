/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package smart_chunking

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

const (
	serviceName      = "genai-vector-store"
	isolationIDParam = "isolationID"
)

// ServiceError represents a non-202 HTTP response from the Smart Chunking service.
type ServiceError struct {
	StatusCode int
	Body       string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("smart-chunking service returned error (code: %d): %s", e.StatusCode, e.Body)
}

// SmartChunkingClient sends requests to the Smart Chunking /job API.
type SmartChunkingClient interface {
	SubmitJob(ctx context.Context, authToken, isolationID string, fileReader io.Reader, fileName string, options JobRequestOptions) (*JobSubmittedResponse, error)
}

type smartChunkingClient struct {
	uri    string
	client http_client.HTTPClient
}

func NewTracedSmartChunkingClient(uri string) (SmartChunkingClient, error) {
	httpClient, err := http_client.NewHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to init SmartHttpClient: %w", err)
	}
	tracedClient, err := http_client.NewTracedHTTPClient(httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create traced client: %w", err)
	}
	return &smartChunkingClient{
		client: tracedClient,
		uri:    uri,
	}, nil
}

func (m *smartChunkingClient) getLogger(isolationID string) *zap.Logger {
	return log.GetNamedLogger(serviceName).With(
		zap.String(isolationIDParam, isolationID),
	)
}

// SubmitJob posts a file to the Smart Chunking /v1/{isolationID}/jobs endpoint
// using io.Pipe to stream the file content without buffering it in memory.
func (m *smartChunkingClient) SubmitJob(ctx context.Context, authToken, isolationID string, fileReader io.Reader, fileName string, options JobRequestOptions) (*JobSubmittedResponse, error) {
	logger := m.getLogger(isolationID)

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job options: %w", err)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write the multipart form in a goroutine so the pipe reader
	// can be consumed concurrently by the HTTP request.
	// Errors are propagated through the pipe via CloseWithError.
	go func() {
		var err error
		defer func() { pw.CloseWithError(err) }()

		if err = writer.WriteField("options", string(optionsJSON)); err != nil {
			err = fmt.Errorf("failed to write options field: %w", err)
			return
		}

		var part io.Writer
		if part, err = writer.CreateFormFile("file", fileName); err != nil {
			err = fmt.Errorf("failed to create form file: %w", err)
			return
		}

		if _, err = io.Copy(part, fileReader); err != nil {
			err = fmt.Errorf("failed to stream file content: %w", err)
			return
		}

		if err = writer.Close(); err != nil {
			err = fmt.Errorf("failed to close multipart writer: %w", err)
		}
	}()

	uri := fmt.Sprintf("%s/v1/%s/jobs", m.uri, isolationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, pr)
	if err != nil {
		pr.Close() // unblock the writer goroutine
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	logger.Info("submitting job to smart-chunking",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.String("fileName", fileName),
	)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("smart-chunking service call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	logger.Debug("smart-chunking service response",
		zap.Int("statusCode", resp.StatusCode),
		zap.String("body", helpers.ToTruncatedString(respBody)),
	)

	if resp.StatusCode != http.StatusAccepted {
		return nil, &ServiceError{StatusCode: resp.StatusCode, Body: helpers.ToTruncatedString(respBody)}
	}

	var jobResp JobSubmittedResponse
	if err := json.Unmarshal(respBody, &jobResp); err != nil {
		return nil, fmt.Errorf("could not unmarshal job response: %w", err)
	}

	return &jobResp, nil
}
