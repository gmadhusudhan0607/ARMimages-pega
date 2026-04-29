/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package collections

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedCollectionsManager struct {
	next ColManager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedCollectionsManager) getIsolationID() string {
	return t.next.getIsolationID()
}

func (t *tracedCollectionsManager) CollectionExists(ctx context.Context, collectionID string) (exists bool, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("check-collection-exists"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", collectionID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CollectionExists(ctx, collectionID)
}

func (t *tracedCollectionsManager) CreateCollection(ctx context.Context, collectionID string) (*Collection, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("create-collection"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", collectionID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CreateCollection(ctx, collectionID)
}

func (t *tracedCollectionsManager) GetCollections(ctx context.Context) ([]Collection, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-collections"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetCollections(ctx)
}

func (t *tracedCollectionsManager) GetCollection(ctx context.Context, collectionID string) (*Collection, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-collection"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", collectionID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetCollection(ctx, collectionID)
}

func (t *tracedCollectionsManager) DeleteCollection(ctx context.Context, collectionID string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-collection"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", collectionID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteCollection(ctx, collectionID)
}
