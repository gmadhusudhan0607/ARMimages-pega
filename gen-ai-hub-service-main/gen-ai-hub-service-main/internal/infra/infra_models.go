/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

type ModelConfig struct {
	ModelMapping                string `json:"ModelMapping"`
	ModelId                     string `json:"ModelId"`
	ModelArn                    string `json:"ModelArn"`
	OIDCRole                    string `json:"OIDCIAMRoleArn"`
	Region                      string `json:"Region"`
	Endpoint                    string `json:"Endpoint"`
	Path                        string `json:"Path"`
	TargetApi                   string `json:"TargetApi"`
	Inactive                    bool   `json:"Inactive"`
	UseRegionalInferenceProfile bool   `json:"UseRegionalInferenceProfile"`
}

type ConfigLoader func(context.Context) ([]ModelConfig, error)

func LoadInfraModelsForContext(ctx context.Context) ([]ModelConfig, error) {

	l := cntx.LoggerFromContext(ctx).Sugar()

	infraModelsDir := cntx.GetInfraModelsDir(ctx)
	subdirs, err := readSubdirectories(infraModelsDir)
	if err != nil {
		fmt.Printf("error reading subdirectories: %v\n", err)
		return nil, err
	}

	infraModels := make([]ModelConfig, 0)
	for _, modelsDir := range subdirs {
		//list all files in the secret mount directory
		files, err := os.ReadDir(modelsDir)
		if err != nil {
			return nil, fmt.Errorf("error when reading genai infra directory %s: %w", ctx, err)
		}

		for _, f := range files {
			if !strings.Contains(f.Name(), "genai") {
				continue
			}

			content, err := os.ReadFile(modelsDir + "/" + f.Name())
			if err != nil {
				l.Error(fmt.Errorf("error when reading file %s: %w", f.Name(), err))
				continue
			}

			model := ModelConfig{}
			err = json.Unmarshal(content, &model)
			if err != nil {
				l.Error(fmt.Errorf("error when unmarshaling json file %s: %w", f.Name(), err))
				continue
			}
			infraModels = append(infraModels, model)
		}
	}

	return infraModels, nil
}

// FindBestMatch look into a GenAI Infra slice and return the *best*
// configuration that matches the mapping asked. It decides which mapping is
// best based on following criteria:
// 1. mapping supports uses Inference Profile and supports targetApi literally and
// 2. mapping supports Inference profile but targetApi is undefined
// 3. mapping supports the targetApi literally
// 4. one of other Active configurations for the requested model
func FindBestMatch(ctx context.Context, configs []ModelConfig, mappingName string, targetApi string) (bool, *ModelConfig) {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	candidates := []*ModelConfig{}
	for _, c := range configs {
		if c.ModelMapping == mappingName && (c.TargetApi == targetApi || c.TargetApi == "") {
			// Found a model config that matches the mapping name and/or target API
			candidates = append(candidates, &c)
		}
	}

	if len(candidates) == 0 {
		return false, nil
	}

	var best *ModelConfig

	for _, m := range candidates {
		if m.TargetApi == targetApi && m.UseRegionalInferenceProfile {
			logger.Debugf("FindBestMatch: Selected model config - exact match with regional inference profile (ModelMapping=%s, TargetApi=%s, UseRegionalInferenceProfile=true)",
				m.ModelMapping, m.TargetApi)
			return true, m
		}
		if m.UseRegionalInferenceProfile {
			best = m
		}
		if m.TargetApi == targetApi && best == nil {
			best = m
		}
	}

	if best == nil {
		// If no model config with UseRegionalInferenceProfile is found or match the TargetApi, return the first one
		best = candidates[0]
	}

	// Log the match for debugging purposes
	if best != nil {
		matchType := "fallback to first candidate"
		if best.UseRegionalInferenceProfile && best.TargetApi == targetApi {
			matchType = "regional inference profile with target API match"
		} else if best.UseRegionalInferenceProfile {
			matchType = "regional inference profile match"
		} else if best.TargetApi == targetApi {
			matchType = "target API match"
		}
		logger.Debugf("FindBestMatch: Selected model config - %s (ModelMapping=%s, TargetApi=%s, UseRegionalInferenceProfile=%v)",
			matchType, best.ModelMapping, best.TargetApi, best.UseRegionalInferenceProfile)
	}

	return true, best
}

func readSubdirectories(dirName string) ([]string, error) {
	var subdirs []string

	err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != dirName {
			subdirs = append(subdirs, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return subdirs, nil
}

func GetInfraModelsForContext(ctx context.Context) ([]ModelConfig, error) {

	l := cntx.LoggerFromContext(ctx).Sugar()

	infraModels := make([]ModelConfig, 0)

	url := os.Getenv("MAPPING_ENDPOINT")

	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		e := fmt.Errorf("error when getting models mapping from %s: %w", url, err)
		return nil, e
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body) //ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error when read the response body obtained from the /mappings endpoint exposed by the genai-ops-gateway svc %s: %w", ctx, err)
	}

	err = json.Unmarshal(body, &infraModels)
	l.Debugf("Got %d models from MAPPING_ENDPOINT", len(infraModels))
	if err != nil {
		return nil, fmt.Errorf("error when trying to unmarshall the json response body %s: %w", ctx, err)
	}

	return infraModels, nil
}
