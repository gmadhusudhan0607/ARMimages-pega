/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

// This file implements a random embedder that generates random vectors.
// It is used for testing purposes only and should not be used in production.

package random

import (
	"context"
	math "math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

const (
	embeddingModel        = "random-generator"
	embeddingModelVersion = "1.0"
)

var logger = log.GetNamedLogger("genai-vector-store")

// RandomEmbedder implements TextEmbedder for testing with random vectors
type RandomEmbedder struct {
	uri         string
	vectorLen   int
	httpHeaders map[string]string
	delay       time.Duration
}

// NewRandomEmbedder creates a new random embedder instance for testing
func NewRandomEmbedder(uri string, vectorLen int, httpHeaders map[string]string) (embedders.TextEmbedder, error) {
	rc := &RandomEmbedder{
		uri:         uri,
		httpHeaders: httpHeaders,
		vectorLen:   vectorLen,
	}
	delayStr := os.Getenv("RANDOM_EMBEDDER_DELAY")
	if delayStr != "" {
		delaySec, err := strconv.ParseFloat(delayStr, 64)
		if err == nil && delaySec > 0 {
			rc.delay = time.Duration(delaySec * float64(time.Second))
		}
	}
	// No delay by default when RANDOM_EMBEDDER_DELAY is not provided
	return rc, nil
}

// NewRandomClient is deprecated: use NewRandomEmbedder instead
// This function is provided for backward compatibility and will be removed in a future version
func NewRandomClient(uri string, vectorLen int, httpHeaders map[string]string) (embedders.Embedder, error) {
	return NewRandomEmbedder(uri, vectorLen, httpHeaders)
}

func (m *RandomEmbedder) GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error) {
	var measurement = servicemetrics.FromContext(ctx).EmbeddingMetrics.NewMeasurement(embeddingModel, embeddingModelVersion)

	// Define empty values for model metrics since we're not making HTTP calls
	var modelCallHostName, modelCallPath, modelCallMethod, modelCallCode string

	measurement.Start() // start timer
	// Track active HTTP connections metrics (even though we don't have real HTTP calls)
	embedders.AddModelHttpMetrics(
		modelCallHostName, modelCallPath, modelCallMethod,
		modelCallCode, embeddingModel, embeddingModelVersion, measurement.Duration().Seconds(), false, measurement.Retries())

	defer func() {
		measurement.Stop() // stop timer
		embedders.AddModelHttpMetrics(
			modelCallHostName, modelCallPath, modelCallMethod,
			modelCallCode, embeddingModel, embeddingModelVersion, measurement.Duration().Seconds(), true, measurement.Retries())
	}()

	// Generate random vector
	vector := make([]float32, m.vectorLen)
	for i := 0; i < m.vectorLen; i++ {
		vector[i] = float32(math.Float64()*2 - 1) // random float between -1 and 1
	}

	// Emulate delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	logger.Info("Generated random vector", zap.Int("vector_length", m.vectorLen))
	return vector, http.StatusOK, nil
}

func (m *RandomEmbedder) GetURL() string {
	return m.uri
}
