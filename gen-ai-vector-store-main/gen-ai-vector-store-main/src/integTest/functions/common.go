/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Create a new random number generator with a custom seed (e.g., current time)
var rndGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Shared HTTP client with optimized connection pooling for integration tests
var sharedHTTPClient = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50, // Increased from default 2 to handle concurrent requests
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
}

func GetEnvOfDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rndGenerator.Intn(len(letterRunes))]
	}
	return string(b)
}

func ReadTestDataFile(file string) string {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	fullPath := fmt.Sprintf("%s/data/%s", path, file)

	fi, err := os.Stat(fullPath)
	Expect(err).To(BeNil())
	size := fi.Size()
	Expect(size).NotTo(BeZero())

	f, err := os.Open(fullPath)
	Expect(err).To(BeNil())
	defer f.Close()

	data := make([]byte, size)
	_, err = f.Read(data)
	Expect(err).To(BeNil())
	return string(data)
}

func ReadFromTesDatatDir(dir, file string) string {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	fullPath := fmt.Sprintf("%s/data/%s/%s", path, dir, file)

	fi, err := os.Stat(fullPath)
	Expect(err).To(BeNil())
	size := fi.Size()
	Expect(size).NotTo(BeZero())

	f, err := os.Open(fullPath)
	Expect(err).To(BeNil())
	defer f.Close()

	data := make([]byte, size)
	_, err = f.Read(data)
	Expect(err).To(BeNil())
	return string(data)
}

func ReadFile(file string) string {
	GinkgoHelper()
	fi, err := os.Stat(file)
	Expect(err).To(BeNil())
	size := fi.Size()
	Expect(size).NotTo(BeZero())

	f, err := os.Open(file)
	Expect(err).To(BeNil())
	defer f.Close()

	data := make([]byte, size)
	_, err = f.Read(data)
	Expect(err).To(BeNil())
	return string(data)
}

func isPortAccessible(host string, port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}

func TrimAllSpacesDuplicates(s string) string {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func HttpCall(method, uri string, params []string, reqBody string) (response *http.Response, respBody []byte, err error) {
	GinkgoHelper()
	By(fmt.Sprintf("  -> HTTP/%s : %s", method, uri))
	request, reqErr := http.NewRequest(method, uri, bytes.NewBuffer([]byte(reqBody)))
	Expect(reqErr).To(BeNil())
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := (&http.Client{Timeout: time.Second * 15}).Do(request)
	Expect(err).To(BeNil())
	defer resp.Body.Close()
	respBody, _ = io.ReadAll(resp.Body)
	return resp, respBody, err
}

func HttpCallWithHeaders(method, uri string, headers map[string]string, reqBody string) (response *http.Response, respBody []byte, err error) {
	// Use shared HTTP client with optimized connection pooling
	var req *http.Request
	if reqBody != "" {
		req, err = http.NewRequest(method, uri, bytes.NewBuffer([]byte(reqBody)))
	} else {
		req, err = http.NewRequest(method, uri, nil)
	}
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	return resp, body, nil
}

func injectHeader(inJson string, name, value string) string {
	return injectIntoJson(inJson, []string{"httpRequest", "headers"}, name, value)
}

func injectIntoJson(inJson string, path []string, key string, value interface{}) string {
	d := []byte(inJson)
	var root interface{}
	if err := json.Unmarshal(d, &root); err != nil {
		log.Fatal(err)
	}

	// Walk down path to target object.
	v := root
	for i, k := range path {
		m, ok := v.(map[string]interface{})
		if !ok {
			log.Fatalf("map not found at %s", strings.Join(path[:i+1], ", "))
		}
		v, ok = m[k]
		if !ok {
			log.Fatalf("value not found at %s", strings.Join(path[:i+1], ", "))
		}
	}

	// Set value in the target object.
	m, ok := v.(map[string]interface{})
	if !ok {
		log.Fatalf("map not found at %s", strings.Join(path, ", "))
	}
	m[key] = value

	// Marshal back to JSON. Variable d is []byte with the JSON
	d, err := json.Marshal(root)
	if err != nil {
		log.Fatal(err)
	}
	return string(d)
}

func ContainsAttribute(attrs []attributes.Attribute, attr attributes.Attribute) bool {
	for _, r := range attrs {
		if r.Name == attr.Name && r.Type == attr.Type {
			if !r.Compare(attr) {
				return false
			}
		}
	}
	return true
}

type MultiformPart struct {
	Type  string
	Name  string
	Value interface{}
}

func HttpCallMultipartForm(method, uri string, mfParts []MultiformPart) (response *http.Response, respBody []byte, err error) {
	return HttpCallMultipartFormWithHeaders(method, uri, mfParts, nil)
}

func HttpCallMultipartFormWithHeaders(method, uri string, mfParts []MultiformPart, headers map[string]string) (response *http.Response, respBody []byte, err error) {
	GinkgoHelper()

	By(fmt.Sprintf("-> HTTP/%s (multipart form) : %s", method, uri))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, mfPart := range mfParts {
		switch mfPart.Type {
		case "field":
			//--- Create a form field writer for field --------------
			wr, err := writer.CreateFormField(mfPart.Name)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create form field '%s': %w", mfPart.Name, err)
			}
			val, err := json.Marshal(mfPart.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal %s : %w", mfPart.Name, err)
			}
			if _, err = wr.Write(val); err != nil {
				return nil, nil, fmt.Errorf("failed to write %s: %w", mfPart.Name, err)
			}
		case "file":
			//--- Create a form file --------------
			filePath := fmt.Sprintf("%s", mfPart.Value)
			wr, err := writer.CreateFormFile(mfPart.Name, filepath.Base(filePath))
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create form file: %w", err)
			}
			file, err := os.Open(filePath)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
			}
			defer file.Close()
			if _, err = io.Copy(wr, file); err != nil {
				return nil, nil, fmt.Errorf("failed to copy file content: %w", err)
			}
			if err = writer.Close(); err != nil {
				return nil, nil, fmt.Errorf("failed to close writer: %w", err)
			}
		default:
			return nil, nil, fmt.Errorf("not supported form type: %s", mfPart.Type)
		}
	}

	request, err := http.NewRequest("PUT", uri, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// Add custom headers if provided
	for key, value := range headers {
		request.Header.Add(key, value)
	}

	httpClient := &http.Client{Timeout: time.Second * 15}

	resp, err := httpClient.Do(request)
	Expect(err).To(BeNil())
	defer resp.Body.Close()
	respBody, _ = io.ReadAll(resp.Body)
	return resp, respBody, err
}

