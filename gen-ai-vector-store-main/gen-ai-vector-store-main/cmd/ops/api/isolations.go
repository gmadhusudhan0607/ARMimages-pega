/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IsolationRegistrationRequest struct {
	ID             string `json:"id" binding:"required"`
	MaxStorageSize string `json:"maxStorageSize,omitempty"`
	PDCEndpointURL string `json:"pdcEndpointURL,omitempty"`
}

type IsolationRegistrationResponse struct {
	ID string `json:"id" binding:"required"`
}

type IsolationUpdateRequest struct {
	MaxStorageSize string `json:"maxStorageSize"`
	PDCEndpointURL string `json:"pdcEndpointURL,omitempty"`
}

func GetIsolation(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// retrieve isolation from param
	id := c.Param(isolationIDParamName)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", isolationIDParamName)})
		return
	}

	logger.Debug("looking for isolation", zap.String("isolation", id))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", id)})
		return
	}

	isoMgr := isolations.NewManager(dbConn.(db.Database), logger)
	isoDetails, err := isoMgr.GetIsolation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while getting isolation details: %s", err)})
		return
	}

	c.JSON(http.StatusOK, isoDetails)
}

func PostIsolation(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	var reg IsolationRegistrationRequest
	if err := c.BindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error unmarshalling the request: %s", err)})
		return
	}
	if reg.ID == "" || reg.MaxStorageSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("invalid request: %v", reg)})
		return
	}
	logger.Debug("initializing isolation", zap.String("isolation", reg.ID), zap.String("maxStorageSize", reg.MaxStorageSize), zap.String("pdcEndpointURL", reg.PDCEndpointURL))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", reg.ID)})
		return
	}

	isoMgr := isolations.NewManager(dbConn.(db.Database), logger)
	isolationExists, err := isoMgr.IsolationExists(c.Request.Context(), reg.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while checking if isolation exists: %s", err)})
		return
	}
	if isolationExists {
		msg := fmt.Sprintf("isolation '%s' already exists", reg.ID)
		c.JSON(http.StatusOK, gin.H{"code": "200", "message": msg, "method": c.Request.Method, "uri": c.Request.RequestURI})
		return
	}
	err = isoMgr.CreateIsolation(c.Request.Context(), reg.ID, reg.MaxStorageSize, reg.PDCEndpointURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while creating isolation: %s", err)})
		return
	}

	c.JSON(http.StatusOK, IsolationRegistrationResponse{
		ID: reg.ID,
	})
	logger.Info("created isolation", zap.String("isolation", reg.ID), zap.String("maxStorageSize", reg.MaxStorageSize), zap.String("pdcEndpointURL", reg.PDCEndpointURL))

}

func PutIsolation(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	id := c.Param(isolationIDParamName)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", isolationIDParamName)})
		return
	}

	var reg IsolationUpdateRequest
	if err := c.BindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error unmarshalling the request: %s", err)})
		return
	}
	if reg.MaxStorageSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("invalid request: %v", reg)})
		return
	}

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", id)})
		return
	}

	isoMgr := isolations.NewManager(dbConn.(db.Database), logger)

	isolationExists, err := isoMgr.IsolationExists(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while checking if isolation exists: %s", err)})
		return
	}
	if !isolationExists {
		msg := fmt.Sprintf("isolation '%s' does not exist", id)
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": msg, "method": c.Request.Method, "uri": c.Request.RequestURI})
		return
	}

	err = isoMgr.UpdateIsolation(c.Request.Context(), id, reg.MaxStorageSize, reg.PDCEndpointURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while registering new isolation %s: %s", id, err)})
		return
	}

	c.JSON(http.StatusOK, IsolationRegistrationResponse{
		ID: id,
	})
}

func DeleteIsolation(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// retrieve isolation from param
	id := c.Param(isolationIDParamName)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", isolationIDParamName)})
		return
	}

	logger.Debug("looking for isolation", zap.String("isolation", id))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error retrieving database connection for isolation %s", id)})
		return
	}

	isoMgr := isolations.NewManager(dbConn.(db.Database), logger)
	err := isoMgr.DeleteIsolation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("error while deleting isolation: %s", err)})
		return
	}

	c.Status(http.StatusOK)
	logger.Info("deleted isolation", zap.String("isolation", id))
}

func GetIsolationRO(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// Call the GetIsolation method
	GetIsolation(c)
}

func PostIsolationRO(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// Extract the ID from the request body
	var reg IsolationRegistrationRequest
	if err := c.BindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error unmarshalling the request: %s", err)})
		return
	}

	// Return a success status without performing any action
	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Read-only mode, no action taken",
		"method":  c.Request.Method,
		"uri":     c.Request.RequestURI,
		"id":      reg.ID,
	})
}

func PutIsolationRO(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request in read-only mode", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// Retrieve isolation ID from param
	id := c.Param(isolationIDParamName)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", isolationIDParamName)})
		return
	}

	var reg IsolationUpdateRequest
	if err := c.BindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("error unmarshalling the request: %s", err)})
		return
	}
	if reg.MaxStorageSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("invalid request: %v", reg)})
		return
	}

	logger.Debug("read-only mode: avoiding updating isolation", zap.String("isolation", id), zap.String("maxStorageSize", reg.MaxStorageSize))

	// Return a success status without performing any update
	c.JSON(http.StatusOK, IsolationRegistrationResponse{
		ID: id,
	})
}

func DeleteIsolationRO(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request in read-only mode", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// Retrieve isolation ID from param
	id := c.Param(isolationIDParamName)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("%s param is required", isolationIDParamName)})
		return
	}

	logger.Debug("read-only mode: avoiding deletion of isolation", zap.String("isolation", id))

	// Return a success status without performing any deletion
	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Read-only mode, no action taken",
		"method":  c.Request.Method,
		"uri":     c.Request.RequestURI,
		"id":      id,
	})
}
