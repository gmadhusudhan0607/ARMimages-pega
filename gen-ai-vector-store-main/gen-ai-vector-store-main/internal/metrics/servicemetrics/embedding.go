/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"context"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
)

// ModelMetrics holds the metrics for a specific model version.
type ModelMetrics struct {
	ModelID               string
	ModelVersion          string
	TotalExecutionTime    time.Duration
	TotalMeasurementCount int
	TotalRetryCount       int
}

type Embedding struct {
	modelToMetrics map[model][]modelMetrics
	mutex          sync.RWMutex
}

func (m *Embedding) NewMeasurement(modelID, modelVersion string) EmbeddingMeasurement {
	return EmbeddingMeasurement{
		model: model{
			id:      modelID,
			version: modelVersion,
		},
		e: m,
	}
}

type EmbeddingMeasurement struct {
	startTime  time.Time
	duration   time.Duration
	model      model
	retryCount int
	e          *Embedding
}

type embMeasurementCtxKey string

const embeddingMeasurementCtxKey embMeasurementCtxKey = "embeddingMeasurement"

func WithEmbeddingMeasurement(ctx context.Context, m *EmbeddingMeasurement) context.Context {
	return context.WithValue(ctx, embeddingMeasurementCtxKey, m)
}

func EmbeddingMeasurementFromContext(ctx context.Context) *EmbeddingMeasurement {
	if v := ctx.Value(embeddingMeasurementCtxKey); v != nil {
		return v.(*EmbeddingMeasurement)
	}

	log.GetLoggerFromContext(ctx).Warn("No embedding measurement found in context")
	return &EmbeddingMeasurement{}
}

func (m *EmbeddingMeasurement) IncreaseRetries() {
	m.retryCount++
}

func (m *EmbeddingMeasurement) Retries() int {
	return m.retryCount
}

func (m *EmbeddingMeasurement) Start() {
	m.startTime = time.Now()
}

func (m *EmbeddingMeasurement) Stop() {
	if m.startTime.IsZero() {
		return
	}

	m.e.mutex.Lock()
	defer m.e.mutex.Unlock()

	if m.e.modelToMetrics == nil {
		m.e.modelToMetrics = make(map[model][]modelMetrics)
	}

	m.duration = time.Since(m.startTime)

	m.e.modelToMetrics[m.model] = append(m.e.modelToMetrics[m.model], modelMetrics{
		executionTime: m.duration,
		retryCount:    m.retryCount,
	})

	m.startTime = time.Time{}
}

func (m *EmbeddingMeasurement) Duration() time.Duration {
	return m.duration
}

type model struct {
	id      string
	version string
}

type modelMetrics struct {
	retryCount    int
	executionTime time.Duration
}

func (e *Embedding) GetMetrics() []ModelMetrics {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var result []ModelMetrics
	for m, metrics := range e.modelToMetrics {
		totalTime := time.Duration(0)
		for _, mm := range metrics {
			totalTime += mm.executionTime
		}
		result = append(result, ModelMetrics{
			ModelID:               m.id,
			ModelVersion:          m.version,
			TotalExecutionTime:    totalTime,
			TotalMeasurementCount: len(metrics),
			TotalRetryCount:       totalRetriesCount(metrics),
		})
	}
	return result
}

func totalRetriesCount(metrics []modelMetrics) int {
	count := 0

	for _, m := range metrics {
		count += m.retryCount
	}

	return count
}
