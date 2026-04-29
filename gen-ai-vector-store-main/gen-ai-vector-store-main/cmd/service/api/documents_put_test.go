/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/smart_chunking"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test helpers for overriding the package-level SC client ---

func resetSmartChunkingClientForTest() {
	scClientOnce = sync.Once{}
	scClient = nil
	scClientErr = nil
}

func overrideSmartChunkingClientForTest(client smart_chunking.SmartChunkingClient, err error) {
	scClientOnce = sync.Once{}
	scClientOnce.Do(func() {}) // mark as done so getSmartChunkingClient won't re-init
	scClient = client
	scClientErr = err
}

// --- mock SmartChunkingClient ---

type mockSCClient struct {
	submitJobFn func(ctx context.Context, authToken, isolationID string, fileReader io.Reader, fileName string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error)
}

func (m *mockSCClient) SubmitJob(ctx context.Context, authToken, isolationID string, fileReader io.Reader, fileName string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
	return m.submitJobFn(ctx, authToken, isolationID, fileReader, fileName, options)
}

func boolPtr(b bool) *bool { return &b }

// --- tests ---

func TestGetSmartChunkingClient_LazyInit(t *testing.T) {
	t.Run("returns overridden client", func(t *testing.T) {
		mock := &mockSCClient{}
		overrideSmartChunkingClientForTest(mock, nil)
		defer resetSmartChunkingClientForTest()

		client, err := getSmartChunkingClient()
		assert.NoError(t, err)
		assert.Equal(t, mock, client)
	})

	t.Run("returns overridden error", func(t *testing.T) {
		overrideSmartChunkingClientForTest(nil, fmt.Errorf("init failed"))
		defer resetSmartChunkingClientForTest()

		client, err := getSmartChunkingClient()
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "init failed")
	})

	t.Run("returns same instance on repeated calls", func(t *testing.T) {
		mock := &mockSCClient{}
		overrideSmartChunkingClientForTest(mock, nil)
		defer resetSmartChunkingClientForTest()

		client1, _ := getSmartChunkingClient()
		client2, _ := getSmartChunkingClient()
		assert.Same(t, client1, client2, "lazy init should return the same instance")
	})
}

func TestSubmitFileJob_ClientUnavailable(t *testing.T) {
	overrideSmartChunkingClientForTest(nil, fmt.Errorf("SAX certificate not found"))
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "file.pdf",
		"doc-1", nil, nil,
	)

	assert.False(t, result)
	assert.Equal(t, http.StatusBadGateway, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "502", body["code"])
	assert.Contains(t, body["message"], "smart-chunking client not available")
}

func TestSubmitFileJob_SCReturnsError(t *testing.T) {
	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, _ smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "file.pdf",
		"doc-1", nil, nil,
	)

	assert.False(t, result)
	assert.Equal(t, http.StatusBadGateway, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "502", body["code"])
	assert.Contains(t, body["message"], "smart-chunking service")
}

func TestSubmitFileJob_Success(t *testing.T) {
	var capturedFileName string
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, fileName string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedFileName = fileName
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{
				OperationID: "op-123",
				IsolationID: "iso-1",
				Status:      "PENDING",
			}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	docAttrs := []attributes.Attribute{{Name: "title", Values: attributes.AttrValues{"Test"}, Type: "string", Kind: "static"}}
	metadata := &documents.DocumentMetadata{EnableOCR: boolPtr(true)}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("file content")), "report.pdf",
		"doc-1", docAttrs, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Nil(t, body["operationID"], "operationID should not be in response")
	assert.Equal(t, "doc-1", body["documentID"])
	assert.Equal(t, "IN_PROGRESS", body["status"])

	// Verify SC received correct parameters
	assert.Equal(t, "report.pdf", capturedFileName)
	assert.Equal(t, []string{"extraction", "chunking", "indexing"}, capturedOptions.Tasks)
	assert.NotNil(t, capturedOptions.TaskOptions.Extraction)
	require.NotNil(t, capturedOptions.TaskOptions.Extraction.EnableOCR)
	assert.True(t, *capturedOptions.TaskOptions.Extraction.EnableOCR)
	assert.Equal(t, "col-1", capturedOptions.TaskOptions.Indexing.CollectionName)
	assert.Equal(t, "doc-1", capturedOptions.TaskOptions.Indexing.DocumentID)
}

