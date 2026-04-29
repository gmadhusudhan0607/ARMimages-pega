/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
)

// SwaggerSpec represents the OpenAPI specification structure
type SwaggerSpec struct {
	Paths map[string]PathItem `yaml:"paths"`
}

type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
	Patch  *Operation `yaml:"patch,omitempty"`
}

type Operation struct {
	Responses map[string]Response `yaml:"responses"`
}

type Response struct {
	Headers map[string]HeaderDef `yaml:"headers,omitempty"`
}

type HeaderDef struct {
	Ref string `yaml:"$ref,omitempty"`
}

// HeaderExpectation represents what headers we expect for an endpoint
type HeaderExpectation struct {
	Endpoint string
	Method   string
	Headers  []string
}

// SwaggerHeadersCache caches the parsed swagger spec
var swaggerHeadersCache *SwaggerSpec

// GetSwaggerSpec fetches and parses the OpenAPI spec from the service
func GetSwaggerSpec(swaggerSpecUrl string) (*SwaggerSpec, error) {
	if swaggerHeadersCache != nil {
		return swaggerHeadersCache, nil
	}

	resp, err := http.Get(swaggerSpecUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch swagger spec: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read swagger spec: %w", err)
	}

	var spec SwaggerSpec
	err = yaml.Unmarshal(body, &spec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse swagger spec: %w", err)
	}

	swaggerHeadersCache = &spec
	return &spec, nil
}

// GetExpectedResponseHeadersForEndpoint returns only the list of expected header names as strings for a given endpoint, method, and response code
func GetExpectedResponseHeadersForEndpoint(serviceBaseURI, path, method string, responseCode int) ([]string, error) {
	spec, err := GetSwaggerSpec(serviceBaseURI)
	if err != nil {
		return nil, err
	}

	normalizedPath := NormalizeApiPath(path)
	pathItem, exists := spec.Paths[normalizedPath]
	if !exists {
		return nil, fmt.Errorf("path %s not found in swagger spec", normalizedPath)
	}

	var operation *Operation
	switch strings.ToUpper(method) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "DELETE":
		operation = pathItem.Delete
	case "PATCH":
		operation = pathItem.Patch
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	if operation == nil {
		return nil, fmt.Errorf("method %s not found for path %s", method, normalizedPath)
	}

	respCodeStr := fmt.Sprintf("%d", responseCode)
	response, exists := operation.Responses[respCodeStr]
	if !exists {
		var discoveredCodes []string
		for code := range operation.Responses {
			discoveredCodes = append(discoveredCodes, code)
		}
		return nil, fmt.Errorf("response code %d not found in API spec for path %s and method %s. Discovered response codes: [%s]", responseCode, normalizedPath, method, strings.Join(discoveredCodes, ", "))
	}

	var headerNames []string
	for headerName := range response.Headers {
		headerName = strings.TrimPrefix(headerName, "#/components/headers/")
		// Validate against Go constants (optional)
		// If you want to enforce only known headers, you can check here
		headerNames = append(headerNames, headerName)
	}

	return headerNames, nil
}

