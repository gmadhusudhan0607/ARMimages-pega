/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributesgroup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrAttributeGroupDoesNotExist = errors.New("AttributeGroup does not exist")

func (m *attrGrpManager) CreateAttributesGroup(ctx context.Context, description string, attrs []string) (attrsGroup *AttributesGroup, err error) {
	defer m.rollbackTransactionIfError(&err)

	gGUID := uuid.New().String()
	query := fmt.Sprintf(`INSERT INTO %[1]s (group_id, description, attributes) VALUES ($1, $2, $3)`, m.attrGrpTable)
	_, err = m.Exec(ctx, query, gGUID, description, attrs)
	if err != nil {
		return nil, fmt.Errorf("error while creating attributes group [%s]: %w", query, err)
	}
	return &AttributesGroup{GroupID: gGUID, Description: description, Attributes: attrs}, nil

}

func (m *attrGrpManager) GetAttributesGroup(ctx context.Context, groupID string) (attrsGroup *AttributesGroup, err error) {
	defer m.rollbackTransactionIfError(&err)
	var rows *sql.Rows

	query := fmt.Sprintf(`SELECT description, array_to_string(attributes, ',') FROM  %s WHERE group_id = $1`, m.attrGrpTable)
	rows, err = m.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("error while getting attributes group [%s]: %w", query, err)
	}
	defer rows.Close()
	if rows.Next() {
		var description, AttributesList string
		err = rows.Scan(&description, &AttributesList)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		return &AttributesGroup{
			GroupID:     groupID,
			Description: description,
			Attributes:  strings.Split(AttributesList, ","),
		}, nil
	}
	return nil, ErrAttributeGroupDoesNotExist
}

func (m *attrGrpManager) GetAttributesGroupDescriptions(ctx context.Context) (agDescr map[string]string, err error) {
	defer m.rollbackTransactionIfError(&err)

	agDescr = make(map[string]string)
	query := fmt.Sprintf(` SELECT group_id, description FROM %s`, m.attrGrpTable)
	var rows *sql.Rows
	rows, err = m.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error while getting attributes group descriptions [%s]: %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		var groupID, description string
		err = rows.Scan(&groupID, &description)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query [%s], %w", query, err)
		}
		agDescr[groupID] = description
	}
	return agDescr, nil
}

func (m *attrGrpManager) DeleteAttributesGroup(ctx context.Context, groupID string) (err error) {
	defer m.rollbackTransactionIfError(&err)

	query := fmt.Sprintf(`DELETE FROM %s WHERE group_id = $1`, m.attrGrpTable)
	_, err = m.Exec(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("error while deleting attributes group '%s' [%s]: %w", groupID, query, err)
	}
	return nil
}
