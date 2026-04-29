/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package filters

import "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"

// RequestFilter type representing the input for "POST query/chunks" endpoint
type RequestFilter struct {
	Query      string                       `json:"query" binding:"required"`
	SubFilters []attributes.AttributeFilter `json:"attributes,omitempty"`
}