// NormalizeApiPath converts actual paths to swagger path patterns
func NormalizeApiPath(actualPath string) string {
	// Remove query params
	if idx := strings.Index(actualPath, "?"); idx != -1 {
		actualPath = actualPath[:idx]
	}
	parts := strings.Split(actualPath, "/")
	if len(parts) < 2 {
		return actualPath
	}

	var normalized []string
	apiVersion := ""
	for i, part := range parts {
		if part == "" && i == 0 {
			normalized = append(normalized, "")
			continue
		}

		prevPart := ""
		if i > 0 {
			prevPart = parts[i-1]
		}

		if isApiVersion(part) {
			apiVersion = part
			normalized = append(normalized, part)
		} else if apiVersion == "v1" && isIsolationID(part, prevPart) {
			normalized = append(normalized, "{isolationID}")
		} else if apiVersion == "v2" && isIsolationID(part, prevPart) {
			normalized = append(normalized, "{isolationID}")
		} else if part == "collections" {
			normalized = append(normalized, part)
		} else if apiVersion == "v1" && isCollectionName(part, prevPart) {
			normalized = append(normalized, "{collectionName}")
		} else if apiVersion == "v2" && isCollectionID(part, prevPart) {
			normalized = append(normalized, "{collectionID}")
		} else if isCollectionID(part, prevPart) {
			// Handle OPS API pattern which uses {collectionID} even for v1
			normalized = append(normalized, "{collectionID}")
		} else if part == "documents" {
			normalized = append(normalized, part)
		} else if apiVersion == "v1" && prevPart == "documents" && part != "" {
			normalized = append(normalized, "{documentID}")
		} else if apiVersion == "v2" && prevPart == "documents" && part != "" {
			normalized = append(normalized, "{documentID}")
		} else if part == "smart-attributes-group" {
			normalized = append(normalized, part)
		} else if isGroupID(part, prevPart) {
			normalized = append(normalized, "{groupID}")
		} else if isStaticSegment(part) {
			normalized = append(normalized, part)
		} else {
			normalized = append(normalized, part)
		}
	}
	return strings.Join(normalized, "/")
}

func isApiVersion(part string) bool {
	return part == "v1" || part == "v2"
}

func isIsolationID(part, prevPart string) bool {
	// Handle direct isolation IDs after v1/v2 (service API pattern)
	if (prevPart == "v1" || prevPart == "v2") && strings.HasPrefix(part, "iso-") {
		return true
	}
	// Handle isolation IDs after /isolations/ (ops API pattern)
	if prevPart == "isolations" && part != "" {
		return true
	}
	// Handle isolation IDs after /ops/ (ops metrics API pattern)
	if prevPart == "ops" && part != "" {
		return true
	}
	return false
}

func isCollectionName(part, prevPart string) bool {
	// For v1, treat any collection segment after 'collections' as collectionName
	return prevPart == "collections"
}

func isCollectionID(part, prevPart string) bool {
	// For v2, treat any collection segment after 'collections' as collectionID
	return prevPart == "collections"
}

func isGroupID(part, prevPart string) bool {
	return prevPart == "smart-attributes-group" && part != ""
}

func isStaticSegment(part string) bool {
	switch part {
	case "query", "chunks", "attributes", "file", "text", "document", "delete-by-id", "find-documents":
		return true
	default:
		return false
	}
}

// GetHeaderNamesFromRefs converts header $ref values to actual header names
func GetHeaderNamesFromRefs(headerRefs []string) []string {
	var headerNames []string
	for _, ref := range headerRefs {
		if strings.HasPrefix(ref, "#/components/headers/") {
			headerName := strings.TrimPrefix(ref, "#/components/headers/")
			headerNames = append(headerNames, headerName)
		} else {
			headerNames = append(headerNames, ref)
		}
	}
	return headerNames
}

// GetNormalizedSpecEndpointsWithResponseCodes returns all normalized endpoints, methods, and response codes from the Swagger spec
func GetNormalizedSpecEndpointsWithResponseCodes(swaggerSpecURI string) (map[string]map[string]map[int]int, error) {
	spec, err := GetSwaggerSpec(swaggerSpecURI)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]map[int]int)

	for path, pathItem := range spec.Paths {
		normalizedPath := NormalizeApiPath(path)
		if _, ok := result[normalizedPath]; !ok {
			result[normalizedPath] = make(map[string]map[int]int)
		}

		methods := map[string]*Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"DELETE": pathItem.Delete,
			"PATCH":  pathItem.Patch,
		}

		for method, op := range methods {
			if op == nil {
				continue
			}
			if _, ok := result[normalizedPath][method]; !ok {
				result[normalizedPath][method] = make(map[int]int)
			}
			for respCodeStr := range op.Responses {
				var respCode int
				_, err := fmt.Sscanf(respCodeStr, "%d", &respCode)
				if err != nil {
					continue // skip non-integer codes like "default"
				}
				result[normalizedPath][method][respCode] = 1
			}
		}
	}
	return result, nil
}
