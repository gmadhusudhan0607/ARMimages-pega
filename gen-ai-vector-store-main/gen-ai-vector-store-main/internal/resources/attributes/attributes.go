/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
)

const (
	AttributeKindStatic       = "static"
	AttributeKindAutoResolved = "auto-resolved"
	AttributeKindIndex        = "index"
)

var ALLOWED_KINDS = []string{AttributeKindStatic, AttributeKindAutoResolved, AttributeKindIndex}

func ContainsAttribute(retrieveAttributes []string, attr Attribute) bool {
	for _, r := range retrieveAttributes {
		if r == attr.Name {
			return true
		}
	}
	return false
}

func (m *attrManager) UpsertAttributes2(ctx context.Context, attrs []Attribute, extraAttributesKinds []string) (attrItemIds []int64, err error) {
	if !slices.Contains(extraAttributesKinds, AttributeKindStatic) {
		extraAttributesKinds = append(extraAttributesKinds, AttributeKindStatic)
	}
	selectedAttrs := make([]Attribute, 0)
	for _, attr := range attrs {
		if slices.Contains(extraAttributesKinds, attr.Kind) {
			selectedAttrs = append(selectedAttrs, attr)
		}
	}
	attrItems := m.attributesToUniqAttributeItems(selectedAttrs)
	if err = m.upsertAttributeItems(ctx, attrItems); err != nil {
		return nil, fmt.Errorf("error while upserting attributes: %w", err)
	}
	return m.getAttributeItemIds2(ctx, attrItems)
}

func (m *attrManager) attributesToUniqAttributeItems(attrs []Attribute) (attrItems []AttributeItem) {
	for _, attr := range attrs {
		for _, val := range attr.Values {
			item := AttributeItem{
				Name:  attr.Name,
				Type:  attr.Type,
				Value: val,
			}
			if !containsAttributeItem(attrItems, item) {
				attrItems = append(attrItems, item)
			}
		}
	}
	return attrItems
}

func containsAttributeItem(items []AttributeItem, item AttributeItem) bool {
	for _, i := range items {
		if i.Name == item.Name && i.Type == item.Type && i.Value == item.Value {
			return true
		}
	}
	return false
}

func (m *attrManager) upsertAttributeItems(ctx context.Context, attrItems []AttributeItem) (err error) {
	if len(attrItems) == 0 {
		return nil
	}

	rowsAffected := int64(0)
	for idx, attr := range attrItems {
		// Check context before each query to save resources, time and decrease not needed DB connection usage
		// If timeout occurs at item 4 of 10, the function will still attempt to execute queries
		//   for items 6-100, each failing with context errors. This means:
		//
		//   - 6 unnecessary database connection attempts
		//   - 6 unnecessary network round-trips
		//   - Significantly longer total execution time (each attempt might take milliseconds to fail)
		//   - More resource consumption
		//   - ExecContext will fail, but we still tried
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("operation timeout exceeded: attribute_upserting - isolation: %s, collection: %s, processedCount: %d, totalCount: %d (original: %w)",
				m.IsolationID, m.CollectionID, idx, len(attrItems), ctx.Err())
		}

		query := fmt.Sprintf(
			`INSERT INTO %[1]s (name, type, value, value_hash) 
                    SELECT $1, $2, $3, md5($3)
                    WHERE NOT EXISTS (
                    	SELECT 1 FROM %[1]s 
                    	WHERE name = $1 AND type = $2 AND value_hash = md5($3) AND value = $3
                    )
                    ON CONFLICT DO NOTHING
					`, m.tableAttr)

		var res sql.Result
		res, err = m.Exec(ctx, query, attr.Name, attr.Type, attr.Value)
		if err != nil {
			return fmt.Errorf("error executing query [%s]: %w", query, err)
		}
		var ra int64
		ra, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error getting rows affected [%s]: %w", query, err)
		}
		rowsAffected += ra
	}
	if rowsAffected > 0 {
		m.logger.Debug("inserted new attribute records",
			zap.Int64("rowsAffected", rowsAffected))
	}
	return nil
}

