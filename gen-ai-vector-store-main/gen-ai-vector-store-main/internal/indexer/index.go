/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
)

func (i *indexer) Index(ctx context.Context, docID string, chunks []embedings.Chunk,
	docAttributes []attributes.Attribute, docMetadata *documents.DocumentMetadata, consistencyLevel string, extraAttributesKinds []string) error {
	var err error

	isoMgr := isolations.NewManager(i.dbConn, i.logger)

	isoExists, err := isoMgr.IsolationExists(ctx, i.IsolationID)
	if err != nil {
		return fmt.Errorf("error while checking if isolation exists: %w", err)
	}

	if !isoExists {
		if helpers.IsIsolationAutoCreationEnabled() {
			maxStorageSize := helpers.GetIsolationAutoCreationMaxStorageSize()
			if err = isoMgr.CreateIsolation(ctx, i.IsolationID, maxStorageSize, ""); err != nil {
				return fmt.Errorf("isolaton auto-creation failed: %w", err)
			}
		} else {
			return fmt.Errorf("isolation %s is not registered. Please install GenAIVectorStoreIsolation SCE first", i.IsolationID)
		}
	}

	//create databases for given isolation & collection if not exist
	colMgr := collections.NewManager(i.dbConn, i.IsolationID, i.logger)
	collectionExists, err := colMgr.CollectionExists(ctx, i.CollectionName)
	if err != nil {
		return fmt.Errorf("error while checking if collection exists: %w", err)
	}
	if !collectionExists {
		if _, err = colMgr.CreateCollection(ctx, i.CollectionName); err != nil {
			return fmt.Errorf("error while creating collection: %w", err)
		}
	}

	chunks = i.initChunks2(chunks, docID)

	switch consistencyLevel {
	//Eventual Consistency is used by default
	case ConsistencyLevelEventual:
		err := i.processAsync(ctx, docID, chunks, docAttributes, docMetadata, extraAttributesKinds)
		if err != nil {
			return fmt.Errorf("error while handling db transaction asynchronously: %w", err)
		}
	case ConsistencyLevelStrong:
		err := i.processSync(ctx, docID, chunks, docAttributes, docMetadata, extraAttributesKinds)
		if err != nil {
			return fmt.Errorf("error while handling db transaction synchronously: %w", err)
		}
	default:
		return fmt.Errorf("unsupported consistency level: %s", consistencyLevel)
	}
	return nil
}

func (i *indexer) getConcatenatedContextFromIndexContextAttributes(ch embedings.Chunk) []string {
	ccItems := make([]string, 0)
	if ch.Metadata == nil || ch.Metadata.SmartIndexContextAttributes == nil {
		return ccItems
	}
	idxCtxAttrs := ch.Metadata.SmartIndexContextAttributes
	if idxCtxAttrs == nil || idxCtxAttrs.Attributes == nil || len(*idxCtxAttrs.Attributes) == 0 {
		return ccItems
	}
	var result []string
	sections := fmt.Sprintf("Sections: %s", strings.Join(*idxCtxAttrs.Attributes, " > "))
	result = append(result, sections)
	return result
}

func (i *indexer) getConcatenatedContextFromAutoResolvedAndStatisAttributes(ctx context.Context, tx *sql.Tx, ch embedings.Chunk, docMetadata *documents.DocumentMetadata) ([]string, error) {
	ccItems := make([]string, 0)
	var mergedAttrNames []string

	if docMetadata != nil && len(docMetadata.StaticEmbeddingAttributes) > 0 {
		mergedAttrNames = append(mergedAttrNames, docMetadata.StaticEmbeddingAttributes...)
	}

	if ch.Metadata != nil {
		if len(ch.Metadata.StaticEmbeddingAttributes) > 0 {
			mergedAttrNames = append(mergedAttrNames, ch.Metadata.StaticEmbeddingAttributes...)
		}
	}
	mergedAttrNames = slices.Compact(mergedAttrNames)

	if len(mergedAttrNames) > 0 {
		attrMgr := attributes.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
		chAttrs, err := attrMgr.GetEmbeddingAttributesProcessing(ctx, ch.DocumentID, ch.ID, mergedAttrNames)
		if err != nil {
			return nil, fmt.Errorf("error while finding attributes: %w", err)
		}
		for _, attr := range chAttrs {
			attrEntry := fmt.Sprintf("%s: %s", attr.Name, strings.Join(attr.Values, ", "))
			ccItems = append(ccItems, attrEntry)
		}
	}
	return ccItems, nil
}

func (i *indexer) getChunkContentToEmbed(ctx context.Context, tx *sql.Tx, ch embedings.Chunk, docMetadata *documents.DocumentMetadata) (string, error) {
	// Add ConcatenatedContext to chunk content for embedding
	ccItems := make([]string, 0)

	// Get data from Index context attributes
	indexCcItems := i.getConcatenatedContextFromIndexContextAttributes(ch)
	ccItems = append(ccItems, indexCcItems...)

	// Get data from AutoResolved and Static attributes
	arAttrs, err := i.getConcatenatedContextFromAutoResolvedAndStatisAttributes(ctx, tx, ch, docMetadata)
	if err != nil {
		return "", fmt.Errorf("error while getting concatenated context from auto-resolved and static attributes: %w", err)
	}
	ccItems = append(ccItems, arAttrs...)

	// Add actual content to the chunk content
	ccItems = append(ccItems, fmt.Sprintf("Content: %s", ch.Content))
	return strings.Join(ccItems, " | "), nil
}

func (i *indexer) embedChunk2(ctx context.Context, contentToEmbed string) ([]float32, error) {
	i.logger.Debug("LLM-EMBEDDING-CONTENT", zap.String("content", strings.ReplaceAll(contentToEmbed, "\n", "\\n")))

	embedding, statusCode, err := i.Embedder.GetEmbedding(ctx, contentToEmbed)
	if err != nil {
		i.logger.Debug("error while getting embedding from LLM", zap.Error(err), zap.Int("statusCode", statusCode))
		return nil, &embeddingError{err: err, statusCode: statusCode}
	}
	i.logger.Debug("successfully received embedding from LLM")

	return embedding, nil
}

func (i *indexer) initChunks2(chs []embedings.Chunk, docID string) []embedings.Chunk {
	for n := range chs {
		id := fmt.Sprintf("%s-EMB-%d", docID, n)
		chs[n].ID = id
		chs[n].DocumentID = docID
	}
	return chs
}
