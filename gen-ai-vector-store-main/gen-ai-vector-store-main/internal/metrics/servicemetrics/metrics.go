/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
)

type ctxKey string

const (
	serviceMetricsKey ctxKey = "serviceMetrics"
)

type ServiceMetrics struct {
	DbMetrics        DB
	EmbeddingMetrics Embedding
	ResponseMetrics  Response
	GatewayMetrics   Gateway
	RequestMetrics   Request
}

// CalculateProcessingOverheads returns (processingDurationMs, overheadMs, embeddingNetOverheadMs).
// processingDurationMs = overheadMs + embeddingNetOverheadMs
// Handles missing gateway headers: empty string or "-1" → treated as 0
func (sm *ServiceMetrics) CalculateProcessingOverheads() (int64, int64, int64) {
	requestMs := sm.RequestMetrics.Duration().Milliseconds()
	dbMs := sm.DbMetrics.QueryExecutionTime().Milliseconds()

	var embMs int64
	embeddingMetrics := sm.EmbeddingMetrics.GetMetrics()
	if len(embeddingMetrics) > 0 {
		embMs = embeddingMetrics[0].TotalExecutionTime.Milliseconds()
	}

	// GatewayResponseTimeMs is "" (headers==nil, no gateway call) or "-1" (default, no response) → 0
	gwMs := parseGatewayResponseTimeMs(sm.GatewayMetrics.GetHeader(headers.GatewayResponseTimeMs))

	overheadMs := clampToZero(requestMs - dbMs - embMs)
	var embNetOverheadMs int64
	if embMs > 0 {
		embNetOverheadMs = clampToZero(embMs - gwMs)
	}
	return overheadMs + embNetOverheadMs, overheadMs, embNetOverheadMs
}

func clampToZero(val int64) int64 {
	if val < 0 {
		return 0
	}
	return val
}

func parseGatewayResponseTimeMs(val string) int64 {
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (sm *ServiceMetrics) LogMetrics(logger *zap.Logger) {
	queryDurationLog := zap.Duration("dbQueryTimeSec", sm.DbMetrics.QueryExecutionTime())

	embeddingMetrics := sm.EmbeddingMetrics.GetMetrics()
	embeddingMetricsLogs := []zap.Field{}
	for _, embMetric := range embeddingMetrics {
		embeddingMetricsLogs = append(embeddingMetricsLogs,
			zap.Dict(embMetric.ModelID,
				zap.String("modelVersion", embMetric.ModelVersion),
				zap.Duration("embeddingTimeSec", embMetric.TotalExecutionTime),
				zap.Int("embedderCallsCount", embMetric.TotalMeasurementCount),
				zap.Int("embedderRetriesCount", embMetric.TotalRetryCount),
			),
		)
	}
	embeddingsLog := zap.Dict("embeddingMetrics", embeddingMetricsLogs...)

	responseLog := zap.Int("itemsReturned", sm.ResponseMetrics.ItemsReturned())

	gatewayLog := zap.Any("gatewayMetrics", sm.GatewayMetrics.GetHeaders())

	requestLog := zap.Duration("requestDurationSec", sm.RequestMetrics.Duration())

	processingDurationMs, overheadMs, embNetOverheadMs := sm.CalculateProcessingOverheads()
	processingDurationLog := zap.Int64("vsProcessingDurationMs", processingDurationMs)
	overheadLog := zap.Int64("vsOverheadMs", overheadMs)
	embNetOverheadLog := zap.Int64("vsEmbeddingNetOverheadMs", embNetOverheadMs)

	logger.Info("Service Metrics",
		queryDurationLog,
		embeddingsLog,
		responseLog,
		requestLog,
		gatewayLog,
		processingDurationLog,
		overheadLog,
		embNetOverheadLog,
	)
}

func FromContext(ctx context.Context) *ServiceMetrics {
	if ctx == nil {
		return nil
	}

	if metrics, ok := ctx.Value(serviceMetricsKey).(*ServiceMetrics); ok {
		return metrics
	}

	return &ServiceMetrics{}
}

func WithMetrics(ctx context.Context) context.Context {
	metrics := &ServiceMetrics{}
	return context.WithValue(ctx, serviceMetricsKey, metrics)
}
