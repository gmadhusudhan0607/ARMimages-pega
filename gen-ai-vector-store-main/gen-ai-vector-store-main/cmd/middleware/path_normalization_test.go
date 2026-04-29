/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPathNormalizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(PathNormalizationMiddleware)
	engine.GET("/*any", func(c *gin.Context) {
		path, exists := c.Get("normalizedPath")
		if !exists {
			c.String(500, "missing normalizedPath")
			return
		}
		c.String(200, path.(string))
	})

	tests := []struct {
		incomingPath string
		expected     string
	}{
		{"/v1/iso-test123/collections/col-test123/documents", "/v1/<isolationID>/collections/<collectionName>/documents"},
		{"/v1/iso-test123/collections/col-test123/documents/doc-test123", "/v1/<isolationID>/collections/<collectionName>/documents/<documentID>"},
		{"/v1/iso-test123/collections/col-test123/query/chunks", "/v1/<isolationID>/collections/<collectionName>/query/chunks"},
		{"/v1/iso-test123/collections/col-test123/query/documents", "/v1/<isolationID>/collections/<collectionName>/query/documents"},
		{"/swagger", "/swagger"},
		{"/v1/iso-test123/collections/col-test123/file/text", "/v1/<isolationID>/collections/<collectionName>/file/text"},
		{"/v1/iso-test123/collections/col-test123/file", "/v1/<isolationID>/collections/<collectionName>/file"},
		{"/v1/iso-test123/collections/col-test123/document/delete-by-id", "/v1/<isolationID>/collections/<collectionName>/document/delete-by-id"},
		{"/v1/iso-test123/collections/col-test123/attributes", "/v1/<isolationID>/collections/<collectionName>/attributes"},
		{"/v1/iso-test123/smart-attributes-group", "/v1/<isolationID>/smart-attributes-group"},
		{"/v1/iso-test123/smart-attributes-group/group-test123", "/v1/<isolationID>/smart-attributes-group/<groupID>"},
		{"/v2/iso-test123/collections", "/v2/<isolationID>/collections"},
		{"/v2/iso-test123/collections/col-test123", "/v2/<isolationID>/collections/<collectionID>"},
		{"/v2/iso-test123/collections/col-test123/documents/doc-test123/chunks", "/v2/<isolationID>/collections/<collectionID>/documents/<documentID>/chunks"},
		{"/v2/iso-test123/collections/col-test123/find-documents", "/v2/<isolationID>/collections/<collectionID>/find-documents"},
	}

	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest("GET", tt.incomingPath, nil))
		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, tt.expected, recorder.Body.String())
	}
}
