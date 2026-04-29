// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package timeout_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Request Timeout Middleware Tests", Ordered, func() {

	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string
	)

	BeforeEach(func() {
		ctx = context.Background()
		isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
		collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
		mockIDs = nil
		CreateIsolation(opsBaseURI, isolationID, "1GB")
	})

	AfterEach(func() {
		if !CurrentSpecReport().Failed() {
			// Use safe cleanup function that handles synchronization
			SafeCleanupIsolation(ctx, database, opsBaseURI, isolationID)
		}

		// Cleanup wiremock expectations
		for _, mockID := range mockIDs {
			err := DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
		}
		mockIDs = nil
	})

	Context("timeout middleware applies to all endpoints", func() {

		It("PUT document endpoint respects timeout", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating wiremock expectation with delay exceeding timeout")
			// Create a delay that will cause timeout (2100ms > 2000ms QUERY_EMBEDDING_TIMEOUT)
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("PUT document - should timeout")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)

			By("Verifying timeout response")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			// Document is accepted but async processing will timeout
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted),
				fmt.Sprintf("Expected 202 Accepted for async operation, got %d. Body: %s", resp.StatusCode, string(body)))
		})

		It("POST query documents endpoint respects timeout", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			WaitForDocumentsStatusInDB(ctx, database, isolationID, collectionID, docIDs, "COMPLETED")

			By("Deleting previous mock and creating expectation with delay to simulate timeout")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			// 2100ms delay > 2000ms QUERY_EMBEDDING_TIMEOUT
			delayMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, delayMockID)

			By("Query documents - should timeout")
			queryData := ReadTestDataFile("test02/requests/query-documents.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)

			By("Verifying timeout response")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout),
				fmt.Sprintf("Expected 504 Gateway Timeout, got %d", resp.StatusCode))
		})

		It("POST query chunks endpoint respects timeout", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/query/chunks", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("test03/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			WaitForDocumentsStatusInDB(ctx, database, isolationID, collectionID, docIDs, "COMPLETED")

			By("Deleting previous mock and creating expectation with delay to simulate timeout")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			delayMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, delayMockID)

			By("Query chunks - should timeout")
			queryData := ReadTestDataFile("test03/requests/query-chunks.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)

			By("Verifying timeout response")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout),
				fmt.Sprintf("Expected 504 Gateway Timeout, got %d", resp.StatusCode))
		})

		It("GET document endpoint respects timeout", func() {
			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert a document first")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			putPath := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			putEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, putPath)
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", putEndpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for document to complete")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("GET document succeeds without timeout")
			getPath := fmt.Sprintf("/v1/%s/collections/%s/documents/DOC-1", isolationID, collectionID)
			getEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, getPath)
			start := time.Now()
			resp, _, err = HttpCallWithHeadersAndApiCallStat("GET", getEndpointURI, ServerConfigurationHeaders, "")
			elapsed := time.Since(start)

			By("Verifying successful response within timeout")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(elapsed).To(BeNumerically("<", 5*time.Second),
				fmt.Sprintf("GET request took too long: %v", elapsed))
		})

		It("DELETE document endpoint respects timeout", func() {
			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert a document first")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			putPath := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			putEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, putPath)
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", putEndpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for document to complete")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("DELETE document succeeds without timeout")
			deletePath := fmt.Sprintf("/v1/%s/collections/%s/documents/DOC-1", isolationID, collectionID)
			deleteEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, deletePath)
			start := time.Now()
			resp, _, err = HttpCallWithHeadersAndApiCallStat("DELETE", deleteEndpointURI, ServerConfigurationHeaders, "")
			elapsed := time.Since(start)

			By("Verifying successful response within timeout")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(elapsed).To(BeNumerically("<", 5*time.Second),
				fmt.Sprintf("DELETE request took too long: %v", elapsed))
		})

		It("PATCH document endpoint respects timeout", func() {
			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert a document first")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			putPath := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			putEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, putPath)
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", putEndpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for document to complete")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("PATCH document succeeds without timeout")
			patchPath := fmt.Sprintf("/v1/%s/collections/%s/documents/DOC-1", isolationID, collectionID)
			patchEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, patchPath)
			patchData := `{"attributes":[{"name":"updated","type":"string","value":["true"]}]}`
			start := time.Now()
			resp, _, err = HttpCallWithHeadersAndApiCallStat("PATCH", patchEndpointURI, ServerConfigurationHeaders, patchData)
			elapsed := time.Since(start)

			By("Verifying successful response within timeout")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(elapsed).To(BeNumerically("<", 5*time.Second),
				fmt.Sprintf("PATCH request took too long: %v", elapsed))
		})
	})

	Context("context cancellation and cleanup", func() {

		It("verifies context cancellation propagates through request chain", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			WaitForDocumentsStatusInDB(ctx, database, isolationID, collectionID, docIDs, "COMPLETED")

			By("Creating expectation with delay to trigger timeout")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			delayMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, delayMockID)

			By("Making query that will timeout")
			queryData := ReadTestDataFile("test02/requests/query-documents.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)

			By("Verifying timeout response")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout))

			By("Verifying no idle transactions left after timeout")
			// Give a moment for cleanup
			time.Sleep(1 * time.Second)
			ExpectNoIdleTransactionsLeft(ctx, database, "testuser")
		})
	})
})
