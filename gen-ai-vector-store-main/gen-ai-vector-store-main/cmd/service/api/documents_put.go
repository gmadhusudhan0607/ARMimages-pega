/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/factory"
	errorshelper "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/errors"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/smart_chunking"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	scClientOnce sync.Once
	scClient     smart_chunking.SmartChunkingClient
	scClientErr  error
)

func getSmartChunkingClient() (smart_chunking.SmartChunkingClient, error) {
	scClientOnce.Do(func() {
		serviceURL := helpers.GetEnvOrDefault("GENAI_SMART_CHUNKING_SERVICE_URL", "")
		scClient, scClientErr = smart_chunking.NewTracedSmartChunkingClient(serviceURL)
		if scClientErr != nil {
			logger := log.GetNamedLogger("documents_put")
			logger.Error("failed to create smart-chunking client",
				zap.String("service_url", serviceURL),
				zap.Error(scClientErr),
			)
		}
	})
	return scClient, scClientErr
}

func setAttributesTypesIfNotSet(doc *documents.PutDocumentRequest) {
	for idx := range doc.Attributes {
		if doc.Attributes[idx].Type == "" {
			doc.Attributes[idx].Type = "string"
		}
		if doc.Attributes[idx].Kind == "" {
			doc.Attributes[idx].Kind = "static"
		}
	}
	for idx := range doc.Chunks {
		if doc.Chunks[idx].Attributes != nil {
			for attrIdx := range doc.Chunks[idx].Attributes {
				if doc.Chunks[idx].Attributes[attrIdx].Type == "" {
					doc.Chunks[idx].Attributes[attrIdx].Type = "string"
				}
				if doc.Chunks[idx].Attributes[attrIdx].Kind == "" {
					doc.Chunks[idx].Attributes[attrIdx].Kind = "static"
				}
			}
		}
	}
}

// rejectQueryParams checks that none of the given parameter names appear as URL query params.
// These parameters must be provided in the request body, not the URL.
// Writes a 400 response and returns false if a rejected parameter is found.
func rejectQueryParams(c *gin.Context, params ...string) bool {
	for _, param := range params {
		if c.Query(param) != "" {
			msg := fmt.Sprintf("%s parameter is not allowed in the URL path. Please provide it in the request body", param)
			c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
			return false
		}
	}
	return true
}

// getDB retrieves a database connection from the gin context by key.
// Writes an HTTP error response and returns nil on failure.
func getDB(c *gin.Context, contextKey string, isolationID string) db.Database {
	dbConnAny, ok := c.Get(contextKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return nil
	}
	dbConn, ok := dbConnAny.(db.Database)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error casting database connection for isolation '%s'", isolationID)})
		return nil
	}
	return dbConn
}

// getIngestDB retrieves the ingest database connection from the gin context.
func getIngestDB(c *gin.Context, isolationID string) db.Database {
	return getDB(c, middleware.DBConnectionIngest, isolationID)
}

// getGenericDB retrieves the generic database connection from the gin context.
func getGenericDB(c *gin.Context, isolationID string) db.Database {
	return getDB(c, middleware.DBConnectionGeneric, isolationID)
}

