/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

// Package runner provides shared helpers for live tests.
package runner

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"
)

// AssertResponseHeaders validates that the HTTP response contains the required headers:
//   - Content-Type is present and non-empty
//   - Content-Length is present and non-empty
func AssertResponseHeaders(t *testing.T, resp *http.Response) {
	t.Helper()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		t.Fatal("Response missing Content-Type header")
	}

	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		t.Fatal("Response missing Content-Length header")
	}
}

// AssertConverseResponse validates a Bedrock converse HTTP response:
//   - HTTP 200 status
//   - Content-Type and Content-Length headers present
//   - Valid JSON body
//   - stopReason == "end_turn"
//
// The body parameter should contain the already-read response body bytes.
func AssertConverseResponse(t *testing.T, resp *http.Response, body []byte) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK but got %d.\nResponse body: %s", resp.StatusCode, string(body))
	}

	AssertResponseHeaders(t, resp)

	bodyStr := string(body)
	if !gjson.Valid(bodyStr) {
		t.Fatalf("Response body should be valid JSON: %s", bodyStr)
	}

	stopReason := gjson.Get(bodyStr, "stopReason")
	if !stopReason.Exists() {
		t.Fatalf("Response should contain 'stopReason' field.\nFull response: %s", bodyStr)
	}
	if stopReason.String() != "end_turn" {
		t.Fatalf("stopReason must be 'end_turn', got: %q.\nFull response: %s", stopReason.String(), bodyStr)
	}
}

// AssertChatCompletionResponse validates a chat completion HTTP response:
//   - HTTP 200 status
//   - Content-Type and Content-Length headers present
//   - Valid JSON body
//   - finish_reason == "stop" in the first choice
//
// The body parameter should contain the already-read response body bytes.
func AssertChatCompletionResponse(t *testing.T, resp *http.Response, body []byte) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK but got %d.\nResponse body: %s", resp.StatusCode, string(body))
	}

	AssertResponseHeaders(t, resp)

	bodyStr := string(body)
	if !gjson.Valid(bodyStr) {
		t.Fatalf("Response body should be valid JSON: %s", bodyStr)
	}

	choices := gjson.Get(bodyStr, "choices")
	if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
		t.Fatalf("Response should contain non-empty 'choices' array.\nFull response: %s", bodyStr)
	}

	finishReason := gjson.Get(bodyStr, "choices.0.finish_reason")
	if finishReason.String() != "stop" {
		t.Fatalf("finish_reason must be 'stop', got: %q.\nFull response: %s", finishReason.String(), bodyStr)
	}
}

// AssertEmbeddingResponse validates an embeddings HTTP response:
//   - HTTP 200 status
//   - Content-Type and Content-Length headers present
//   - Valid JSON body
//   - Contains embedding data in one of these formats:
//   - OpenAI format: object == "list" with "data" array containing embedding objects
//   - Bedrock Titan format: direct "embedding" array with "inputTextTokenCount"
//   - Bedrock Nova format: "embeddings" array containing objects with "embedding" arrays
//
// The body parameter should contain the already-read response body bytes.
func AssertEmbeddingResponse(t *testing.T, resp *http.Response, body []byte) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK but got %d.\nResponse body: %s", resp.StatusCode, string(body))
	}

	AssertResponseHeaders(t, resp)

	bodyStr := string(body)
	if !gjson.Valid(bodyStr) {
		t.Fatalf("Response body should be valid JSON: %s", bodyStr)
	}

	// Check for Bedrock Titan format (direct "embedding" array)
	titanEmbedding := gjson.Get(bodyStr, "embedding")
	if titanEmbedding.Exists() && titanEmbedding.IsArray() && len(titanEmbedding.Array()) > 0 {
		return // Bedrock Titan format is valid
	}

	// Check for Bedrock Nova format ("embeddings" array with objects containing "embedding")
	novaEmbedding := gjson.Get(bodyStr, "embeddings.0.embedding")
	if novaEmbedding.Exists() && novaEmbedding.IsArray() && len(novaEmbedding.Array()) > 0 {
		return // Bedrock Nova format is valid
	}

	// Check for OpenAI format (object: "list" with "data" array)
	obj := gjson.Get(bodyStr, "object")
	if obj.String() == "list" {
		openaiEmbedding := gjson.Get(bodyStr, "data.0.embedding")
		if openaiEmbedding.Exists() && openaiEmbedding.IsArray() && len(openaiEmbedding.Array()) > 0 {
			return // OpenAI format is valid
		}
		t.Fatalf("OpenAI format requires non-empty 'data.0.embedding' array.\nFull response: %s", bodyStr)
	}

	t.Fatalf("Response should have 'embedding' array, 'embeddings.0.embedding' array, or object='list' with 'data.0.embedding'.\nFull response: %s", bodyStr)
}

