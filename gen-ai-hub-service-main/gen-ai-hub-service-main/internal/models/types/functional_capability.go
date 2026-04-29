/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

import "fmt"

// FunctionalCapability represents the functional capability of a model
type FunctionalCapability string

const (
	FunctionalCapabilityChatCompletion FunctionalCapability = "chat_completion"
	FunctionalCapabilityImage          FunctionalCapability = "image"
	FunctionalCapabilityEmbedding      FunctionalCapability = "embedding"
	FunctionalCapabilityAudio          FunctionalCapability = "audio"
	FunctionalCapabilityRealtime       FunctionalCapability = "realtime"
)

// IsValidFunctionalCapability checks if the given capability is valid
func IsValidFunctionalCapability(capability FunctionalCapability) bool {
	switch capability {
	case FunctionalCapabilityChatCompletion, FunctionalCapabilityImage, FunctionalCapabilityEmbedding, FunctionalCapabilityAudio, FunctionalCapabilityRealtime:
		return true
	default:
		return false
	}
}

// ParseFunctionalCapability parses a string into a FunctionalCapability
func ParseFunctionalCapability(s string) (FunctionalCapability, error) {
	capability := FunctionalCapability(s)
	if !IsValidFunctionalCapability(capability) {
		return "", fmt.Errorf("invalid functional capability: %s", s)
	}
	return capability, nil
}
