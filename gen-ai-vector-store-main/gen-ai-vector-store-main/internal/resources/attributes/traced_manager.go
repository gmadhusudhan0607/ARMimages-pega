/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedAttributesManager struct {
	next Manager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedAttributesManager) getIsolationID() string {
	return t.next.getIsolationID()
}

func (t *tracedAttributesManager) getCollectionID() string {
	return t.next.getCollectionID()
}

func (t *tracedAttributesManager) UpsertAttributes2(ctx context.Context, attrs []Attribute, extraAttributesKinds []string) (attrItemIds []int64, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("upsert-attributes"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.UpsertAttributes2(ctx, attrs, extraAttributesKinds)
}

func (t *tracedAttributesManager) GetAttributesByIDs(ctx context.Context, attrIDs []int64) ([]Attribute, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-attributes-by-ids"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetAttributesByIDs(ctx, attrIDs)
}

func (t *tracedAttributesManager) GetAttributesIDs(ctx context.Context, attrs []Attribute) (attrIDs []int64, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-attributes-ids"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetAttributesIDs(ctx, attrs)
}

func (t *tracedAttributesManager) FindAttributes(ctx context.Context, names []string) ([]Attribute, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("find-attributes"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.FindAttributes(ctx, names)
}

func (t *tracedAttributesManager) GetEmbeddingAttributes(ctx context.Context, docId, embID string, filterNames []string) ([]Attribute, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-embedding-attributes"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docId),
				attribute.String("embeddingID", embID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetEmbeddingAttributes(ctx, docId, embID, filterNames)
}

func (t *tracedAttributesManager) GetEmbeddingAttributesProcessing(ctx context.Context, docId, embID string, filterNames []string) ([]Attribute, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-embedding-attributes-processing"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docId),
				attribute.String("embeddingID", embID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetEmbeddingAttributesProcessing(ctx, docId, embID, filterNames)
}
