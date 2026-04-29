/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/ginctx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy"

	"gopkg.in/yaml.v3"
)

const (
	IsolationIdParamName = "isolationId"
	ModelIdParamName     = "modelId"
	BuddyIdParamName     = "buddyId"

	SaxAuthDisabled = false

	PrivateModelFilePath   = "/private-model-config"
	PrivateModelFilePrefix = "genai_private_model"
)

func GetBodyBytes(c *gin.Context) []byte {
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
	}
	return body
}

// CallTarget calls target URL of the AWS/GCP/Azure/buddy API endpoint.
func CallTarget(c *gin.Context, ctx context.Context, url string, saxAuthEnabled bool) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	reqID := c.Request.Header.Get("pega-genai-service-request-id")
	c.Set(ginctx.ModelURLContextKey, url)

	var bodyReader io.ReadCloser
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			l.Errorf("Failed to read request body: %s", err)
			c.JSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to read request body: %s", err),
			})
			return
		}
		bodyReader = io.NopCloser(strings.NewReader(string(bodyBytes)))
	}

	var client *proxy.Client
	if saxAuthEnabled {
		saxConfig := cntx.GetSaxClientConfigFromContext(ctx)
		if saxConfig == nil {
			l.Errorf("SAX authentication enabled but no SAX configuration found in context")
			c.JSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    "SAX authentication enabled but no SAX configuration found in context",
			})
			return
		}
		pk, err := saxConfig.GetPrivateKeyPEMFormat()
		if err != nil {
			l.Errorf("Failed to get private key PEM format: %s", err)
			c.JSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			})
			return
		}
		scp := strings.Split(saxConfig.Scopes, " ")
		client = proxy.GetClientWithSaxAuth(url, saxConfig.ClientId, saxConfig.TokenEndpoint, scp, pk)
		l.Debugf("[%s] SAX client created for %s", reqID, url)
	} else {
		client = proxy.GetClient(url)
		l.Debugf("[%s] Client created for %s", reqID, url)
	}
	l.Debugf("[%s] Calling %s %s", reqID, c.Request.Method, url)
	modelCallStart := time.Now()
	_, resp, err := client.CallURL(c.Request.Context(), url, c.Request.Method, c.Request.Header.Clone(), bodyReader)
	responseReceivedAt := time.Now()
	modelCallDuration := int(time.Since(modelCallStart).Milliseconds())
	if err != nil {
		l.Errorf("[%s] Failed to call %s: %s", reqID, url, err)
		c.JSON(http.StatusInternalServerError, RespErr{
			StatusCode: http.StatusInternalServerError,
			Message:    "upstream request failed",
		})
		return
	}
	defer resp.Body.Close()
	l.Debugf("[%s] Received %d from %s in %dms", reqID, resp.StatusCode, url, modelCallDuration)

	if isGoogleDeploymentRequest(c.Request.URL.Path) && resp.StatusCode >= http.StatusBadRequest {
		logGoogleErrorResponse(ctx, reqID, resp, modelCallStart, responseReceivedAt)
	}

	c.Status(resp.StatusCode)
	for name, values := range resp.Header {
		for _, value := range values {
			c.Header(name, value)
		}
	}
	middleware.SetModelCallDuration(c, modelCallDuration)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		l.Error("Error during copy of response body: %s", err)
	}
}

func isGoogleDeploymentRequest(path string) bool {
	return strings.HasPrefix(path, "/google/deployments/")
}

// logGoogleErrorResponse logs Google upstream error details with the trace context and UTC timestamps.
// It reads and restores the response body so it can still be proxied to the client unchanged.
func logGoogleErrorResponse(ctx context.Context, reqID string, resp *http.Response, callStartTime time.Time, callResponseTime time.Time) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	traceContext := resp.Header.Get("X-Cloud-Trace-Context")

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	payload := escapePayload(string(bodyBytes))
	if err != nil {
		payload = escapePayload(string(bodyBytes)) + " (truncated: " + err.Error() + ")"
	}

	l.Errorf("[%s] Google upstream error status=%d x_cloud_trace_context=%s call_start_time=%s call_response_time=%s payload=%s",
		reqID, resp.StatusCode, traceContext, callStartTime.UTC().Format(time.RFC3339Nano), callResponseTime.UTC().Format(time.RFC3339Nano), payload)
}

// escapePayload replaces newlines and carriage returns in the response body so the log entry stays on a single line.
func escapePayload(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\\r\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// GetEntityEndpointUrl returns Entity URL : model or buddy
func GetEntityEndpointUrl(urlBase, endpointSuffix string) string {
	return urlBase + endpointSuffix
}

// setApiVersionParam adds the service-governed api-version parameter to Azure OpenAI URLs.
// This ensures all OpenAI API calls use a consistent, service-controlled API version
// regardless of what the client provided.
func setApiVersionParam(url string) string {
	const governedApiVersion = "2024-10-21"

	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + "api-version=" + governedApiVersion
}

// reads the models
func RetrieveMappingImpl(ctx context.Context, fileName string) (mapping *Mapping, err error) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	filePath, err := filepath.Abs(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute file path for %s: %w", fileName, err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	mapping = &Mapping{}
	err = yaml.Unmarshal(content, &mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s: %w", fileName, err)
	}

	l.Infof("mapping :%+v\n", mapping)

	return mapping, nil
}

func CallTargetWithResponse(ctx context.Context, url string, method string, headers http.Header, body io.ReadCloser, saxAuthEnabled bool) (*http.Response, error) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	// Read the body once and create a new reader to avoid data races
	var bodyReader io.ReadCloser
	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		bodyReader = io.NopCloser(strings.NewReader(string(bodyBytes)))
		// Close the original body
		body.Close()
	}

	// If headers are not provided, try to extract from gin.Context
	if headers == nil {
		headers = http.Header{}
	}

	if ginCtx := cntx.GetGinContext(ctx); ginCtx != nil {
		for key, values := range ginCtx.Request.Header {
			for _, value := range values {
				headers.Add(key, value)
			}
		}
		l.Debugf("Copied headers from gin.Context: %v", headers)
	} else {
		l.Warn("No gin.Context found in context — cannot copy request headers")
	}

	// Log the final headers that will be sent
	l.Debugf("Final headers used in request to %s: %v", url, headers)

	var client *proxy.Client
	if saxAuthEnabled {
		saxConfig := cntx.GetSaxClientConfigFromContext(ctx)
		if saxConfig == nil {
			return nil, fmt.Errorf("SAX authentication enabled but no SAX configuration found in context")
		}
		pk, err := saxConfig.GetPrivateKeyPEMFormat()
		if err != nil {
			return nil, fmt.Errorf("failed to get private key PEM format: %w", err)
		}
		scp := strings.Split(saxConfig.Scopes, " ")
		client = proxy.GetClientWithSaxAuth(url, saxConfig.ClientId, saxConfig.TokenEndpoint, scp, pk)
		l.Debugf("SAX client created for %s", url)
	} else {
		client = proxy.GetClient(url)
		l.Debugf("Client created for %s", url)
	}
	l.Debugf("Calling %s %s", method, url)
	_, resp, err := client.CallURL(ctx, url, method, headers, bodyReader)
	if err != nil {
		l.Errorf("Request to %s failed: %v", url, err)
		return nil, fmt.Errorf("failed to call %s: %w", url, err)
	}
	l.Debugf("Received %s from %s", resp.Status, url)
	return resp, nil
}
