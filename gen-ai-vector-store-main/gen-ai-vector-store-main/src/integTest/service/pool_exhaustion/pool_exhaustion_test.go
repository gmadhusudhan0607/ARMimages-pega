/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package pool_exhaustion_test

// This test file reproduces the ck-290 production incident (2026-04-15) where
// Pega application pods received java.net.SocketException: Socket closed errors
// when calling the GenAI Vector Store's PUT /documents endpoint.
//
// Root cause: when the upstream embedding service (genai-hub-service) is slow,
// background goroutines hold open DB transactions from the ingestion pool for the
// full duration of the embedding calls.  With a finite ingestion pool
// (MAX_CONNS_INGESTION=3 in this suite), just three concurrent slow documents are
// enough to exhaust every available connection.  New PUT /documents requests then
// block in the HTTP handler waiting for a free connection; after HTTP_REQUEST_TIMEOUT
// (5 s in this suite) they receive a 504 Gateway Timeout instead of the expected
// 202 Accepted.
//
// The test below would fail against the pre-fix implementation and now serves as
// a regression test and acceptance criterion for the ck-290 fix.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("PUT /v1/{isolationID}/collections/{collectionName}/documents - DB Pool Exhaustion", Ordered, func() {

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
		CreateCollection(svcBaseURI, isolationID, collectionID)
	})

	AfterEach(func() {
		if !CurrentSpecReport().Failed() {
			SafeCleanupIsolation(ctx, database, opsBaseURI, isolationID)
		}

		for _, mockID := range mockIDs {
			err := DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
		}
		mockIDs = nil
	})

	Context("when the embedding service is slow", func() {

		// This test demonstrates the cascade that caused the ck-290 incident:
		//
		//  1. N concurrent PUT /documents (eventual consistency) saturate the 3-slot
		//     ingestion pool: each background goroutine acquires a DB connection and
		//     holds it open while waiting for the slow embedder to respond.
		//  2. A new PUT /documents arrives and blocks in UpsertDocStatus trying to
		//     obtain a DB connection from the now-empty pool.
		//  3. After HTTP_REQUEST_TIMEOUT (5 s), the request timeout middleware fires
		//     and returns 504 Gateway Timeout — the same symptom Pega observed as a
		//     socket closed error after its own 30-second socket timeout.
		//
		// Expected behaviour (PASS after a fix): the probe request should return
		// 202 Accepted within 2 seconds regardless of how many background goroutines
		// are in flight, because eventual-consistency uploads should never be blocked
		// by background processing.
		It("should accept new documents immediately even while background goroutines hold all DB connections", func() {
			endpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			By("Setting up a slow embedding mock (8 s delay > HTTP_REQUEST_TIMEOUT of 5 s)")
			// The delay (8000 ms) is intentionally longer than HTTP_REQUEST_TIMEOUT (5 s)
			// so that the background goroutines keep their DB connections open past the
			// point where the probe request would time out.
			// QUERY_EMBEDDING_TIMEOUT_MS is 60 s (suite config), so the embedding calls
			// do NOT time out during the 8 s WireMock delay.
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 8000)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By(fmt.Sprintf("Firing %d concurrent documents to saturate all %d ingestion pool connections",
				3, 3))
			// One document per pool slot (MAX_CONNS_INGESTION=3).
			// Each document has 2 chunks; the embedding calls run in parallel inside
			// the background goroutine, so the goroutine holds its connection for the
			// full 8 s WireMock delay.
			var wg sync.WaitGroup
			for i := 1; i <= 3; i++ {
				wg.Add(1)
				go func(idx int) {
					defer GinkgoRecover()
					defer wg.Done()

					docData := fmt.Sprintf(
						`{"id":"SATURATE-%d","chunks":[{"content":"saturate chunk one %d"},{"content":"saturate chunk two %d"}]}`,
						idx, idx, idx,
					)
					resp, _, callErr := HttpCallWithHeadersAndApiCallStat(
						"PUT", endpointURI, ServerConfigurationHeaders, docData,
					)
					Expect(callErr).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusAccepted),
						fmt.Sprintf("saturating document %d should be accepted immediately", idx))
				}(i)
			}
			wg.Wait()

			// Brief pause to let all three background goroutines start and call
			// BeginTx, acquiring their DB connections before the probe arrives.
			// Background goroutines are spawned immediately after Phase 1 completes
			// (< 1 ms), so 200 ms is a comfortable margin.
			time.Sleep(200 * time.Millisecond)

			By("Sending a probe request while the pool is exhausted")
			probeData := ReadTestDataFile("pool-exhaustion/DOC.json")
			start := time.Now()
			probeResp, _, probeErr := HttpCallWithHeadersAndApiCallStat(
				"PUT", endpointURI, ServerConfigurationHeaders, probeData,
			)
			elapsed := time.Since(start)

			By(fmt.Sprintf("Probe responded in %v with status %d", elapsed, probeResp.StatusCode))

			// --- Assertions that should PASS after a fix, and FAIL today ---
			//
			// Eventual-consistency uploads must be fast-path operations: Phase 1 is
			// a lightweight DB write that completes in milliseconds.  The background
			// goroutines' DB usage must NOT block the synchronous Phase 1 path.
			//
			// Currently FAILS because all three ingestion pool connections are held
			// by background goroutines and Phase 1 (UpsertDocStatus) cannot acquire
			// a connection — it blocks until HTTP_REQUEST_TIMEOUT (5 s) fires and
			// returns 504 Gateway Timeout.
			Expect(probeErr).To(BeNil(), "probe HTTP call itself must not error")
			Expect(probeResp.StatusCode).To(Equal(http.StatusAccepted),
				fmt.Sprintf(
					"probe returned %d after %v; expected 202 Accepted — "+
						"this indicates the ingestion DB pool was fully held by "+
						"background goroutines doing embedding calls, starving new requests",
					probeResp.StatusCode, elapsed,
				),
			)
			Expect(elapsed).To(BeNumerically("<", 2*time.Second),
				fmt.Sprintf(
					"probe took %v; eventual-consistency uploads should complete in "+
						"under 2 s regardless of background processing load",
					elapsed,
				),
			)
		})
	})
})
