/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
)

func (m *docManager) GetChunksContent2(ctx context.Context, docID string) ([]embedings.Chunk, error) {
	var chunks []embedings.Chunk
	tableEmb := fmt.Sprintf("%s.%s_emb", m.schemaName, m.prefix)

	query := fmt.Sprintf("SELECT emb_id, content FROM %s WHERE doc_id=$1", tableEmb)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		chunk := embedings.Chunk{DocumentID: docID}
		if err = rows.Scan(&chunk.ID, &chunk.Content); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (m *docManager) GetAttributeIDs(ctx context.Context, docID string) ([]int64, error) {
	var docAttrIDs []int64

	query := fmt.Sprintf("SELECT unnest(attr_ids) FROM %s WHERE doc_id=$1", m.tableDoc)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute IDs for document '%s': %w", docID, err)
	}
	if rows.Next() {
		var attrID int64
		if err = rows.Scan(&attrID); err != nil {
			return nil, fmt.Errorf("failed to scan attribute IDs for document '%s': %w", docID, err)
		}
		docAttrIDs = append(docAttrIDs, attrID)
	}
	return docAttrIDs, nil
}