// minStreamDuration is the minimum expected duration for streaming responses with many chunks.
// If more than minChunksForStreamValidation chunks arrive faster than this, it indicates
// the response may not be truly streaming (e.g., buffered and sent all at once).
const minStreamDuration = 50 * time.Millisecond

// minChunksForStreamValidation is the minimum number of chunks required before
// stream duration validation is applied.
const minChunksForStreamValidation = 5

// readCounter wraps an io.Reader and counts the number of Read() calls.
type readCounter struct {
	reader io.Reader
	count  int
}

func (r *readCounter) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.count++
	}
	return n, err
}

// timedChunk holds a raw streaming chunk JSON with its arrival time.
type timedChunk struct {
	json       string
	receivedAt time.Time
}

// AssertStreamingChatCompletionResponse validates a streaming chat completion SSE response:
//   - HTTP 200 status
//   - Content-Type contains "text/event-stream"
//   - Body contains SSE "data:" lines with valid JSON chunks
//   - At least 1 logical chunk received
//   - All non-final chunks have finish_reason == null (JSON null)
//   - Last chunk has finish_reason == "stop"
//   - Stream terminates with "data: [DONE]"
//
// Logs both logical chunks (SSE events) and data chunks (HTTP reads) for visibility.
// The resp.Body is consumed by this function via streaming read.
func AssertStreamingChatCompletionResponse(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected HTTP 200 OK but got %d.\nResponse body: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Expected Content-Type to contain 'text/event-stream', got: %q", contentType)
	}

	// Wrap response body with a read counter to track data chunks
	counter := &readCounter{reader: resp.Body}
	scanner := bufio.NewScanner(counter)

	var chunks []timedChunk
	gotDone := false

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and SSE comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream terminator
		if data == "[DONE]" {
			gotDone = true
			break
		}

		// Validate JSON and record arrival time
		if !gjson.Valid(data) {
			t.Fatalf("SSE data chunk is not valid JSON: %s", data)
		}
		chunks = append(chunks, timedChunk{json: data, receivedAt: time.Now()})
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading SSE stream: %v", err)
	}

	logicalChunks := len(chunks)
	dataChunks := counter.count

	if logicalChunks == 0 {
		t.Fatal("Expected at least 1 SSE data chunk, got 0")
	}

	if !gotDone {
		t.Fatal("Streaming response did not end with 'data: [DONE]'")
	}

	// Validate all non-final chunks have finish_reason == null (JSON null)
	for i := 0; i < len(chunks)-1; i++ {
		chunk := chunks[i].json
		choices := gjson.Get(chunk, "choices")
		if !choices.Exists() || len(choices.Array()) == 0 {
			continue // some intermediate chunks may not have choices (e.g. role-only chunks)
		}
		finishReason := gjson.Get(chunk, "choices.0.finish_reason")
		// In JSON, null values exist but have Type == gjson.Null
		if finishReason.Exists() && finishReason.Type != gjson.Null {
			t.Fatalf("Chunk %d (non-final) must have finish_reason: null, got: %q.\nChunk: %s",
				i, finishReason.String(), chunk)
		}
	}

	// Validate the last chunk has finish_reason == "stop"
	lastChunk := chunks[len(chunks)-1].json
	choices := gjson.Get(lastChunk, "choices")
	if !choices.Exists() || len(choices.Array()) == 0 {
		t.Fatalf("Last chunk should contain 'choices' array.\nLast chunk: %s", lastChunk)
	}

	finishReason := gjson.Get(lastChunk, "choices.0.finish_reason")
	if finishReason.String() != "stop" {
		t.Fatalf("Last chunk finish_reason must be 'stop', got: %q.\nLast chunk: %s",
			finishReason.String(), lastChunk)
	}

	// Validate streaming duration for responses with many chunks
	if len(chunks) > 1 {
		firstTime := chunks[0].receivedAt
		lastTime := chunks[len(chunks)-1].receivedAt
		streamDuration := lastTime.Sub(firstTime)

		// If many chunks arrived faster than minStreamDuration, streaming may not be working
		if len(chunks) > minChunksForStreamValidation && streamDuration < minStreamDuration {
			t.Fatalf("Streaming response appears buffered: %d chunks arrived in %v (expected > %v). "+
				"Response may not be truly streaming.", len(chunks), streamDuration, minStreamDuration)
		}

		logVerbosef("  Streaming validated: %d logical chunks, %d data chunks over %v\n",
			logicalChunks, dataChunks, streamDuration)
	} else {
		logVerbosef("  Streaming validated: %d logical chunk, %d data chunks (short response)\n",
			logicalChunks, dataChunks)
	}
}

