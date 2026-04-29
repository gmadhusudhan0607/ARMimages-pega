/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package extensions

import "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"

// RequestConfig defines JSON paths for request processing
type RequestConfig struct {
	MaxTokens    string // Single exact path for max_tokens
	SystemPrompt string // Single exact path for system prompt injection
}

// ResponseConfig defines JSON paths for response processing
type ResponseConfig struct {
	UsedTokens   string // Single exact path to extract used tokens
	FinishReason string // Single exact path to extract finish reason
}

// ExtensionConfiguration defines JSON paths for request and response processing
type ExtensionConfiguration struct {
	// Request processing paths
	Request RequestConfig

	// Response processing paths
	Response ResponseConfig
}

// ProcessedResponse contains the result of response processing
type ProcessedResponse struct {
	UsedTokens      *int
	ReasoningTokens *int
	WasTruncated    bool
	FinishReason    string
}

// ProcessingConfig defines configuration for request processing
type ProcessingConfig struct {
	OutputTokensStrategy          config.OutputTokensStrategy // Strategy for handling max_tokens
	OutputTokensBaseValue         *int                        // Value to set when strategy is FIXED
	OutputTokensAdjustmentForced  bool                        // Force adjustment even when max_tokens exists
	OutputTokensAdjustmentStreams bool                        // Enable max_tokens adjustment for streaming requests (default: false)
	CopyrightProtectionEnabled    bool                        // Enable copyright protection (injects DefaultCopyrightMessage)
}
