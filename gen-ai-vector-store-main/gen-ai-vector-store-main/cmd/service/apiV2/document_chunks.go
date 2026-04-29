/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package apiV2

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/pagination"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GetDocumentChunksResponse struct {
	DocumentID string                `json:"documentID,omitempty"`
	Chunks     []DocumentChunk       `json:"chunks" binding:"required"`
	Pagination pagination.Pagination `json:"pagination,omitempty"`
}

type DocumentChunk struct {
	ID         string                `json:"id,omitempty"`
	Content    string                `json:"content"`
	Attributes attributes.Attributes `json:"attributes,omitempty"`
}

func GetDocumentChunks(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "get-document-chunks")
	defer span.End()

	startTime := time.Now()

	isolationID, err := getIsolationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": err.Error()})
		return
	}
	collectionID, err := getCollectionID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": err.Error()})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logResponse(logger, c, startTime)
	logRequest(logger, c)

	documentID, err := getDocumentID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": err.Error()})
		return
	}

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to get db schema metadata: %s", err)})
		return
	}
	if !schemaMgr.IsolationExists(isolationID) {
		msg := fmt.Sprintf("isolation '%s' not found. Please install GenAIVectorStoreIsolation SCE first", isolationID)
		if helpers.IsIsolationAutoCreationEnabled() {
			msg = fmt.Sprintf("isolation '%s' not found. Please insert data before retrieving", isolationID)
		}
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": msg})
		return
	}

	if !schemaMgr.CollectionExists(isolationID, collectionID) {
		msg := fmt.Sprintf("collection '%s' not found. Please insert data before retrieving", collectionID)
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": msg})
		return
	}

	// get pagination parameters
	cursor, limit, err := pagination.GetPaginationParameters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": err.Error()})
		return
	}

	var response GetDocumentChunksResponse
	embMgr := embedings.NewManager(dbConn.(db.Database), nil, isolationID, collectionID, logger)
	chunks, itemsTotal, itemsLeft, err := embMgr.GetDocumentChunksPaginated(ctx, documentID, cursor, limit)
	if err != nil {
		switch {
		case errors.Is(err, documents.ErrDocumentNotFound):
			c.JSON(http.StatusNotFound, response)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error processing the request: %s", err)})
		}
		return
	}

	// build response
	response.DocumentID = documentID
	response.Pagination = pagination.CalculatePagination(chunks, limit, itemsTotal, itemsLeft, func(ch *embedings.Chunk) string { return ch.ID })
	for _, chunk := range chunks {
		docChunk := DocumentChunk{
			ID:         chunk.ID,
			Content:    chunk.Content,
			Attributes: chunk.Attributes,
		}
		response.Chunks = append(response.Chunks, docChunk)
	}

	logger.Debug("returned document", zap.String("document", helpers.ToTruncatedString(response)))
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(response.Chunks))
	c.JSON(http.StatusOK, response)
}
