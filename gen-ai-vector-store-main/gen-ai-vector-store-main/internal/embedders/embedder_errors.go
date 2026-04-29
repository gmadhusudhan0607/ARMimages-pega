/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"errors"
	"fmt"
	"io"
)

func ConstructModelForbiddenError(respBody io.ReadCloser) error {
	message := `Unable to call the Embedding Model. Received HTTP 403 (Access Denied) from LLM/GatewayService.
Verify your permissions and ensure your host/IP is correctly whitelisted to access the LLM Gateway. If the issue persists, contact your administrator`

	if respBody != nil {
		defer respBody.Close()
		body, err := io.ReadAll(respBody)
		if err == nil {
			message = fmt.Sprintf("%s\nModel response: %s", message, string(body))
		}
	}

	return errors.New(message)
}

func ConstructModelNotFoundError(respBody io.ReadCloser) error {
	message := `Unable to call the Embedding Model. Received HTTP 404 (Not Found) from LLM/GatewayService.
Ensure you have correct Gateway URL set and the model is available. If the issue persists, contact your administrator`

	if respBody != nil {
		defer respBody.Close()
		body, err := io.ReadAll(respBody)
		if err == nil {
			message = fmt.Sprintf("%s\nModel response: %s", message, string(body))
		}
	}

	return errors.New(message)
}
