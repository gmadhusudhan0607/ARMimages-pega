/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"errors"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/go-sax/ginsax"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName           = "genai-vector-store"
	attrValidationSuccess = attribute.Key("validation_success")
	contextKeyClaims      = "saxClaims"
)

// Config holds the configuration for the SAX validator.
type Config struct {
	Audience     string
	Issuer       string
	JWKSEndpoint string
}

// Validator defines the interface for request validation middleware.
type Validator interface {
	ValidateRequest(scopes ...string) gin.HandlerFunc
}

// validator handles the authentication of request tokens.
type validator struct {
	config Config
	tracer trace.Tracer
}

// New creates a new Validator for Authentication.
func New(cfg Config) (Validator, error) {
	if cfg.Audience == "" || cfg.Issuer == "" || cfg.JWKSEndpoint == "" {
		return nil, errors.New("sax: audience, issuer, and jwksEndpoint must be provided")
	}

	return &validator{
		config: cfg,
		tracer: otel.Tracer(helpers.LibraryNameFromPkgPath()),
	}, nil
}

func (v *validator) ValidateRequest(scopes ...string) gin.HandlerFunc {
	saxHandler := ginsax.Auth(ginsax.Config{
		Expected: ginsax.Expected{
			Issuer:   v.config.Issuer,
			Audience: v.config.Audience,
			Scopes:   scopes,
		},
		ContextKey:   contextKeyClaims,
		JWKSEndpoint: v.config.JWKSEndpoint,
	})

	return func(c *gin.Context) {
		// Extract the original trace context
		originalCtx := c.Request.Context()
		spanCtx := trace.SpanContextFromContext(originalCtx)

		// Create a new context with the same trace ID, but as a new root span
		// This preserves the trace ID for correlation while starting a new hierarchy segment,
		// We use originalCtx as the base to retain all existing context values while
		// applying a new span context that inherits just the trace ID but not the parent span ID
		ctx := trace.ContextWithSpanContext(originalCtx, spanCtx)

		ctx, span := v.tracer.Start(ctx,
			serviceName+": token_validation", // Explicit naming shows this span's specific purpose
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithLinks(trace.Link{SpanContext: spanCtx}), // Link to original trace for correlation)
		)
		defer span.End()

		c.Set("token_validation_span", span)

		c.Request = c.Request.WithContext(ctx)

		saxHandler(c)
		if c.IsAborted() {
			span.SetAttributes(attrValidationSuccess.Bool(false))
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last().Err)
			}
			return
		}

		span.SetStatus(codes.Ok, "Validation successful")
		span.SetAttributes(attrValidationSuccess.Bool(true))

		c.Request = c.Request.WithContext(originalCtx)
		c.Next()
	}
}
