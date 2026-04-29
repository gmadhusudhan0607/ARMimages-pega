/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func PatchDocument(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "patch-document")
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

	docID := c.Param(docIDParamName)
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", docIDParamName)})
		return
	}

	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Failed to read request body: %s", readErr.Error())})
		return
	}
	var req documents.PatchDocumentRequest
	if err = json.Unmarshal(bodyBytes, &req); err != nil {
		msg := fmt.Sprintf("Invalid request. Failed to bind request [request.body: %s]: %s", string(bodyBytes), err.Error())
		logger.Warn("Invalid request",
			zap.String("details", msg),
		)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	// Validate that at least one field (attributes or status) is provided
	if len(req.Attributes) == 0 && req.Status == nil {
		msg := "Invalid request. At least one of 'attributes' or 'status' must be provided"
		logger.Warn(msg)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	// Validate status value if provided
	if req.Status != nil {
		if !slices.Contains(resources.AllowedDocumentStatuses(), *req.Status) {
			msg := fmt.Sprintf("Invalid status value: %s. Allowed values: %v", *req.Status, resources.AllowedDocumentStatuses())
			logger.Warn(msg,
				zap.String("providedStatus", *req.Status),
				zap.Strings("allowedStatuses", resources.AllowedDocumentStatuses()),
			)
			c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
			return
		}
	}

	// Use transaction to ensure atomicity of updates
	tx, err := dbConn.(db.Database).GetConn().BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to begin transaction: %s", err)})
		return
	}
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				logger.Error("rollback failed", zap.Error(rbErr))
			}
		}
	}()

	docMgr := documents.NewManagerTx(tx, nil, isolationID, collectionName, logger)

	// Check if document exists before any updates
	exists, err := docMgr.DocumentExists(ctx, docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error checking document existence: %s", err)})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": fmt.Sprintf("document '%s' not found", docID)})
		return
	}

	// Update status if provided
	if req.Status != nil {
		errorMsg := ""
		if req.ErrorMessage != nil {
			errorMsg = *req.ErrorMessage
		}
		tableDoc := db.GetTableDoc(isolationID, collectionName)
		err = dbConn.(db.Database).UpsertDocStatusTx(ctx, tx, tableDoc, docID, *req.Status, errorMsg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while updating the document status: %s", err)})
			return
		}
	}

	// Update attributes if provided
	// Note: SetAttributes internally re-checks documentExists — redundant but harmless
	// because both checks run in the same transaction. Removing it would require auditing
	// all other SetAttributes callers, so we tolerate the extra round-trip.
	if len(req.Attributes) > 0 {
		err = docMgr.SetAttributes(ctx, docID, req.Attributes)
		if err != nil {
			if errors.Is(err, documents.ErrDocumentNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": fmt.Sprintf("document '%s' not found", docID)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while patching the document attributes: %s", err)})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to commit transaction: %s", err)})
		return
	}
	committed = true

	c.Status(http.StatusOK)
}
