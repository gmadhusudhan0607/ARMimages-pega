/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package headers

// Header names as constants for type safety and maintainability
const (
	RequestDurationMs          = "X-Genai-Vectorstore-Request-Duration-Ms"
	DbQueryTimeMs              = "X-Genai-Vectorstore-Db-Query-Time-Ms"
	ModelId                    = "X-Genai-Vectorstore-Model-Id"
	ModelVersion               = "X-Genai-Vectorstore-Model-Version"
	EmbeddingTimeMs            = "X-Genai-Vectorstore-Embedding-Time-Ms"
	EmbeddingCallsCount        = "X-Genai-Vectorstore-Embedding-Calls-Count"
	EmbeddingRetryCount        = "X-Genai-Vectorstore-Embedding-Retry-Count"
	ResponseReturnedItemsCount = "X-Genai-Vectorstore-Response-Returned-Items-Count"
	ProcessingDurationMs       = "X-Genai-Vectorstore-Processing-Duration-Ms"
	OverheadMs                 = "X-Genai-Vectorstore-Overhead-Ms"
	EmbeddingNetOverheadMs     = "X-Genai-Vectorstore-Embedding-Net-Overhead-Ms"
	VectorsCount               = "X-Genai-Vectorstore-Vectors-Count"
	DocumentsCount             = "X-Genai-Vectorstore-Documents-Count"
	GatewayResponseTimeMs      = "X-Genai-Gateway-Response-Time-Ms"
	GatewayInputTokens         = "X-Genai-Gateway-Input-Tokens"
	GatewayModelId             = "X-Genai-Gateway-Model-Id"
	GatewayRegion              = "X-Genai-Gateway-Region"
	GatewayOutputTokens        = "X-Genai-Gateway-Output-Tokens"
	GatewayTokensPerSecond     = "X-Genai-Gateway-Tokens-Per-Second"
	GatewayRetryCount          = "X-Genai-Gateway-Retry-Count"

	// Configuration headers for runtime behavior modification
	ForceFreshDbMetrics = "X-Genai-Vectorstore-Force-Fresh-DB-Metrics"
	ServiceMode         = "X-Genai-Vectorstore-Service-Mode"
	DbSchemaVersion     = "X-Genai-Vectorstore-Db-Schema-Version"
	DbSchemaMigration   = "X-Genai-Vectorstore-Db-Schema-Migration"
	IsolationId         = "X-Genai-Vectorstore-Isolation-Id"
)
