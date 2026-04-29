/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package helpers

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy"
)

type helperTools struct {
	FileReader           fileReaderFunc
	HttpCaller           httpCaller
	GetEnvOrDefault      func(key, fallback string) string
	RandStringRunes      func(n int) string
	GetEnvOrFalse        func(key string) bool
	GetEnvOrPanic        func(name string) string
	FileExists           func(string) (bool, error)
	CreateServiceContext func(name string) context.Context
	SelectValue          func(inputs ...string) string
	GetEnabledProviders  func(ctx context.Context) []string
}

type httpCaller func(address, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error)

var HelperSuite = &helperTools{
	FileReader:           os.ReadFile,
	HttpCaller:           proxy.AtomicUnauthenticatedCall,
	RandStringRunes:      RandStringRunes,
	GetEnvOrDefault:      GetEnvOrDefault,
	GetEnvOrFalse:        GetEnvOrFalse,
	GetEnvOrPanic:        GetEnvOrPanic,
	FileExists:           fileExists,
	CreateServiceContext: nil,
	SelectValue:          selectValue,
	GetEnabledProviders:  GetEnabledProviders,
}

func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, err
	}
	return err == nil, err
}

func (h *helperTools) Reset() {
	h.FileReader = os.ReadFile
	h.HttpCaller = proxy.AtomicUnauthenticatedCall
	h.RandStringRunes = RandStringRunes
	h.GetEnvOrDefault = GetEnvOrDefault
	h.GetEnvOrFalse = GetEnvOrFalse
	h.GetEnvOrPanic = GetEnvOrPanic
}

type fileReaderFunc func(string) ([]byte, error)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterRunes))))
		if err != nil {
			panic(fmt.Sprintf("crypto/rand failed: %v", err))
		}
		b[i] = letterRunes[idx.Int64()]
	}
	return string(b)
}

func GetEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GetEnvOrFalse(key string) bool {
	return GetEnvOrDefault(key, "false") == "true"
}

func GetEnvOrPanic(name string) string {
	val, present := os.LookupEnv(name)
	if !present {
		panic(fmt.Sprintf("Env variable '%s' is required", name))
	}
	return val
}

func ExtractTokenValue(bearer string) (string, error) {
	r := regexp.MustCompile(`^Bearer (\S*)$`)
	if match := r.FindStringSubmatch(bearer); len(match) == 2 {
		return match[1], nil
	}
	return "", fmt.Errorf("please provide bearer token")
}

func selectValue(inputs ...string) string {
	var zero string
	for _, input := range inputs {
		if input != zero {
			return input
		}
	}
	return zero
}

func GetEnabledProviders(ctx context.Context) []string {
	enabledProviders := GetEnvOrDefault("ENABLED_PROVIDERS", "Azure,Vertex,Bedrock")

	providers := strings.Split(enabledProviders, ",")
	// Trim whitespace from each provider
	for i, provider := range providers {
		providers[i] = strings.TrimSpace(provider)
	}
	return providers
}
