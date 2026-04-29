/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// SaveResponseFiles writes the HTTP response to two files:
//   - filePrefix+".resp.headers" — status line and response headers
//   - filePrefix+".resp.body.json" — raw response body
func SaveResponseFiles(t *testing.T, resp *http.Response, body []byte, filePrefix string) {
	t.Helper()

	// Save headers
	headersFile := filePrefix + ".resp.headers"
	var headersBuf bytes.Buffer
	fmt.Fprintf(&headersBuf, "%s %s\n", resp.Proto, resp.Status)
	_ = resp.Header.Write(&headersBuf)
	if err := os.WriteFile(headersFile, headersBuf.Bytes(), 0644); err != nil {
		t.Logf("Warning: failed to write headers file %s: %v", headersFile, err)
	}

	// Save body
	bodyFile := filePrefix + ".resp.body.json"
	if err := os.WriteFile(bodyFile, body, 0644); err != nil {
		t.Logf("Warning: failed to write body file %s: %v", bodyFile, err)
	}
}
