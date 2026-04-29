/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	errorshelpers "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/errors"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/factory"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const QueryEmbeddingTimeoutEnvVar = "QUERY_EMBEDDING_TIMEOUT_MS"
const QueryEmbeddingMaxRetriesEnvVar = "QUERY_EMBEDDING_MAX_RETRIES"

const defaultQueryEmbeddingTimeout = 5000 * time.Millisecond
const defaultQueryEmbeddingMaxRetries = 1

func initQueryHttpClientConfig() *http_client.HTTPClientConfig {
	timeout := defaultQueryEmbeddingTimeout
	retries := defaultQueryEmbeddingMaxRetries

	if tStr := helpers.GetEnvOrDefault(QueryEmbeddingTimeoutEnvVar, ""); tStr != "" {
		if tVal, err := strconv.Atoi(tStr); err == nil && tVal > 0 {
			timeout = time.Duration(tVal) * time.Millisecond
		}
	}
	if rStr := helpers.GetEnvOrDefault(QueryEmbeddingMaxRetriesEnvVar, ""); rStr != "" {
		if rVal, err := strconv.Atoi(rStr); err == nil && rVal >= 0 {
			retries = rVal
		}
	}
	logger := log.GetNamedLogger(serviceName)
	logger.Info("Query embedding HTTP client configuration", zap.Int64("timeout_ms", timeout.Milliseconds()), zap.Int("max_retries", retries))

	return &http_client.HTTPClientConfig{
		Timeout:    timeout,
		MaxRetries: retries,
	}
}

var queryHttpClientConfig = initQueryHttpClientConfig()

func getEmbedderTimeoutErrorMessage() string {
	return fmt.Sprintf(
		"LLM Response timeout. Failed to get embedding for question during %d milliseconds . Max retries (%d) reached",
		queryHttpClientConfig.Timeout.Milliseconds(), queryHttpClientConfig.MaxRetries,
	)
}

func QueryDocuments(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "query-documents")
	defer span.End()

	var docs []*documents.DocumentQueryResponse
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

	dbConn, ok := c.Get(middleware.DBConnectionSearch)
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

	var query documents.QueryDocumentsRequest
	if err = c.BindJSON(&query); err != nil {
		bodyBytes, err1 := io.ReadAll(c.Request.Body)
		if err1 != nil {
			bodyBytes = []byte(fmt.Sprintf("error reading request body: %v", err))
		}
		msg := fmt.Sprintf("Invalid request. Failed to bind request [request.body: %v]: %s", bodyBytes, err.Error())
		logger.Warn("Invalid request",
			zap.String("details", msg),
		)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	// Set valid limit
	if query.Limit <= 0 {
		query.Limit = math.MaxInt32
	}

	// APIv1 backward compatibility:
	if len(query.Filters.SubFilters) > 0 {
		for idx := range query.Filters.SubFilters {
			if query.Filters.SubFilters[idx].Operator == "" {
				query.Filters.SubFilters[idx].Operator = "in"
			}
		}
	}

	embProfile := factory.DefaultEmbeddingProfileID
	a, err := factory.CreateTextEmbedder(dbConn.(db.Database), isolationID, collectionName, embProfile, queryHttpClientConfig, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to init embedding client: %s", err.Error())})
		return
	}

	mgr := documents.NewManager(dbConn.(db.Database), a, isolationID, collectionName, logger)

	// Use new schema (FindDocuments4) only if both conditions are met:
	// 1. Legacy attribute IDs are not forced via env var
	// 2. Attribute replication to v0.19.0 is completed
	replicationCompleted := sql.IsAttributeReplicationCompleted(dbConn.(db.Database))

	if !helpers.UseLegacyAttributesIDs() && replicationCompleted {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "completed")
		docs, err = mgr.FindDocuments4(ctx, &query)
	} else {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "incompleted")
		docs, err = mgr.FindDocuments2(ctx, &query)
	}
	if err != nil {
		if errorshelpers.IsTimeout(err) {
			// Return 504 with custom timeout message
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":    "504",
				"message": getEmbedderTimeoutErrorMessage(),
			})
			return
		}
		// All other errors return 500
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "500",
			"message": fmt.Sprintf("error while querying documents: %s", err.Error()),
		})
		return
	}

	// Differences in floating-point behavior are particularly noticeable on different architectures (e.g., x86 vs. ARM) or even different CPU models within the same architecture.
	// To avoid such issues, we cut off the precision if env variable PGVECTOR_DISTANCE_PRECISION is set.
	if helpers.IsCutOffDistancePrecisionEnabled() {
		for i := range docs {
			docs[i].Distance = helpers.CutOffDistancePrecision(docs[i].Distance)
		}
	}

	logger.Info("returned items",
		zap.Int("count", len(docs)),
	)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(docs))
	c.JSON(http.StatusOK, docs)
}

