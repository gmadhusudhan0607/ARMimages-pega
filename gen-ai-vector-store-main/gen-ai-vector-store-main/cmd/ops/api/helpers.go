/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	isolationIDParamName = "isolationID"
	maxIsolationIDLength = 36
)

func getIsolationIDName(c *gin.Context) (string, error) {
	isolationID := c.Param(isolationIDParamName)

	if isolationID == "" {
		return isolationID, fmt.Errorf("%s param is required", isolationIDParamName)
	}

	if len(isolationID) > maxIsolationIDLength {
		return isolationID, fmt.Errorf("%s param cannot exceed %d characters", isolationIDParamName, maxIsolationIDLength)
	}
	return strings.ToLower(isolationID), nil
}
