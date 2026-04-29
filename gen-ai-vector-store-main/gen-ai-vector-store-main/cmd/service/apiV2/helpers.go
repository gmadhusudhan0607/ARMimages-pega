/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package apiV2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func logRequest(logger *zap.Logger, c *gin.Context) {
	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI))
}

func logResponse(logger *zap.Logger, c *gin.Context, startTime time.Time) {
	dimeDiff := time.Since(startTime).Seconds()
	logger.Info("served request",
		zap.Int("status", c.Writer.Status()),
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
		zap.Float64("duration_seconds", dimeDiff))
	_ = logger.Sync()
}

func getIsolationID(c *gin.Context) (isolationID string, err error) {
	isolationID = c.Param(paramIsolationID)
	if isolationID == "" {
		return "", fmt.Errorf("%s cannot be empty", paramIsolationID)
	}
	if len(isolationID) < minIsolationIDLength || len(isolationID) > maxIsolationIDLength {
		return "", fmt.Errorf("isolationID must be between %d and %d characters",
			minIsolationIDLength, maxIsolationIDLength)

	}
	return isolationID, nil
}

func getCollectionID(c *gin.Context) (collectionID string, err error) {
	// Decode the collectionID from the URL
	collectionID, err = url.PathUnescape(c.Param(paramCollectionID))
	if err != nil {
		return "", fmt.Errorf("error decoding %s: %w", paramCollectionID, err)
	}
	if collectionID == "" {
		return "", fmt.Errorf("%s cannot be empty", paramCollectionID)
	}
	if len(collectionID) < minCollectionIDLength || len(collectionID) > maxCollectionIDLength {
		return "", fmt.Errorf("%s must be between %d and %d characters",
			paramCollectionID, minCollectionIDLength, maxCollectionIDLength)
	}
	return collectionID, nil
}

func getDocumentID(c *gin.Context) (documentID string, err error) {
	// Decode the documentID from the URL
	documentID, err = url.PathUnescape(c.Param(paramDocumentID))
	if err != nil {
		return "", fmt.Errorf("error decoding %s: %w", paramDocumentID, err)
	}
	if documentID == "" {
		return "", fmt.Errorf("%s cannot be empty", paramDocumentID)
	}
	return documentID, nil
}

func getFindDocumentsBodyFromRequest(c *gin.Context) (FindDocumentsRequestBody, error) {
	// Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return FindDocumentsRequestBody{}, fmt.Errorf("could not read request body: %w", err)
	}

	// Recreate body for further processing
	c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse request body into FindDocumentsRequestBody
	var findDocumentsBody FindDocumentsRequestBody
	err = c.BindJSON(&findDocumentsBody)
	if err != nil {
		return FindDocumentsRequestBody{}, fmt.Errorf("could not parse request body: %w", err)
	}

	return findDocumentsBody, nil
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
