/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package schema

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedVsSchemaManager struct {
	next VsSchemaManager
}

func (t *tracedVsSchemaManager) Load(ctx context.Context, isolationID, collectionID any) (VsSchemaManager, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			fmt.Sprintf("%s : schema-manager : %s", serviceName, "load-schema-info"),
			trace.WithAttributes(
				attribute.String("isolationID", fmt.Sprintf("%v", isolationID)),
				attribute.String("collectionID", fmt.Sprintf("%v", collectionID)),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.Load(ctx, isolationID, collectionID)
}

func (t *tracedVsSchemaManager) GetIsolations() []*Isolation {
	// No need to trace this method as it is not a database operation
	return t.next.GetIsolations()
}

func (t *tracedVsSchemaManager) GetCollections() []*Collection {
	// No need to trace this method as it is not a database operation
	return t.next.GetCollections()
}

func (t *tracedVsSchemaManager) GetIsolation(isolationID string) *Isolation {
	// No need to trace this method as it is not a database operation
	return t.next.GetIsolation(isolationID)
}

func (t *tracedVsSchemaManager) IsolationExists(isolationID string) bool {
	// No need to trace this method as it is not a database operation
	return t.next.IsolationExists(isolationID)
}

func (t *tracedVsSchemaManager) CollectionExists(isolationID, collectionID string) bool {
	// No need to trace this method as it is not a database operation
	return t.next.CollectionExists(isolationID, collectionID)
}
