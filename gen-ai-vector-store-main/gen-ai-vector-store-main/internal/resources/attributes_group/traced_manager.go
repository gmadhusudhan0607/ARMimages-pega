/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package attributesgroup

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "attributes_group"

type tracedAttrGrpManager struct {
	next Manager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedAttrGrpManager) getIsolationID() string {
	return t.next.getIsolationID()
}

func (t *tracedAttrGrpManager) CreateTables(ctx context.Context) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("create-tables"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CreateTables(ctx)
}

func (t *tracedAttrGrpManager) DropTables(ctx context.Context) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("drop-tables"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DropTables(ctx)
}

func (t *tracedAttrGrpManager) CreateAttributesGroup(ctx context.Context, description string, attrs []string) (ag *AttributesGroup, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("create-attributes-group"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CreateAttributesGroup(ctx, description, attrs)
}

func (t *tracedAttrGrpManager) GetAttributesGroup(ctx context.Context, groupID string) (ag *AttributesGroup, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-attributes-group"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("groupID", groupID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetAttributesGroup(ctx, groupID)
}

func (t *tracedAttrGrpManager) GetAttributesGroupDescriptions(ctx context.Context) (agDescriptions map[string]string, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-attributes-group-descriptions"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetAttributesGroupDescriptions(ctx)
}

func (t *tracedAttrGrpManager) DeleteAttributesGroup(ctx context.Context, groupID string) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-attributes-group"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("groupID", groupID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteAttributesGroup(ctx, groupID)
}
