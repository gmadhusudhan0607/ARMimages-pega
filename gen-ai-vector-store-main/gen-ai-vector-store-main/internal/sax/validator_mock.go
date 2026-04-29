/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"github.com/gin-gonic/gin"
)

type validatorMock struct{}

func NewValidatorMock() Validator {
	return &validatorMock{}
}

func (a *validatorMock) ValidateRequest(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
