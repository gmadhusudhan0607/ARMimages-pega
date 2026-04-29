/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package background

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/factory"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/queue"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

const (
	isolationIDParamName    = "isolationID"
	collectionNameParamName = "collectionName"
	docIDParamName          = "documentID"
	embIDParamName          = "embeddingID"
	requestIDParamName      = "requestID"
	spanIDParamName         = "spanID"
	traceIDParamName        = "traceID"
)

const serviceName = "genai-vector-store"

type EmbeddingsProcessingHandler func() error

func GetEmbeddingsProcessingHandler2(ctx context.Context, database db.Database) EmbeddingsProcessingHandler {
	return func() error {
		for {
			if err := process2(ctx, database); err != nil {
				log.GetNamedLogger(serviceName).Error("EmbeddingsProcessing2() failed with error", zap.Error(err))
			}
		}
	}
}

func process2(ctx context.Context, database db.Database) error {
	var err error
	defer func() {
		// recover from panic if one occurred. Set err to nil otherwise.
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("failed with exception: %v", err1)
		}
	}()

	var item *queue.EmbeddingQueueItem
	eq2 := queue.NewEmbeddingQueue2(ctx, database)

	item, err = eq2.Get2()
	if errors.Is(err, queue.ErrEmptyQueue) || errors.Is(err, queue.ErrIsolationDoesNotExist) {
		time.Sleep(time.Second)
		return nil
	}
	logger := getLogger(item)
	if errors.Is(err, queue.ErrInvalidEntry) {
		logger.Warn("ignored invalid queue item",
			zap.String("embPath", item.GetEmbPath()),
			zap.Any("item", item),
			zap.String("error", err.Error()))
		return nil
	}

	if err != nil {
		if errors.Is(err, queue.ErrInvalidEntry) {
			logger.Warn("error reading from queue",
				zap.String("error", err.Error()))
			return nil
		}
	}

	if item.RetryCount >= queue.MaxRetryCount {
		logger.Warn("Max retry count reached",
			zap.Int("maxRetryCount", queue.MaxRetryCount),
			zap.String("embPath", item.GetEmbPath()))

		err := updateEmbStatus(database, item, resources.StatusError, "Max retry count reached", 555)
		if err != nil {
			logger.Warn("error update embedding status when reached max retries",
				zap.String("embPath", item.GetEmbPath()),
				zap.String("error", err.Error()))
		}

		// Remove from queue since this is a terminal state
		err = eq2.DropEmbedding2(item.IsolationID, item.CollectionID, item.DocumentID, item.EmbeddingID)
		if err != nil {
			logger.Warn("failed to drop embedding from queue after max retries",
				zap.String("embPath", item.GetEmbPath()),
				zap.Error(err))
		}
		return nil
	}

	item.RetryCount += 1

	if err = renewDataEmbedding(database, item); err != nil {
		logger.Warn("error processing",
			zap.String("embPath", item.GetEmbPath()),
			zap.String("error", err.Error()))
		err = eq2.Put2(item) // reschedule again
		if err != nil {
			logger.Error("failed to reschedule embedding",
				zap.String("embPath", item.GetEmbPath()),
				zap.String("error", err.Error()))
		}
		return nil
	}

	if count, err := dbUpdateEmbedding(database, item); err != nil || count == 0 {
		logger.Warn("error while processing",
			zap.String("embPath", item.GetEmbPath()),
			zap.String("error", err.Error()))
		err = eq2.Put2(item) // reschedule again
		if err != nil {
			logger.Error("failed to reschedule embedding",
				zap.String("embPath", item.GetEmbPath()),
				zap.String("error", err.Error()))
		}
		return nil
	}

	// Reschedule if needed
	switch item.Data.ResponseCode {
	case http.StatusOK:
		logger.Debug("embedding processed successfully",
			zap.String("embPath", item.GetEmbPath()))
	case http.StatusTooManyRequests:
		// Check if max retries reached before rescheduling
		if item.RetryCount >= queue.MaxRetryCount {
			logger.Warn("Max retry count reached, not rescheduling",
				zap.Int("maxRetryCount", queue.MaxRetryCount),
				zap.String("embPath", item.GetEmbPath()))
			return nil
		}
		// Add 3 sec to be able to do checks in integration tests
		delay := 3 + item.RetryCount
		err = eq2.PutPostponed2(item, delay) // reschedule again with delay
		if err != nil {
			logger.Warn("failed to reschedule embedding",
				zap.String("embPath", item.GetEmbPath()),
				zap.String("error", err.Error()))
		}
		time.Sleep(time.Second)
	case http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusBadGateway:
		// Transient errors - retry with exponential backoff but limit retries
		if item.RetryCount >= queue.MaxRetryCount {
			logger.Warn("Max retry count reached for transient error. Stop retrying",
				zap.Int("maxRetryCount", queue.MaxRetryCount),
				zap.Int("statusCode", item.Data.ResponseCode),
				zap.String("embPath", item.GetEmbPath()))
			return nil
		}
		// Add 3 sec to be able to do checks in integration tests
		delay := 3 + item.RetryCount*item.RetryCount
		err = eq2.PutPostponed2(item, delay) // reschedule again after X seconds
		if err != nil {
			logger.Warn("failed to reschedule embedding",
				zap.String("embPath", item.GetEmbPath()),
				zap.String("error", err.Error()))
		}
	case http.StatusForbidden, http.StatusUnauthorized,
		http.StatusInternalServerError, http.StatusNotImplemented,
		http.StatusVariantAlsoNegotiates, http.StatusInsufficientStorage,
		http.StatusLoopDetected, http.StatusNotExtended:
		// Permanent errors - do NOT retry, these will never succeed
		logger.Warn("Permanent error encountered. Stop retrying. Not rescheduling. Cleanup embedding_queue",
			zap.Int("statusCode", item.Data.ResponseCode),
			zap.String("embPath", item.GetEmbPath()),
			zap.String("errorMessage", item.Data.ErrorMessage))
		// Document is already marked as ERROR in dbUpdateEmbedding
		// Remove from queue since this is a terminal state
		// This will allow to keep empty queue. Not empty queue cause problem in some integration tests.
		err = eq2.DropEmbedding2(item.IsolationID, item.CollectionID, item.DocumentID, item.EmbeddingID)
		if err != nil {
			logger.Warn("failed to drop embedding from queue after permanent error",
				zap.String("embPath", item.GetEmbPath()),
				zap.Error(err))
		}
		return nil
	}
	return nil

}

