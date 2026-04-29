//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing OPS /v1/* ", func() {

	ctx := context.TODO()
	var isolationID string

	BeforeEach(func() {
		isolationID = fmt.Sprintf("test-%s", RandStringRunes(20))
	})

	_ = Context("environment ready", func() {

		It("can create/delete isolation", func() {
			CreateIsolation(baseOpsURI, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			By(fmt.Sprintf("Checking if the tables are created in the database for the isolation %s", isolationID))
			schemaName := db.GetSchema(isolationID)
			ExpectTableExistsInDB(ctx, database, fmt.Sprintf("%s.collections", schemaName))
			ExpectTableExistsInDB(ctx, database, fmt.Sprintf("%s.emb_profiles", schemaName))
			ExpectTableExistsInDB(ctx, database, fmt.Sprintf("%s.collection_emb_profiles", schemaName))
			ExpectTableExistsInDB(ctx, database, fmt.Sprintf("%s.smart_attributes_group", schemaName))

			By(fmt.Sprintf("Cleanup isolation %s", isolationID))
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("do not returns error when trying to create isolation that already exists (MRDR support)", func() {

			By(fmt.Sprintf("Creating isolation %s", isolationID))
			uri := fmt.Sprintf("%s/v1/isolations", baseOpsURI)
			var jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, "10GB")

			resp, body, err := HttpCall("POST", uri, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(fmt.Sprintf(`{"id":"%s"}`, isolationID)))
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			By(fmt.Sprintf("Creating isolation that already exists: %s", isolationID))
			jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, "111GB")

			resp, body, err = HttpCall("POST", uri, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(fmt.Sprintf(`{"code":"200","message":"isolation '%s' already exists","method":"POST","uri":"/v1/isolations"}`, isolationID)))

			//Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("can get isolation", func() {
			CreateIsolation(baseOpsURI, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			resp, body, err := GetIsolation(baseOpsURI, isolationID)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var isoResp GetIsolationResponse
			err = json.Unmarshal(body, &isoResp)
			Expect(err).To(BeNil())
			Expect(isoResp.ID).To(Equal(isolationID))
			Expect(isoResp.MaxStorageSize).To(Equal("10GB"))

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("can update isolation", func() {
			CreateIsolation(baseOpsURI, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			UpdateIsolation(baseOpsURI, isolationID, "20GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "20GB")

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("can delete isolation", func() {
			CreateIsolation(baseOpsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			By(fmt.Sprintf("Deleting isolation %s", isolationID))
			uri := fmt.Sprintf("%s/v1/isolations/%s", baseOpsURI, isolationID)
			resp, _, err := HttpCall("DELETE", uri, nil, "{}")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			ExpectIsolationDoesNotExistInDB(ctx, database, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)

			ExpectTableDoesNotExistInDB(ctx, database, db.GetTableCollections(isolationID))
			ExpectTableDoesNotExistInDB(ctx, database, db.GetTableSmartAttrGroup(isolationID))
		})

		It("returns 404 for updating non-existing isolation", func() {
			uri := fmt.Sprintf("%s/v1/isolations/%s", baseOpsURI, isolationID)
			jsonData := fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, "20GB")

			resp, body, err := HttpCall("PUT", uri, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("404"))
			Expect(response["message"]).To(Equal(fmt.Sprintf("isolation '%s' does not exist", isolationID)))
			Expect(response["uri"]).To(Equal(fmt.Sprintf("/v1/isolations/%s", isolationID)))
			Expect(response["method"]).To(Equal("PUT"))
			ExpectIsolationDoesNotExistInDB(ctx, database, isolationID)
		})

		It("returns 200 without creating isolation using PostIsolationRO (MRDR support)", func() {
			uri := fmt.Sprintf("%s/v1/isolationsRO", baseOpsURI)
			jsonData := fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, "10GB")

			resp, body, err := HttpCall("POST", uri, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["id"]).To(Equal(isolationID))
			Expect(response["code"]).To(Equal("200"))
			Expect(response["message"]).To(Equal("Read-only mode, no action taken"))
			Expect(response["uri"]).To(Equal("/v1/isolationsRO"))

			// Verify no changes in the database
			ExpectIsolationDoesNotExistInDB(ctx, database, isolationID)
		})

		It("cannot delete isolation using DeleteIsolationRO (MRDR support)", func() {
			CreateIsolation(baseOpsURI, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			uri := fmt.Sprintf("%s/v1/isolationsRO/%s", baseOpsURI, isolationID)
			resp, body, err := HttpCall("DELETE", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["id"]).To(Equal(isolationID))
			Expect(response["code"]).To(Equal("200"))
			Expect(response["message"]).To(Equal("Read-only mode, no action taken"))
			Expect(response["uri"]).To(Equal(fmt.Sprintf("/v1/isolationsRO/%s", isolationID)))

			// Verify no changes in the database
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("returns 200 without updating isolation using PutIsolationRO (MRDR support)", func() {
			CreateIsolation(baseOpsURI, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			uri := fmt.Sprintf("%s/v1/isolationsRO/%s", baseOpsURI, isolationID)
			jsonData := fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, "20GB")

			resp, body, err := HttpCall("PUT", uri, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(ContainSubstring(fmt.Sprintf(`{"id":"%s"}`, isolationID)))

			// Verify no changes in the database
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURI, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})
	})
})

type GetIsolationResponse struct {
	ID             string    `json:"id"`
	MaxStorageSize string    `json:"maxStorageSize"`
	CreatedAt      time.Time `json:"createdAt"`
	ModifiedAt     time.Time `json:"modifiedAt"`
}
