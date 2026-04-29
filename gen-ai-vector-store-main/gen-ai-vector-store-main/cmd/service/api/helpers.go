/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

const (
	isolationIDParamName    = "isolationID"
	collectionNameParamName = "collectionName"
	docIDParamName          = "documentID"
	maxIsolationIDLength    = 36
	maxCollectionNameLength = 255
	serviceName             = "genai-vector-store"
	groupIDParamName        = "groupID"
)

func GetVsHeaders(isolationID, collectionID string) map[string]string {
	return map[string]string{
		"vs-isolation-id":  isolationID,
		"vs-collection-id": collectionID,
	}
}

func getIsolationIDAndCollectionName(c *gin.Context) (string, string, error) {
	isolationID := c.Param(isolationIDParamName)
	collectionName := c.Param(collectionNameParamName)

	if isolationID == "" {
		return isolationID, collectionName, fmt.Errorf("%s param is required", isolationIDParamName)
	}
	if len(isolationID) > maxIsolationIDLength {
		return isolationID, collectionName, fmt.Errorf("%s param cannot exceed %d characters", isolationIDParamName, maxIsolationIDLength)
	}

	if collectionName == "" {
		return isolationID, collectionName, fmt.Errorf("%s param is required", collectionNameParamName)
	}
	if len(collectionName) > maxCollectionNameLength {
		return isolationID, collectionName, fmt.Errorf("%s param cannot exceed %d characters", collectionNameParamName, maxCollectionNameLength)
	}

	return strings.ToLower(isolationID), strings.ToLower(collectionName), nil
}

func getIsolationIDName(c *gin.Context) (string, error) {
	isolationID := c.Param(isolationIDParamName)

	if isolationID == "" {
		return isolationID, fmt.Errorf("%s param is required", isolationIDParamName)
	}

	if len(isolationID) > maxIsolationIDLength {
		return isolationID, fmt.Errorf("%s param cannot exceed %d characters", isolationIDParamName, maxIsolationIDLength)
	}
	return strings.ToLower(isolationID), nil
}

func getGroupIDName(c *gin.Context) (string, error) {
	groupID := c.Param(groupIDParamName)

	if groupID == "" {
		return groupID, fmt.Errorf("%s param is required", groupIDParamName)
	}

	return groupID, nil
}

func logRequest(logger *zap.Logger, c *gin.Context) {
	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)
}

func logResponse(logger *zap.Logger, c *gin.Context, startTime time.Time) {
	dimeDiff := time.Since(startTime).Seconds()
	logger.Info("served request",
		zap.Int("status", c.Writer.Status()),
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
		zap.Float64("duration_sec", dimeDiff),
	)
	_ = logger.Sync()
}

func startAPIHandlerSpan(c *gin.Context, name string) (context.Context, trace.Span) {
	// End the token_validation span if it exists
	if validationSpan, exists := c.Get("token_validation_span"); exists {
		if span, ok := validationSpan.(trace.Span); ok {
			span.End()
		}
	}
	// Start a new span for the API handler
	spanName := fmt.Sprintf("%s  %s", serviceName, name)
	ctx := c.Request.Context()
	spanCtx, span := otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(ctx, spanName)
	return spanCtx, span
}
