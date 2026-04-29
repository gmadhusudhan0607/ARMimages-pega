/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Database Connection Injection
 * -----------------------------------------
 * Purpose: Injects a database connection into the Gin context for use in downstream handlers.
 * Usage: Add DatabaseHandler(connection) to your Gin middleware chain to make the DB connection
 *        available via context key DBConnection.
 * Configuration: Pass a db.Database instance to DatabaseHandler.
 */

package middleware

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/gin-gonic/gin"
)

const (
	// DBConnectionSearch - Database connection for search key for gin context
	DBConnectionSearch = "db.connection.search"
	// DBConnectionIngest - Database connection for ingest key for gin context
	DBConnectionIngest = "db.connection.ingest"
	// DBConnectionGeneric - Generic database connection key for gin context
	DBConnectionGeneric = "db.connection.generic"
	// DBConnectionRead - Read-only database connection key for gin context
)

type DatabasesConfig struct {
	Generic db.Database
	Ingest  db.Database
	Search  db.Database
}

// DatabaseHandler returns a Gin middleware that injects the provided database connection
// into the Gin context under the key DBConnection. This allows handlers to retrieve the
// database connection from the context for query execution.
func DatabaseHandler(dbs DatabasesConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(DBConnectionSearch, dbs.Search)
		c.Set(DBConnectionIngest, dbs.Ingest)
		c.Set(DBConnectionGeneric, dbs.Generic)
		c.Next()
	}
}
