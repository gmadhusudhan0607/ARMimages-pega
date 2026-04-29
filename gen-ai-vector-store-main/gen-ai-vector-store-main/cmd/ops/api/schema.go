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
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Configuration struct {
	Key   string `db:"key" binding:"required"`
	Value string `db:"value" binding:"required"`
}

type SchemaSyncStatus struct {
	IsolationID  string `db:"iso_id" binding:"required"`
	CollectionID string `db:"col_id" binding:"required"`
	UpdatedDoc   int    `db:"updated_doc" binding:"required"`
	UpdatedEmb   int    `db:"updated_emb" binding:"required"`
	UpdatedAttr  int    `db:"updated_attr" binding:"required"`
	PendingDoc   int    `db:"pending_doc" binding:"required"`
	PendingEmb   int    `db:"pending_emb" binding:"required"`
	PendingAttr  int    `db:"pending_attr" binding:"required"`
}

type SchemaSyncStatus1 struct {
	IsolationID  string `db:"iso_id" binding:"required"`
	CollectionID string `db:"col_id" binding:"required"`
	V1Doc        int    `db:"v1_doc" binding:"required"`
	V2Doc        int    `db:"v2_doc" binding:"required"`
	V1Emb        int    `db:"v1_emb" binding:"required"`
	V2Emb        int    `db:"v2_emb" binding:"required"`
	V1DocAttr    int    `db:"v1_doc_attr" binding:"required"`
	V2DocAttr    int    `db:"v2_doc_attr" binding:"required"`
	V1EmbAttr    int    `db:"v1_emb_attr" binding:"required"`
	V2EmbAttr    int    `db:"v2_emb_attr" binding:"required"`
}

type DatabaseSize struct {
	UsedBytes int64 `json:"used_bytes"`
}

func GetDatabaseSize(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": "Error retrieving database connection for db-size check"})
		return
	}

	dbDatabase, ok := dbConn.(db.Database)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": "Database connection is not of expected type"})
		return
	}

	query := "SELECT pg_database_size(current_database()) AS used_bytes"
	rows, err := dbDatabase.GetConn().Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get database size: %v", err)})
		return
	}
	defer rows.Close()

	var size DatabaseSize
	if rows.Next() {
		if err = rows.Scan(&size.UsedBytes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to scan database size: %v", err)})
			return
		}
	}
	c.JSON(http.StatusOK, size)
}

func GetConfiguration(c *gin.Context) {
	logger := log.GetLoggerFromContext(c.Request.Context())
	logger.Info("serving request", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))

	dbConn, ok := c.Get(middleware.DBConnectionGeneric)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": "Error retrieving database connection for db-configuration check"})
		return
	}

	query := "select * from vector_store.configuration"
	rows, err := dbConn.(db.Database).GetConn().Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to get VS configuration: %v", err)})
		return
	}
	defer rows.Close()

	var objects []Configuration
	for rows.Next() {
		var object Configuration
		err = rows.Scan(&object.Key, &object.Value)
		if err != nil {
			logger.Warn("error while scanning rows for query", zap.String("query", query), zap.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving SchemaSyncStatus data: %v", err)})
			return
		}
		objects = append(objects, object)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Error retrieving VS configuration: %v", err)})
		return
	}
	c.JSON(http.StatusOK, objects)
}
