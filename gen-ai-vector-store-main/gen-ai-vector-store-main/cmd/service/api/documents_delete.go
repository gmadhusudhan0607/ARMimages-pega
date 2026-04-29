/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func DeleteDocument(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "delete-document")
	defer span.End()

	startTime := time.Now()
	// read parameters
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

	// read document ID
	docID := c.Param(docIDParamName)
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", docIDParamName)})
		return
	}

	rowsAffected := int64(0)
	mgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionName, logger)
	rowsAffected, err = mgr.DeleteDocument2(ctx, docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while deleting documents: %s", err)})
		return
	}

	c.JSON(http.StatusOK, DeleteDocumentsResponse{DeletedDocuments: rowsAffected})
}

func DeleteDocuments(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "delete-documents")
	defer span.End()

	startTime := time.Now()
	// read parameters
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

	// parse request body
	var delDocReq = &documents.DeleteDocumentRequest{}

	if err = c.BindJSON(&delDocReq.Items); err != nil {
		msg := fmt.Sprintf("Invalid request. Failed to bind request 'items' [%v]: %s", c.Request, err.Error())
		logger.Warn("Invalid request",
			zap.String("details", msg),
		)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	if len(delDocReq.Items) == 0 {
		c.JSON(http.StatusOK, map[string]int64{"deletedDocuments": 0})
		return
	}

	rowsAffected := int64(0)
	mgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionName, logger)

	// Use new schema (DeleteDocumentsByFilters3) only if both conditions are met:
	// 1. Legacy attribute IDs are not forced via env var
	// 2. Attribute replication to v0.19.0 is completed
	replicationCompleted := sql.IsAttributeReplicationCompleted(dbConn.(db.Database))

	if !helpers.UseLegacyAttributesIDs() && replicationCompleted {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "completed")
		rowsAffected, err = mgr.DeleteDocumentsByFilters3(ctx, delDocReq.Items)
	} else {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "incompleted")
		rowsAffected, err = mgr.DeleteDocumentsByFilters(ctx, delDocReq.Items)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error processing the request: %s", err)})
		return
	}

	c.JSON(http.StatusOK, DeleteDocumentsResponse{DeletedDocuments: rowsAffected})
}

func DeleteDocumentById(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "delete-document-by-id")
	defer span.End()

	startTime := time.Now()
	// read parameters
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

	// Get document ID from request body
	var delDocReq = &documents.DeleteDocumentByIdRequest{}

	if err = c.BindJSON(&delDocReq); err != nil {
		bodyBytes, err2 := io.ReadAll(c.Request.Body)
		if err2 != nil {
			bodyBytes = []byte(fmt.Sprintf("error reading request body: %v", err))
		}
		msg := fmt.Sprintf("Invalid request. Failed to bind request [request.body: %v]: %s", bodyBytes, err.Error())
		logger.Warn(msg)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}
	if delDocReq.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "document ID is required"})
		return
	}

	rowsAffected := int64(0)
	mgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionName, logger)
	rowsAffected, err = mgr.DeleteDocument2(ctx, delDocReq.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while deleting documents: %s", err)})
		return
	}

	c.JSON(http.StatusOK, DeleteDocumentsResponse{DeletedDocuments: rowsAffected})
}
