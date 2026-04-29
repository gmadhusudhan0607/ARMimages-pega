/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestHostKey(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"full URL", "https://api.openai.com/v1/chat/completions?api-version=2024", "https://api.openai.com"},
		{"with port", "http://localhost:8080/api/v1", "http://localhost:8080"},
		{"host only", "https://example.com", "https://example.com"},
		{"with path and query", "https://host.com:443/a/b/c?x=1&y=2", "https://host.com:443"},
		{"invalid URL", "://bad", "://bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hostKey(tt.url))
		})
	}
}

func TestClient_Calls(t *testing.T) {

	as := testutils.NewLocalAuthServer()
	defer as.Close()

	type saxParams struct {
		cid string
		end string
		scp []string
		pk  []byte
	}
	tests := []struct {
		name      string
		method    string
		headers   http.Header
		headerKey string
		wantErr   error
		sax       *saxParams
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			headers: http.Header{
				"Header-Key": []string{"Value", "Diff-Value"},
			},
			headerKey: "Header-Key",
		},
		{
			name:   "Success with sax",
			method: http.MethodPost,
			headers: http.Header{
				"Header-Key": []string{"Value", "Diff-Value"},
			},
			headerKey: "Header-Key",
			sax: &saxParams{
				cid: "cid",
				end: as.URL,
				scp: []string{"scp"},
				pk:  testutils.GeneratePrivateKey(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			var c *Client
			if tt.sax == nil {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check if headers are copied
					assert.Equal(t, tt.headers.Values(tt.headerKey), r.Header.Values(tt.headerKey))
					assert.Equal(t, tt.method, r.Method)
					w.WriteHeader(http.StatusOK)
				}))
				c = NewClient(ts.URL)
			} else {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check if headers are copied
					assert.Equal(t, tt.headers.Values(tt.headerKey), r.Header.Values(tt.headerKey))
					assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
					assert.Equal(t, tt.method, r.Method)
					w.WriteHeader(http.StatusOK)
				}))
				c = NewClientWithSaxAuth(ts.URL, tt.sax.cid, tt.sax.end, tt.sax.scp, tt.sax.pk)
			}
			_, resp, err := c.Call(context.Background(), tt.method, tt.headers, nil)
			ts.Close()
			ts = nil
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers and method
		assert.Equal(t, []string{"HeaderValue"}, r.Header.Values("Test-Header"))
		//assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	header := http.Header{}
	header.Set("Test-Header", "HeaderValue")

	tests := []struct {
		name          string
		addr          string
		method        string
		errorExpected bool
	}{
		{
			name:          "Valid GET request",
			addr:          ts.URL,
			method:        http.MethodGet,
			errorExpected: false,
		},
		{
			name:          "Invalid request",
			addr:          ts.URL,
			method:        "]]]]]",
			errorExpected: true,
		},
		{
			name:          "Request error",
			addr:          ts.URL + "1",
			method:        http.MethodGet,
			errorExpected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, resp, err := AtomicUnauthenticatedCall(test.addr, test.method, header, nil)
			if test.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestGzipDecompression(t *testing.T) {
	expected := "hello gzip world"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(expected))
		assert.NoError(t, err)
		gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(buf.Bytes())
		assert.NoError(t, err)
	}))
	defer ts.Close()

	header := http.Header{}
	header.Set("Test-Header", "HeaderValue")

	c := NewClient(ts.URL)
	_, resp, err := c.Call(context.Background(), http.MethodGet, header, nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Content-Encoding should be removed after decompression
	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(body))
}

func TestDo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"HeaderValue"}, r.Header.Values("Test-Header"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	h := http.Header{}
	h.Set("Test-Header", "HeaderValue")

	tests := []struct {
		name          string
		addr          string
		method        string
		errorExpected bool
	}{
		{
			name:          "Valid GET request",
			addr:          ts.URL,
			method:        http.MethodGet,
			errorExpected: false,
		},
		{
			name:          "Invalid request",
			addr:          ts.URL,
			method:        "]]]]]",
			errorExpected: true,
		},
		{
			name:          "Request error",
			addr:          ts.URL + "1",
			method:        http.MethodGet,
			errorExpected: true,
		},
	}

	c := NewClient(ts.URL)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, e1 := http.NewRequest(test.method, test.addr, nil)
			var err error
			var resp *http.Response
			if e1 == nil {
				req.Header = h
				resp, err = c.Do(req)
			} else {
				err = e1
			}

			if test.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}
