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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/opsmetrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RetrieveDocumentsMetrics(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// read parameters
	isolationID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger.Debug("looking for isolation", zap.String("isolation", isolationID))

	var req opsmetrics.DocumentsMetricsRequest
	if c.Request.ContentLength > 0 {
		err = c.BindJSON(&req)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error unmarshalling the request for isolation '%s': %v", isolationID, err)})
		return
	}

	logger.Info("requested data", zap.String("isolation", isolationID), zap.Any("params", req.Metrics))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isolationID)})
		return
	}
	o := opsmetrics.NewOpsMetrics(dbConn.(db.Database), isolationID)
	im, err := o.GetIsolationMetrics(req.Metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while getting isolation details for isolation '%s': %v", isolationID, err)})
		return
	}
	c.JSON(http.StatusOK, im)

}

func RetrieveDocumentsMetricsDetails(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	defer logger.Sync() //nolint:errcheck

	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	// read parameters
	isoID, err := getIsolationIDName(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "400", "message": fmt.Sprintf("Error parsing the request for isolation '%s': %v", isoID, err)})
		return
	}

	logger.Debug("looking for isolation", zap.String("isolation", isoID))

	var req opsmetrics.DocumentsMetricsDetailsRequest
	if c.Request.ContentLength > 0 {
		err = c.BindJSON(&req)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error unmarshalling the request for isolation '%s': %v", isoID, err)})
		return
	}

	logger.Info("requested data", zap.String("isolation", isoID), zap.Any("params", req.Metrics))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving database connection for isolation '%s'", isoID)})
		return
	}
	o := opsmetrics.NewOpsMetrics(dbConn.(db.Database), isoID)
	cm, err := o.GetCollectionsMetrics(req.Metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error while getting isolation details for isolation '%s': %v", isoID, err)})
		return
	}

	c.JSON(http.StatusOK, cm)
}
