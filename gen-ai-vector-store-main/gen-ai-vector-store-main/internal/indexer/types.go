/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

const (
	BatchSize = 250

	ConsistencyLevelStrong   = "strong"
	ConsistencyLevelEventual = "eventual"
)

type embeddingError struct {
	err        error
	statusCode int
}

func (e *embeddingError) Error() string {
	return e.err.Error()
}

func (e *embeddingError) Unwrap() error {
	return e.err
}
