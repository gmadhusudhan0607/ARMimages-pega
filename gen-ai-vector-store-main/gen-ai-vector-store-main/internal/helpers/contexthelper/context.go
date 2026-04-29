/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package contexthelper

type ContextKey string

const (
	IsolationIDKey  ContextKey = "isolationID"
	CollectionIDKey ContextKey = "collectionID"
	TraceIDKey      ContextKey = "traceID"
	SpanIDKey       ContextKey = "spanID"
	RequestIDKey    ContextKey = "requestID"
	DocumentIDKey   ContextKey = "documentID"
)
