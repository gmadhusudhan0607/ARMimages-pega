/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package heimdallgzip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipPluginCompression(t *testing.T) {
	// Set the environment variable to enable compression
	os.Setenv("USE_COMPRESSION", "true")
	defer os.Unsetenv("USE_COMPRESSION")

	// Create a new GzipPlugin
	plugin := NewGzipPlugin()

	// Create a test request with a body
	body := "test request body"
	req, err := http.NewRequest("POST", "http://example.com", io.NopCloser(bytes.NewBufferString(body)))
	assert.NoError(t, err)

	// Apply the BeforeRequest method
	plugin.OnRequestStart(req)

	// Check if the Content-Encoding header is set to gzip
	assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))

	// Check if the body is compressed
	gzipReader, err := gzip.NewReader(req.Body)
	assert.NoError(t, err)
	decompressedBody, err := io.ReadAll(gzipReader)
	assert.NoError(t, err)
	assert.Equal(t, body, string(decompressedBody))
}

func TestGzipPluginDecompression(t *testing.T) {
	// Create a new GzipPlugin
	plugin := NewGzipPlugin()

	// Create a compressed response body
	body := "test response body! test request body! test request body! test request body! test request body! test request body! test request body! "
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write([]byte(body))
	assert.NoError(t, err)
	gzipWriter.Close()
	compressedBody := buf.Bytes()

	// Create a test response with a compressed body
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
			"Content-Length":   []string{fmt.Sprintf("%d", len(compressedBody))},
		},
		Body: io.NopCloser(bytes.NewReader(compressedBody)),
	}

	// Apply the AfterResponse method
	plugin.OnRequestEnd(nil, resp)
	assert.NoError(t, err)

	// Content length should have been recalculated
	assert.NotEqual(t, resp.Header.Get("Content-Length"), strconv.Itoa(len(compressedBody)))
	assert.Equal(t, resp.Header.Get("Content-Length"), strconv.Itoa(len(body)))

	// Check if the Content-Encoding header is removed
	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	// Check if the body is decompressed
	decompressedBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, string(decompressedBody))
}

func TestGzipPluginNoCompression(t *testing.T) {
	// Create a new GzipPlugin
	plugin := NewGzipPlugin()

	// Create a test request with a body
	body := "test request body!"
	req, err := http.NewRequest("POST", "http://example.com", io.NopCloser(bytes.NewBufferString(body)))
	assert.NoError(t, err)

	// Apply the BeforeRequest method
	plugin.OnRequestStart(req)
	assert.NoError(t, err)

	// Check if the Content-Encoding header is not set
	assert.Empty(t, req.Header.Get("Content-Encoding"))

	// Check if the body is not compressed
	readBody, err := io.ReadAll(req.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, string(readBody))
}

func TestGzipPluginDecompressionError(t *testing.T) {
	plugin := NewGzipPlugin()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte("Invalid gzip data"))),
	}

	plugin.OnRequestEnd(nil, resp)
	assert.Contains(t, resp.Header, "Content-Encoding")
}

func TestGzipPluginCompressionError(t *testing.T) {
	os.Setenv("USE_COMPRESSION", "true")
	defer os.Unsetenv("USE_COMPRESSION")

	plugin := NewGzipPlugin()

	req, err := http.NewRequest("POST", "http://example.com", &ErrorReader{})
	assert.NoError(t, err)

	plugin.OnRequestStart(req)
	assert.NotContains(t, req.Header, "Content-Encoding")
}

type ErrorReader struct{}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated error")
}

func (e *ErrorReader) Close() error {
	return nil
}
