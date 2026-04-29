// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package documents

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
)

// buildDocJSONBContainmentClause creates a single JSONB containment check for doc_attributes filtering
// Returns: doc_attributes @> '{"attrName": {"values": ["value"]}}'::jsonb
func buildDocJSONBContainmentClause(attrName, value string, forEmbAttributes bool) string {
	// Build the JSONB structure for containment check
	// We need to properly escape the value for JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		// If marshaling fails, fall back to simple escaping
		valueBytes = []byte(fmt.Sprintf("%q", value))
	}

	attributesColumn := "doc_attributes"
	if forEmbAttributes {
		attributesColumn = "emb_attributes"
	}

	return fmt.Sprintf("%s @> '{%q: {\"values\": [%s]}}'::jsonb", attributesColumn, attrName, string(valueBytes))
}

// attributeToDocJSONBSqlClause converts a single AttributeFilter to JSONB SQL condition for doc_attributes
// For multiple values with "in/or/any" operator, generates OR conditions with containment checks
// For multiple values with "all/and" operator, generates AND conditions with containment checks
func attributeToDocJSONBSqlClause(attr attributes.AttributeFilter, forEmbAttributes bool) (string, error) {
	if len(attr.Values) == 0 {
		return "", nil
	}

	// Determine the logical operator based on the filter operator
	var logicalOp string
	switch strings.ToLower(attr.Operator) {
	case "in", "or", "any":
		logicalOp = "OR"
	case "and", "all":
		logicalOp = "AND"
	default:
		return "", fmt.Errorf("unsupported operator '%s'. Must be one of [in,or,any] or one of [all,and]", attr.Operator)
	}

	// Build individual containment clauses for each value
	var clauses []string
	for _, val := range attr.Values {
		clause := buildDocJSONBContainmentClause(attr.Name, val, forEmbAttributes)
		clauses = append(clauses, clause)
	}

	// Single value: return as-is
	if len(clauses) == 1 {
		return clauses[0], nil
	}

	// Multiple values: wrap in parentheses and join with appropriate operator
	return fmt.Sprintf("(\n        %s\n      )", strings.Join(clauses, fmt.Sprintf(" %s \n        ", logicalOp))), nil
}

// getDocJSONBAttrsWhereClause converts array of AttributeFilters to WHERE clause for doc_attributes
// Returns empty string if no filters, otherwise returns formatted WHERE clause
func (m *docManager) getDocJSONBAttrsWhereClause(filterAttrs []attributes.AttributeFilter, forEmbAttributes bool) (string, error) {
	if len(filterAttrs) == 0 {
		return "", nil
	}

	var attrClauses []string
	for _, attr := range filterAttrs {
		clause, err := attributeToDocJSONBSqlClause(attr, forEmbAttributes)
		if err != nil {
			return "", fmt.Errorf("failed to create JSONB WHERE clause from attribute '%s': %w", attr.Name, err)
		}
		if clause != "" {
			attrClauses = append(attrClauses, clause)
		}
	}

	if len(attrClauses) == 0 {
		return "", nil
	}

	return strings.Join(attrClauses, "\n     AND "), nil
}

// getDocJSONBAttrsProcessingWhereClause creates WHERE clause for querying both doc and doc_processing tables with JSONB filtering
// This is the JSONB equivalent of getAttrsProcessingWhereClause()
func (m *docManager) getDocJSONBAttrsProcessingWhereClause(filterAttrs []attributes.AttributeFilter, status string) (string, error) {
	if len(filterAttrs) == 0 {
		return "", nil
	}

	statusFilter := ""
	if status != "" {
		statusFilter = fmt.Sprintf(" AND status = '%s'", status)
	}

	attrsFilter, err := m.getDocJSONBAttrsWhereClause(filterAttrs, false)
	if err != nil {
		return "", err
	}

	if attrsFilter == "" {
		return "", nil
	}

	embAttrsFilter, err := m.getDocJSONBAttrsWhereClause(filterAttrs, true)
	if err != nil {
		return "", err
	}

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
			%[5]s
		UNION ALL
		SELECT doc_id
		FROM %[1]s.%[2]s_emb_processing
		WHERE
			%[5]s
	)`, m.schemaName, m.prefix, attrsFilter, statusFilter, embAttrsFilter), nil
}
