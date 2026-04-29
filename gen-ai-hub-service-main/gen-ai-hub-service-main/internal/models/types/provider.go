/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// Provider represents the AI provider type
type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderBedrock   Provider = "bedrock"
	ProviderAnthropic Provider = "anthropic"
	ProviderMeta      Provider = "meta"
	ProviderAmazon    Provider = "amazon"
	ProviderVertex    Provider = "vertex"
	ProviderAzure     Provider = "azure"
)
