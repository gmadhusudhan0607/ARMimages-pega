/*
 * Copyright (c) 2024 Pegasystems Inc.
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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func PostCollection(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "put-document")
	defer span.End()

	startTime := time.Now()

	isolationID, err := getIsolationID(c)
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

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
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

	var createReq CollectionCreateRequest
	err = c.BindJSON(&createReq)
	if err != nil {
		msg := fmt.Sprintf("invalid request. Could not bind request: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": msg})
		return
	}

	colMgr := collections.NewManager(dbConn.(db.Database), isolationID, logger)
	collectionExists, err := colMgr.CollectionExists(ctx, createReq.CollectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while checking collection existence: %s", err)})
		return
	}
	if collectionExists {
		msg := fmt.Sprintf("collection '%s' already exists in isolation '%s'", createReq.CollectionID, isolationID)
		c.JSON(http.StatusConflict, gin.H{"code": "409", "message": msg})
		return
	}

	co, err := colMgr.CreateCollection(ctx, createReq.CollectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while creating collection: %s", err)})
		return
	}
	c.JSON(http.StatusCreated, co)
}

func GetCollections(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "get-collections")
	defer span.End()

	startTime := time.Now()

	isolationID, err := getIsolationID(c)
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

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
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

	colMgr := collections.NewManager(dbConn.(db.Database), isolationID, logger)
	collList, err := colMgr.GetCollections(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while getting collection: %s", err)})
		return
	}

	response := ListCollectionsResponse{
		IsolationID: isolationID,
		Collections: collList,
	}

	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(response.Collections))
	c.JSON(http.StatusOK, response)
}

func GetCollection(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "get-collection")
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

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, collectionID)
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

	colMgr := collections.NewManager(dbConn.(db.Database), isolationID, logger)
	co, err := colMgr.GetCollection(ctx, collectionID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			msg := fmt.Sprintf("collection '%s' not found in isolation '%s'", collectionID, isolationID)
			c.JSON(http.StatusBadRequest, gin.H{"code": "404", "message": msg})
			return
		}
	}
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(1)
	c.JSON(http.StatusOK, co)
}

func DeleteCollection(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "delete-collection")
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

	logger.Info("Deleting collection",
		zap.String("collectionID", collectionID))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, collectionID)
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

	colMgr := collections.NewManager(dbConn.(db.Database), isolationID, logger)
	err = colMgr.DeleteCollection(ctx, collectionID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			c.Status(http.StatusOK)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while getting collection: %s", err)})
		return
	}
	c.Status(http.StatusOK)
}
