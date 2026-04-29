/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"fmt"
	"os"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"gopkg.in/yaml.v3"
)

// loadStaticMapping loads the static model and buddy mapping from the CONFIGURATION_FILE
// This file contains configurations for Azure OpenAI, GCP Vertex, AWS Bedrock (legacy), and Buddies
func loadStaticMapping(configFile string) (*api.Mapping, error) {
	if configFile == "" {
		return nil, fmt.Errorf("configuration file path is empty")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", configFile, err)
	}

	var mapping api.Mapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration file %s: %w", configFile, err)
	}

	return &mapping, nil
}

// findModelInMapping searches for a model by name in the mapping
// Returns the model and true if found, otherwise returns nil and false
func findModelInMapping(mapping *api.Mapping, modelName string) (*api.Model, bool) {
	if mapping == nil {
		return nil, false
	}

	for i := range mapping.Models {
		if mapping.Models[i].Name == modelName {
			return &mapping.Models[i], true
		}
	}

	return nil, false
}

// findBuddyInMapping searches for a buddy by name in the mapping
// Returns the buddy and true if found, otherwise returns nil and false
func findBuddyInMapping(mapping *api.Mapping, buddyName string) (*api.Buddy, bool) {
	if mapping == nil {
		return nil, false
	}

	for i := range mapping.Buddies {
		if mapping.Buddies[i].Name == buddyName {
			return &mapping.Buddies[i], true
		}
	}

	return nil, false
}
