/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var waitTimeout = time.Duration(GetEnvOrDefaultInt("DOCUMENT_STATUS_UPDATE_PERIOD_MS", 30000)) * 2 * time.Millisecond

func WaitForDocumentsStatusInDB(ctx context.Context, db *pgxpool.Pool, isolationID string, collectionID string, docIDs []string, status string) {
	GinkgoHelper()
	for _, docID := range docIDs {
		WaitForDocumentStatusInDB(context.Background(), db, isolationID, collectionID, docID, resources.StatusCompleted)
	}
}

func WaitForDocumentStatusInDB(ctx context.Context, db *pgxpool.Pool, isolationID string, collectionID string, docID string, status string) {
	GinkgoHelper()
	var docStatus, docErr string
	backoff := time.Second
	for stay, timeout := true, time.After(waitTimeout); stay; {
		select {
		case <-timeout:
			lastState := fmt.Sprintf("[status=%s, error=%s]", docStatus, docErr)
			By(fmt.Sprintf("!!! ERROR: timeout reached while waiting for %s/%s/%s status to be %s: Last document state: %s", isolationID, collectionID, docID, status, lastState))
			Expect(true).To(Equal(false))
			stay = false
		default:
			docStatus, docErr = GetDocumentStatusAndErrorFromDB(ctx, db, isolationID, collectionID, docID)
			if docStatus != status {
				time.Sleep(backoff)
				if backoff < time.Duration(10)*time.Second {
					backoff *= 2
				}
			} else {
				stay = false
			}
		}
	}
}

// WaitForAttributesMigration waits for attributes to be migrated
func WaitForAttributesMigration(ctx context.Context, database *pgxpool.Pool, isolationID, collectionID, profileID string, timeout time.Duration) error {
	GinkgoHelper()
	By(fmt.Sprintf("Waiting for attributes migration for %s/%s/%s", isolationID, collectionID, profileID))

	configKey := fmt.Sprintf("attribute_replication_v0.19.0_%s_%s_%s", isolationID, collectionID, profileID)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		query := `SELECT value FROM vector_store.configuration WHERE key = $1`
		var status string
		err := database.QueryRow(ctx, query, configKey).Scan(&status)

		if err == nil && status == "completed" {
			By(fmt.Sprintf("Attributes migration completed for %s/%s/%s", isolationID, collectionID, profileID))
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("migration did not complete within %v", timeout)
}

// GetQueuedDocumentsCount returns the number of documents queued for the given isolation
func GetQueuedDocumentsCount(ctx context.Context, db *pgxpool.Pool, isolationID string) int {
	GinkgoHelper()
	By(fmt.Sprintf("Checking queue count for isolation %s", isolationID))
	query := `SELECT COUNT(*) FROM vector_store.embedding_queue WHERE (content->'iso_id')::jsonb ? $1`
	var count int
	err := db.QueryRow(ctx, query, isolationID).Scan(&count)
	Expect(err).To(BeNil())
	return count
}

// WaitForQueueEmpty waits until the embedding queue is empty for the given isolation
// This ensures all background processing has completed before cleanup
func WaitForQueueEmpty(ctx context.Context, db *pgxpool.Pool, isolationID string, timeout string) {
	GinkgoHelper()
	By(fmt.Sprintf("Waiting for queue to be empty for isolation %s", isolationID))
	Eventually(func() int {
		return GetQueuedDocumentsCount(ctx, db, isolationID)
	}, timeout, "200ms").Should(Equal(0),
		fmt.Sprintf("Queue should be empty for isolation %s before cleanup", isolationID))
}
