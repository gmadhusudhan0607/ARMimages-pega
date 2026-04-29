/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package saxtypes

import (
	"encoding/base64"
	"errors"
)

// SaxAuthClientConfig is a helper to hold the data need to generate a SAX Token. It is used when instantiating a
// Http Proxy for calling model deployments that require SAX authentication.
type SaxAuthClientConfig struct {
	ClientId            string `json:"client_id"`
	PrivateKey          string `json:"private_key"`
	Scopes              string `json:"scopes"`
	TokenEndpoint       string `json:"token_endpoint"`
	SecretArn           string
	privateKeyPEMFormat []byte
}

func (s *SaxAuthClientConfig) GetPrivateKeyPEMFormat() ([]byte, error) {
	if s == nil {
		return nil, errors.New("SAX Configuration not found")
	}

	if len(s.privateKeyPEMFormat) > 0 {
		return s.privateKeyPEMFormat, nil
	}
	var err error
	s.privateKeyPEMFormat, err = base64.StdEncoding.DecodeString(s.PrivateKey)
	return s.privateKeyPEMFormat, err
}
