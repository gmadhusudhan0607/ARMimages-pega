/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/filters"
)

func float64Ptr(f float64) *float64 {
	return &f
}

func Test_manager_buildFindChunksSqlQuery(t *testing.T) {
	type fields struct {
		IsoID          string
		ColID          string
		Embedder       embedders.Embedder
		dbConn         db.Database
		dbSchemaName   string
		dbTablesPrefix string
	}
	type args struct {
		chReq *QueryChunksRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "without filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters:            filters.RequestFilter{},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 2147483647
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 1.0
				ORDER BY distance, emb_id
				LIMIT 2147483647
			`,
		},
		{
			name: "with limit and max distance",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              10,
					MaxDistance:        float64Ptr(0.5),
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters:            filters.RequestFilter{},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 10
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 0.500000
				ORDER BY distance, emb_id
				LIMIT 10
			`,
		},
		{
			name: "with filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Name:   "attr1",
								Type:   "type1",
								Values: []string{"val1", "val2"},
							},
						},
					},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
				    WHERE attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad'))
							)
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 2147483647
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 1.0
				ORDER BY distance, emb_id
				LIMIT 2147483647
			`,
		},
		{
			name: "with limit and filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              7,
					MaxDistance:        nil,
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Name:   "attr1",
								Type:   "type1",
								Values: []string{"val1", "val2"},
							},
						},
					},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
				    WHERE attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad'))
							)
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 7
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 1.0
				ORDER BY distance, emb_id
				LIMIT 7
			`,
		},
		{
			name: "with two attributes",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Name:   "attr1",
								Type:   "type1",
								Values: []string{"val1", "val2"},
							},
							{
								Name:   "attr2",
								Type:   "type2",
								Values: []string{"val3", "val4"},
							},
						},
					},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
				    WHERE attr_ids2 && (
							SELECT array_agg(attr_id)
							FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
							WHERE (name = 'attr1' AND type = 'type1' AND value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					  AND attr_ids2 && (
							SELECT array_agg(attr_id)
							FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
							WHERE (name = 'attr2' AND type = 'type2' AND  value_hash IN ('9163c8c66d03c512404cca8549a250e7', 'a8516b0468eb31028c3dd669867c15b1')))
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 2147483647
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 1.0
				ORDER BY distance, emb_id
				LIMIT 2147483647
			`,
		},
		{
			name: "with three attributes",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				chReq: &QueryChunksRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveVector:     false,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Name:   "attr1",
								Type:   "type1",
								Values: []string{"val1", "val2"},
							},
							{
								Name:   "attr2",
								Type:   "type2",
								Values: []string{"val3", "val4"},
							},
							{
								Name:   "attr3",
								Type:   "type3",
								Values: []string{"val5", "val6"},
							},
						},
					},
				},
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
                    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
                    FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
				    WHERE attr_ids2 && (
							SELECT array_agg(attr_id)
							FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
							WHERE (name = 'attr1' AND type = 'type1' AND value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					  AND attr_ids2 && (
							SELECT array_agg(attr_id)
							FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
							WHERE (name = 'attr2' AND type = 'type2' AND  value_hash IN ('9163c8c66d03c512404cca8549a250e7', 'a8516b0468eb31028c3dd669867c15b1')))
					  AND attr_ids2 && (
							SELECT array_agg(attr_id)
							FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
							WHERE (name = 'attr3' AND type = 'type3' AND  value_hash IN ('294a6e0d759cdbcf55753c4a58161721', '1b18b7cd2d63787cbe1d97e57a54853c')))
					/* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
                    ORDER BY distance
                    LIMIT 2147483647
                )
				SELECT EMB.emb_id,
					   EMB.doc_id,
					   content,
					   vector_store.attributes_as_jsonb_by_ids( 'vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr' , EMB.attr_ids2 ) as attributes,
					   distance
				FROM filtered_embeddings_with_distance FDEMB LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON FDEMB.emb_id = EMB.emb_id
				WHERE distance <= 1.0
				ORDER BY distance, emb_id
				LIMIT 2147483647
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &embManager{
				IsolationID:  tt.fields.IsoID,
				CollectionID: tt.fields.ColID,
				Embedder:     tt.fields.Embedder,
				database:     tt.fields.dbConn,
				schemaName:   tt.fields.dbSchemaName,
				tablesPrefix: tt.fields.dbTablesPrefix,
			}

			got := mgr.buildFindChunksSqlQuery(tt.args.chReq)

			got = TrimSpaces(got)
			want := TrimSpaces(tt.want)

			if got != want {
				t.Errorf("buildFindChunksSqlQuery() = %v, want %v", got, want)
			}
			println(got)
		})
	}
}

func TrimSpaces(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\) ,`).ReplaceAllString(s, "),")
	s = regexp.MustCompile(`\s\)`).ReplaceAllString(s, ")")
	s = regexp.MustCompile(`\(\s`).ReplaceAllString(s, "(")
	return s
}