// ensureIsolation verifies the isolation exists in the DB, auto-creating it when enabled.
// Writes an HTTP error response and returns false on failure.
func ensureIsolation(c *gin.Context, ctx context.Context, dbConn db.Database, logger *zap.Logger, isolationID string) bool {
	schemaMgr, err := schema.NewVsSchemaManager(dbConn, logger).Load(ctx, isolationID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get db metadata for isolation '%s': %v", isolationID, err)})
		return false
	}
	if !schemaMgr.IsolationExists(isolationID) {
		if !helpers.IsIsolationAutoCreationEnabled() {
			msg := fmt.Sprintf("isolation '%s' not found. Please install GenAIVectorStoreIsolation SCE first", isolationID)
			c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": msg})
			return false
		}
		maxStorageSize := helpers.GetIsolationAutoCreationMaxStorageSize()
		isoMgr := isolations.NewManager(dbConn, logger)
		err = isoMgr.CreateIsolation(ctx, isolationID, maxStorageSize, "")
		if err != nil {
			msg := fmt.Sprintf("failed to create isolation '%s': %s", isolationID, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": msg})
			return false
		}
	}
	return true
}

// submitFileJob submits a file (or text) to the Smart Chunking /job API and writes the 202 response.
// Writes an HTTP error response and returns false on failure.
func submitFileJob(
	c *gin.Context,
	ctx context.Context,
	logger *zap.Logger,
	isolationID string,
	collectionName string,
	authToken string,
	fileReader io.Reader,
	fileName string,
	documentID string,
	docAttributes []attributes.Attribute,
	docMetadata *documents.DocumentMetadata,
) bool {
	client, err := getSmartChunkingClient()
	if err != nil {
		logger.Error("smart-chunking client not available", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"code": "502", "message": fmt.Sprintf("smart-chunking client not available: %s", err.Error())})
		return false
	}

	// Coerce nil Values slices to empty slice so SC's Pydantic model
	// receives "value": [] instead of "value": null.
	// Nil arises when Pega sends {"value": null} for an unpopulated attribute.
	var nilValueAttrNames []string
	for i := range docAttributes {
		if docAttributes[i].Values == nil {
			nilValueAttrNames = append(nilValueAttrNames, docAttributes[i].Name)
			docAttributes[i].Values = attributes.AttrValues{}
		}
	}
	if len(nilValueAttrNames) > 0 {
		logger.Debug("Coerced document attributes with null value list to empty list to prevent Smart Chunking validation error",
			zap.String("isolationID", isolationID),
			zap.String("collectionName", collectionName),
			zap.String("documentID", documentID),
			zap.Int("nilAttributeCount", len(nilValueAttrNames)),
			zap.Strings("nilAttributeNames", nilValueAttrNames),
		)
	}

	var extractionOpts *smart_chunking.ExtractionOptions
	indexingOpts := &smart_chunking.IndexingOptions{
		CollectionName:     collectionName,
		DocumentID:         documentID,
		DocumentAttributes: docAttributes,
	}
	var chunkingOpts *smart_chunking.ChunkingOptions
	if docMetadata != nil {
		if docMetadata.EnableOCR != nil {
			extractionOpts = &smart_chunking.ExtractionOptions{EnableOCR: docMetadata.EnableOCR}
		}
		indexingOpts.EmbeddingAttributes = docMetadata.StaticEmbeddingAttributes
		if docMetadata.EnableSmartAttribution != nil {
			chunkingOpts = &smart_chunking.ChunkingOptions{
				EnableSmartAttribution: docMetadata.EnableSmartAttribution,
				ExcludeSmartAttributes: docMetadata.ExcludeSmartAttributes,
			}
		}
		if docMetadata.EmbedSmartAttributes != nil {
			indexingOpts.EmbedSmartAttributes = docMetadata.EmbedSmartAttributes
		}
	}

	jobRequest := smart_chunking.JobRequestOptions{
		Tasks: []string{"extraction", "chunking", "indexing"},
		TaskOptions: &smart_chunking.JobTaskOptions{
			Extraction: extractionOpts,
			Chunking:   chunkingOpts,
			Indexing:   indexingOpts,
		},
	}

	scMgr := smart_chunking.NewManager(client, isolationID, collectionName)
	_, err = scMgr.SubmitJob(ctx, authToken, fileReader, fileName, jobRequest)
	if err != nil {
		logger.Error("failed to submit job to smart-chunking", zap.Error(err))

		var scErr *smart_chunking.ServiceError
		if errors.As(err, &scErr) && scErr.StatusCode >= 400 && scErr.StatusCode < 500 {
			c.JSON(scErr.StatusCode, gin.H{"code": fmt.Sprintf("%d", scErr.StatusCode), "message": scErr.Body})
			return false
		}

		if errorshelper.IsTimeout(err) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"code": "504", "message": fmt.Sprintf("smart-chunking service (%s) request timed out: %s", scMgr.GetServiceURL(), err.Error())})
			return false
		}
		msg := fmt.Sprintf("smart-chunking service (%s) is unavailable: %s", scMgr.GetServiceURL(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"code": "502", "message": msg})
		return false
	}

	c.JSON(http.StatusAccepted, gin.H{
		"documentID": documentID,
		"status":     "IN_PROGRESS",
	})
	return true
}