func AttrsToAttrFilter(docAttrs []attributes.Attribute, operator string) []attributes.AttributeFilter {
	var docFilter []attributes.AttributeFilter
	for _, attr := range docAttrs {
		a := attributes.AttributeFilter{
			Operator: operator, Name: attr.Name, Type: attr.Type, Values: attr.Values}
		docFilter = append(docFilter, a)
	}
	return docFilter
}

func GetEnvOrDefaultInt(key string, fallback int) int {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return i
}

func ReadResponseBody(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", fmt.Errorf("response or response body is nil")
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

// EndpointsCallsStats structure:
// map[endpoint]map[method]map[statusCode]count
var EndpointsCallsStats map[string]map[string]map[int]int
var endpointsCallsStatsMu sync.Mutex

func init() {
	EndpointsCallsStats = make(map[string]map[string]map[int]int)
}

// HttpCallWithHeadersAndApiCallStat wraps HttpCallWithHeaders and tracks API endpoint calls
func HttpCallWithHeadersAndApiCallStat(method, uri string, headers map[string]string, reqBody string) (response *http.Response, respBody []byte, err error) {
	response, respBody, err = HttpCallWithHeaders(method, uri, headers, reqBody)
	if response != nil {
		recordEndpointCallStat(uri, method, response.StatusCode)
	}
	return
}

// HttpCallMultipartFormWithHeadersAndApiCallStat wraps HttpCallMultipartFormWithHeaders and tracks API endpoint calls
func HttpCallMultipartFormWithHeadersAndApiCallStat(method, uri string, mfParts []MultiformPart, headers map[string]string) (response *http.Response, respBody []byte, err error) {
	response, respBody, err = HttpCallMultipartFormWithHeaders(method, uri, mfParts, headers)
	if response != nil {
		recordEndpointCallStat(uri, method, response.StatusCode)
	}
	return
}

// recordEndpointCallStat records a single API call into EndpointsCallsStats with proper synchronization.
func recordEndpointCallStat(uri, method string, statusCode int) {
	path := uri
	if strings.HasPrefix(path, "http") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			path = "/" + parts[3]
		}
	}
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	normalized := NormalizeApiPath(path)

	endpointsCallsStatsMu.Lock()
	if _, ok := EndpointsCallsStats[normalized]; !ok {
		EndpointsCallsStats[normalized] = make(map[string]map[int]int)
	}
	if _, ok := EndpointsCallsStats[normalized][method]; !ok {
		EndpointsCallsStats[normalized][method] = make(map[int]int)
	}
	EndpointsCallsStats[normalized][method][statusCode]++
	endpointsCallsStatsMu.Unlock()
}

// SetupDatabaseConnectionFromString creates a database connection pool from a connection string
func SetupDatabaseConnectionFromString(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	database, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("error while connecting to DB: %w", err)
	}

	return database, nil
}
