/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"go.uber.org/zap"
)

// Function to extract the number after "-EMB-"
func extractIndexNumber(input string) (int, error) {
	// Regex to match "-EMB-" followed by digits
	re := regexp.MustCompile(`-EMB-(\d+)$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no index number found in: %s", input)
	}

	// Convert extracted string to integer
	indexNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("error converting index number: %v", err)
	}
	return indexNumber, nil
}

func (m *embManager) getDocumentChunksItemsTotal(ctx context.Context, documentID string) (itemsTotal int, err error) {
	// Get total number of chunks for a document
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %[1]s.%[2]s_emb WHERE doc_id = $1 
	`, m.schemaName, m.tablesPrefix)
	rows, err := m.query(ctx, query, documentID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query[%s]: %w", query, err)
	}
	defer rows.Close()

	if rows.Next() {
		if err = rows.Scan(&itemsTotal); err != nil {
			return 0, fmt.Errorf("rows.scan error query[%s]: %s", query, err)
		}
	}
	return itemsTotal, err
}

func (m *embManager) getDocumentChunksItemsLeft(ctx context.Context, documentID string, idx int) (itemsLeft int, err error) {
	// Get total number of chunks for a document
	query := fmt.Sprintf(`
	    SELECT COUNT(*) FROM %[1]s.%[2]s_emb 
	    WHERE doc_id = $1 AND CAST(regexp_replace(emb_id, '.*-(\d+)$', '\1', 'g') AS INTEGER) > $2 
	`, m.schemaName, m.tablesPrefix)

	m.logger.Debug("query",
		zap.String("sql", query))

	rows, err := m.query(ctx, query, documentID, idx)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query[%s]: %w", query, err)
	}
	defer rows.Close()

	if rows.Next() {
		if err = rows.Scan(&itemsLeft); err != nil {
			return 0, fmt.Errorf("rows.scan error query[%s]: %s", query, err)
		}
	}
	return itemsLeft, err
}

func (m *embManager) GetDocumentChunksPaginated(ctx context.Context, documentID string, cursor string, limit int) (chunks []*Chunk, itemsTotal, itemsLeft int, err error) {
	cursorIdx := -1
	if cursor != "" {
		cursorIdx, err = extractIndexNumber(cursor)
		if err != nil {
			return nil, itemsTotal, itemsLeft, fmt.Errorf("failed to extract index from cursor '%s': %w", cursor, err)
		}
	}
	if limit < 1 {
		limit = 2147483647
	}

	// Select total first
	itemsTotal, err = m.getDocumentChunksItemsTotal(ctx, documentID)
	if err != nil {
		return nil, itemsTotal, itemsLeft, fmt.Errorf("failed to get totalItems for document '%s': %w", documentID, err)
	}

	//// Select total first
	//itemsAfterNextCursor, err = m.getDocumentChunksItemsAfter(documentID, cursorIdx)
	//if err != nil {
	//	return nil, 0, 0, fmt.Errorf("failed to get totalItems for document '%s': %w", documentID, err)
	//}

	// Embedding ID is created like DOC-2-EMB-0, DOC-2-EMB-1, DOC-2-EMB-2 ...
	// We want to do pagination and sorting based on number, and behave like numbers not strings
	// otherwise a string  comparison will give us "DOC-2-EMB-8" > "DOC-2-EMB-22", we do not want that.
	// Therefore, we extract numbers from emb_id to compare them as numbers.
	query := fmt.Sprintf(`
		SELECT emb_id,
			   content,
			   vector_store.attributes_as_jsonb_by_ids('%[1]s.%[2]s_attr', attr_ids2 ) as attributes
		FROM %[1]s.%[2]s_emb
		WHERE doc_id = $1
		AND CAST(regexp_replace(emb_id, '.*-(\d+)$', '\1', 'g') AS INTEGER) >  $2
		ORDER BY CAST(regexp_replace(emb_id, '.*-(\d+)$', '\1', 'g') AS INTEGER)
		LIMIT $3
    `, m.schemaName, m.tablesPrefix)

	rows, err := m.query(ctx, query, documentID, cursorIdx, limit)
	if err != nil {
		return nil, itemsTotal, itemsLeft, fmt.Errorf(
			"failed to execute query[%s] params[documentID='%s', cursorIdx='%d', limit'%d']: %w",
			query, documentID, cursorIdx, limit, err)
	}
	defer rows.Close()

	for rows.Next() {
		ch := &Chunk{}
		err = rows.Scan(&ch.ID, &ch.Content, &ch.Attributes)
		if err != nil {
			return nil, itemsTotal, itemsLeft, fmt.Errorf("rows.scan error: %s", err)
		}
		chunks = append(chunks, ch)
	}

	// Select items left to return after the current page
	if len(chunks) > 0 {
		cursor = chunks[len(chunks)-1].ID
		nextCursorIdx, err := extractIndexNumber(cursor)
		if err != nil {
			return nil, itemsTotal, itemsLeft, fmt.Errorf("failed to extract index from cursor '%s': %w", cursor, err)
		}
		itemsLeft, err = m.getDocumentChunksItemsLeft(ctx, documentID, nextCursorIdx)
		if err != nil {
			return nil, itemsTotal, itemsLeft, fmt.Errorf("failed to get items left for document '%s': %w", documentID, err)
		}
	}

	return chunks, itemsTotal, itemsLeft, nil
}
