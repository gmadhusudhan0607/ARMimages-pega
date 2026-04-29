/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	documents_sql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"go.uber.org/zap"
)

func (m *docManager) GetDocument2(ctx context.Context, docID string) (doc Document, err error) {
	dbQuery := fmt.Sprintf(documents_sql.GetDocumentSqlQueryTemplate, m.schemaName, m.prefix)
	m.logger.Debug("getDocument with query", zap.String("query", dbQuery))

	rows, err := m.query(ctx, dbQuery, docID)
	if err != nil {
		return doc, err
	}
	defer func() {
		cErr := rows.Close()
		if cErr != nil {
			m.logger.Error("error closing rows", zap.Error(cErr))
		}
	}()

	if rows.Next() {
		err = rows.Scan(&doc.ID, &doc.Status, &doc.Error)
		if err != nil {
			return doc, fmt.Errorf("rows.scan error: %s", err)
		}
	} else {
		return doc, ErrDocumentNotFound
	}
	return doc, nil
}

func (m *docManager) SetAttributes(ctx context.Context, docID string, attrs attributes.Attributes) error {

	exist, err := m.documentExists(ctx, docID)
	if err != nil {
		return fmt.Errorf("failed to check if document '%s' exists: %w", docID, err)
	}
	if !exist {
		return ErrDocumentNotFound
	}

	// OLD attrs
	oldAttrIDs, err := m.getAttributeIDs(ctx, docID)
	if err != nil {
		return fmt.Errorf("failed to get attributes: %w", err)
	}
	oldAttrs, err := m.attrMgr.GetAttributesByIDs(ctx, oldAttrIDs)
	if err != nil {
		return fmt.Errorf("failed to get attributes by IDs: %w", err)
	}

	// Patch attributes
	patchedAttrs := attributes.PatchAttributes(oldAttrs, attrs)

	patchedAttrsKindsSet := make(map[string]struct{})
	for _, attr := range patchedAttrs {
		patchedAttrsKindsSet[attr.Kind] = struct{}{}
	}

	patchedAttrsKinds := make([]string, 0, len(patchedAttrsKindsSet))
	for kind := range patchedAttrsKindsSet {
		patchedAttrsKinds = append(patchedAttrsKinds, kind)
	}

	patchedAttrsIDs, err := m.attrMgr.UpsertAttributes2(ctx, patchedAttrs, patchedAttrsKinds)
	if err != nil {
		return fmt.Errorf("error while upserting attributes (docID: '%s', attrs: %v): %w", docID, attrs, err)
	}

	// Update document with new attributes
	query := fmt.Sprintf(` UPDATE %s SET attr_ids = $2 WHERE doc_id = $1 `, m.tableDoc)
	_, err = m.exec(ctx, query, docID, patchedAttrsIDs)
	if err != nil {
		return fmt.Errorf("failed to update doc='%s' attributes: %w", docID, err)
	}
	return nil
}

func (m *docManager) DocumentExists(ctx context.Context, docID string) (bool, error) {
	return m.documentExists(ctx, docID)
}

func (m *docManager) documentExists(ctx context.Context, docID string) (bool, error) {
	query := fmt.Sprintf("SELECT true FROM %s WHERE doc_id = $1", m.tableDoc)
	r, err := m.query(ctx, query, docID)
	if err != nil {
		return false, fmt.Errorf("failed to check if document '%s' exists: %w", docID, err)
	}
	defer func() {
		cerr := r.Close()
		if cerr != nil {
			m.logger.Error("error closing rows", zap.Error(cerr))
		}
	}()
	return r.Next(), nil
}

func (m *docManager) getAttributeIDs(ctx context.Context, docID string) ([]int64, error) {
	query := fmt.Sprintf("SELECT unnest(attr_ids) FROM %s WHERE doc_id = $1", m.tableDoc)
	r, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query [%s], %w", query, err)
	}
	defer func() {
		cerr := r.Close()
		if cerr != nil {
			m.logger.Error("error closing rows", zap.Error(cerr))
		}
	}()
	var attrIDs []int64
	for r.Next() {
		attrID := int64(0)
		err = r.Scan(&attrID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		attrIDs = append(attrIDs, attrID)
	}
	return attrIDs, nil
}
