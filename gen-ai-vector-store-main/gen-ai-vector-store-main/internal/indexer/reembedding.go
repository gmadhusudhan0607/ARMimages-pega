/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/contexthelper"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/queue"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

func (i *indexer) putDocumentIntoQueueForReembedding(ctx context.Context, docID string, delay int) error {
	tableEmbProcessing := db.GetTableEmbProcessing(i.IsolationID, i.CollectionName)
	dbQueryTpl := "SELECT emb_id FROM %s WHERE doc_id=$1"
	dbQuery := fmt.Sprintf(dbQueryTpl, tableEmbProcessing)
	rows, err := i.genericDbConn.GetConn().Query(dbQuery, docID)
	if err != nil {
		return err
	}
	defer rows.Close()

	embQueue := queue.NewEmbeddingQueue2(ctx, i.genericDbConn)

	var embID string
	for rows.Next() {
		err = rows.Scan(&embID)
		if err != nil {
			i.logger.Warn("error while reading rows from query", zap.String("query", dbQuery), zap.String("error", err.Error()))
			return err
		}

		eqItem := queue.EmbeddingQueueItem{
			IsolationID:    i.IsolationID,
			CollectionID:   i.CollectionName,
			DocumentID:     docID,
			EmbeddingID:    embID,
			AdditionalData: constructAdditionalData(ctx),
		}

		if err = embQueue.PutPostponed2(&eqItem, delay); err != nil {
			return fmt.Errorf("error while putting %s into queue: %w", eqItem.GetEmbPath(), err)
		} else {
			i.logger.Info("rescheduled embedding", zap.String("embPath", eqItem.GetEmbPath()), zap.Int("delay", delay))
		}
	}
	return nil
}

func constructAdditionalData(ctx context.Context) queue.EmbeddingQueueItemAdditionalData {
	data := queue.EmbeddingQueueItemAdditionalData{}

	if traceID, ok := ctx.Value(contexthelper.TraceIDKey).(string); ok {
		data.TraceID = traceID
	}
	if spanID, ok := ctx.Value(contexthelper.SpanIDKey).(string); ok {
		data.SpanID = spanID
	}
	if reqID, ok := ctx.Value(contexthelper.RequestIDKey).(string); ok {
		data.RequestID = reqID
	}

	return data
}