func TestSubmitFileJob_WithMetadata(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-456"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		StaticEmbeddingAttributes: []string{"title", "category"},
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Verify embeddingAttributes from metadata are forwarded
	assert.Nil(t, capturedOptions.TaskOptions.Extraction, "extraction opts should be nil when enableOCR is false")
	assert.Equal(t, []string{"title", "category"}, capturedOptions.TaskOptions.Indexing.EmbeddingAttributes)
}

func TestSubmitFileJob_SC4xxError_PropagatesStatusCode(t *testing.T) {
	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, _ smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			return nil, &smart_chunking.ServiceError{StatusCode: 422, Body: "Unsupported file type: .xyz"}
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "file.xyz",
		"doc-1", nil, nil,
	)

	assert.False(t, result)
	assert.Equal(t, 422, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "422", body["code"])
	assert.Equal(t, "Unsupported file type: .xyz", body["message"])
}

func TestSubmitFileJob_SC5xxError_Returns502(t *testing.T) {
	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, _ smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			return nil, &smart_chunking.ServiceError{StatusCode: 500, Body: "internal failure"}
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "file.pdf",
		"doc-1", nil, nil,
	)

	assert.False(t, result)
	assert.Equal(t, http.StatusBadGateway, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "502", body["code"])
	assert.Contains(t, body["message"], "smart-chunking service")
}

