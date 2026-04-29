/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package isolations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedIsolationsManager struct {
	next IsoManager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedIsolationsManager) IsolationExists(ctx context.Context, isolationID string) (bool, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("check-isolation-exists"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.IsolationExists(ctx, isolationID)
}

func (t *tracedIsolationsManager) CreateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("create-isolation"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CreateIsolation(ctx, isolationID, maxStorageSize, pdcEndpointURL)
}

func (t *tracedIsolationsManager) UpdateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("update-isolation"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.UpdateIsolation(ctx, isolationID, maxStorageSize, pdcEndpointURL)
}

func (t *tracedIsolationsManager) GetIsolation(ctx context.Context, isolationID string) (*Details, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-isolation"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetIsolation(ctx, isolationID)
}

func (t *tracedIsolationsManager) GetIsolations(ctx context.Context) ([]*Details, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-isolations"),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetIsolations(ctx)
}

func (t *tracedIsolationsManager) DeleteIsolation(ctx context.Context, isolationID string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-isolation"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteIsolation(ctx, isolationID)
}

func (t *tracedIsolationsManager) GetIsolationProfiles(ctx context.Context, isolationID string) ([]EmbeddingProfile, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-isolation-profiles"),
			trace.WithAttributes(
				attribute.String("isolationID", isolationID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetIsolationProfiles(ctx, isolationID)
}
