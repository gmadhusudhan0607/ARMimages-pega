/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/repository"
	"github.com/gin-gonic/gin"
)

func HandlePostEventRequest(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		l := cntx.LoggerFromContext(ctx).Sugar()

		var evt repository.Event

		if err := c.ShouldBindJSON(&evt); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(evt.Isolation) == 0 || evt.Timestamp == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "isolation and timestamp are mandatory fields"})
			return
		}

		l.Infof("storing new event about isolation: %s at %v", evt.Isolation, evt.Timestamp)
		repository.Insert(evt)
		count := repository.Count()
		l.Infof("records in the events store: %d", count)
	}
}

func HandleGetIsolationMetrics(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		isolation := c.Param("isolationId")
		if len(isolation) == 0 {
			l.Error("isolation parameter is mandatory")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "isolation parameter is mandatory"})
			return
		}

		twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
		from, err := timeOrDefault(c.Query("from"), twentyFourHoursAgo)
		if err != nil {
			l.Errorf("error parsing query parameter `from`. Error: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		to, err := timeOrDefault(c.Query("to"), time.Now())
		if err != nil {
			l.Errorf("error parsing query parameter `to`. Error: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		l.Infof("fetching events for isolation: %s, from %v to %v", isolation, from.Unix(), to.Unix())
		events := repository.Read(isolation, from.Unix(), to.Unix())
		l.Infof("number of events found: %d", len(events))
		response := IsolationMetrics{
			Isolation: isolation,
			Requests:  len(events),
		}
		c.JSON(http.StatusOK, response)
	}
}

func timeOrDefault(s string, t time.Time) (time.Time, error) {
	if len(s) == 0 {
		return t, nil
	}

	unix, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return time.Time{}, e
	}
	return time.Unix(unix, 0), nil
}

type IsolationMetrics struct {
	Isolation string `json:"isolationId"`
	Requests  int    `json:"totalRequests"`
}

type MetricSummary struct {
	Model   string `json:"modelId"`
	Request int    `json:"requests"`
}
