/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package testutils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
)

type FileSystemMock struct {
	fs map[string][]byte
}

func (f *FileSystemMock) With(v ...string) {
	f.fs = make(map[string][]byte)
	for i := 0; i < len(v)-1; i += 2 {
		key := v[i]
		value := v[i+1]
		f.fs[key] = []byte(value)
	}
}

func (f *FileSystemMock) FileReader() func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		r := f.fs[path]
		return r, nil
	}
}

func (f *FileSystemMock) FileExists() func(path string) (bool, error) {
	return func(path string) (bool, error) {
		if v := f.fs[path]; v != nil {
			return true, nil
		}
		return false, os.ErrNotExist
	}
}

func GeneratePrivateKey() []byte {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("failed to generate RSA key pair: %s\n", err)
		return nil
	}
	// parse as string of PEM encoded PKCS1 key
	pk := x509.MarshalPKCS1PrivateKey(privKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pk,
	}

	var output bytes.Buffer
	_ = pem.Encode(&output, privateKeyBlock)
	return output.Bytes()
}

func NewLocalAuthServer() *httptest.Server {
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if headers are copied
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-token"}`))
	}))

	return as
}

func ToJSONString(v interface{}) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
