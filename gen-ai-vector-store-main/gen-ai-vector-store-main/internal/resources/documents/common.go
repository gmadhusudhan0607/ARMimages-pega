/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
)

func (m *docManager) getAttrsWhereClause(filterAttrs []attributes.AttributeFilter, attrIDsColumn string) (string, error) {

	if !(attrIDsColumn == "attr_ids" || attrIDsColumn == "attr_ids2") {
		return "", fmt.Errorf("unsupported column '%s'. Must be one of [ attr_ids, attr_ids2 ]", attrIDsColumn)
	}

	if len(filterAttrs) == 0 {
		return "", nil
	}
	attrClauses := []string{}
	var comparisonOperator string
	for _, attr := range filterAttrs {
		attrSelectClause, err := attributeToSqlClause(attr)
		if err != nil {
			return "", fmt.Errorf("failed to create WHERE clause from attributes: %w", err)
		}

		switch strings.ToLower(attr.Operator) {
		case "in", "or", "any":
			comparisonOperator = "&&"
		case "and", "all":
			comparisonOperator = "@>"
		default:
			return "", fmt.Errorf("unsupported operator '%s'. Must be one of [in,or,any] or one of [all,and] ", attr.Operator)
		}

		attrClauseTpl := `
            %[5]s %[4]s (
                SELECT array_agg(attr_id)
                FROM %[1]s.%[2]s_attr
                WHERE %[3]s
            )`
		attrClause := fmt.Sprintf(attrClauseTpl, m.schemaName, m.prefix, attrSelectClause, comparisonOperator, attrIDsColumn)
		attrClauses = append(attrClauses, attrClause)
	}
	return fmt.Sprintf("WHERE %s", strings.Join(attrClauses, "\n     AND ")), nil
}

func attributeToSqlClause(attrFilter attributes.AttributeFilter) (string, error) {
	var valuesHashes []string
	for _, val := range attrFilter.Values {
		hash := md5.Sum([]byte(val))
		valuesHashes = append(valuesHashes, fmt.Sprintf("'%s'", hex.EncodeToString(hash[:])))
	}
	// TODO : refactor this later (default must not be set on this level)
	attrType := attrFilter.Type
	if attrType == "" {
		attrType = "string"
	}

	return fmt.Sprintf("( name = '%s' AND type = '%s' AND value_hash IN ( %s ) )", attrFilter.Name, attrType, strings.Join(valuesHashes, ", ")), nil
}

func (m *docManager) getAttrsProcessingWhereClause(filterAttrs []attributes.AttributeFilter, status string) (string, error) {
	if len(filterAttrs) == 0 {
		return "", nil
	}

	statusFilter := ""
	if status != "" {
		statusFilter = fmt.Sprintf(" AND status = '%s'", status)
	}

	attrClauses := []string{}
	var comparisonOperator string
	for _, attr := range filterAttrs {
		attrSelectClause, err := attributeToSqlClause(attr)
		if err != nil {
			return "", fmt.Errorf("failed to create WHERE clause from attributes: %w", err)
		}

		switch strings.ToLower(attr.Operator) {
		case "in", "or", "any":
			comparisonOperator = "&&"
		case "and", "all":
			comparisonOperator = "@>"
		default:
			return "", fmt.Errorf("unsupported operator '%s'. Must be one of [in,or,any] or one of [all,and] ", attr.Operator)
		}

		attrClauseTpl := `attr_ids %[4]s (
				SELECT array_agg(attr_id)
				FROM %[1]s.%[2]s_attr
				WHERE %[3]s
			)`
		attrClause := fmt.Sprintf(attrClauseTpl, m.schemaName, m.prefix, attrSelectClause, comparisonOperator)
		attrClauses = append(attrClauses, attrClause)
	}

	attrsFilter := strings.Join(attrClauses, "\n     AND ")

	return fmt.Sprintf(`WHERE DOC.doc_id IN (
		SELECT doc_id
		FROM %[1]s.%[2]s_doc
		WHERE 
			%[3]s
			%[4]s
		UNION ALL
		SELECT doc_id
		FROM %[1]s.%[2]s_doc_processing
		WHERE 
			%[3]s
		UNION ALL
		SELECT doc_id
		FROM %[1]s.%[2]s_emb
		WHERE
			%[3]s
		UNION ALL
		SELECT doc_id
		FROM %[1]s.%[2]s_emb_processing
		WHERE
			%[3]s
	)`, m.schemaName, m.prefix, attrsFilter, statusFilter), nil
}