func QueryChunks(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "query-chunks")
	defer span.End()

	var chs []*embedings.Chunk
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

	dbConn, ok := c.Get(middleware.DBConnectionSearch)
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

	startApiTime := time.Now()
	if helpers.IsLogPerformanceTrace() {
		logger.Info("x-genai-vs-api-time-start-ms",
			zap.Int64("start_ms", startApiTime.UnixMilli()),
		)
	}

	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)

	var query embedings.QueryChunksRequest
	if err = c.BindJSON(&query); err != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			bodyBytes = []byte(fmt.Sprintf("error reading request body: %v", err))
		}
		msg := fmt.Sprintf("Invalid request. Failed to bind request [request.body: %v]: %s", bodyBytes, err.Error())
		logger.Warn("Invalid request",
			zap.String("details", msg),
		)
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	// LEGACY:
	// Use 'retrieveVector' from parameters if provided
	retrieveVectorParam := c.DefaultQuery("retrieveVector", "")
	if retrieveVectorParam != "" {
		retrieveVector, err := strconv.ParseBool(c.DefaultQuery("retrieveVector", "false"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error parsing retrieveVector parameter: %s", err)})
			return
		}
		query.RetrieveVector = retrieveVector
	}

	// Set valid limit
	if query.Limit <= 0 {
		query.Limit = math.MaxInt32
	}

	embProfile := factory.DefaultEmbeddingProfileID
	a, err := factory.CreateTextEmbedder(dbConn.(db.Database), isolationID, collectionName, embProfile, queryHttpClientConfig, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("failed to init embedding client: %s", err)})
		return
	}

	mgr := embedings.NewManager(dbConn.(db.Database), a, isolationID, collectionName, logger)

	// Use new schema (FindChunks4) only if both conditions are met:
	// 1. Legacy attribute IDs are not forced via env var
	// 2. Attribute replication to v0.19.0 is completed
	replicationCompleted := sql.IsAttributeReplicationCompleted(dbConn.(db.Database))

	if !helpers.UseLegacyAttributesIDs() && replicationCompleted {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "completed")
		chs, err = mgr.FindChunks4(ctx, &query)
	} else {
		// Header Used in integration tests
		c.Header(headers.DbSchemaMigration, "incompleted")
		chs, err = mgr.FindChunks2(ctx, &query)
	}
	if err != nil {
		if errorshelpers.IsTimeout(err) {
			// Return 504 with custom timeout message
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":    "504",
				"message": getEmbedderTimeoutErrorMessage(),
			})
			return
		}
		// All other errors return 500
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "500",
			"message": fmt.Sprintf("error while querying documents: %s", err.Error()),
		})
		return
	}

	logger.Debug("received input",
		zap.Any("query", query),
	)

	// we don't want to return nil in case there aren't any chunks returned
	// it's preferred to return empty slice of chunks in this case
	if chs == nil {
		chs = []*embedings.Chunk{}
	}

	// Differences in floating-point behavior are particularly noticeable on different architectures (e.g., x86 vs. ARM) or even different CPU models within the same architecture.
	// To avoid such issues, we cut off the precision if env variable PGVECTOR_DISTANCE_PRECISION is set.
	if helpers.IsCutOffDistancePrecisionEnabled() {
		for i := range chs {
			chs[i].Distance = helpers.CutOffDistancePrecision(chs[i].Distance)
		}
	}

	endApiTime := time.Now()
	durationApi := endApiTime.Sub(startApiTime)
	if helpers.IsLogPerformanceTrace() {
		logger.Info("x-genai-vs-api-time-end-ms",
			zap.Int64("end_ms", endApiTime.UnixMilli()),
		)
		logger.Info("x-genai-vs-api-duration-ms",
			zap.Int64("duration_ms", durationApi.Milliseconds()),
		)
	}

	logger.Info("returned items",
		zap.Int("count", len(chs)),
	)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(chs))
	c.JSON(http.StatusOK, chs)
}

func QueryAttributes(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "query-attributes")
	defer span.End()

	var attrs []attributes.Attribute
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

	var attReq attributes.RetrieveAttributesRequest
	if c.Request.ContentLength > 0 {
		err = c.BindJSON(&attReq)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error unmarshalling the request: %s", err)})
		return
	}

	mgr := attributes.NewManager(dbConn.(db.Database), isolationID, collectionName, logger)
	attrs, err = mgr.FindAttributes(ctx, attReq.RetrieveAttributes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error processing the request: %s", err)})
		return
	}

	logger.Info("returned items",
		zap.Int("count", len(attrs)),
	)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(attrs))
	c.JSON(http.StatusOK, attrs)
}