func renewDataEmbedding(database db.Database, item *queue.EmbeddingQueueItem) error {
	logger := getLogger(item)
	logger.Info("[ background2 ] reembedding",
		zap.String("embPath", item.GetEmbPath()))

	embProfile := factory.DefaultEmbeddingProfileID
	a, err := factory.CreateTextEmbedder(database, item.IsolationID, item.CollectionID, embProfile, nil, logger)
	if err != nil {
		item.Data.Status = fmt.Sprintf("ERROR:%d", 0)
		item.Data.ErrorMessage = fmt.Sprintf("failed to get ADA client: %s", err.Error())
		return nil
	}
	var allValues []string
	if item.Data != nil {
		if len(item.Data.Metadata.Attribute) > 0 {
			allValues = append(allValues, item.Data.Metadata.Attribute...)
		}
		if len(item.Data.DocMetadata.Attribute) > 0 {
			allValues = append(allValues, item.Data.DocMetadata.Attribute...)
		}
	}
	allValues = slices.Compact(allValues)
	ctx := context.Background()
	attrMgr := attributes.NewManager(database, item.IsolationID, item.CollectionID, logger)
	var chAttrs []attributes.Attribute
	chAttrs, err = attrMgr.GetEmbeddingAttributesProcessing(ctx, item.DocumentID, item.EmbeddingID, allValues)
	if err != nil {
		logger.Error("Error while getting chunk content to embed", zap.Error(err))
		return fmt.Errorf("error while getting chunk content to embed")
	}

	var chAttrsEntries []string
	for _, attr := range chAttrs {
		attrEntry := fmt.Sprintf("%s: %s", attr.Name, strings.Join(attr.Values, ", "))
		chAttrsEntries = append(chAttrsEntries, attrEntry)
	}

	contentToEmbed := fmt.Sprintf("%s | Content: %s", strings.Join(chAttrsEntries, " | "), item.Data.Content)
	embedding, code, err := a.GetEmbedding(context.Background(), contentToEmbed)
	if err != nil {
		item.Data.Status = resources.StatusError
		item.Data.ResponseCode = code
		item.Data.ErrorMessage = err.Error()
		return nil
	}

	if code >= http.StatusBadRequest {
		item.Data.Status = resources.StatusError
		item.Data.ResponseCode = code
		item.Data.ErrorMessage = err.Error()
		return nil
	}

	v := pgvector.NewVector(embedding)
	item.Data.Embedding = &v
	item.Data.Status = resources.StatusCompleted
	item.Data.ResponseCode = code
	item.Data.ErrorMessage = ""
	return nil

}

func dbUpdateEmbedding(database db.Database, item *queue.EmbeddingQueueItem) (int64, error) {
	logger := getLogger(item)
	tableEmbProcessing := db.GetTableEmbProcessing(item.IsolationID, item.CollectionID)

	dbQuery := fmt.Sprintf(`
	UPDATE %s
	SET
		status=$1,
		response_code=$2,
	    error_message=$3,
	    embedding=$4,
	    record_timestamp=CURRENT_TIMESTAMP,
		retry_count = $6
	WHERE emb_id=$5
    `, tableEmbProcessing)
	res, err := database.GetConn().Exec(dbQuery, item.Data.Status, item.Data.ResponseCode, item.Data.ErrorMessage, item.Data.Embedding, item.EmbeddingID, item.RetryCount)
	if err != nil {
		return 0, fmt.Errorf("filed to execute dbQuery [%s]: %w", dbQuery, err)
	}
	logger.Info("updated embedding",
		zap.String("embPath", item.GetEmbPath()),
		zap.String("status", item.Data.Status))
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected, nil
}

func updateEmbStatus(database db.Database, item *queue.EmbeddingQueueItem, status, msg string, code int) error {
	tableEmbProcessing := db.GetTableEmbProcessing(item.IsolationID, item.CollectionID)
	query := fmt.Sprintf(`
	UPDATE %s
	SET status=$2, error_message=$3,
        response_code=$4, record_timestamp=CURRENT_TIMESTAMP
	WHERE emb_id = $1
    `, tableEmbProcessing)

	_, err := database.GetConn().Exec(query, item.EmbeddingID, status, msg, code)
	if err != nil {
		return fmt.Errorf("failed to execute [%s]: %w", query, err)
	}
	return nil
}

func getLogger(item *queue.EmbeddingQueueItem) *zap.Logger {
	return log.GetNamedLogger("genai-vector-store-embedding-processing").
		With(
			zap.String(isolationIDParamName, item.IsolationID),
			zap.String(collectionNameParamName, item.CollectionID),
			zap.String(docIDParamName, item.DocumentID),
			zap.String(embIDParamName, item.EmbeddingID),
			zap.String(requestIDParamName, item.AdditionalData.RequestID),
			zap.String(spanIDParamName, item.AdditionalData.SpanID),
			zap.String(traceIDParamName, item.AdditionalData.TraceID),
		)
}
