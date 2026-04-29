/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func UpdateIsolation(baseURI, isolationID, maxStorageSize string) {
	GinkgoHelper()
	By(fmt.Sprintf("Updating isolation %s (maxStorageSize=%s)", isolationID, maxStorageSize))
	uri := fmt.Sprintf("%s/v1/isolations/%s", baseURI, isolationID)
	var jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, maxStorageSize)

	resp, body, err := HttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(string(body)).To(Equal(fmt.Sprintf(`{"id":"%s"}`, isolationID)))
}

func UpsertDoc(baseURI, isolationID, collectionID, consistencyLevel, jsonData string) {
	GinkgoHelper()
	By(fmt.Sprintf("Upserting document '%s/%s'", isolationID, collectionID))
	uri := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, consistencyLevel)

	resp, _, err := HttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	if consistencyLevel == indexer.ConsistencyLevelStrong {
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	} else {
		Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	}
}

func UpsertDocumentsFromDir(dir, baseURI, isolationID, collectionID string) (docIDs []string) {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	dir = fmt.Sprintf("%s/data/%s", path, dir)

	By(fmt.Sprintf("-> Upserting documents from dir %s", dir))
	f, err := os.Open(dir)
	Expect(err).To(BeNil())

	files, err := f.Readdir(0)
	Expect(err).To(BeNil())

	for _, file := range files {
		fPath := fmt.Sprintf("%s/%s", dir, file.Name())
		jsonData := ReadFile(fPath)
		d := &Item{}
		err = json.Unmarshal([]byte(jsonData), &d)
		Expect(err).To(BeNil())
		docIDs = append(docIDs, d.ID)
		UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, jsonData)
	}
	return docIDs
}

func UpsertDocumentsFromDirAndWaitForCOMPLETED(database *pgxpool.Pool, dir, baseURI, isolationID, collectionID string) (docIDs []string) {
	GinkgoHelper()
	docIDs = UpsertDocumentsFromDir(dir, baseURI, isolationID, collectionID)
	By("Waiting for completion")
	for _, docID := range docIDs {
		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, resources.StatusCompleted)
	}
	return docIDs
}

func UpsertDocumentsFromDirWithStrongConsistencyLevel(dir, baseURI, isolationID, collectionID string) (docIDs []string) {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	dir = fmt.Sprintf("%s/data/%s", path, dir)

	By(fmt.Sprintf("-> Upserting documents from dir %s", dir))
	f, err := os.Open(dir)
	Expect(err).To(BeNil())

	files, err := f.Readdir(0)
	Expect(err).To(BeNil())

	for _, file := range files {
		fPath := fmt.Sprintf("%s/%s", dir, file.Name())
		jsonData := ReadFile(fPath)
		d := &Item{}
		err = json.Unmarshal([]byte(jsonData), &d)
		Expect(err).To(BeNil())
		docIDs = append(docIDs, d.ID)
		UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong, jsonData)
	}
	return docIDs
}

type Item struct {
	ID string `json:"id" binding:"required"`
}
