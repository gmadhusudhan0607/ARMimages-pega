/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/contexthelper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

const (
	isolationIDParamName    = "isolationID"
	collectionNameParamName = "collectionName"
	docIDParamName          = "documentID"
)

func ContextInfoMidleware(c *gin.Context) {
	ctx := c.Request.Context()

	// set data from endpoint
	if isoID := getIsolationID(c); isoID != "" {
		ctx = context.WithValue(ctx, contexthelper.IsolationIDKey, isoID)
	}
	if colID := getCollectionID(c); colID != "" {
		ctx = context.WithValue(ctx, contexthelper.CollectionIDKey, colID)
	}
	if docID := getDocumentID(c); docID != "" {
		ctx = context.WithValue(ctx, contexthelper.DocumentIDKey, docID)
	}

	// set traceID and spanID
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		ctx = context.WithValue(ctx, contexthelper.TraceIDKey, spanCtx.TraceID().String())
		ctx = context.WithValue(ctx, contexthelper.SpanIDKey, spanCtx.SpanID().String())
	}

	// set requestID
	ctx = context.WithValue(ctx, contexthelper.RequestIDKey, uuid.New().String())

	// set ctx back
	c.Request = c.Request.WithContext(ctx)
}

func getIsolationID(c *gin.Context) string {
	isolationID := c.Param(isolationIDParamName)

	return strings.ToLower(isolationID)
}

func getCollectionID(c *gin.Context) string {
	collectionID := c.Param(collectionNameParamName)

	return strings.ToLower(collectionID)
}

func getDocumentID(c *gin.Context) string {
	documentID := c.Param(docIDParamName)

	return documentID
}
