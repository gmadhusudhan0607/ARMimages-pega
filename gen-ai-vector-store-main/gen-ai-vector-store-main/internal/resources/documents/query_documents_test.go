/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"math"
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

func Test_manager_buildFindDocumentsSqlQuery(t *testing.T) {
	type fields struct {
		IsoID          string
		ColID          string
		Embedder       embedders.Embedder
		dbConn         db.Database
		dbSchemaName   string
		dbTablesPrefix string
	}
	type args struct {
		docReq   *QueryDocumentsRequest
		cteLimit int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
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
				docReq: &QueryDocumentsRequest{
					Limit:              10,
					MaxDistance:        float64Ptr(0.5),
					RetrieveAttributes: nil,
					Filters:            filters.RequestFilter{},
				},
				cteLimit: 100,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 100
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 0.500000	AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 10 `,
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
				docReq: &QueryDocumentsRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Operator: "in",
								Name:     "attr1",
								Type:     "type1",
								Values:   []string{"val1", "val2"},
							},
						},
					},
				},
				cteLimit: 2147483647,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   WHERE
							attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND
								value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 2147483647
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 1.0 AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 2147483647 `,
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
				docReq: &QueryDocumentsRequest{
					Limit:              5,
					MaxDistance:        nil,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Operator: "in",
								Name:     "attr1",
								Type:     "type1",
								Values:   []string{"val1", "val2"},
							},
						},
					},
				},
				cteLimit: 50,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   WHERE
							attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND
								value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 50
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 1.0 AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 5 `,
		},
		{
			name: "with max distance and filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				docReq: &QueryDocumentsRequest{
					Limit:              0,
					MaxDistance:        float64Ptr(0.5),
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Operator: "in",
								Name:     "attr1",
								Type:     "type1",
								Values:   []string{"val1", "val2"},
							},
						},
					},
				},
				cteLimit: 2147483647,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   WHERE
							attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND
								value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */	
					   ORDER BY distance
					   LIMIT 2147483647
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 0.500000 AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 2147483647 `,
		},
		{
			name: "with limit, max distance and filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				docReq: &QueryDocumentsRequest{
					Limit:              7,
					MaxDistance:        float64Ptr(0.5),
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Operator: "in",
								Name:     "attr1",
								Type:     "type1",
								Values:   []string{"val1", "val2"},
							},
						},
					},
				},
				cteLimit: 70,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   WHERE
							attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND
								value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad')))
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 70
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 0.500000 AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 7 `,
		},

		{
			name: "with two filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				docReq: &QueryDocumentsRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveAttributes: nil,
					Filters: filters.RequestFilter{
						SubFilters: []attributes.AttributeFilter{
							{
								Operator: "in",
								Name:     "attr1",
								Type:     "type1",
								Values:   []string{"val1", "val2"},
							},
							{
								Operator: "in",
								Name:     "attr2",
								Type:     "type2",
								Values:   []string{"val3", "val4"},
							},
						},
					},
				},
				cteLimit: 2147483647,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   WHERE
							attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr1' AND type = 'type1' AND
								value_hash IN ('8de92ce2033cf3ca03fa8cc63e7a703f', '38ceaa3b09c5a07d329888ba1ccde9ad'))
							)
						AND attr_ids2 && (
								SELECT array_agg(attr_id)
								FROM vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr
								WHERE (name = 'attr2' AND type = 'type2' AND
								value_hash IN ('9163c8c66d03c512404cca8549a250e7', 'a8516b0468eb31028c3dd669867c15b1'))
							)
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 2147483647
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 1.0 AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 2147483647 `,
		},
		{
			name: "without limit, max distance and filters",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				docReq: &QueryDocumentsRequest{
					Limit:              0,
					MaxDistance:        nil,
					RetrieveAttributes: nil,
					Filters:            filters.RequestFilter{},
				},
				cteLimit: 2147483647,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 2147483647
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 1.0	AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 2147483647 `,
		},
		{
			name: "CTE limit differs from final limit",
			fields: fields{
				IsoID:          "iso1",
				ColID:          "col1",
				Embedder:       nil,
				dbConn:         nil,
				dbSchemaName:   "vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd",
				dbTablesPrefix: "t_8c43de7b01bca674276c43e09b3ec5ba",
			},
			args: args{
				docReq: &QueryDocumentsRequest{
					Limit:              10,
					MaxDistance:        float64Ptr(0.8),
					RetrieveAttributes: nil,
					Filters:            filters.RequestFilter{},
				},
				cteLimit: 200,
			},
			want: `
				WITH filtered_embeddings_with_distance AS (
					SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1	as distance
					   FROM  vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB
					   /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
					   ORDER BY distance
					   LIMIT 200
				), runked_filtered_embeddings AS (
					SELECT emb_id, distance, ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
					FROM filtered_embeddings_with_distance
					 ORDER BY distance, emb_id
				)
				SELECT DOC.doc_id,
					   distance,
					   vector_store.attributes_as_jsonb_by_ids('vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_attr', EMB.attr_ids2 ) as attributes
				FROM runked_filtered_embeddings RFE
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_emb EMB ON RFE.emb_id = EMB.emb_id
						 LEFT JOIN vector_store_464328fbc04e8cf4f2d0f6109f7fbfdd.t_8c43de7b01bca674276c43e09b3ec5ba_doc DOC ON DOC.doc_id = EMB.doc_id
				WHERE distance <= 0.800000	AND rank = 1
				ORDER BY distance, doc_id
				LIMIT 10 `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &docManager{
				IsolationID:  tt.fields.IsoID,
				CollectionID: tt.fields.ColID,
				Embedder:     tt.fields.Embedder,
				database:     tt.fields.dbConn,
				schemaName:   tt.fields.dbSchemaName,
				prefix:       tt.fields.dbTablesPrefix,
			}

			got, err := mgr.buildFindDocumentsSqlQuery(tt.args.docReq, tt.args.cteLimit)
			// ensure no error occurred
			if err != nil {
				t.Errorf("buildFindDocumentsSqlQuery() error = %v", err)
			}

			got = TrimSpaces(got)
			want := TrimSpaces(tt.want)

			if got != want {
				t.Errorf("buildFindDocumentsSqlQuery() = %v, want %v", got, tt.want)
			}
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

func Test_safeMul(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "normal multiplication",
			a:    10,
			b:    10,
			want: 100,
		},
		{
			name: "multiplier of 1",
			a:    42,
			b:    1,
			want: 42,
		},
		{
			name: "zero a returns MaxInt32",
			a:    0,
			b:    10,
			want: math.MaxInt32,
		},
		{
			name: "zero b returns MaxInt32",
			a:    10,
			b:    0,
			want: math.MaxInt32,
		},
		{
			name: "negative a returns MaxInt32",
			a:    -1,
			b:    10,
			want: math.MaxInt32,
		},
		{
			name: "negative b returns MaxInt32",
			a:    10,
			b:    -5,
			want: math.MaxInt32,
		},
		{
			name: "overflow returns MaxInt32",
			a:    math.MaxInt32,
			b:    2,
			want: math.MaxInt32,
		},
		{
			name: "large values that don't overflow",
			a:    1000,
			b:    1000,
			want: 1000000,
		},
		{
			name: "MaxInt32 times 1 returns MaxInt32",
			a:    math.MaxInt32,
			b:    1,
			want: math.MaxInt32,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeMul(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("safeMul(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