// PutDocumentFile forwards a file upload to Smart Chunking /job API.
// VS acts as a thin proxy: validate, forward, return 202 immediately.
func PutDocumentFile(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "put-document-file")
	defer span.End()

	startTime := time.Now()
	logger := log.GetLoggerFromContext(ctx)
	defer logResponse(logger, c, startTime)
	logRequest(logger, c)

	if !rejectQueryParams(c, "documentID", "documentAttributes", "documentFile") {
		return
	}

	isolationID, collectionName, err := getIsolationIDAndCollectionName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	dbConn := getDB(c, middleware.DBConnectionGeneric, isolationID)
	if dbConn == nil {
		return
	}
	if !ensureIsolation(c, ctx, dbConn, logger, isolationID) {
		return
	}

	if err := c.Request.ParseMultipartForm(0); err != nil {
		msg := fmt.Sprintf("Invalid request. failed to parse multipart form: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	documentID := c.Request.FormValue("documentID")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentID parameter is required"})
		return
	}

	documentAttributesParam := c.Request.FormValue("documentAttributes")
	if documentAttributesParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentAttributes parameter is required. Must be not empty"})
		return
	}
	if !json.Valid([]byte(documentAttributesParam)) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "failed to parse documentAttributes parameter. Valid JSON required (List of attributes)"})
		return
	}
	var docAttributes []attributes.Attribute
	if err := json.Unmarshal([]byte(documentAttributesParam), &docAttributes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("failed to unmarshal documentAttributes parameter: %s", err.Error())})
		return
	}
	if len(docAttributes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentAttributes must not be empty"})
		return
	}

	var docMetadata *documents.DocumentMetadata
	documentMetadataParam := c.Request.FormValue("documentMetadata")
	if documentMetadataParam != "" {
		docMetadata = &documents.DocumentMetadata{}
		if err := json.Unmarshal([]byte(documentMetadataParam), docMetadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("failed to unmarshal documentMetadata parameter: %s", err.Error())})
			return
		}
	}

	documentFile, headers, err := c.Request.FormFile("documentFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("documentFile parameter is required: %s", err.Error())})
		return
	}
	defer documentFile.Close()

	if headers.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentFile contains no data"})
		return
	}

	scFileName := filepath.Base(headers.Filename)
	if scFileName == "" || scFileName == "." {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentFile has an invalid filename"})
		return
	}
	// Legacy form parameter: overrides documentMetadata.enableOCR when explicitly set.
	if ocrParam := c.Request.FormValue("enableOCR"); ocrParam != "" {
		if docMetadata == nil {
			docMetadata = &documents.DocumentMetadata{}
		}
		enableOCR := strings.EqualFold(ocrParam, "true")
		docMetadata.EnableOCR = &enableOCR
	}

	var enableOCRFlag *bool
	if docMetadata != nil {
		enableOCRFlag = docMetadata.EnableOCR
	}
	logger.Debug("File upload request parsed successfully — all form parameters except file binary",
		zap.String("isolationID", isolationID),
		zap.String("collectionName", collectionName),
		zap.String("documentID", documentID),
		zap.String("fileName", scFileName),
		zap.Int64("fileSizeBytes", headers.Size),
		zap.Boolp("enableOCR", enableOCRFlag),
		zap.String("documentAttributes", documentAttributesParam),
		zap.String("documentMetadata", documentMetadataParam),
	)

	_ = submitFileJob(c, ctx, logger, isolationID, collectionName, c.GetHeader("Authorization"), documentFile, scFileName, documentID, docAttributes, docMetadata)
}

