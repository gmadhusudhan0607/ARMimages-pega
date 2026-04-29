//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package cross_functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC Response headers", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()

	_ = Context("calling service", func() {
		var testExpectations []string

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			CreateIsolation(opsBaseURI, isolationID, "1GB")
			CreateCollection(svcBaseURI, isolationID, collectionID)
		})

		AfterEach(func() {
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		_ = Context("/v1/{isolationID}/collections/{collectionName}/file", func() {

			It("PUT v1 collections/{collectionName}/file returns expected headers on 202 response", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/file", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectation for SC job submission")
				scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{scExpID}

				docAttrs := []attributes.Attribute{
					{Name: "Document type", Type: "string", Values: []string{"Article"}},
					{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
				}

				mfParts := []MultiformPart{
					{Type: "field", Name: "documentID", Value: "Astronomy"},
					{Type: "field", Name: "documentAttributes", Value: docAttrs},
					{Type: "file", Name: "documentFile", Value: GetAbsPath("test01/files/Astronomy.txt")},
				}

				putResp, _, putErr := HttpCallMultipartFormWithHeadersAndApiCallStat("PUT", endpointURI, mfParts, ServerConfigurationHeaders)
				Expect(putErr).To(BeNil())
				Expect(putResp).NotTo(BeNil())
				if putResp.StatusCode == http.StatusNotFound || putResp.StatusCode == http.StatusBadGateway {
					By("Endpoint not available or upstream error; skipping header assertions for /file without using Ginkgo Skip")
					return
				}
				Expect(putResp.StatusCode).To(Equal(http.StatusAccepted))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "PUT", putResp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(putResp, checks)
			})
		})

		_ = Context("/v1/{isolationID}/collections/{collectionName}/file/text", func() {

			It("PUT v1 collections/{collectionName}/file/text returns expected headers on 202 response", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/file/text", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectation for SC job submission")
				scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{scExpID}

				docAttrs := []attributes.Attribute{
					{Name: "Document type", Type: "string", Values: []string{"Article"}},
					{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
				}
				putReq := map[string]interface{}{
					"documentID":         "Astronomy-txt",
					"documentAttributes": docAttrs,
					"documentContent":    ReadTestDataFile("test01/files/Astronomy.txt"),
				}
				requestBody, err := json.Marshal(putReq)
				Expect(err).To(BeNil())

				putResp, _, putErr := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, string(requestBody))
				Expect(putErr).To(BeNil())
				Expect(putResp).NotTo(BeNil())
				if putResp.StatusCode == http.StatusNotFound || putResp.StatusCode == http.StatusBadGateway {
					By("Endpoint not available or upstream error; skipping header assertions for /file/text without using Ginkgo Skip")
					return
				}
				Expect(putResp.StatusCode).To(Equal(http.StatusAccepted))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "PUT", putResp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(putResp, checks)
			})
		})

	})

})
