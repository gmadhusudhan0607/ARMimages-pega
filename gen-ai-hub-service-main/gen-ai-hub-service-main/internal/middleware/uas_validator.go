/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/patrickmn/go-cache"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
)

var (
	uasIssuer         = "urn:uas-service"
	cacheKey          = "uas-jwks"
	tokenSkewDuration = 2 * time.Minute
	jwksURL           = os.Getenv("UAS_JWKS_URL")
	gocache           = cache.New(60*time.Minute, 120*time.Minute)
)

func FetchKeySet(url string) (jwk.Set, error) {
	if set, exists := gocache.Get(cacheKey); exists {
		return set.(jwk.Set), nil
	} else {
		if fetch, err := jwk.Fetch(context.Background(), url); err == nil {
			gocache.Set(cacheKey, fetch, 60*time.Minute)
			return fetch, nil
		} else {
			return nil, err
		}
	}
}

func UasValidator(ctx context.Context) gin.HandlerFunc {

	return func(c *gin.Context) {
		value := os.Getenv("PLATFORM_TYPE")
		if len(value) != 0 && value == "launchpad" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "please provide access token"})
				c.Abort()
				return
			}

			// validates if bearer token is provided
			authToken, err := helpers.ExtractTokenValue(authHeader)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			// fetch jwks and cache
			jwks, err := FetchKeySet(jwksURL)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			// verifies signature
			token, err := jwt.ParseString(authToken, jwt.WithKeySet(jwks))
			if err != nil {
				// fmt.Printf(err.Error())
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			// check issuer and expiry
			err = jwt.Validate(token, jwt.WithIssuer(uasIssuer), jwt.WithAcceptableSkew(tokenSkewDuration))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
		}
	}
}
