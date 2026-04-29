/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// Creator represents the AI model creator type
type Creator string

const (
	CreatorOpenAI    Creator = "openai"
	CreatorGoogle    Creator = "google"
	CreatorMeta      Creator = "meta"
	CreatorAmazon    Creator = "amazon"
	CreatorAnthropic Creator = "anthropic"
	CreatorBedrock   Creator = "bedrock"
	CreatorVertex    Creator = "vertex"
	CreatorStability Creator = "stability"
	CreatorMistral   Creator = "mistral"
)

// IsValidCreator checks if the given creator is a known/valid creator
func IsValidCreator(creator Creator) bool {
	switch creator {
	case CreatorOpenAI, CreatorGoogle, CreatorMeta, CreatorAmazon, CreatorAnthropic, CreatorBedrock, CreatorVertex, CreatorStability, CreatorMistral:
		return true
	default:
		return false
	}
}
