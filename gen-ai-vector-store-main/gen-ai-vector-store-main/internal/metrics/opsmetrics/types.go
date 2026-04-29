/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package opsmetrics

import (
	"context"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
)

type DocumentsMetricsResponse struct {
	Total DocumentsMetrics `json:"total" binding:"required"`
}

type DocumentsMetricsDetailsResponse struct {
	CollectionsMetrics []CollectionsMetrics `json:"collectionsMetrics" binding:"required"`
}

type CollectionsMetrics struct {
	ID               string           `json:"id" binding:"required"`
	DocumentsMetrics DocumentsMetrics `json:"documentsDetailsMetrics" binding:"required"`
}

type DocumentsMetricsRequest struct {
	Metrics []string `json:"metrics,omitempty"`
}

type DocumentsMetricsDetailsRequest struct {
	Metrics []string `json:"metrics,omitempty"`
}

type DocumentsMetrics struct {
	DiskUsage             int64      `json:"diskUsage"`
	DocumentsCount        int64      `json:"documentsCount"`
	DocumentsModification *time.Time `json:"documentsModification"`
}

type OpsMetrics interface {
	DocumentsMetricsPerIsolation(ctx context.Context, req DocumentsMetricsRequest, isoId string) (*DocumentsMetrics, error)
	DocumentsMetricsPerCollection(ctx context.Context, req DocumentsMetricsRequest, isoId string) ([]CollectionsMetrics, error)
}

type opsMetrics struct {
	database    db.Database
	isolationID string
}

func NewOpsMetrics(dbConn db.Database, isolationID string) *opsMetrics {
	return &opsMetrics{
		database:    dbConn,
		isolationID: isolationID,
	}
}
