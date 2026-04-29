/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"fmt"
	"time"

	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func DeleteMockServerExpectation(expectationID string) {
	GinkgoHelper()
	By(fmt.Sprintf("-> Deleting mockserver expectation %s", expectationID))
	uri := fmt.Sprintf("%s/mockserver/clear", genaiUrl)
	jsonData := fmt.Sprintf("{ \"id\": \"%s\" }", expectationID)

	resp, _, err := HttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func DeleteIsolation(baseURI, isolationID string) {
	GinkgoHelper()
	By(fmt.Sprintf("Deleting isolation %s", isolationID))
	uri := fmt.Sprintf("%s/v1/isolations/%s", baseURI, isolationID)

	resp, _, err := HttpCall("DELETE", uri, nil, "{}")
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

// SafeCleanupIsolation performs safe cleanup of isolation data
// It waits for queue to be empty before deleting isolation
func SafeCleanupIsolation(ctx context.Context, db *pgxpool.Pool,
	opsBaseURI, isolationID string) {
	GinkgoHelper()
	By(fmt.Sprintf("Performing safe cleanup for isolation %s", isolationID))

	// Wait for queue to be empty (10s to allow for reschedule delay + processing time)
	WaitForQueueEmpty(ctx, db, isolationID, "10s")

	// Stabilization delay to ensure all background operations complete
	time.Sleep(500 * time.Millisecond)

	// Remove from queue (if any stragglers)
	RemovedIsolationFromEmbeddingQueue(ctx, db, isolationID)

	// Delete isolation
	DeleteIsolation(opsBaseURI, isolationID)

	By(fmt.Sprintf("Isolation %s cleaned up successfully", isolationID))
}
