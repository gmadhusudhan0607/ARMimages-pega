/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

var ErrEmptyQueue = errors.New("queue is empty")
var ErrInvalidEntry = errors.New("invalid queue entry")
var ErrIsolationDoesNotExist = errors.New("isolation does not exist")

var logger = log.GetNamedLogger("genai-vector-store-queue")

type queue2 struct {
	ctx      context.Context
	database db.Database
}

type QueueItem struct {
	ID            string    `json:"id" binding:"required"`
	CreatedAt     time.Time `json:"created_at" binding:"required"`
	PostponeUntil time.Time `json:"postpone_until" binding:"required"`
	Content       string    `json:"content" binding:"required"`
}

func (q *queue2) Get2() (*QueueItem, error) {
	item := QueueItem{}

	dbQuery := "SELECT * FROM vector_store.embeddings_queue_get()"
	rows, err := q.database.Query(context.Background(), dbQuery)
	if err != nil || rows == nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrEmptyQueue
	}

	err = rows.Scan(&item.ID, &item.CreatedAt, &item.PostponeUntil, &item.Content)
	if err != nil {
		return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", dbQuery, err)
	}

	return &item, nil
}

func (q *queue2) Put2(content interface{}) error {
	jsonData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal %#v to queue content: %w", content, err)
	}

	dbQuery := fmt.Sprintf("select vector_store.embeddings_queue_put('%s'::json, null)", jsonData)
	_, err = q.database.GetConn().Exec(dbQuery)
	if err != nil {
		logger.Error("failed to queue", zap.String("jsonData", string(jsonData)), zap.Error(err))
		return err
	}
	return nil
}

func (q *queue2) PutPostponed2(content interface{}, seconds int) error {
	jsonData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal %#v to queue content: %w", content, err)
	}

	dbQuery := fmt.Sprintf("select vector_store.embeddings_queue_put('%s'::json, $1 )", jsonData)
	_, err = q.database.GetConn().Exec(dbQuery, seconds)
	if err != nil {
		logger.Error("failed to queue", zap.ByteString("jsonData", jsonData), zap.Error(err))
		return err
	}
	return nil
}
