/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package otel

import (
	"context"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"os"
)

func SetupTracing(ctx context.Context) (*trace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Return a no-op tracer provider or nil
		return trace.NewTracerProvider(), nil
	}
	// Initialize real tracing when endpoint is configured
	return initTracer(ctx)
}

func initTracer(ctx context.Context) (*trace.TracerProvider, error) {
	// OTEL will automatically configure itself based on the following environment variables:
	// - OTEL_EXPORTER_OTLP_ENDPOINT 	(i.e. 'http://tracing-agent.tracing:4317')
	// - OTEL_TRACES_SAMPLER 			(i.e. 'parentbased_traceidratio')
	// - OTEL_TRACES_SAMPLER_ARG 		(i.e. '1')
	// - OTEL_RESOURCE_ATTRIBUTES 		(i.e. 'service.name=genai-service')
	// The above environment variables are automatically injected by service-base.
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	// Propagators are not auto-configured based on OTEL_PROPAGATORS environment variable.
	// The service-base sets it to 'tracecontext,baggage,b3' by default.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
			b3.New()))
	return tp, nil
}
