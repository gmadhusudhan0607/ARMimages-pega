/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
)

func (b *BuddyUrlParams) String() string {
	return fmt.Sprintf("buddyId=%s", b.BuddyId)
}

func GetBuddy(cd *Mapping, buddyId string) (*Buddy, *AppError) {
	for _, b := range cd.Buddies {
		if b.Name == buddyId {
			return &b, nil
		}
	}
	errMsg := fmt.Sprintf("unrecognized buddyId: %s", buddyId)
	return nil, &AppError{Message: errMsg, Error: fmt.Errorf("%s", errMsg)}
}

func GetBuddyRequestParams(c *gin.Context) *BuddyUrlParams {
	return &BuddyUrlParams{
		IsolationId: c.Param(IsolationIdParamName),
		BuddyId:     c.Param(BuddyIdParamName),
	}
}

// HandleBuddyRequest method for handling buddy API requests
func HandleBuddyRequest(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		buddyUrlParams := GetBuddyRequestParams(c)

		if buddyUrlParams.IsolationId == "" || buddyUrlParams.BuddyId == "" {
			l.Error("isolationId and buddyId are required")
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "isolationId and buddyId params are required",
			})
			return
		}

		// check if the buddy is recognized
		b, err := GetBuddy(mapping, buddyUrlParams.BuddyId)
		if err != nil {
			l.Error(err.Message)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		// check if the buddy URL is configured
		if b.RedirectURL == "" {
			errMsg := fmt.Sprintf("buddy '%s' is not mapped to any provider URL", buddyUrlParams.BuddyId)
			l.Error(errMsg)
			c.JSON(http.StatusNotFound, RespErr{
				StatusCode: http.StatusNotFound,
				Message:    errMsg,
			})
			return
		}

		// construct prefix string
		PrefixPath := fmt.Sprintf("/v1/%s/buddies/%s", buddyUrlParams.IsolationId, buddyUrlParams.BuddyId)
		operationPath := ""
		if strings.HasPrefix(c.Request.URL.RequestURI(), PrefixPath) {
			operationPath = strings.TrimPrefix(c.Request.URL.RequestURI(), PrefixPath)
		} else {
			// the request does not fit the pattern
			msg := fmt.Sprintf("Error while parsing the request: Unrecognized request URI %s", c.Request.URL.RequestURI())
			l.Error(msg)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		buddyUrl := GetEntityEndpointUrl(b.RedirectURL, operationPath)

		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, buddyUrl)

		CallTarget(c, ctx, buddyUrl, SaxAuthDisabled)
	}
	return fn
}
