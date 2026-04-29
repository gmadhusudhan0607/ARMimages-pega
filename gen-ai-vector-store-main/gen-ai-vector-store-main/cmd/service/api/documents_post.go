/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func getFiltersFromRequest(c *gin.Context) ([]attributes.AttributeFilter, error) {
	// Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		_ = c.Request.Body.Close()
		return nil, fmt.Errorf("could not read request body: %w", err)
	}

	// Recreate body for further processing
	errClose := c.Request.Body.Close()
	if errClose != nil {
		return nil, fmt.Errorf("could not close request body: %w", errClose)
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// trim body before analyze
	body := regexp.MustCompile(`\s+`).ReplaceAllString(string(bodyBytes), " ")
	body = strings.TrimSpace(body)

	var attrFilters []attributes.AttributeFilter
	if body == "" {
		return attrFilters, nil
	}

	// Validate body format
	if (!strings.HasPrefix(body, "[")) || (!strings.HasSuffix(body, "]")) {
		return nil, fmt.Errorf("invalid body format: %s", body)
	}

	// parse filters
	err = c.BindJSON(&attrFilters)
	if err != nil {
		return nil, fmt.Errorf("could not parse request body: %w", err)
	}
	return attrFilters, nil
}

func RetrieveDocuments(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "retrieve-documents")
	defer span.End()

	startTime := time.Now()
	var docs []documents.Document

	docStatus := c.Query("status")

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

	attrFilters, err := getFiltersFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error parsing the request: %s", err)})
		return
	}

	docMgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionName, logger)

	// Use new schema (ListDocuments3) only if both conditions are met:
	// 1. Legacy attribute IDs are not forced via env var
	// 2. Attribute replication to v0.19.0 is completed
	replicationCompleted := sql.IsAttributeReplicationCompleted(dbConn.(db.Database))

	if !helpers.UseLegacyAttributesIDs() && replicationCompleted {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "completed")
		docs, err = docMgr.ListDocuments3(ctx, docStatus, attrFilters)
	} else {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "incompleted")
		docs, err = docMgr.ListDocuments2(ctx, docStatus, attrFilters)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error processing the request: %s", err)})
		return
	}

	logger.Info("returned items",
		zap.Int("count", len(docs)),
	)
	helpers.LogTruncated(logger, "returning documents", 20, docs)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(docs))
	c.JSON(http.StatusOK, docs)
}
