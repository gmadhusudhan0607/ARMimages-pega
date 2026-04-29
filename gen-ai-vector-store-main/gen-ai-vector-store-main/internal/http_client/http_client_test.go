/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package http_client

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// mockRoundTripper simulates HTTP errors and timeouts
// errorType: "timeout" or "other"
type mockRoundTripper struct {
	calls     *int
	errorType string
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	(*m.calls)++
	if m.errorType == "timeout" {
		return nil, &timeoutError{}
	}
	return nil, &temporaryError{}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

type temporaryError struct{}

func (e *temporaryError) Error() string   { return "temporary error" }
func (e *temporaryError) Temporary() bool { return true }
func (e *temporaryError) Timeout() bool   { return false }

func init() {
	// Force retry logic for tests
	os.Setenv("SAX_CLIENT_DISABLED", "true")
}

func newTestHTTPClient(maxRetries int, timeout time.Duration, errorType string, calls *int) HTTPClient {
	cfg := HTTPClientConfig{
		Timeout:    timeout,
		MaxRetries: maxRetries,
	}
	client, _ := NewHTTPClientWithConfig(cfg)
	// Replace the transport with our mock
	if uc, ok := client.(*httpClient); ok {
		if uc.retryClient != nil && uc.retryClient.HTTPClient != nil {
			uc.retryClient.HTTPClient.Transport = &mockRoundTripper{calls: calls, errorType: errorType}
		}
	}
	return client
}

func TestHTTPClient_RetriesOnError(t *testing.T) {
	calls := 0
	maxRetries := 3
	client := newTestHTTPClient(maxRetries, time.Second, "other", &calls)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if calls != maxRetries+1 {
		t.Errorf("expected %d calls, got %d", maxRetries+1, calls)
	}
}

func TestHTTPClient_RetriesOnTimeout(t *testing.T) {
	calls := 0
	maxRetries := 2
	client := newTestHTTPClient(maxRetries, time.Millisecond, "timeout", &calls)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if calls != maxRetries+1 {
		t.Errorf("expected %d calls, got %d", maxRetries+1, calls)
	}
}

// Helper type for inline RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPClient_TimeoutThenSuccess(t *testing.T) {
	calls := 0
	maxRetries := 1
	client := newTestHTTPClient(maxRetries, time.Millisecond, "", &calls)
	// Override transport with custom logic
	if uc, ok := client.(*httpClient); ok {
		if uc.retryClient != nil && uc.retryClient.HTTPClient != nil {
			uc.retryClient.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return nil, &timeoutError{}
				}
				return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
			})
		}
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200 response, got: %v", resp)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestHTTPClient_DoesNotRetryOnContextCanceled(t *testing.T) {
	calls := 0
	maxRetries := 3
	client := newTestHTTPClient(maxRetries, time.Second, "", &calls)
	if uc, ok := client.(*httpClient); ok {
		if uc.retryClient != nil && uc.retryClient.HTTPClient != nil {
			uc.retryClient.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				return nil, context.Canceled
			})
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the context is already done
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// A pre-canceled context may prevent the request from reaching the transport.
	// If it does reach the transport and returns context.Canceled, it must not retry.
	if calls > 1 {
		t.Errorf("expected at most 1 transport call for context.Canceled, got %d", calls)
	}
}

func TestHTTPClient_RetriesOnContextDeadlineExceededWhenContextIsActive(t *testing.T) {
	calls := 0
	maxRetries := 3
	client := newTestHTTPClient(maxRetries, time.Second, "", &calls)
	if uc, ok := client.(*httpClient); ok {
		if uc.retryClient != nil && uc.retryClient.HTTPClient != nil {
			uc.retryClient.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				return nil, context.DeadlineExceeded
			})
		}
	}
	// The request context is still active, so a per-attempt DeadlineExceeded should be retried.
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != maxRetries+1 {
		t.Errorf("expected %d calls for retried context.DeadlineExceeded, got %d", maxRetries+1, calls)
	}
}


func TestSaxClientSingleton(t *testing.T) {
	os.Setenv("SAX_CLIENT_DISABLED", "false")
	os.Setenv("SAX_CLIENT_ID", "test-client-id")
	os.Setenv("SAX_CLIENT_SECRET", "dGVzdC1rZXk=") // base64 for 'test-key'
	os.Setenv("SAX_CLIENT_SCOPES", "scope1 scope2")
	os.Setenv("SAX_CLIENT_TOKEN_ENDPOINT", "https://example.com/token")
	os.Setenv("CLOUD_PROVIDER", "aws")
	os.Setenv("REGION", "us-east-1")

	// Reset singleton for test isolation
	saxClientOnce = sync.Once{}
	singleHTTPClientSax = nil
	saxClientInitErr = nil

	// Mock secret retrieval
	getAWSSaxClientPrivateKeyFunc = func(ctx context.Context, secretArn string) ([]byte, error) {
		return []byte("dummy-key"), nil
	}
	defer func() { getAWSSaxClientPrivateKeyFunc = getAWSSaxClientPrivateKey }()

	cfg := HTTPClientConfig{Timeout: 1 * time.Second, MaxRetries: 1}
	client1, err1 := NewHTTPClientWithConfig(cfg)
	client2, err2 := NewHTTPClientWithConfig(cfg)

	if err1 != nil || err2 != nil {
		t.Fatalf("Expected no error, got err1=%v, err2=%v", err1, err2)
	}
	if client1 != client2 {
		t.Errorf("Expected singleton SAX client, got different instances")
	}
}

func TestSaxClientSingletonUnknownProvider(t *testing.T) {
	os.Setenv("SAX_CLIENT_DISABLED", "false")
	os.Setenv("CLOUD_PROVIDER", "unknown")

	// Reset singleton for test isolation
	saxClientOnce = sync.Once{}
	singleHTTPClientSax = nil
	saxClientInitErr = nil

	cfg := HTTPClientConfig{Timeout: 1 * time.Second, MaxRetries: 1}
	_, err := NewHTTPClientWithConfig(cfg)
	if err == nil {
		t.Errorf("Expected error for unknown provider, got nil")
	}
}