// AssertGatewayMetricsHeaders validates that the GenAI Gateway metrics headers are present
// and well-formed in non-streaming responses. For reasoning models (e.g., o1, o3, o4-mini, GPT-5),
// the X-Genai-Gateway-Reasoning-Tokens header should be present and contain a valid non-negative integer.
// For non-reasoning models, it may be absent.
//
// This function logs the reasoning tokens value when present for visibility during live test runs.
func AssertGatewayMetricsHeaders(t *testing.T, resp *http.Response, modelName string) {
	t.Helper()

	// Basic gateway headers that should always be present for successful requests
	responseTimeMs := resp.Header.Get("X-Genai-Gateway-Response-Time-Ms")
	if responseTimeMs != "" {
		if _, err := strconv.Atoi(responseTimeMs); err != nil {
			t.Fatalf("X-Genai-Gateway-Response-Time-Ms should be a valid integer, got: %q", responseTimeMs)
		}
	}

	inputTokens := resp.Header.Get("X-Genai-Gateway-Input-Tokens")
	if inputTokens != "" {
		if val, err := strconv.Atoi(inputTokens); err != nil {
			t.Fatalf("X-Genai-Gateway-Input-Tokens should be a valid integer, got: %q", inputTokens)
		} else if val < 0 {
			t.Fatalf("X-Genai-Gateway-Input-Tokens should be non-negative, got: %d", val)
		}
	}

	outputTokens := resp.Header.Get("X-Genai-Gateway-Output-Tokens")
	if outputTokens != "" {
		if val, err := strconv.Atoi(outputTokens); err != nil {
			t.Fatalf("X-Genai-Gateway-Output-Tokens should be a valid integer, got: %q", outputTokens)
		} else if val < 0 {
			t.Fatalf("X-Genai-Gateway-Output-Tokens should be non-negative, got: %d", val)
		}
	}

	// Reasoning tokens header: present only for reasoning models with reasoning_tokens > 0
	reasoningTokens := resp.Header.Get("X-Genai-Gateway-Reasoning-Tokens")
	if reasoningTokens != "" {
		val, err := strconv.Atoi(reasoningTokens)
		if err != nil {
			t.Fatalf("X-Genai-Gateway-Reasoning-Tokens should be a valid integer, got: %q", reasoningTokens)
		}
		if val <= 0 {
			t.Fatalf("X-Genai-Gateway-Reasoning-Tokens should be > 0 when present, got: %d", val)
		}
		logVerbosef("  Gateway metrics: model=%s, input_tokens=%s, output_tokens=%s, reasoning_tokens=%s\n",
			modelName, inputTokens, outputTokens, reasoningTokens)
	} else {
		logVerbosef("  Gateway metrics: model=%s, input_tokens=%s, output_tokens=%s (no reasoning tokens)\n",
			modelName, inputTokens, outputTokens)
	}
}

// AssertStreamingGatewayMetricsTrailers validates that the GenAI Gateway metrics trailers are present
// and well-formed in streaming responses. For reasoning models, the X-Genai-Gateway-Reasoning-Tokens
// trailer should be present with a valid positive integer value.
//
// This should be called after the response body has been fully consumed (trailers are sent after body).
func AssertStreamingGatewayMetricsTrailers(t *testing.T, resp *http.Response, modelName string) {
	t.Helper()

	// Trailers are only available after body is fully consumed
	inputTokens := resp.Trailer.Get("X-Genai-Gateway-Input-Tokens")
	outputTokens := resp.Trailer.Get("X-Genai-Gateway-Output-Tokens")

	if inputTokens != "" {
		if val, err := strconv.Atoi(inputTokens); err != nil {
			t.Fatalf("Trailer X-Genai-Gateway-Input-Tokens should be a valid integer, got: %q", inputTokens)
		} else if val < 0 {
			t.Fatalf("Trailer X-Genai-Gateway-Input-Tokens should be non-negative, got: %d", val)
		}
	}

	if outputTokens != "" {
		if val, err := strconv.Atoi(outputTokens); err != nil {
			t.Fatalf("Trailer X-Genai-Gateway-Output-Tokens should be a valid integer, got: %q", outputTokens)
		} else if val < 0 {
			t.Fatalf("Trailer X-Genai-Gateway-Output-Tokens should be non-negative, got: %d", val)
		}
	}

	// Reasoning tokens trailer: present only for reasoning models with reasoning_tokens > 0
	reasoningTokens := resp.Trailer.Get("X-Genai-Gateway-Reasoning-Tokens")
	if reasoningTokens != "" {
		val, err := strconv.Atoi(reasoningTokens)
		if err != nil {
			t.Fatalf("Trailer X-Genai-Gateway-Reasoning-Tokens should be a valid integer, got: %q", reasoningTokens)
		}
		if val <= 0 {
			t.Fatalf("Trailer X-Genai-Gateway-Reasoning-Tokens should be > 0 when present, got: %d", val)
		}
		logVerbosef("  Gateway streaming trailers: model=%s, input_tokens=%s, output_tokens=%s, reasoning_tokens=%s\n",
			modelName, inputTokens, outputTokens, reasoningTokens)
	} else {
		logVerbosef("  Gateway streaming trailers: model=%s, input_tokens=%s, output_tokens=%s (no reasoning tokens)\n",
			modelName, inputTokens, outputTokens)
	}
}

