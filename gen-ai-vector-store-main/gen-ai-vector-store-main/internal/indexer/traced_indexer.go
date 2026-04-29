/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedIndexer struct {
	next Indexer
}

func (t *tracedIndexer) getIsolationID() string {
	return t.next.getIsolationID()
}
func (t *tracedIndexer) getCollectionID() string {
	return t.next.getCollectionID()
}

func (t *tracedIndexer) Index(ctx context.Context, docID string, chunks []embedings.Chunk, attributes []attributes.Attribute, docMetadata *documents.DocumentMetadata, consistencyLevel string, extraAttributesKinds []string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			fmt.Sprintf("%s index-document", serviceName),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.Index(ctx, docID, chunks, attributes, docMetadata, consistencyLevel, extraAttributesKinds)
}

func (t *tracedIndexer) MoveDataToPermanentTablesTx(ctx context.Context, tx *sql.Tx, docID string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			fmt.Sprintf("%s move-emb-data", serviceName),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.MoveDataToPermanentTablesTx(ctx, tx, docID)
}
