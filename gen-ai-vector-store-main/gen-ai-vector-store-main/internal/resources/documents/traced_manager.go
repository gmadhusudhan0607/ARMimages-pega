/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracedDocumentsManager struct {
	next Manager
}

func spanName(name string) string {
	return fmt.Sprintf("%s : resources-manager : %s", serviceName, name)
}

func (t *tracedDocumentsManager) getIsolationID() string {
	return t.next.getIsolationID()
}

func (t *tracedDocumentsManager) getCollectionID() string {
	return t.next.getCollectionID()
}

func (t *tracedDocumentsManager) FindDocuments2(ctx context.Context, query *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("find-documents"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.FindDocuments2(ctx, query)
}

func (t *tracedDocumentsManager) FindDocuments4(ctx context.Context, query *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("find-documents-4"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.FindDocuments4(ctx, query)
}
func (t *tracedDocumentsManager) ListDocuments2(ctx context.Context, status string, filters []attributes.AttributeFilter) ([]Document, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("list-documents"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.ListDocuments2(ctx, status, filters)
}

func (t *tracedDocumentsManager) ListDocuments3(ctx context.Context, status string, filters []attributes.AttributeFilter) ([]Document, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("list-documents-3"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.ListDocuments3(ctx, status, filters)
}

func (t *tracedDocumentsManager) DocumentExists(ctx context.Context, docID string) (bool, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("document-exists"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DocumentExists(ctx, docID)
}

func (t *tracedDocumentsManager) GetDocument2(ctx context.Context, docID string) (Document, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-document"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetDocument2(ctx, docID)
}

func (t *tracedDocumentsManager) DeleteDocument2(ctx context.Context, docID string) (int64, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-document"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteDocument2(ctx, docID)
}

func (t *tracedDocumentsManager) DeleteDocumentsByFilters(ctx context.Context, filters []attributes.AttributeFilter) (int64, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-documents-by-filters"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteDocumentsByFilters(ctx, filters)
}

func (t *tracedDocumentsManager) DeleteDocumentsByFilters3(ctx context.Context, filters []attributes.AttributeFilter) (int64, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("delete-documents-by-filters-3"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.DeleteDocumentsByFilters3(ctx, filters)
}

func (t *tracedDocumentsManager) SetAttributes(ctx context.Context, docID string, attrs attributes.Attributes) error {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("set-attributes"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.SetAttributes(ctx, docID, attrs)
}

func (t *tracedDocumentsManager) GetChunksContent2(ctx context.Context, docID string) ([]embedings.Chunk, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-chunks-content"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetChunksContent2(ctx, docID)
}

func (t *tracedDocumentsManager) GetAttributeIDs(ctx context.Context, docID string) ([]int64, error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-attribute-ids"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", docID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetAttributeIDs(ctx, docID)
}

func (t *tracedDocumentsManager) CalculateDocumentStatus2(ctx context.Context, documentID string) (status, msg string, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("calculate-document-status"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", documentID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.CalculateDocumentStatus2(ctx, documentID)
}

func (t *tracedDocumentsManager) SetDocumentStatus(ctx context.Context, documentID, status, msg string) (err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("set-document-status"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
				attribute.String("documentID", documentID),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.SetDocumentStatus(ctx, documentID, status, msg)
}

func (t *tracedDocumentsManager) GetDocumentStatuses(ctx context.Context, status string, fields []string, filter attributes.Filter, cursor string, limit int) (documentStatuses []DocumentStatus, itemsTotal int, itemsLeft int, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-document-statuses"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetDocumentStatuses(ctx, status, fields, filter, cursor, limit)
}

func (t *tracedDocumentsManager) GetDocumentStatuses3(ctx context.Context, status string, fields []string, filter attributes.Filter, cursor string, limit int) (documentStatuses []DocumentStatus, itemsTotal int, itemsLeft int, err error) {
	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			spanName("get-document-statuses-3"),
			trace.WithAttributes(
				attribute.String("isolationID", t.getIsolationID()),
				attribute.String("collectionID", t.getCollectionID()),
			),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		defer span.End()
	}
	return t.next.GetDocumentStatuses3(ctx, status, fields, filter, cursor, limit)
}