// AssertConverseStreamResponse validates a Bedrock converse-stream HTTP response:
//   - HTTP 200 status
//   - Response body is readable and non-empty
//
// The body parameter should contain the already-read response body bytes.
func AssertConverseStreamResponse(t *testing.T, resp *http.Response, body []byte) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK but got %d.\nResponse body: %s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		t.Fatal("Converse-stream response body is empty")
	}
}

// AssertImageGenerationResponse validates an image generation HTTP response:
//   - HTTP 200 (success) required — configuration errors and invalid requests fail the test
//   - HTTP 429 (rate limit) fails the test — RunImageGenerationTest retries 429s with
//     exponential backoff before calling this function, so a 429 here means retries exhausted
//   - Other 4xx errors are acceptable (quota exceeded, content policy)
//   - Contains image data in one of these formats:
//   - Gemini format: candidates[0].content.parts[0].inlineData with mimeType and data fields
//   - Imagen format: predictions[0] with bytesBase64Encoded and mimeType fields
//   - OpenAI format: data[0].b64_json with base64-encoded image data
//
// The body parameter should contain the already-read response body bytes.
func AssertImageGenerationResponse(t *testing.T, resp *http.Response, body []byte) {
	t.Helper()

	bodyStr := string(body)

	// Fail on rate limiting — retries with exponential backoff are performed by the caller;
	// reaching here means all retries were exhausted.
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Fatalf("Expected HTTP 200 but got 429 (rate limited after %d retries). Try again later.\nResponse body: %s", maxRetries429, bodyStr)
	}

	// Only fail on errors that indicate configuration problems (model not found, wrong payload format)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Check if it's a configuration error (model not found, unrecognized model, wrong payload)
		if strings.Contains(bodyStr, "unrecognized model") ||
			strings.Contains(bodyStr, "model not found") ||
			strings.Contains(bodyStr, "Model name must be specified") ||
			strings.Contains(bodyStr, "INVALID_ARGUMENT") {
			t.Fatalf("Expected HTTP 200 but got %d (configuration error).\nResponse body: %s", resp.StatusCode, bodyStr)
		}
		// Fail on invalid request errors — these indicate our payload format is wrong
		if strings.Contains(bodyStr, "invalid_request_error") ||
			strings.Contains(bodyStr, "unknown_parameter") ||
			strings.Contains(bodyStr, "invalid_value") {
			t.Fatalf("Expected HTTP 200 but got %d (invalid request).\nResponse body: %s", resp.StatusCode, bodyStr)
		}
		// Other 4xx errors are acceptable (e.g., quota exceeded, content policy)
		logVerbosef("  Image generation validated (4xx non-config error): Status %d, request format correct\n", resp.StatusCode)
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 but got %d.\nResponse body: %s", resp.StatusCode, bodyStr)
	}

	AssertResponseHeaders(t, resp)

	if !gjson.Valid(bodyStr) {
		t.Fatalf("Response body should be valid JSON: %s", bodyStr)
	}

	img := extractImageData(bodyStr)
	if img != nil {
		if img.Format == "imagen" && img.MimeType == "" {
			t.Fatalf("Imagen prediction should contain non-empty 'mimeType' field.\nFull response: %s", bodyStr)
		}
		logVerbosef("  Image generation validated (%s): mimeType=%s, data length=%d bytes\n",
			img.Format, img.MimeType, len(img.B64Data))
		return
	}

	// No image data extracted — check if Gemini structure is still valid (e.g., thinking steps)
	candidates := gjson.Get(bodyStr, "candidates")
	if !candidates.Exists() || !candidates.IsArray() || len(candidates.Array()) == 0 {
		t.Fatalf("Response should contain non-empty 'candidates' array (Gemini), 'predictions' array (Imagen), or 'data' array (OpenAI).\nFull response: %s", bodyStr)
	}

	parts := gjson.Get(bodyStr, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		t.Fatalf("Response should contain 'candidates[0].content.parts' array.\nFull response: %s", bodyStr)
	}

	logVerbosef("  Image generation validated (Gemini): Response has valid structure (candidates with %d parts)\n", len(parts.Array()))
}
