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
)

// maxDecompressSize is the maximum number of bytes allowed when decompressing a response body.
// This protects against decompression bomb attacks (G110/CWE-409).
const maxDecompressSize = 100 * 1024 * 1024 // 100 MB

// GzipPlugin implements the heimdall.Plugin interface to handle gzip compression. Inspired in the Heimdall SAX plugin.
type GzipPlugin struct {
	compressPayload   bool
	decompressPayload bool
}

// NewGzipPlugin creates a new GzipPlugin instance.
func NewGzipPlugin() *GzipPlugin {
	useCompression := os.Getenv("USE_COMPRESSION") == "true"
	return &GzipPlugin{
		compressPayload:   useCompression,
		decompressPayload: true,
	}
}

// BeforeRequest modifies the request before it is sent.
func (p *GzipPlugin) OnRequestStart(req *http.Request) {
	if p.compressPayload && req.Body != nil {
		cl, compressedBody, err := compress(req.Body)
		if err != nil {
			return
		}
		req.Body = compressedBody
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", cl))
	}
}

// AfterResponse processes the response after it is received.
func (p *GzipPlugin) OnRequestEnd(req *http.Request, resp *http.Response) {
	if p.decompressPayload && resp.Header.Get("Content-Encoding") == "gzip" {
		cl, decompressedBody, err := decompress(resp.Body)
		if err != nil {
			return
		}
		resp.Body = decompressedBody
		resp.Header.Del("Content-Encoding")
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", cl))
	}
}

// OnError is called when an error occurs during the request.
func (p *GzipPlugin) OnError(_ *http.Request, _ error) {
	// No specific error handling for gzip plugin
}

func compress(src io.ReadCloser) (int, io.ReadCloser, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := io.Copy(gz, src)
	if err != nil {
		gz.Close()
		return 0, nil, fmt.Errorf("failed to gzip request body: %w", err)
	}
	if err := gz.Close(); err != nil {
		return 0, nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return buf.Len(), io.NopCloser(&buf), nil
}

func decompress(src io.ReadCloser) (int, io.ReadCloser, error) {
	defer src.Close()
	gr, err := gzip.NewReader(src)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, io.LimitReader(gr, maxDecompressSize+1))
	if err != nil {
		gr.Close()
		return 0, nil, fmt.Errorf("failed to decompress gzip body: %w", err)
	}
	if n > maxDecompressSize {
		gr.Close()
		return 0, nil, fmt.Errorf("decompressed response body exceeds %d byte limit", maxDecompressSize)
	}
	if err := gr.Close(); err != nil {
		return 0, nil, fmt.Errorf("failed to close gzip reader: %w", err)
	}
	return buf.Len(), io.NopCloser(buf), nil
}
