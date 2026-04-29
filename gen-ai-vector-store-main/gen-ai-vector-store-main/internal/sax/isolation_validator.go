/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	internalerrors "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/errors"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/go-sax"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	pathParamIsolationID      = "isolationID"
	attrValidationGUID        = attribute.Key("sax.validation.guid")
	attrValidationIsolationID = attribute.Key("sax.validation.isolation_id")
	attrIsolationID           = attribute.Key("sax.isolation_id")
)

// IsolationValidator handles the authorization of requests by checking GUID/IsolationID.
// This validator needs to run after the SAX authentication middleware.
// Currently, it cannot be coupled with the sax.Validator wrapper due to sax.Auth calling c.Next().
type IsolationValidator struct {
	tracer  trace.Tracer
	enabled bool
}

// NewIsolationValidator creates a new instance for checking isolation access.
func NewIsolationValidator() *IsolationValidator {
	enabled := true
	if helpers.IsIsolationIDVerificationDisabled() {
		enabled = false
	}
	return &IsolationValidator{
		tracer:  otel.Tracer(helpers.LibraryNameFromPkgPath()),
		enabled: enabled,
	}
}

// Validate checks if the authenticated user has access to the requested IsolationID.
func (v *IsolationValidator) Validate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !v.enabled {
			c.Next()
			return
		}

		spanCtx := trace.SpanContextFromContext(c.Request.Context())
		_, span := v.tracer.Start(c.Request.Context(), serviceName+": isolation_check",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithLinks(trace.Link{SpanContext: spanCtx}),
		)
		defer span.End()

		if err := v.verifyAccess(c, span); err != nil {
			v.handleError(c, span, err)
			return
		}

		span.SetStatus(codes.Ok, "Access granted")
		c.Next()
	}
}

func (v *IsolationValidator) verifyAccess(c *gin.Context, span trace.Span) error {
	claims, err := v.getClaimsFromContext(c)
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}

	reqID := c.Param(pathParamIsolationID)
	if reqID == "" {
		return internalerrors.ResponseError{
			Code:    http.StatusBadRequest,
			Message: "isolationID path parameter is missing",
		}
	}

	if claims.GUID == reqID {
		span.SetAttributes(
			attrValidationGUID.String(claims.GUID),
			attrIsolationID.String(reqID),
		)
		return nil
	}

	isolationID, err := v.extractIsolationIDFromToken(c.Request.Header.Get("Authorization"))
	if err != nil {
		span.RecordError(err)
		return internalerrors.ResponseError{
			Code:    http.StatusForbidden,
			Message: "access denied: unable to extract isolationID from token",
		}
	}

	if isolationID == reqID {
		span.SetAttributes(
			attrValidationIsolationID.String(reqID),
			attrIsolationID.String(reqID),
		)
		return nil
	}

	span.SetAttributes(attrIsolationID.String(reqID))
	if claims.GUID != "" {
		span.SetAttributes(attrValidationGUID.String(claims.GUID))
	}
	if isolationID != "" {
		span.SetAttributes(attrValidationIsolationID.String(isolationID))
	}

	return internalerrors.ResponseError{
		Code:    http.StatusForbidden,
		Message: "access denied: neither guid nor isolationID matches the request",
	}
}

func (v *IsolationValidator) getClaimsFromContext(c *gin.Context) (sax.Claims, error) {
	claimsData, exists := c.Get(contextKeyClaims)
	if !exists {
		return sax.Claims{}, fmt.Errorf("claims not found in context")
	}

	claims, ok := claimsData.(sax.Claims)
	if !ok {
		return sax.Claims{}, fmt.Errorf("invalid claims format: expected sax.Claims, got %T", claimsData)
	}
	return claims, nil
}

func (v *IsolationValidator) extractIsolationIDFromToken(authHeader string) (string, error) {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader || token == "" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	if val, ok := claims["isolationId"].(string); ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("isolationId not found in token claims")
}

func (v *IsolationValidator) handleError(c *gin.Context, span trace.Span, err error) {
	respErr := internalerrors.ToResponseError(err)

	span.SetStatus(codes.Error, respErr.Message)
	span.RecordError(err)

	c.AbortWithStatusJSON(respErr.Code, respErr)
}
