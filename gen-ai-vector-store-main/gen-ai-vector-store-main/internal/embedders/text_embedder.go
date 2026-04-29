/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"context"
)

// TextEmbedder defines the interface for text embedding operations
type TextEmbedder interface {
	GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error)
	GetURL() string
}

// Embedder is deprecated: use TextEmbedder instead
// This alias is provided for backward compatibility and will be removed in a future version
type Embedder = TextEmbedder
