/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package http_client

import (
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracedHTTPClient wraps an HTTPClient and adds tracing to all requests
type TracedHTTPClient struct {
	client HTTPClient
	prop   propagation.TextMapPropagator
}

// NewTracedHTTPClient creates a new TracedHTTPClient
func NewTracedHTTPClient(client HTTPClient) (*TracedHTTPClient, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	return &TracedHTTPClient{
		client: client,
		prop:   otel.GetTextMapPropagator(),
	}, nil
}

// Do executes an HTTP request with tracing
func (c *TracedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	var span trace.Span
	if trace.SpanFromContext(ctx).SpanContext().IsValid() {
		ctx, span = otel.Tracer(helpers.LibraryNameFromPkgPath()).Start(
			ctx,
			fmt.Sprintf("%s %s %s", serviceName, req.Method, req.URL.String()),
			trace.WithAttributes(
				attribute.String("component", "http-client"),
				attribute.String("http.method", req.Method),
				attribute.String("http.url", req.URL.String()),
			),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		// Inject the trace context into the request headers
		c.prop.Inject(ctx, propagation.HeaderCarrier(req.Header))
	}

	// Execute the actual request with the updated context
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)

	// Add response status to span
	if span != nil {
		if resp != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		}
		if err != nil {
			span.RecordError(err)
		}
	}
	return resp, err
}
