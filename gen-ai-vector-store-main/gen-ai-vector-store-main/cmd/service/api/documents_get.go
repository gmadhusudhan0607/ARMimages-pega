/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GetDocument(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "get-document")
	defer span.End()

	startTime := time.Now()
	//read parameters
	isolationID, collectionName, err := getIsolationIDAndCollectionName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error parsing the request: %s", err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logResponse(logger, c, startTime)
	logRequest(logger, c)

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, collectionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to get db metadata: %s", err)})
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
	if !schemaMgr.CollectionExists(isolationID, collectionName) {
		msg := fmt.Sprintf("collection '%s' not found. Please insert data before retrieving", collectionName)
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": msg})
		return
	}

	docID := c.Param(docIDParamName)
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", docIDParamName)})
		return
	}

	var docResponse documents.GetDocumentResponse
	mgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionName, logger)
	doc, err := mgr.GetDocument2(ctx, docID)
	if err != nil {
		switch {
		case errors.Is(err, documents.ErrDocumentNotFound):
			c.JSON(http.StatusNotFound, docResponse)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error processing the request: %s", err)})
			return
		}
		return
	}
	docResponse.ID = doc.ID
	docResponse.Status = doc.Status
	docResponse.Error = doc.Error
	logger.Debug("returned document",
		zap.String("doc", helpers.ToTruncatedString(docResponse)),
	)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(1)
	c.JSON(http.StatusOK, docResponse)
}
