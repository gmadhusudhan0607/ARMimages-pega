/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package helpers

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"runtime"
	"strings"
)

const serviceName = "genai-vector-store"

func LibraryNameFromPkgPath() string {
	pc, _, _, _ := runtime.Caller(1)
	fullName := runtime.FuncForPC(pc).Name()
	if idx := strings.Index(fullName, ".("); idx != -1 {
		return fullName[:idx]
	}
	parts := strings.Split(fullName, ".")
	return strings.Join(parts[:len(parts)-1], ".")
}

func NewRootSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s  %s", serviceName, name)
	newCtx, bgSpan := otel.Tracer(LibraryNameFromPkgPath()).Start(
		ctx,
		spanName,
		trace.WithNewRoot(), // This creates a new root span instead of continuing current trace
		trace.WithSpanKind(trace.SpanKindClient),
	)
	return newCtx, bgSpan
}
