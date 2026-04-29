/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

import (
	"fmt"
	"strings"
)

// Endpoint represents the API endpoint type
type Endpoint string

const (
	EndpointChatCompletions          Endpoint = "chat/completions"
	EndpointEmbeddings               Endpoint = "embeddings"
	EndpointImagesGenerations        Endpoint = "images/generations"
	EndpointGenerateImages           Endpoint = "generateImages"
	EndpointConverse                 Endpoint = "converse"
	EndpointConverseStream           Endpoint = "converse-stream"
	EndpointInvoke                   Endpoint = "invoke"
	EndpointPredict                  Endpoint = "predict"
	EndpointInvokeStream             Endpoint = "invoke-stream"
	EndpointInvokeWithResponseStream Endpoint = "invoke-with-response-stream"
	EndpointGenerateContent          Endpoint = "generateContent"
	EndpointStreamGenerateContent    Endpoint = "streamGenerateContent"
	EndpointResponses                Endpoint = "v1/responses"
	EndpointRealtimeClientSecrets    Endpoint = "v1/realtime/client_secrets"
	EndpointRealtimeCalls            Endpoint = "v1/realtime/calls"
)

// NormalizeEndpoint converts raw endpoint strings to standardized Endpoint
func NormalizeEndpoint(rawEndpoint string) (Endpoint, error) {
	// Remove any leading and trailing slashes and normalize
	endpoint := strings.Trim(rawEndpoint, "/")

	switch endpoint {
	case "chat/completions":
		return EndpointChatCompletions, nil
	case "embeddings":
		return EndpointEmbeddings, nil
	case "images/generations":
		return EndpointImagesGenerations, nil
	case "generateImages":
		return EndpointGenerateImages, nil
	case "converse":
		return EndpointConverse, nil
	case "converse-stream":
		return EndpointConverseStream, nil
	case "invoke":
		return EndpointInvoke, nil
	case "predict", ":predict":
		return EndpointPredict, nil
	case "invoke-stream":
		return EndpointInvokeStream, nil
	case "invoke-with-response-stream":
		return EndpointInvokeWithResponseStream, nil
	case "generateContent":
		return EndpointGenerateContent, nil
	case "streamGenerateContent", ":streamGenerateContent":
		return EndpointStreamGenerateContent, nil
	case "v1/responses":
		return EndpointResponses, nil
	case "v1/realtime/client_secrets":
		return EndpointRealtimeClientSecrets, nil
	case "v1/realtime/calls":
		return EndpointRealtimeCalls, nil
	default:
		return "", fmt.Errorf("unknown endpoint: %s", rawEndpoint)
	}
}
