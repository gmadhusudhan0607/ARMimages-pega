//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"net/http"
)

// Test constants for consistent values across emulation mode tests
const (
	// Test identifiers
	TestDocumentID  = "DOC-1"
	TestDocumentID2 = "DOC-2"
	TestGroupID     = "test-group-1"
	TestChunkID     = "test-doc-1"

	// API version paths
	APIv1Path = "/v1"
	APIv2Path = "/v2"

	// Collection paths
	CollectionsPath     = "/collections"
	DocumentsPath       = "/documents"
	AttributesPath      = "/attributes"
	QueryChunksPath     = "/query/chunks"
	QueryDocumentsPath  = "/query/documents"
	SmartAttributesPath = "/smart-attributes-group"
	FindDocumentsPath   = "/find-documents"

	// File paths
	FileUploadPath     = "/file"
	FileTextUploadPath = "/file/text"
	DeleteByIDPath     = "/document/delete-by-id"

	// Expected HTTP status codes
	StatusOK       = http.StatusOK
	StatusCreated  = http.StatusCreated
	StatusAccepted = http.StatusAccepted

	// Test data files
	DocumentsPutRequestFile    = "requests/documents-put-DOC-1.json"
	DocumentsDeleteRequestFile = "requests/documents-delete-request-in-1.json"
	DocumentsPatchRequestFile  = "requests/documents-patch-request-1.json"
	QueryChunksRequestFile     = "requests/query-chunks-v2-query-limit-1.json"
	QueryDocumentsRequestFile  = "requests/query-documents-query-limit-1.json"
	QueryAttributesRequestFile = "requests/query-retrieveAttributes.json"
	AstronomyPDFFile           = "requests/Astronomy.pdf"
	AstronomyMarkdownFile      = "requests/Astronomy.md"
)

// Common request payloads
var (
	FindDocumentsPayload = `{"query": "test query", "limit": 10}`
	DeleteByIDPayload    = `{"id": "DOC-1"}`
	AttributesPayload    = `{"retrieveAttributes": ["attr1", "attr2"]}`
)

// Helper function to generate collection request body
func CollectionRequestBody(collectionID string) string {
	return `{"collectionID":"` + collectionID + `"}`
}

// Helper function to generate smart attributes group request body
func SmartAttributesGroupRequestBody(isolationID, description string) string {
	return `{"description":"` + description + `","attributes":["version","category"]}`
}
