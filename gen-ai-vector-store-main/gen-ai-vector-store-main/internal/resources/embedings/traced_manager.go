/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedEmbeddingsManager struct {
	next EmbManager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedEmbeddingsManager) getIsolationID() string {
	return t.next.getIsolationID()
}

func (t *tracedEmbeddingsManager) getCollectionID() string {
	return t.next.getCollectionID()
}

func (t *tracedEmbeddingsManager) FindChunks2(ctx context.Context, query *QueryChunksRequest) ([]*Chunk, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("find-chunks"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.FindChunks2(ctx, query)
}

func (t *tracedEmbeddingsManager) FindChunks4(ctx context.Context, query *QueryChunksRequest) ([]*Chunk, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("find-chunks-4"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.FindChunks4(ctx, query)
}

func (t *tracedEmbeddingsManager) GetDocumentChunksPaginated(ctx context.Context, documentID string, cursor string, limit int) (chunks []*Chunk, itemsTotal, itemsLeft int, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-document-chunks"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", documentID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetDocumentChunksPaginated(ctx, documentID, cursor, limit)
}