// PutDocumentFileText forwards text content to Smart Chunking /job API.
// VS acts as a thin proxy: validate, forward, return 202 immediately.
func PutDocumentFileText(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "put-document-file-text")
	defer span.End()

	startTime := time.Now()
	logger := log.GetLoggerFromContext(ctx)
	defer logResponse(logger, c, startTime)
	logRequest(logger, c)

	if !rejectQueryParams(c, "documentID", "documentAttributes", "documentContent") {
		return
	}

	isolationID, collectionName, err := getIsolationIDAndCollectionName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	dbConn := getDB(c, middleware.DBConnectionGeneric, isolationID)
	if dbConn == nil {
		return
	}
	if !ensureIsolation(c, ctx, dbConn, logger, isolationID) {
		return
	}

	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Failed to read request body: %s", readErr.Error())})
		return
	}
	var docTxt documents.PutFileTextRequest
	if err := json.Unmarshal(bodyBytes, &docTxt); err != nil {
		msg := fmt.Sprintf("Invalid request. Failed to bind request [request.body: %s]: %s", string(bodyBytes), err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}
	if docTxt.DocumentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentID parameter is required"})
		return
	}
	if len(docTxt.DocumentAttributes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentAttributes parameter is required. Must be not empty"})
		return
	}
	if docTxt.DocumentContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": "documentContent parameter is required"})
		return
	}

	logger.Debug("File-text upload request parsed successfully — all body parameters except document content",
		zap.String("isolationID", isolationID),
		zap.String("collectionName", collectionName),
		zap.String("documentID", docTxt.DocumentID),
		zap.Int("documentContentLength", len(docTxt.DocumentContent)),
		zap.String("documentAttributes", helpers.ToTruncatedString(docTxt.DocumentAttributes)),
		zap.String("documentMetadata", helpers.ToTruncatedString(docTxt.DocumentMetadata)),
	)

	scFileName := docTxt.DocumentID + ".txt"
	textReader := bytes.NewReader([]byte(docTxt.DocumentContent))

	_ = submitFileJob(c, ctx, logger, isolationID, collectionName, c.GetHeader("Authorization"), textReader, scFileName, docTxt.DocumentID, docTxt.DocumentAttributes, docTxt.DocumentMetadata)
}

