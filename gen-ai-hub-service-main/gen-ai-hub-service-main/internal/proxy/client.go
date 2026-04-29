/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gojektech/heimdall/v6/httpclient"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/pkg/heimdallgzip"
	"github.com/Pega-CloudEngineering/go-sax/heimdallsax"
)

type Client struct {
	cli     *httpclient.Client
	BaseURL string // scheme://host used as pool key, not the full request URL
}

// clientPool is a thread-safe pool of reusable proxy clients keyed by host.
// Reusing clients avoids creating a new http.Transport (and its connection pool)
// on every request, which would cause connections to accumulate and leak.
var clientPool sync.Map // map[string]*Client

// hostKey extracts scheme://host from a URL to use as the pool key.
// This aligns with how http.Transport pools connections (by host:port),
// so all requests to the same host share one client regardless of path.
func hostKey(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Scheme + "://" + u.Host
}

// GetClient returns a cached unauthenticated Client for the given URL's host.
// The pool is keyed by scheme://host so all requests to the same host share
// one client and connection pool. Use CallURL to send requests to specific paths.
func GetClient(rawURL string) *Client {
	key := hostKey(rawURL)
	if v, ok := clientPool.Load(key); ok {
		return v.(*Client)
	}
	c := NewClient(key)
	// LoadOrStore ensures only one client wins under concurrent first access.
	actual, _ := clientPool.LoadOrStore(key, c)
	return actual.(*Client)
}

// GetClientWithSaxAuth returns a cached SAX-authenticated Client for the given
// URL's host and SAX client identity. The cache key incorporates both the host
// and the clientId (cid) so that different SAX identities always get their own
// client. heimdallsax handles token refresh internally, so the cached client
// remains safe to reuse across requests.
func GetClientWithSaxAuth(rawURL, cid, end string, scp []string, pk []byte) *Client {
	cacheKey := "sax:" + cid + ":" + hostKey(rawURL)
	if v, ok := clientPool.Load(cacheKey); ok {
		return v.(*Client)
	}
	c := NewClientWithSaxAuth(rawURL, cid, end, scp, pk)
	actual, _ := clientPool.LoadOrStore(cacheKey, c)
	return actual.(*Client)
}

// NewClientWithSaxAuth creates a new proxy client with Sax Authentication headers.
// Prefer GetClientWithSaxAuth for long-lived usage to avoid transport proliferation.
func NewClientWithSaxAuth(url, cid, end string, scp []string, pk []byte) *Client {
	c := NewClient(url)
	p := &heimdallsax.Plugin{
		ClientID:      cid,
		PrivateKey:    pk,
		Scopes:        scp,
		TokenEndpoint: end,
	}
	c.cli.AddPlugin(p)

	return c
}

// NewClient creates a new proxy client.
// Prefer GetClient for long-lived usage to avoid transport proliferation.
func NewClient(url string) *Client {
	envTimeout := os.Getenv("MODEL_TIMEOUT_SECONDS")
	timeout, e := time.ParseDuration(envTimeout + "s")
	if e != nil {
		timeout = 1800 * time.Second
	}

	c := httpclient.NewClient(
		httpclient.WithHTTPTimeout(timeout),
	)

	gzipPlugin := heimdallgzip.NewGzipPlugin()
	c.AddPlugin(gzipPlugin)

	return &Client{cli: c, BaseURL: url}
}

// Call redirects a request to the client's stored URL.
// It copies headers and body from the original request and returns the upstream response.
func (c *Client) Call(ctx context.Context, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
	return c.CallURL(ctx, c.BaseURL, method, header, body)
}

// CallURL is like Call but sends the request to targetURL instead of the
// client's stored URL. This allows a cached Client (keyed by base URL) to
// make requests to varying URLs (e.g. with different query strings) while
// reusing the same underlying transport and connection pool.
func (c *Client) CallURL(ctx context.Context, targetURL, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
	tracer := otel.Tracer("github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy")
	ctx, span := tracer.Start(
		ctx,
		fmt.Sprintf("%s %s", method, targetURL),
		trace.WithAttributes(
			attribute.String("method", method),
			attribute.String("url", targetURL),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to read request body.")
			return nil, nil, fmt.Errorf("failed to read request body: %w", err)
		}
		body.Close()
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create a request.")
		return req, nil, fmt.Errorf("failed to create a request: %w", err)
	}

	if bodyBytes != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	req.Header = header

	resp, err := c.cli.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send a request.")
		return req, nil, fmt.Errorf("failed to send a request: %w", err)
	}
	span.SetAttributes(attribute.Int("statusCode", resp.StatusCode))

	return req, resp, nil
}

func AtomicUnauthenticatedCall(address string, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
	c := GetClient(address)
	return c.CallURL(context.Background(), address, method, header, body)
}

// TODO: need to simplify this file as a whole
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	tracer := otel.Tracer("github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy")
	ctx, span := tracer.Start(
		req.Context(),
		fmt.Sprintf("%s %s", req.Method, c.BaseURL),
		trace.WithAttributes(
			attribute.String("method", req.Method),
			attribute.String("url", c.BaseURL),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	req = req.WithContext(ctx)

	resp, err := c.cli.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send a request.")
		return nil, fmt.Errorf("failed to send a request: %w", err)
	}
	span.SetAttributes(attribute.Int("statusCode", resp.StatusCode))

	return resp, nil
}
