/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package errors

import (
	"errors"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestVersionNotFoundError(t *testing.T) {
	tests := []struct {
		name           string
		err            *VersionNotFoundError
		expectedString string
	}{
		{
			name: "Version not found with available versions",
			err: &VersionNotFoundError{
				Provider:          types.ProviderAzure,
				Creator:           types.CreatorAmazon,
				ModelName:         "gpt-35-turbo",
				RequestedVersion:  "nonexistent",
				AvailableVersions: []string{"1106", "0125"},
			},
			expectedString: "version nonexistent not found for model azure/amazon/gpt-35-turbo, available versions: [1106 0125]",
		},
		{
			name: "Model not found (no available versions)",
			err: &VersionNotFoundError{
				Provider:          types.ProviderAzure,
				Creator:           types.CreatorAmazon,
				ModelName:         "nonexistent-model",
				RequestedVersion:  "v1",
				AvailableVersions: []string{},
			},
			expectedString: "model azure/amazon/nonexistent-model not found",
		},
		{
			name: "Model not found (nil available versions)",
			err: &VersionNotFoundError{
				Provider:          types.ProviderAzure,
				Creator:           types.CreatorAmazon,
				ModelName:         "nonexistent-model",
				RequestedVersion:  "v1",
				AvailableVersions: nil,
			},
			expectedString: "model azure/amazon/nonexistent-model not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expectedString {
				t.Errorf("Expected error string %q, got %q", tt.expectedString, result)
			}
		})
	}
}

func TestModelNotFoundError(t *testing.T) {
	err := &ModelNotFoundError{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorAmazon,
		ModelName: "nonexistent-model",
	}

	expected := "model azure/amazon/nonexistent-model not found"
	result := err.Error()

	if result != expected {
		t.Errorf("Expected error string %q, got %q", expected, result)
	}
}

func TestIsVersionNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "VersionNotFoundError",
			err: &VersionNotFoundError{
				Provider:         types.ProviderAzure,
				Creator:          types.CreatorAmazon,
				ModelName:        "gpt-35-turbo",
				RequestedVersion: "nonexistent",
			},
			expected: true,
		},
		{
			name: "ModelNotFoundError",
			err: &ModelNotFoundError{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorAmazon,
				ModelName: "nonexistent-model",
			},
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVersionNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsModelNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "ModelNotFoundError",
			err: &ModelNotFoundError{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorAmazon,
				ModelName: "nonexistent-model",
			},
			expected: true,
		},
		{
			name: "VersionNotFoundError",
			err: &VersionNotFoundError{
				Provider:         types.ProviderAzure,
				Creator:          types.CreatorAmazon,
				ModelName:        "gpt-35-turbo",
				RequestedVersion: "nonexistent",
			},
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsModelNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestErrorInterfaces(t *testing.T) {
	// Test that our error types implement the error interface
	var err error

	err = &VersionNotFoundError{
		Provider:         types.ProviderAzure,
		Creator:          types.CreatorAmazon,
		ModelName:        "gpt-35-turbo",
		RequestedVersion: "nonexistent",
	}
	if err.Error() == "" {
		t.Error("VersionNotFoundError should implement error interface")
	}

	err = &ModelNotFoundError{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorAmazon,
		ModelName: "nonexistent-model",
	}
	if err.Error() == "" {
		t.Error("ModelNotFoundError should implement error interface")
	}
}

func TestErrorTypeAssertions(t *testing.T) {
	// Test type assertions work correctly
	var err error

	// Test VersionNotFoundError
	err = &VersionNotFoundError{
		Provider:         types.ProviderAzure,
		Creator:          types.CreatorAmazon,
		ModelName:        "gpt-35-turbo",
		RequestedVersion: "nonexistent",
	}

	if versionErr, ok := err.(*VersionNotFoundError); !ok {
		t.Error("Should be able to assert VersionNotFoundError")
	} else {
		if versionErr.Provider != types.ProviderAzure {
			t.Error("Provider field should be accessible after type assertion")
		}
		if versionErr.ModelName != "gpt-35-turbo" {
			t.Error("ModelName field should be accessible after type assertion")
		}
		if versionErr.RequestedVersion != "nonexistent" {
			t.Error("RequestedVersion field should be accessible after type assertion")
		}
	}

	// Test ModelNotFoundError
	err = &ModelNotFoundError{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorAmazon,
		ModelName: "nonexistent-model",
	}

	if modelErr, ok := err.(*ModelNotFoundError); !ok {
		t.Error("Should be able to assert ModelNotFoundError")
	} else {
		if modelErr.Provider != types.ProviderAzure {
			t.Error("Provider field should be accessible after type assertion")
		}
		if modelErr.ModelName != "nonexistent-model" {
			t.Error("ModelName field should be accessible after type assertion")
		}
	}
}

func TestErrorFieldAccess(t *testing.T) {
	// Test that all fields are properly accessible
	versionErr := &VersionNotFoundError{
		Provider:          types.ProviderAzure,
		Creator:           types.CreatorAmazon,
		ModelName:         "gpt-35-turbo",
		RequestedVersion:  "nonexistent",
		AvailableVersions: []string{"1106", "0125"},
	}

	if versionErr.Provider != types.ProviderAzure {
		t.Errorf("Expected provider 'openai', got '%s'", versionErr.Provider)
	}
	if versionErr.ModelName != "gpt-35-turbo" {
		t.Errorf("Expected model name 'gpt-35-turbo', got '%s'", versionErr.ModelName)
	}
	if versionErr.RequestedVersion != "nonexistent" {
		t.Errorf("Expected requested version 'nonexistent', got '%s'", versionErr.RequestedVersion)
	}
	if len(versionErr.AvailableVersions) != 2 {
		t.Errorf("Expected 2 available versions, got %d", len(versionErr.AvailableVersions))
	}

	modelErr := &ModelNotFoundError{
		Provider:  types.ProviderGoogle,
		Creator:   types.CreatorGoogle,
		ModelName: "nonexistent-model",
	}

	if modelErr.Provider != types.ProviderGoogle {
		t.Errorf("Expected provider 'google', got '%s'", modelErr.Provider)
	}
	if modelErr.ModelName != "nonexistent-model" {
		t.Errorf("Expected model name 'nonexistent-model', got '%s'", modelErr.ModelName)
	}
}
