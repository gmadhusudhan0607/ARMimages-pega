// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

// PartialFailureError indicates that some chunks were sent successfully before failure
type PartialFailureError struct {
	Err              error
	SuccessfullySent int // Number of metrics successfully sent before failure
	TotalMetrics     int // Total number of metrics
	FailedSegment    int // Which segment failed
}

func (e *PartialFailureError) Error() string {
	return fmt.Sprintf("partial failure at segment %d: %v (sent %d/%d metrics)",
		e.FailedSegment, e.Err, e.SuccessfullySent, e.TotalMetrics)
}

func (e *PartialFailureError) Unwrap() error {
	return e.Err
}

// PDCSender handles HTTP communication with PDC endpoints including automatic chunking
// It uses Go generics to support any metric type T
type PDCSender[T any] struct {
	httpClient          *http.Client
	logger              *zap.Logger
	maxPayloadSizeBytes int
}

// PDCSenderConfig holds configuration for PDC sender
type PDCSenderConfig struct {
	RequestTimeoutSeconds int
	MaxPayloadSizeBytes   int
}

// NewPDCSender creates a new generic PDC sender with the given configuration
func NewPDCSender[T any](config PDCSenderConfig, loggerName string) *PDCSender[T] {
	return &PDCSender[T]{
		httpClient: &http.Client{
			Timeout: time.Duration(config.RequestTimeoutSeconds) * time.Second,
		},
		logger:              log.GetNamedLogger(loggerName),
		maxPayloadSizeBytes: config.MaxPayloadSizeBytes,
	}
}

// Send sends metrics to PDC with automatic chunking if payload exceeds size limit
// This method does NOT include retry logic - callers should implement their own retry strategy
func (s *PDCSender[T]) Send(ctx context.Context, pdcURL string, metrics []T) error {
	if len(metrics) == 0 {
		s.logger.Debug("No metrics to send")
		return nil
	}

	// Convert URL to PDC format
	convertedURL := ConvertUsageDataURL(pdcURL)

	// Create payload to check size
	payload := s.createPayload(metrics, 1, 1)
	payloadSize := s.calculatePayloadSize(payload)

	s.logger.Debug("Preparing to send metrics to PDC",
		zap.String("originalURL", pdcURL),
		zap.String("convertedURL", convertedURL),
		zap.Int("metricsCount", len(metrics)),
		zap.Int("payloadSize", payloadSize))

	// If payload is small enough, send as single chunk
	if payloadSize <= s.maxPayloadSizeBytes {
		return s.sendChunk(ctx, convertedURL, metrics, 1, 1)
	}

	// Split into chunks
	return s.sendChunked(ctx, convertedURL, metrics, payloadSize)
}

// sendChunked splits metrics into multiple chunks and sends them
// Returns PartialFailureError if some chunks succeeded but others failed
func (s *PDCSender[T]) sendChunked(ctx context.Context, url string, metrics []T, totalSize int) error {
	totalMetrics := len(metrics)

	// Calculate chunk size
	numChunks := (totalSize / s.maxPayloadSizeBytes) + 1
	chunkSize := totalMetrics / numChunks
	if chunkSize == 0 {
		chunkSize = 1
	}

	s.logger.Info("Chunking large PDC payload",
		zap.Int("totalMetrics", totalMetrics),
		zap.Int("totalSizeBytes", totalSize),
		zap.Int("numChunks", numChunks),
		zap.Int("chunkSize", chunkSize))

	// Send each chunk
	for i := 0; i < totalMetrics; i += chunkSize {
		end := i + chunkSize
		if end > totalMetrics {
			end = totalMetrics
		}

		chunk := metrics[i:end]
		segmentNumber := (i / chunkSize) + 1
		segmentsTotal := numChunks

		if err := s.sendChunk(ctx, url, chunk, segmentNumber, segmentsTotal); err != nil {
			// Return partial failure error with information about which metrics weren't sent
			return &PartialFailureError{
				Err:              err,
				SuccessfullySent: i,
				TotalMetrics:     totalMetrics,
				FailedSegment:    segmentNumber,
			}
		}
	}

	return nil
}

// sendChunk sends a single chunk of metrics to PDC
func (s *PDCSender[T]) sendChunk(ctx context.Context, url string, metrics []T, segmentNumber, segmentsTotal int) error {
	payload := s.createPayload(metrics, segmentNumber, segmentsTotal)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	s.logger.Debug("Sending metrics chunk to PDC",
		zap.String("url", url),
		zap.Int("metricsCount", len(metrics)),
		zap.Int("segmentNumber", segmentNumber),
		zap.Int("segmentsTotal", segmentsTotal),
		zap.Int("payloadSize", len(jsonData)))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PDC endpoint responded with status %d: %s", resp.StatusCode, resp.Status)
	}

	s.logger.Info("Successfully sent metrics chunk to PDC",
		zap.String("url", url),
		zap.Int("metricsCount", len(metrics)),
		zap.Int("segmentNumber", segmentNumber),
		zap.Int("segmentsTotal", segmentsTotal))

	return nil
}

// createPayload creates a payload structure for PDC
func (s *PDCSender[T]) createPayload(metrics []T, segmentNumber, segmentsTotal int) UsageDataPayload[T] {
	return UsageDataPayload[T]{
		Data: metrics,
		Metadata: UsageDataMetadata{
			SegmentNumber: segmentNumber,
			SegmentsTotal: segmentsTotal,
			Source:        metricSourceGenAIVectorStore,
		},
	}
}

// calculatePayloadSize estimates the JSON payload size in bytes
func (s *PDCSender[T]) calculatePayloadSize(payload UsageDataPayload[T]) int {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		// Return a conservative estimate
		return s.maxPayloadSizeBytes + 1
	}
	return len(jsonData)
}