func TestSubmitFileJob_EnableSmartAttribution(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-789"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		EnableSmartAttribution: boolPtr(true),
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// enableSmartAttribution=true should set ChunkingOptions
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	assert.True(t, *capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	// embedSmartAttributes should NOT be set
	assert.Nil(t, capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
}

func TestSubmitFileJob_BothSmartFlags(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-101"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		StaticEmbeddingAttributes: []string{"department"},
		EnableSmartAttribution:    boolPtr(true),
		EmbedSmartAttributes:      boolPtr(true),
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Both flags should be set independently
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	assert.True(t, *capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	require.NotNil(t, capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
	assert.True(t, *capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
	assert.Equal(t, []string{"department"}, capturedOptions.TaskOptions.Indexing.EmbeddingAttributes)
}

func TestSubmitFileJob_EmbedSmartAttributesWithoutEnableSmartAttribution(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-202"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		EmbedSmartAttributes: boolPtr(true),
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// embedSmartAttributes=true without enableSmartAttribution should NOT create ChunkingOptions
	assert.Nil(t, capturedOptions.TaskOptions.Chunking, "chunking opts should be nil when enableSmartAttribution is false")
	// But embedSmartAttributes should still be forwarded (SC will silently ignore it)
	require.NotNil(t, capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
	assert.True(t, *capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
}

func TestSubmitFileJob_ExcludeSmartAttributes(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-300"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	excludeAttrs := []string{"title", "author"}
	metadata := &documents.DocumentMetadata{
		EnableSmartAttribution: boolPtr(true),
		ExcludeSmartAttributes: &excludeAttrs,
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Verify excludeSmartAttributes is passed to ChunkingOptions
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	assert.True(t, *capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
	assert.Equal(t, &excludeAttrs, capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
	assert.Equal(t, []string{"title", "author"}, *capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
}

func TestSubmitFileJob_ExcludeSmartAttributesNil(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-301"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		EnableSmartAttribution: boolPtr(true),
		ExcludeSmartAttributes: nil, // explicitly nil
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Verify nil excludeSmartAttributes is passed as nil
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	assert.Nil(t, capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
}

func TestSubmitFileJob_ExcludeSmartAttributesEmpty(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-302"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	emptySlice := []string{}
	metadata := &documents.DocumentMetadata{
		EnableSmartAttribution: boolPtr(true),
		ExcludeSmartAttributes: &emptySlice,
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Verify empty array is passed correctly
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
	assert.Equal(t, &emptySlice, capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
	assert.Empty(t, *capturedOptions.TaskOptions.Chunking.ExcludeSmartAttributes)
}

func TestSubmitFileJob_ExcludeSmartAttributesWithoutEnableSmartAttribution(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-303"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	excludeAttrs := []string{"title"}
	metadata := &documents.DocumentMetadata{
		// EnableSmartAttribution is nil/false
		ExcludeSmartAttributes: &excludeAttrs,
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// excludeSmartAttributes without enableSmartAttribution should NOT create ChunkingOptions
	assert.Nil(t, capturedOptions.TaskOptions.Chunking, "chunking opts should be nil when enableSmartAttribution is not set")
}

func TestSubmitFileJob_NilAttributeValuesCoercedToEmpty(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions

	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-303"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	docAttrs := []attributes.Attribute{
		{Name: "contentKey", Values: attributes.AttrValues{"PEGAFW-KB-WORK-ARTICLE KB-31062"}, Type: "string"},
		{Name: "contentSourceSystem", Values: attributes.AttrValues{"Modern"}, Type: "string"},
		{Name: "contentID", Values: attributes.AttrValues{"KB-31062"}, Type: "string"},
		{Name: "contentFormat", Values: attributes.AttrValues{"file"}, Type: "string"},
		{Name: "contentURL", Values: nil, Type: "string"},
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "article.pdf",
		"doc-1", docAttrs, nil,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	attrs := capturedOptions.TaskOptions.Indexing.DocumentAttributes
	require.Len(t, attrs, 5)
	for i, attr := range attrs {
		assert.NotNil(t, attr.Values, "attribute %d (%s) Values must not be nil", i, attr.Name)
	}
	// The previously-nil attribute is now an empty (non-nil) slice
	assert.Equal(t, attributes.AttrValues{}, attrs[4].Values)
}

func TestPutDocumentRequest_SmartAttrsDeserializedButIgnored(t *testing.T) {
	// The JSON PUT endpoint (/documents) uses DocumentMetadata which includes
	// enableSmartAttribution and embedSmartAttributes fields. These fields are
	// only acted upon by /file and /file-text (via submitFileJob). The JSON PUT
	// handler ignores them. This test confirms they deserialize without error
	// and don't contaminate the fields the JSON endpoint actually uses.
	payload := `{
		"id": "DOC-1",
		"chunks": [{"content": "chunk text"}],
		"metadata": {
			"embeddingAttributes": ["version"],
			"extraAttributesKinds": ["auto-resolved"],
			"enableSmartAttribution": true,
			"embedSmartAttributes": true
		}
	}`

	var doc documents.PutDocumentRequest
	err := json.Unmarshal([]byte(payload), &doc)
	require.NoError(t, err)

	// Smart attribution fields deserialize into the struct
	require.NotNil(t, doc.Metadata)
	require.NotNil(t, doc.Metadata.EnableSmartAttribution)
	assert.True(t, *doc.Metadata.EnableSmartAttribution)
	require.NotNil(t, doc.Metadata.EmbedSmartAttributes)
	assert.True(t, *doc.Metadata.EmbedSmartAttributes)

	// The fields the JSON PUT endpoint actually reads are unaffected
	assert.Equal(t, []string{"version"}, doc.Metadata.StaticEmbeddingAttributes)
	assert.Equal(t, []string{"auto-resolved"}, doc.Metadata.ExtraAttributesKinds)
}

func TestPutDocumentRequest_MetadataWithoutSmartAttrs(t *testing.T) {
	// Confirms the base Metadata fields work alone (the common case for JSON PUT)
	payload := `{
		"id": "DOC-2",
		"chunks": [{"content": "chunk text"}],
		"metadata": {
			"embeddingAttributes": ["region"]
		}
	}`

	var doc documents.PutDocumentRequest
	err := json.Unmarshal([]byte(payload), &doc)
	require.NoError(t, err)

	require.NotNil(t, doc.Metadata)
	assert.Equal(t, []string{"region"}, doc.Metadata.StaticEmbeddingAttributes)
	assert.Nil(t, doc.Metadata.EnableSmartAttribution)
	assert.Nil(t, doc.Metadata.EmbedSmartAttributes)
	assert.Empty(t, doc.Metadata.ExtraAttributesKinds)
}

func TestSubmitFileJob_NilMetadataFlags_NotForwarded(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions
	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-nil"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	// Metadata with all three bool flags left nil
	metadata := &documents.DocumentMetadata{
		StaticEmbeddingAttributes: []string{"title"},
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// None of the nil flags should create extraction or chunking options
	assert.Nil(t, capturedOptions.TaskOptions.Extraction)
	assert.Nil(t, capturedOptions.TaskOptions.Chunking)
	assert.Nil(t, capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
}

func TestSubmitFileJob_ExplicitFalse_Forwarded(t *testing.T) {
	var capturedOptions smart_chunking.JobRequestOptions
	mock := &mockSCClient{
		submitJobFn: func(_ context.Context, _, _ string, _ io.Reader, _ string, options smart_chunking.JobRequestOptions) (*smart_chunking.JobSubmittedResponse, error) {
			capturedOptions = options
			return &smart_chunking.JobSubmittedResponse{OperationID: "op-false"}, nil
		},
	}
	overrideSmartChunkingClientForTest(mock, nil)
	defer resetSmartChunkingClientForTest()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)

	metadata := &documents.DocumentMetadata{
		EnableOCR:              boolPtr(false),
		EnableSmartAttribution: boolPtr(false),
		EmbedSmartAttributes:   boolPtr(false),
	}
	logger := log.GetNamedLogger("test")
	result := submitFileJob(
		c, context.Background(), logger,
		"iso-1", "col-1", "Bearer token",
		bytes.NewReader([]byte("content")), "doc.pdf",
		"doc-1", nil, metadata,
	)

	assert.True(t, result)
	assert.Equal(t, http.StatusAccepted, recorder.Code)

	// Explicit false should be forwarded (options created, not nil)
	require.NotNil(t, capturedOptions.TaskOptions.Extraction)
	assert.False(t, *capturedOptions.TaskOptions.Extraction.EnableOCR)
	require.NotNil(t, capturedOptions.TaskOptions.Chunking)
	assert.False(t, *capturedOptions.TaskOptions.Chunking.EnableSmartAttribution)
	require.NotNil(t, capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
	assert.False(t, *capturedOptions.TaskOptions.Indexing.EmbedSmartAttributes)
}

func TestFilenameSanitization(t *testing.T) {
	// Verify that filepath.Base strips path components from malicious filenames.
	// This is the sanitization applied in PutDocumentFile before sending to SC.
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal filename", "report.pdf", "report.pdf"},
		{"path traversal unix", "../../etc/passwd", "passwd"},
		// Note: on Unix, backslash is a valid filename char; filepath.Base only splits on '/'.
		// Windows-style paths are not a concern since the service runs on Linux.
		{"absolute unix path", "/tmp/evil.pdf", "evil.pdf"},
		{"nested path", "a/b/c/d/file.txt", "file.txt"},
		{"filename with spaces", "my report.pdf", "my report.pdf"},
		{"empty string", "", "."},
		{"dot only", ".", "."},
		{"slash only", "/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, filepath.Base(tt.input))
		})
	}
}
