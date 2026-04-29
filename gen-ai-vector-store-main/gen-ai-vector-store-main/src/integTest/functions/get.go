/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"fmt"
	"net/http"

	db2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func GetIsolation(baseURI, isolationID string) (resp *http.Response, body []byte, err error) {
	GinkgoHelper()
	By(fmt.Sprintf("Getting isolation %s", isolationID))
	uri := fmt.Sprintf("%s/v1/isolations/%s", baseURI, isolationID)
	resp, body, err = HttpCall("GET", uri, nil, "")
	Expect(err).To(BeNil())
	return resp, body, err
}

func GetDocumentStatusAndErrorFromDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string) (status, docErr string) {
	GinkgoHelper()
	By(fmt.Sprintf("-> Getting document status of %s/%s/%s from DB", isolationID, collectionID, docID))
	tableDoc := db2.GetTableDoc(isolationID, collectionID)
	sqlQuery := fmt.Sprintf("SELECT status, error_message FROM %s WHERE doc_id=$1", tableDoc)
	err1 := db.QueryRow(ctx, sqlQuery, docID).Scan(&status, &docErr)
	Expect(err1).To(BeNil())
	return status, docErr
}

func GetChunksCountFromDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string) (count int) {
	GinkgoHelper()
	By(fmt.Sprintf("-> Getting chunks of %s/%s/%s from DB", isolationID, collectionID, docID))
	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	sqlQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE doc_id=$1", tableEmb)
	err := db.QueryRow(ctx, sqlQuery, docID).Scan(&count)
	Expect(err).To(BeNil())
	return count
}

func GetChunksProcessingCountFromDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string) (count int) {
	GinkgoHelper()
	By(fmt.Sprintf("-> Getting chunks of %s/%s/%s from DB", isolationID, collectionID, docID))
	tableEmbProcessing := db2.GetTableEmbProcessing(isolationID, collectionID)
	sqlQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE doc_id=$1", tableEmbProcessing)
	err := db.QueryRow(ctx, sqlQuery, docID).Scan(&count)
	Expect(err).To(BeNil())
	return count
}

type EmbeddingEntry struct {
	EmbID    string  `json:"emb_id" binding:"required"`
	DocID    string  `json:"doc_id" binding:"required"`
	AttrIDs  []int64 `json:"attr_ids,omitempty"`
	AttrIDs2 []int64 `json:"attr_ids2,omitempty"`
}

func GetEmbeddingFromDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID, embID string) *EmbeddingEntry {
	GinkgoHelper()
	By(fmt.Sprintf("-> Getting embedding of %s/%s/%s from DB", isolationID, collectionID, docID))
	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	sqlQuery := fmt.Sprintf("SELECT emb_id, doc_id, attr_ids, attr_ids2 FROM %s WHERE doc_id=$1 AND emb_id = $2 ", tableEmb)
	rows, err := db.Query(ctx, sqlQuery, docID, embID)
	Expect(err).To(BeNil())
	defer rows.Close()

	if rows.Next() {
		var entry EmbeddingEntry
		err := rows.Scan(&entry.EmbID, &entry.DocID, &entry.AttrIDs, &entry.AttrIDs2)
		Expect(err).To(BeNil())
		return &entry
	}
	return nil
}

func GetAttrIdFormDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, attrName, attrType, attrValue string) (attrID int64) {
	GinkgoHelper()
	tableAttr := db2.GetTableAttr(isolationID, collectionID)
	sqlQuery := fmt.Sprintf("SELECT attr_id FROM %s WHERE name=$1 AND type = $2 AND value = $3", tableAttr)
	err1 := db.QueryRow(ctx, sqlQuery, attrName, attrType, attrValue).Scan(&attrID)
	Expect(err1).To(BeNil())
	return attrID
}