func (m *attrManager) getAttributeItemIds2(ctx context.Context, attrItems []AttributeItem) (attrItemIds []int64, err error) {
	if len(attrItems) == 0 {
		return nil, nil
	}

	var sqlClauses []string
	var params []interface{}
	for i, item := range attrItems {
		sqlClause := fmt.Sprintf("(name = $%d AND type = $%d AND value_hash = $%d)", i*3+1, i*3+2, i*3+3)
		sqlClauses = append(sqlClauses, sqlClause)
		params = append(params, item.Name, item.Type, db.GetMD5Hash(item.Value))
	}

	query := fmt.Sprintf("SELECT attr_id FROM %s WHERE %s", m.tableAttr, strings.Join(sqlClauses, " OR "))

	var rows *sql.Rows
	rows, err = m.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("error executing query [%s]: %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		attrId := int64(0)
		if err = rows.Scan(&attrId); err != nil {
			return nil, fmt.Errorf("error scanning attrItemIds: %w", err)
		}
		attrItemIds = append(attrItemIds, attrId)
	}
	if len(attrItems) != len(attrItemIds) {
		return nil, fmt.Errorf("error getting attrItemIds expected %d ids , got %d : %w", len(attrItems), len(attrItemIds), err)
	}
	return attrItemIds, nil
}

func (m *attrManager) GetAttributesIDs(ctx context.Context, attrs []Attribute) (attrItemIds []int64, err error) {
	if len(attrs) == 0 {
		return nil, nil
	}

	attrItems := m.attributesToUniqAttributeItems(attrs)

	var sqlClauses []string
	var params []interface{}
	for i, item := range attrItems {
		sqlClause := fmt.Sprintf("(name = $%d AND type = $%d AND value_hash = $%d)", i*3+1, i*3+2, i*3+3)
		sqlClauses = append(sqlClauses, sqlClause)
		params = append(params, item.Name, item.Type, db.GetMD5Hash(item.Value))
	}

	query := fmt.Sprintf("SELECT attr_id FROM %s WHERE %s", m.tableAttr, strings.Join(sqlClauses, " OR "))
	var rows *sql.Rows
	rows, err = m.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("error executing query [%s]: %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		attrId := int64(0)
		if err = rows.Scan(&attrId); err != nil {
			return nil, fmt.Errorf("error scanning attrItemIds: %w", err)
		}
		attrItemIds = append(attrItemIds, attrId)
	}
	m.logger.Info("attrItemIds",
		zap.Any("attrItemIds", attrItemIds))

	if len(attrItems) != len(attrItemIds) {
		return nil, fmt.Errorf("error getting attr IDs:  expected %d ids , got %d : %w", len(attrItems), len(attrItemIds), err)
	}
	return attrItemIds, nil
}

func (m *attrManager) GetAttributesByIDs(ctx context.Context, attrIDs []int64) (result []Attribute, err error) {
	query := fmt.Sprintf("SELECT vector_store.attributes_as_jsonb_by_ids('%s', $1::bigint[] ) ", m.tableAttr)

	var rows *sql.Rows
	rows, err = m.Query(ctx, query, attrIDs)

	if err != nil {
		return nil, fmt.Errorf("failed to execute query [%s], %w", query, err)
	}
	defer rows.Close()

	//var attrs Attributes
	if rows.Next() {
		attrs := Attributes{}
		err = rows.Scan(&attrs)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		return attrs, nil
	}
	return nil, nil
}

func PatchAttributes(srcAttrs, patchAttrs []Attribute) []Attribute {

	// initialize map with srcAttrs
	attrMap := map[string]Attribute{}
	for _, a := range srcAttrs {
		attrMap[a.Name] = Attribute{
			Name:   a.Name,
			Values: a.Values,
			Type:   a.Type,
			Kind:   a.Kind,
		}
	}

	// update map with patchAttrs
	for _, a := range patchAttrs {
		_, ok := attrMap[a.Name]
		if ok {
			// Update existing attribute
			attrMap[a.Name] = Attribute{
				Name:   a.Name,
				Values: a.Values,
				Type:   a.Type,
				Kind:   a.Kind,
			}
		} else {
			// Add new attribute
			attrMap[a.Name] = a
		}
	}
	return values(attrMap)
}