// PutDocument handles the PUT /documents endpoint (used by direct callers and SC indexing task).
func PutDocument(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "put-document")
	defer span.End()

	startTime := time.Now()
	consistencyLevel := c.DefaultQuery("consistencyLevel", indexer.ConsistencyLevelEventual)

	isolationID, collectionName, err := getIsolationIDAndCollectionName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logResponse(logger, c, startTime)
	logRequest(logger, c)

	// Use the generic pool for quick, request-scoped DB operations (isolation
	// checks, collection creation, status updates).  The ingestion pool is
	// passed to the indexer for heavy transactional work (sync processing and
	// background goroutines).  This separation prevents background goroutines
	// that hold ingestion connections from blocking new incoming requests.
	genericDB := getGenericDB(c, isolationID)
	if genericDB == nil {
		return
	}
	ingestDB := getIngestDB(c, isolationID)
	if ingestDB == nil {
		return
	}
	if !ensureIsolation(c, ctx, genericDB, logger, isolationID) {
		return
	}

	var doc documents.PutDocumentRequest
	if err := c.BindJSON(&doc); err != nil {
		bodyBytes, err1 := io.ReadAll(c.Request.Body)
		if err1 != nil {
			bodyBytes = []byte(fmt.Sprintf("error reading request body: %v", err1))
		}
		msg := fmt.Sprintf("Invalid request. Failed to bind request [body: %v]: %s", bodyBytes, err.Error())
		logger.Warn(msg)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	for idx, chunk := range doc.Chunks {
		chunkSize := len(chunk.Content)
		if chunkSize > MaxChunkContentSize {
			msg := fmt.Sprintf("max content size is %d (chunk %d size is %d)", MaxChunkContentSize, idx, chunkSize)
			c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
			return
		}
	}
	setAttributesTypesIfNotSet(&doc)

	//create databases for given isolation & collection if they do not exist
	colMgr := collections.NewManager(genericDB, isolationID, logger)
	collectionExists, err := colMgr.CollectionExists(ctx, collectionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while checking if collection exists for isolation '%s': %v", isolationID, err)})
		return
	}
	if !collectionExists {
		if _, err = colMgr.CreateCollection(ctx, collectionName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while creating collection for isolation '%s': %v", isolationID, err)})
			return
		}
	}

	var extraKinds []string
	if doc.Metadata != nil {
		for _, kind := range doc.Metadata.ExtraAttributesKinds {
			if !slices.Contains(attributes.ALLOWED_KINDS, kind) {
				c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Invalid value in extraAttributesKinds: %s", kind)})
				return
			}
		}
		extraKinds = doc.Metadata.ExtraAttributesKinds
	}

	tableDoc := db.GetTableDoc(isolationID, collectionName)

	// Empty chunks = register document placeholder with IN_PROGRESS status.
	// No processing rows are created, so the background worker ignores this document
	// and the status stays IN_PROGRESS until the next PUT with real chunks.
	// Both the status upsert and optional attribute update run in a single transaction.
	if len(doc.Chunks) == 0 {
		tx, txErr := genericDB.GetConn().BeginTx(ctx, nil)
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to begin transaction: %s", txErr)})
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

		// Create/upsert document placeholder with IN_PROGRESS status
		if err = genericDB.UpsertDocStatusTx(ctx, tx, tableDoc, doc.ID, resources.StatusInProgress, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while setting document '%s' status to IN_PROGRESS for isolation '%s': %v", doc.ID, isolationID, err)})
			return
		}

		// Apply attributes if provided
		if len(doc.Attributes) > 0 {
			attrMgr := attributes.NewManagerTx(tx, isolationID, collectionName, logger)
			docAttrIDs, uErr := attrMgr.UpsertAttributes2(ctx, doc.Attributes, extraKinds)
			if uErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to upsert attributes: %s", uErr)})
				return
			}

			docAttrsV2 := attributes.ConvertAttributesV1ToV2(doc.Attributes)
			if uErr = genericDB.UpdateDocAttributesTx(ctx, tx, tableDoc, doc.ID, docAttrIDs, docAttrsV2); uErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to update document attributes: %s", uErr)})
				return
			}
		}

		if cErr := tx.Commit(); cErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to commit: %s", cErr)})
			return
		}
		committed = true
		c.Status(http.StatusAccepted)
		return
	}

	// Non-empty chunks path: create embedder and index
	embProfile := factory.DefaultEmbeddingProfileID
	a, err := factory.CreateTextEmbedder(genericDB, isolationID, collectionName, embProfile, queryHttpClientConfig, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to init embedding client for isolation '%s': %v", isolationID, err)})
		return
	}

	// Set document status to IN_PROGRESS after validating all request parts (non-empty chunks path)
	err = genericDB.UpsertDocStatus(ctx, tableDoc, doc.ID, resources.StatusInProgress, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while setting document '%s' status to IN_PROGRESS for isolation '%s': %v", doc.ID, isolationID, err)})
		return
	}

	i := indexer.NewIndexer(ingestDB, genericDB, a, isolationID, collectionName, logger)
	err = i.Index(ctx, doc.ID, doc.Chunks, doc.Attributes, doc.Metadata, consistencyLevel, extraKinds)
	if err != nil {
		if errorshelper.IsTimeout(err) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":    "504",
				"message": "Operation timeout. Failed to complete embedding operation.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while embedding the chunks for isolation '%s': %v", isolationID, err)})
		return
	}

	if consistencyLevel == indexer.ConsistencyLevelStrong {
		c.Status(http.StatusCreated)
	} else {
		c.Status(http.StatusAccepted)
	}
}
