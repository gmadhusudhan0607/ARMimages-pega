/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer/processing"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.uber.org/zap"
)

func (i *indexer) replaceChunkMetadataTx(ctx context.Context, tx *sql.Tx, ch embedings.Chunk) error {
	tableEmbMeta := db.GetTableEmbMeta(i.IsolationID, i.CollectionName)

	_, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE emb_id = $1`, tableEmbMeta), ch.ID)
	if err != nil {
		return fmt.Errorf("error while deleting existing chunk metadata for emb_id [%s]: %w", ch.ID, err)
	}

	query := "INSERT INTO " + tableEmbMeta + " (emb_id, metadata_key, metadata_value) VALUES ($1, $2, $3)"

	// create deep copy of StaticEmbeddingAttributes to remove duplicates by compacting
	staticEmbeddingAttributes := make([]string, len(ch.Metadata.StaticEmbeddingAttributes))
	copy(staticEmbeddingAttributes, ch.Metadata.StaticEmbeddingAttributes)
	metadataValue := strings.Join(slices.Compact(staticEmbeddingAttributes), ",")

	_, err = tx.ExecContext(ctx, query, ch.ID, embedings.MetadataKeyStaticEmbeddingAttributes, metadataValue)
	if err != nil {
		return fmt.Errorf("error while inserting chunk metadata [%s]: %w", query, err)
	}

	return nil
}

// chunkEmbedFailure captures a single chunk embedding failure so that errors
// can be persisted to the database in a separate, short-lived transaction
// after all embedding HTTP calls have completed.
type chunkEmbedFailure struct {
	chunkIdx   int
	err        error
	statusCode int
}

// precomputeChunkContent reads the content-to-embed for every chunk from the
// processing tables inside the given transaction.  The returned slice is
// parallel to chunks (same length, same order).
func (i *indexer) precomputeChunkContent(ctx context.Context, tx *sql.Tx, chunks []embedings.Chunk, docMetadata *documents.DocumentMetadata) ([]string, error) {
	content := make([]string, len(chunks))
	for idx := range chunks {
		c, err := i.getChunkContentToEmbed(ctx, tx, chunks[idx], docMetadata)
		if err != nil {
			return nil, fmt.Errorf("error while getting chunk content to embed for chunk %d: %w", idx, err)
		}
		content[idx] = c
	}
	return content, nil
}

func (i *indexer) embedChunksInParallel(ctx context.Context, chunks []embedings.Chunk, content []string) []chunkEmbedFailure {
	if len(chunks) == 0 {
		return nil
	}

	var (
		mu       sync.Mutex
		failures []chunkEmbedFailure
	)

	g, ctx := getParallelizationGroup().WithContext(ctx)

	for idx := range chunks {
		g.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				i.logger.Debug("context cancelled, stopping embedding", zap.Error(ctx.Err()))
				return nil
			default:
				embedding, err := i.embedChunk2(ctx, content[idx])
				if err != nil {
					i.logger.Debug("error while embedding chunk",
						zap.Error(err),
						zap.String("chunkID", chunks[idx].ID),
						zap.String("docID", chunks[idx].DocumentID))

					f := chunkEmbedFailure{chunkIdx: idx, err: err}
					embErr := &embeddingError{}
					if errors.As(err, &embErr) {
						f.statusCode = embErr.statusCode
					}
					mu.Lock()
					failures = append(failures, f)
					mu.Unlock()

					return err
				}

				chunks[idx].Embedding = embedding
				return nil
			}
		})
	}

	err := g.Wait()
	if err != nil {
		i.logger.Error("error during parallel embedding", zap.Error(err))
	}

	return failures
}

// persistChunkErrorsTx writes previously collected chunk embedding failures
// into the processing table inside the provided transaction.
func (i *indexer) persistChunkErrorsTx(ctx context.Context, tx *sql.Tx, chunks []embedings.Chunk, failures []chunkEmbedFailure) {
	procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
	for _, f := range failures {
		ch := chunks[f.chunkIdx]
		if err := procMgr.SetChunkError(ctx, ch.DocumentID, ch.ID, f.err.Error(), f.statusCode); err != nil {
			i.logger.Error("error while persisting chunk error",
				zap.String("chunkID", ch.ID),
				zap.Error(err))
		}
	}
}
