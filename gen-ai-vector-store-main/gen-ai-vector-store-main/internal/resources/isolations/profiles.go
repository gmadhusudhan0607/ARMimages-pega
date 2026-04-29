/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package isolations

import (
	"context"
	"fmt"
)

func (m *isoManager) GetIsolationProfiles(ctx context.Context, isolationID string) ([]EmbeddingProfile, error) {
	query := fmt.Sprintf(`
               SELECT profile_id, provider_name, model_name, model_version, vector_len, max_tokens 
               FROM %s.emb_profiles 
               WHERE isolation_id = $1
       `, m.dbSchemaName)

	rows, err := m.query(ctx, query, isolationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query embedding profiles for isolation %s: %w", isolationID, err)

	}
	defer rows.Close()

	var profiles []EmbeddingProfile
	for rows.Next() {
		var profile EmbeddingProfile
		if err := rows.Scan(
			&profile.ID,
			&profile.ProviderName,
			&profile.ModelName,
			&profile.ModelVersion,
			&profile.VectorLen,
			&profile.MaxTokens,
		); err != nil {
			return nil, fmt.Errorf("failed to scan embedding profile row: %w", err)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading embedding profiles: %w", err)
	}
	return profiles, nil
}
