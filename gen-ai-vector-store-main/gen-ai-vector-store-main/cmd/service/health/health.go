/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package health

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetLiveness(c *gin.Context) {
	c.Status(http.StatusOK)
}

func GetReadiness(c *gin.Context) {
	c.Status(http.StatusOK)
}
