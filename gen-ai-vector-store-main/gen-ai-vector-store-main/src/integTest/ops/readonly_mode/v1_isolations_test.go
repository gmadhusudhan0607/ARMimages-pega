//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing OPS /v1/*/isolations in ReadOnly mode", func() {

	ctx := context.TODO()
	var isolationID string

	BeforeEach(func() {
		isolationID = fmt.Sprintf("test-%s", RandStringRunes(20))
	})

	_ = Context("when performing CRUD operations", func() {

		It("should reject isolation creation with 405 status", func() {
			By("Creating isolation request")
			uri := fmt.Sprintf("%s/v1/isolations", baseOpsURL)
			var jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "10GB" }`, isolationID)

			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", uri, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

			// Verify no changes in the database
			ExpectIsolationDoesNotExistInDB(ctx, database, isolationID)
		})

		It("should allow isolation retrieval successfully", func() {
			// First create isolation without readonly mode
			CreateIsolation(baseOpsURL, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			By("Retrieving isolation details")
			uri := fmt.Sprintf("%s/v1/isolations/%s", baseOpsURL, isolationID)

			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var isoResp GetIsolationResponse
			err = json.Unmarshal(body, &isoResp)
			Expect(err).To(BeNil())
			Expect(isoResp.ID).To(Equal(isolationID))
			Expect(isoResp.MaxStorageSize).To(Equal("10GB"))

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURL, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("should reject isolation updates with 405 status", func() {
			// First create isolation without readonly mode
			CreateIsolation(baseOpsURL, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			By("Updating isolation configuration")
			uri := fmt.Sprintf("%s/v1/isolations/%s", baseOpsURL, isolationID)
			var jsonData = fmt.Sprintf(`{ "maxStorageSize": "20GB" }`)

			resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", uri, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

			// Verify no changes in the database - should still be 10GB
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURL, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})

		It("should reject isolation deletion with 405 status", func() {
			// First create isolation without readonly mode
			CreateIsolation(baseOpsURL, isolationID, "10GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			By("Deleting isolation")
			uri := fmt.Sprintf("%s/v1/isolations/%s", baseOpsURL, isolationID)

			resp, body, err := HttpCallWithHeadersAndApiCallStat("DELETE", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

			// Verify isolation still exists in the database
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "10GB")

			// Cleanup
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(baseOpsURL, isolationID)
			ExpectNoTablesForIsolation(ctx, database, isolationID)
		})
	})
})
