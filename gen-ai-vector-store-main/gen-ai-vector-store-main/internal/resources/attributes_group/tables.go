/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributesgroup

import (
	"context"
	"fmt"
)

func (m *attrGrpManager) CreateTables(ctx context.Context) (err error) {
	defer m.rollbackTransactionIfError(&err)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
		    group_id    VARCHAR(40) NOT NULL UNIQUE,
		    description TEXT,
		    attributes  TEXT[],
		 PRIMARY KEY (group_id) )
    `, m.attrGrpTable)

	_, err = m.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error while creating table: %w", err)
	}

	query = fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx1_smart_attributes_group__id on %s (group_id)", m.attrGrpTable)
	_, err = m.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error while creating index: %w", err)
	}
	return nil
}

func (m *attrGrpManager) DropTables(ctx context.Context) (err error) {
	defer m.rollbackTransactionIfError(&err)

	// Drop the table will also drop the index
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", m.attrGrpTable)
	_, err = m.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error while dropping table [%s]: %w", query, err)
	}
	return nil
}
