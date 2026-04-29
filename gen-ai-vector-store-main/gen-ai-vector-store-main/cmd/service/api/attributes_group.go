/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"errors"
	"fmt"
	"io"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	attributesgroup "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes_group"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"

	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func PostSmartAttributesGroup(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "post-smart-attributes-group")
	defer span.End()

	// read parameters
	isolationID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get db metadata for isolation '%s': %v", isolationID, err)})
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

	var attrs attributesgroup.AttributesGroup
	if err = c.BindJSON(&attrs); err != nil {
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

	mgr := attributesgroup.NewManager(dbConn.(db.Database), isolationID, logger)
	attrsGrp, err := mgr.CreateAttributesGroup(ctx, attrs.Description, attrs.Attributes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error upserting attributes group in database for isolation '%s': %v", isolationID, err)})
		return
	}
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(1)
	c.JSON(http.StatusOK, attrsGrp)
}

func GetSmartAttributesGroup(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "get-smart-attributes-group")
	defer span.End()

	// read parameters
	isolationID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get db metadata for isolation '%s': %v", isolationID, err)})
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

	groupID, err := getGroupIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}
	if groupID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("group_id parameter is required for isolation '%s'", isolationID)})
		return
	}

	mgr := attributesgroup.NewManager(dbConn.(db.Database), isolationID, logger)
	attrsGroups, err := mgr.GetAttributesGroup(ctx, groupID)
	if err != nil {
		if errors.Is(err, attributesgroup.ErrAttributeGroupDoesNotExist) {
			c.JSON(http.StatusNotFound, fmt.Sprintf("attribute group with id '%s' not found", groupID))
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error getting attribute from database for isolation '%s': %v", isolationID, err)})
		return
	}
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(1)
	c.JSON(http.StatusOK, attrsGroups)
}

func ListSmartAttributesGroups(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "list-smart-attributes-groups")
	defer span.End()

	// read parameters
	isolationID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get db metadata for isolation '%s': %v", isolationID, err)})
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

	mgr := attributesgroup.NewManager(dbConn.(db.Database), isolationID, logger)
	agDescr, err := mgr.GetAttributesGroupDescriptions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving list of attributes groups from database for isolation '%s': %v", isolationID, err)})
		return
	}

	var resp []ListAttributesGroupsItem
	for name, value := range agDescr {
		resp = append(resp, ListAttributesGroupsItem{GroupID: name, Description: value})
	}

	logger.Info("returned items",
		zap.Int("count", len(resp)),
	)
	servicemetrics.FromContext(c.Request.Context()).ResponseMetrics.SetItemsReturned(len(resp))
	c.JSON(http.StatusOK, resp)
}

func PutSmartAttributesGroup(c *gin.Context) {
	_, span := startAPIHandlerSpan(c, "put-smart-attributes-group")
	defer span.End()

	// This endpoint is not implemented, but we need to return 405 for readonly mode
	c.JSON(http.StatusNotImplemented, gin.H{"code": "501", "message": "PUT operation not implemented"})
}

func DeleteSmartAttributesGroup(c *gin.Context) {
	ctx, span := startAPIHandlerSpan(c, "delete-smart-attributes-group")
	defer span.End()

	// read parameters
	isolationID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger := log.GetLoggerFromContext(ctx)
	defer logger.Sync() //nolint:errcheck

	groupID, err := getGroupIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "499", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("group_id parameter is required for isolation '%s'", isolationID)})
		return
	}

	logger.Info("serving request",
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
	)

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return
	}

	schemaMgr, err := schema.NewVsSchemaManager(dbConn.(db.Database), logger).Load(ctx, isolationID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get db metadata for isolation '%s': %v", isolationID, err)})
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

	mgr := attributesgroup.NewManager(dbConn.(db.Database), isolationID, logger)
	err = mgr.DeleteAttributesGroup(ctx, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error deleting attributes groups from database for isolation '%s': %v", isolationID, err)})
		return
	}
	c.JSON(http.StatusOK, groupID)
}
