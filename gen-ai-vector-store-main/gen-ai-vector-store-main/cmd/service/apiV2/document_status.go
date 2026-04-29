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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/pagination"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FindDocumentsResponse struct {
	Documents  []DocumentStatus      `json:"documents" binding:"required"`
	Pagination pagination.Pagination `json:"pagination,omitempty"`
}

type DocumentStatus struct {
	DocumentID              string                `json:"documentID" binding:"required"`
	Status                  string                `json:"ingestionStatus,omitempty"`
	IngestionStart          string                `json:"ingestionTime,omitempty"`
	LastSuccessfulIngestion string                `json:"updateTime,omitempty"`
	ErrorMessage            string                `json:"errorMessage,omitempty"`
	ChunkStatus             map[string]int        `json:"chunkStatus,omitempty"`
	DocumentAttributes      []DocumentAttributeV2 `json:"documentAttributes,omitempty"`
}

type DocumentAttributeV2 struct {
	Name   string   `json:"name" binding:"required"`
	Values []string `json:"values" binding:"required"`
	Type   string   `json:"type,omitempty"`
}

func ConvertToDocumentAttributes(attrs []attributes.Attribute) []DocumentAttributeV2 {
	var docAttrs []DocumentAttributeV2
	for _, attr := range attrs {
		docAttrs = append(docAttrs, DocumentAttributeV2{
			Name:   attr.Name,
			Values: attr.Values,
			Type:   attr.Type,
		})
	}
	return docAttrs
}

// Refactored to align with get_status.go functionality
func FindDocuments(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "find-documents")
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

	// Get request body
	requestBody, err := getFindDocumentsBodyFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": err.Error()})
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

	var response FindDocumentsResponse
	docMgr := documents.NewManager(dbConn.(db.Database), nil, isolationID, collectionID, logger)

	// Use new schema (GetDocumentStatuses3) only if both conditions are met:
	// 1. Legacy attribute IDs are not forced via env var
	// 2. Attribute replication to v0.19.0 is completed
	replicationCompleted := sql.IsAttributeReplicationCompleted(dbConn.(db.Database))

	var statuses []documents.DocumentStatus
	var itemsTotal, itemsLeft int
	if !helpers.UseLegacyAttributesIDs() && replicationCompleted {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "completed")
		statuses, itemsTotal, itemsLeft, err = docMgr.GetDocumentStatuses3(ctx, requestBody.Filter.Status, requestBody.Fields, convertFilterToAttributes(requestBody.Filter.Attributes), cursor, limit)
	} else {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "incompleted")
		statuses, itemsTotal, itemsLeft, err = docMgr.GetDocumentStatuses(ctx, requestBody.Filter.Status, requestBody.Fields, convertFilterToAttributes(requestBody.Filter.Attributes), cursor, limit)
	}
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
	response.Pagination = pagination.CalculatePagination(
		statuses, limit, itemsTotal, itemsLeft, func(status documents.DocumentStatus) string { return status.DocumentID })

	for _, status := range statuses {
		docStatus := DocumentStatus{
			DocumentID:              status.DocumentID,
			Status:                  status.Status,
			ErrorMessage:            status.ErrorMessage,
			IngestionStart:          status.IngestionStart,
			LastSuccessfulIngestion: status.LastSuccessfulIngestion,
			ChunkStatus:             status.ChunkStatus,
			DocumentAttributes:      ConvertToDocumentAttributes(status.DocumentAttributes),
		}

		// Filter fields if the 'fields' attribute is provided
		if len(requestBody.Fields) > 0 {
			filteredDocStatus := DocumentStatus{}
			for _, field := range requestBody.Fields {
				switch field {
				case "documentID":
					filteredDocStatus.DocumentID = docStatus.DocumentID
				case "status":
					filteredDocStatus.Status = docStatus.Status
				case "errorMessage":
					filteredDocStatus.ErrorMessage = docStatus.ErrorMessage
				case "ingestionStart":
					filteredDocStatus.IngestionStart = docStatus.IngestionStart
				case "lastSuccessfulIngestion":
					filteredDocStatus.LastSuccessfulIngestion = docStatus.LastSuccessfulIngestion
				case "chunkStatus":
					filteredDocStatus.ChunkStatus = docStatus.ChunkStatus
				case "documentAttributes":
					filteredDocStatus.DocumentAttributes = docStatus.DocumentAttributes
				}
			}
			response.Documents = append(response.Documents, filteredDocStatus)
		} else {
			response.Documents = append(response.Documents, docStatus)
		}
	}

	logger.Debug("returned documents", zap.String("documents", helpers.ToTruncatedString(response)))
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(response.Documents))
	c.JSON(http.StatusOK, response)
}
