/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestTracerInitialization(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://tracing-agent.tracing:4317")
	t.Setenv("OTEL_TRACES_SAMPLER", "parentbased_traceidratio")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "1")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=genai-hub-service")
	ctx := context.Background()
	tp, err := InitTracer(ctx)
	assert.NoError(t, err)

	tracer := tp.Tracer("test")
	_, s := tracer.Start(ctx, "test span")

	assert.True(t, s.IsRecording())
	assert.Equal(t, 1, len(resource.Environment().Attributes()))
	assert.Equal(t, "genai-hub-service", resource.Environment().Attributes()[0].Value.AsString())
}
