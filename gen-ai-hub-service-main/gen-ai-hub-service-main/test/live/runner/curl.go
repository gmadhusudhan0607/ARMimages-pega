/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"text/template"

	http2curl "moul.io/http2curl/v2"
)

//go:embed templates/curl.sh.tmpl
var curlTemplateStr string

var curlTemplate = template.Must(template.New("curl.sh").Parse(curlTemplateStr))

// curlTemplateData is the data passed to the curl.sh.tmpl template.
type curlTemplateData struct {
	Curl        string
	SaxProfile  string
	SaxSecretID string
	SaxRegion   string
}

// SaveCurlFile generates a reproducible curl command from the given HTTP request,
// writes a portable version to filePrefix+".curl.sh", and registers a t.Cleanup
// that logs the curl command on test failure.
//
// The request body is cloned internally so the caller's request remains usable.
// The portable version replaces baseURL with http://localhost:8080 and token with $JWT.
func SaveCurlFile(t *testing.T, req *http.Request, filePrefix, baseURL, token string, sax SaxConfig) {
	t.Helper()

	// Read and restore body so the caller's request is not consumed
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Clone for GetCurlCommand (which consumes the body)
	reqCopy := req.Clone(req.Context())
	reqCopy.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	curlCmd, _ := http2curl.GetCurlCommand(reqCopy)
	curlStr := curlCmd.String()

	// Make curl file portable: use fixed port 8080 and $JWT variable
	portableCurl := strings.ReplaceAll(curlStr, baseURL, "http://localhost:8080")
	portableCurl = strings.ReplaceAll(portableCurl, token, "$JWT")
	// Use double quotes for the Authorization header so bash expands $JWT
	portableCurl = strings.ReplaceAll(portableCurl, "'Authorization: Bearer $JWT'", `"Authorization: Bearer $JWT"`)
	// Add --silent and --no-buffer (needed for streaming) flags
	portableCurl = strings.Replace(portableCurl, "curl ", "curl --silent --no-buffer ", 1)

	// Render curl template
	var buf bytes.Buffer
	data := curlTemplateData{
		Curl:        portableCurl,
		SaxProfile:  sax.Profile,
		SaxSecretID: sax.SecretID,
		SaxRegion:   sax.Region,
	}
	if err := curlTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to render curl template: %v", err)
	}

	curlFile := filePrefix + ".curl.sh"
	if err := os.WriteFile(curlFile, buf.Bytes(), 0755); err != nil {
		t.Logf("Warning: failed to write curl file %s: %v", curlFile, err)
	}

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("\nReproduce with:\n%s\n", curlStr)
			t.Logf("Curl file: %s", curlFile)
		}
	})
}