// Values returns all values from a map
func values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func (m *attrManager) FindAttributes(ctx context.Context, names []string) (attrs []Attribute, err error) {
	tableDoc := db.GetTableDoc(m.IsolationID, m.CollectionID)
	tableEmb := db.GetTableEmb(m.IsolationID, m.CollectionID)
	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)
	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	nameFilter := ""
	if len(names) > 0 {
		nameFilter = fmt.Sprintf("AND ATTR.name IN ('%s')", strings.Join(names, "','"))
	}

	query := fmt.Sprintf(`
		SELECT row_to_json(attr_row) as attribute
		FROM (
				 SELECT name, type, array_agg(DISTINCT ATTR.value ORDER BY ATTR.value) as value
				 FROM %[1]s ATTR
				 WHERE attr_id IN (
				    -- Select only attributes actually used in documents and embeddings
					SELECT unnest(attr_ids) as attr_id from %[2]s
					UNION DISTINCT
					SELECT unnest(attr_ids) as attr_id from %[3]s
					UNION DISTINCT
					SELECT unnest(attr_ids) as attr_id from %[5]s
					UNION DISTINCT
					SELECT unnest(attr_ids) as attr_id from %[6]s
				 )
				 %[4]s
				 GROUP BY ATTR.name, ATTR.type
			 ) attr_row
		ORDER BY attr_row.name
    `, m.tableAttr, tableDoc, tableEmb, nameFilter, tableDocProcessing, tableEmbProcessing)

	var r *sql.Rows
	r, err = m.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query [%s], %w", query, err)
	}
	defer r.Close()

	for r.Next() {
		var attr Attribute
		err = r.Scan(&attr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

// Compare compares two Attribute objects for equality.
func (a *Attribute) Compare(other Attribute) bool {
	if a.Name != other.Name || a.Type != other.Type {
		return false
	}
	if len(a.Values) != len(other.Values) {
		return false
	}
	// Compare values ignoring order
	for _, v := range a.Values {
		if !slices.Contains(other.Values, v) {
			return false
		}
	}
	return true
}

func (m *attrManager) GetEmbeddingAttributes(ctx context.Context, docId, embID string, filterNames []string) (attrs []Attribute, err error) {
	tableDoc := db.GetTableDoc(m.IsolationID, m.CollectionID)
	tableEmb := db.GetTableEmb(m.IsolationID, m.CollectionID)

	return m.getEmbeddingAttributesProcessing(ctx, docId, embID, filterNames, tableDoc, tableEmb)
}

func (m *attrManager) GetEmbeddingAttributesProcessing(ctx context.Context, docId, embID string, filterNames []string) (attrs []Attribute, err error) {
	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)
	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	return m.getEmbeddingAttributesProcessing(ctx, docId, embID, filterNames, tableDocProcessing, tableEmbProcessing)
}

func (m *attrManager) getEmbeddingAttributesProcessing(ctx context.Context, docId, embID string, filterNames []string, tableDoc, tableEmb string) (attrs []Attribute, err error) {
	nameFilter := ""
	if len(filterNames) > 0 {
		nameFilter = fmt.Sprintf("AND ATTR.name IN ('%s')", strings.Join(filterNames, "','"))
	}

	query := fmt.Sprintf(`
		SELECT row_to_json(attr_row) as attribute
		FROM (
				 SELECT name, type, array_agg(DISTINCT ATTR.value ORDER BY ATTR.value) as value
				 FROM %[1]s ATTR
				 WHERE attr_id IN (
				    -- Select only attributes actually used in documents and embeddings
					SELECT unnest(attr_ids) as attr_id
                        FROM %[2]s
                        WHERE doc_id = $1
					UNION DISTINCT
					SELECT unnest(attr_ids) as attr_id
                        FROM %[3]s
                        WHERE doc_id = $1 AND emb_id = $2
				  )
				 %[4]s
				 GROUP BY ATTR.name, ATTR.type
			 ) attr_row
		ORDER BY attr_row.name
    `, m.tableAttr, tableDoc, tableEmb, nameFilter)

	var rows *sql.Rows
	rows, err = m.Query(ctx, query, docId, embID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query [%s], %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var attr Attribute
		err = rows.Scan(&attr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}
