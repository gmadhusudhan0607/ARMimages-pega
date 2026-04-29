/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// NewAuthorizedRequest creates an HTTP POST request with JSON content type
// and Bearer token authorization. The body is set from the provided payload bytes.
func NewAuthorizedRequest(t *testing.T, url string, payload []byte, token string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// DefaultTimeout is the default HTTP request timeout for live test scenarios.
const DefaultTimeout = 300 * time.Second

// maxRetries429 is the maximum number of retries for HTTP 429 (rate limited) responses.
const maxRetries429 = 3

// retryBackoffBase is the initial backoff duration before the first retry on HTTP 429.
// Subsequent retries double the delay: 5s, 10s, 20s.
const retryBackoffBase = 5 * time.Second

// newTestFilePrefix generates a unique file prefix for test artifact files (curl, headers, body).
// The prefix has the form "/tmp/live-test-{uuid}-{sanitized_test_name}" and is logged for
// easy tracing of which files belong to which test run.
func newTestFilePrefix(t *testing.T, target ModelTarget) string {
	t.Helper()
	testID := uuid.Must(uuid.NewV7()).String()
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	filePrefix := fmt.Sprintf("/tmp/live-test-%s-%s", testID, safeName)
	logVerbosef("Test ID: %s (target: %s, files: %s.*)\n", testID, target, filePrefix)
	return filePrefix
}

// RunChatCompletionSuite runs chat completion tests for all discovered targets in parallel.
// Each scenario's run_test.go should call this from its test function.
func RunChatCompletionSuite(t *testing.T, env *TestEnvironment,
	systemPromptFile, userPromptFile string, timeout time.Duration) {
	t.Helper()
	t.Parallel()

	targets := env.ChatCompletionTargets
	if len(targets) == 0 {
		t.Skip("No chat completion targets discovered from /models")
	}

	for _, target := range targets {
		t.Run(fmt.Sprintf("should complete successfully for %s", target), func(t *testing.T) {
			t.Parallel()
			RunChatCompletionTest(t, env, target, systemPromptFile, userPromptFile, timeout)
		})
	}
}

// RunChatCompletionTest executes a chat completion request against the given target,
// saves curl/response files, and validates the response.
// It generates a unique test ID, builds the payload from the given prompt files,
// and asserts the response based on the provider type.
func RunChatCompletionTest(t *testing.T, env *TestEnvironment, target ModelTarget,
	systemPromptFile, userPromptFile string, timeout time.Duration) {
	t.Helper()

	filePrefix := newTestFilePrefix(t, target)

	payloadBytes := BuildChatCompletionPayload(t, target, systemPromptFile, userPromptFile)

	url := env.SvcBaseURL + target.RequestPath()
	req := NewAuthorizedRequest(t, url, payloadBytes, env.JWTToken)

	// Generate reproducible curl command and save to file
	SaveCurlFile(t, req, filePrefix, env.SvcBaseURL, env.JWTToken, SaxConfigForCell(env.SaxCell))

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Save response headers and body to files
	SaveResponseFiles(t, resp, respBody, filePrefix)

	// Validate response based on provider type
	if target.isConverseProvider() {
		AssertConverseResponse(t, resp, respBody)
	} else {
		AssertChatCompletionResponse(t, resp, respBody)
	}

	// Validate gateway metrics headers (including reasoning tokens for reasoning models)
	AssertGatewayMetricsHeaders(t, resp, target.Model)
}

// RunStreamingChatCompletionSuite runs streaming chat completion tests for all discovered targets in parallel.
// Each scenario's run_test.go should call this from its test function.
func RunStreamingChatCompletionSuite(t *testing.T, env *TestEnvironment,
	systemPromptFile, userPromptFile string, timeout time.Duration) {
	t.Helper()
	t.Parallel()

	targets := env.ChatCompletionTargets
	if len(targets) == 0 {
		t.Skip("No chat completion targets discovered from /models")
	}

	for _, target := range targets {
		t.Run(fmt.Sprintf("should stream successfully for %s", target), func(t *testing.T) {
			t.Parallel()
			RunStreamingChatCompletionTest(t, env, target, systemPromptFile, userPromptFile, timeout)
		})
	}
}

// RunStreamingChatCompletionTest executes a streaming chat completion request against the given target,
// saves curl files, and validates the streaming response.
// For chat completion targets (OpenAI, Google), the request body includes "stream": true and
// the response is validated as an SSE stream.
// For Anthropic targets, the converse-stream endpoint is used and the response body is validated
// as non-empty.
func RunStreamingChatCompletionTest(t *testing.T, env *TestEnvironment, target ModelTarget,
	systemPromptFile, userPromptFile string, timeout time.Duration) {
	t.Helper()

	filePrefix := newTestFilePrefix(t, target)

	payloadBytes := BuildStreamingChatCompletionPayload(t, target, systemPromptFile, userPromptFile)

	url := env.SvcBaseURL + target.StreamingRequestPath()
	req := NewAuthorizedRequest(t, url, payloadBytes, env.JWTToken)

	// Generate reproducible curl command and save to file
	SaveCurlFile(t, req, filePrefix, env.SvcBaseURL, env.JWTToken, SaxConfigForCell(env.SaxCell))

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Validate streaming response based on provider type
	if target.isConverseProvider() {
		// For converse-stream, read the full body and validate
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Save response headers and body to files
		SaveResponseFiles(t, resp, respBody, filePrefix)

		AssertConverseStreamResponse(t, resp, respBody)
	} else {
		// For SSE streaming (OpenAI, Google), validate the stream directly
		// Save response headers (body is streamed, so we save what we can)
		SaveResponseFiles(t, resp, []byte("<streaming response>"), filePrefix)

		AssertStreamingChatCompletionResponse(t, resp)

		// Validate gateway metrics trailers (including reasoning tokens for reasoning models)
		// Trailers are available after body is fully consumed by AssertStreamingChatCompletionResponse
		AssertStreamingGatewayMetricsTrailers(t, resp, target.Model)
	}
}

// RunEmbeddingSuite runs embedding tests for all discovered embedding targets in parallel.
// Each scenario's run_test.go should call this from its test function.
func RunEmbeddingSuite(t *testing.T, env *TestEnvironment,
	inputFilePath string, timeout time.Duration) {
	t.Helper()
	t.Parallel()

	targets := env.EmbeddingTargets
	if len(targets) == 0 {
		t.Skip("No embedding targets discovered from /models")
	}

	for _, target := range targets {
		t.Run(fmt.Sprintf("should embed successfully for %s", target), func(t *testing.T) {
			t.Parallel()
			RunEmbeddingTest(t, env, target, inputFilePath, timeout)
		})
	}
}

// RunEmbeddingTest executes an embedding request against the given target.
// It reads the input from inputFilePath, builds the payload using the embeddings template,
// saves curl/response files, and validates the response.
func RunEmbeddingTest(t *testing.T, env *TestEnvironment, target ModelTarget, inputFilePath string, timeout time.Duration) {
	t.Helper()

	filePrefix := newTestFilePrefix(t, target)

	payload := BuildEmbeddingPayload(t, target, inputFilePath)

	url := env.SvcBaseURL + target.EmbeddingsPath()
	req := NewAuthorizedRequest(t, url, payload, env.JWTToken)

	// Generate reproducible curl command and save to file
	SaveCurlFile(t, req, filePrefix, env.SvcBaseURL, env.JWTToken, SaxConfigForCell(env.SaxCell))

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Save response headers and body to files
	SaveResponseFiles(t, resp, respBody, filePrefix)

	// Validate embedding response
	AssertEmbeddingResponse(t, resp, respBody)
}

// RunImageGenerationSuite runs image generation tests for all discovered image generation targets in parallel.
// Each scenario's run_test.go should call this from its test function.
func RunImageGenerationSuite(t *testing.T, env *TestEnvironment, userPromptFile string, timeout time.Duration) {
	t.Helper()
	t.Parallel()

	targets := env.ImageGenerationTargets
	if len(targets) == 0 {
		t.Skip("No image generation targets discovered from /models")
	}

	for _, target := range targets {
		t.Run(fmt.Sprintf("should generate image successfully for %s", target), func(t *testing.T) {
			t.Parallel()
			RunImageGenerationTest(t, env, target, userPromptFile, timeout)
		})
	}
}

// RunImageGenerationTest executes an image generation request against the given target,
// saves curl/response files, and validates the response.
// It generates a unique test ID, builds the payload from the given prompt file,
// and asserts the response contains valid image data in Gemini native format.
// HTTP 429 (rate limited) responses are retried up to maxRetries429 times with
// exponential backoff starting at retryBackoffBase before the test is failed.
func RunImageGenerationTest(t *testing.T, env *TestEnvironment, target ModelTarget, userPromptFile string, timeout time.Duration) {
	t.Helper()

	filePrefix := newTestFilePrefix(t, target)

	payloadBytes := BuildImageGenerationPayload(t, target, userPromptFile)

	url := env.SvcBaseURL + target.ImageGenerationPath()

	// Generate reproducible curl command and save to file (use the first request for curl)
	firstReq := NewAuthorizedRequest(t, url, payloadBytes, env.JWTToken)
	SaveCurlFile(t, firstReq, filePrefix, env.SvcBaseURL, env.JWTToken, SaxConfigForCell(env.SaxCell))

	client := &http.Client{Timeout: timeout}

	var resp *http.Response
	var respBody []byte
	for attempt := 0; attempt <= maxRetries429; attempt++ {
		if attempt > 0 {
			backoff := retryBackoffBase * (1 << (attempt - 1))
			logVerbosef("  Rate limited (HTTP 429), retrying in %s (attempt %d/%d)...\n", backoff, attempt, maxRetries429)
			time.Sleep(backoff)
		}

		req := NewAuthorizedRequest(t, url, payloadBytes, env.JWTToken)
		var err error
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
	}

	// Restore body for SaveResponseFiles (which needs a readable Body)
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	// Save response headers and body to files
	SaveResponseFiles(t, resp, respBody, filePrefix)

	// Validate image generation response
	AssertImageGenerationResponse(t, resp, respBody)

	// Save the decoded image to a file for manual inspection (only on success)
	if resp.StatusCode == http.StatusOK {
		SaveImageFile(t, respBody, filePrefix)
	}
}

// imageExtraction holds base64-encoded image data extracted from an image generation response.
type imageExtraction struct {
	B64Data  string // base64-encoded image bytes
	MimeType string // e.g. "image/png"
	Format   string // "imagen", "openai", or "gemini"
}

// extractImageData detects the response format (Imagen, OpenAI DALL-E, or Gemini) and
// extracts the first base64-encoded image along with its MIME type. Returns nil if no
// image data is found.
func extractImageData(bodyStr string) *imageExtraction {
	// Try Imagen format: predictions[0].bytesBase64Encoded
	if v := gjson.Get(bodyStr, "predictions.0.bytesBase64Encoded"); v.Exists() && v.String() != "" {
		return &imageExtraction{
			B64Data:  v.String(),
			MimeType: gjson.Get(bodyStr, "predictions.0.mimeType").String(),
			Format:   "imagen",
		}
	}

	// Try OpenAI DALL-E format: data[0].b64_json
	if v := gjson.Get(bodyStr, "data.0.b64_json"); v.Exists() && v.String() != "" {
		return &imageExtraction{
			B64Data:  v.String(),
			MimeType: "image/png", // Default to PNG when output_format is not indicated in the response
			Format:   "openai",
		}
	}

	// Try Gemini format: candidates[0].content.parts[].inlineData
	parts := gjson.Get(bodyStr, "candidates.0.content.parts")
	if parts.Exists() && parts.IsArray() {
		for _, part := range parts.Array() {
			inlineData := part.Get("inlineData")
			if inlineData.Exists() {
				d := inlineData.Get("data")
				m := inlineData.Get("mimeType")
				if d.Exists() && d.String() != "" {
					return &imageExtraction{
						B64Data:  d.String(),
						MimeType: m.String(),
						Format:   "gemini",
					}
				}
			}
		}
	}

	return nil
}

// SaveImageFile extracts base64-encoded image data from an image generation response,
// decodes it, and writes it to a file at {filePrefix}.image.{ext}. It supports Imagen,
// OpenAI DALL-E, and Gemini response formats. Failures are logged as warnings and do
// not fail the test — the image file is a convenience artifact for manual inspection.
func SaveImageFile(t *testing.T, body []byte, filePrefix string) {
	t.Helper()

	bodyStr := string(body)
	if !gjson.Valid(bodyStr) {
		t.Logf("Warning: cannot save image file — response is not valid JSON")
		return
	}

	img := extractImageData(bodyStr)
	if img == nil {
		t.Logf("Warning: cannot save image file — no base64 image data found in response")
		return
	}

	ext := mimeTypeToExtension(img.MimeType)
	imageBytes, err := base64.StdEncoding.DecodeString(img.B64Data)
	if err != nil {
		t.Logf("Warning: failed to base64-decode image data: %v", err)
		return
	}

	filePath := filePrefix + ".image." + ext
	if err := os.WriteFile(filePath, imageBytes, 0600); err != nil {
		t.Logf("Warning: failed to write image file %s: %v", filePath, err)
		return
	}

	logVerbosef("  Image saved to %s (%d bytes)\n", filePath, len(imageBytes))
}

// mimeTypeToExtension maps an image MIME type to a file extension.
func mimeTypeToExtension(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png"
	}
}
