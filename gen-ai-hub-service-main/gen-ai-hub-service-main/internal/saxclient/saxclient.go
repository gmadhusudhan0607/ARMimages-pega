/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package saxclient

import (
	"encoding/json"
	"fmt"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func GetJwtValidTo(clientId, scopes, tokenEndpoint string, privateKeyPEMFormat []byte, validTo time.Time) (string, error) {

	helperSuite := helpers.HelperSuite

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEMFormat)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": clientId,
		"sub": clientId,
		"aud": tokenEndpoint,
		"jti": fmt.Sprintf("%d", now.UnixNano()),
		"iat": now.Unix(),
		"exp": validTo.Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign the token with the private key
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	issueTokenPayload := url.Values{}
	issueTokenPayload.Set("grant_type", "client_credentials")
	issueTokenPayload.Set("scope", scopes)
	issueTokenPayload.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	issueTokenPayload.Set("client_assertion", signedToken)

	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	_, resp, err := helperSuite.HttpCaller(tokenEndpoint, "POST", headers, io.NopCloser(strings.NewReader(issueTokenPayload.Encode())))
	if err != nil {
		return "", err
	}

	// Read the response
	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return "", err
	}

	type tokenBody struct {
		AccessToken string `json:"access_token"`
	}

	accessToken := tokenBody{}
	if err = json.Unmarshal(body, &accessToken); err != nil {
		return "", err
	}

	return accessToken.AccessToken, nil
}
