/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package smart_chunking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmitJob_Success(t *testing.T) {
	expectedResp := JobSubmittedResponse{
		OperationID:        "test-op-123",
		IsolationID:        "iso-1",
		Status:             "PENDING",
		RequestedTasks:     []string{"extraction", "chunking", "indexing"},
		CallbackRegistered: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/iso-1/jobs" {
			t.Errorf("expected path /v1/iso-1/jobs, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected auth header, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") == "" {
			t.Error("expected Content-Type header")
		}

		// Verify multipart form
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		// Verify options field
		optionsStr := r.FormValue("options")
		if optionsStr == "" {
			t.Error("expected options field in form")
		}
		var opts JobRequestOptions
		if err := json.Unmarshal([]byte(optionsStr), &opts); err != nil {
			t.Fatalf("failed to unmarshal options: %v", err)
		}
		if len(opts.Tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(opts.Tasks))
		}
		if opts.TaskOptions == nil || opts.TaskOptions.Indexing == nil {
			t.Fatal("expected taskOptions.indexing to be set")
		}
		if opts.TaskOptions.Indexing.CollectionName != "my-col" {
			t.Errorf("expected collectionName my-col, got %s", opts.TaskOptions.Indexing.CollectionName)
		}
		if opts.TaskOptions.Indexing.DocumentID != "doc-123" {
			t.Errorf("expected documentID doc-123, got %s", opts.TaskOptions.Indexing.DocumentID)
		}

		// Verify file field
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "doc-123.pdf" {
			t.Errorf("expected filename doc-123.pdf, got %s", header.Filename)
		}
		content, _ := io.ReadAll(file)
		if string(content) != "test file content" {
			t.Errorf("expected file content 'test file content', got %s", string(content))
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer server.Close()

	client := &smartChunkingClient{
		uri:    server.URL,
		client: server.Client(),
	}

	options := JobRequestOptions{
		Tasks: []string{"extraction", "chunking", "indexing"},
		TaskOptions: &JobTaskOptions{
			Indexing: &IndexingOptions{
				CollectionName: "my-col",
				DocumentID:     "doc-123",
			},
		},
	}

	fileReader := bytes.NewReader([]byte("test file content"))
	resp, err := client.SubmitJob(context.Background(), "Bearer test-token", "iso-1", fileReader, "doc-123.pdf", options)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OperationID != "test-op-123" {
		t.Errorf("expected operationID test-op-123, got %s", resp.OperationID)
	}
	if resp.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", resp.Status)
	}
}

func TestSubmitJob_SCReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"VALIDATION_ERROR","message":"file must be provided"}}`))
	}))
	defer server.Close()

	client := &smartChunkingClient{
		uri:    server.URL,
		client: server.Client(),
	}

	options := JobRequestOptions{
		Tasks: []string{"extraction", "chunking"},
	}

	fileReader := bytes.NewReader([]byte("content"))
	resp, err := client.SubmitJob(context.Background(), "", "iso-1", fileReader, "test.pdf", options)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}

	var scErr *ServiceError
	if !errors.As(err, &scErr) {
		t.Fatal("expected error to be *ServiceError")
	}
	if scErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected StatusCode 400, got %d", scErr.StatusCode)
	}
}

func TestSubmitJob_SCReturns5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal failure"}`))
	}))
	defer server.Close()

	client := &smartChunkingClient{
		uri:    server.URL,
		client: server.Client(),
	}

	options := JobRequestOptions{
		Tasks: []string{"extraction"},
	}

	fileReader := bytes.NewReader([]byte("content"))
	resp, err := client.SubmitJob(context.Background(), "", "iso-1", fileReader, "test.pdf", options)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}

	var scErr *ServiceError
	if !errors.As(err, &scErr) {
		t.Fatal("expected error to be *ServiceError")
	}
	if scErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode 500, got %d", scErr.StatusCode)
	}
}

func TestSubmitJob_NetworkError(t *testing.T) {
	client := &smartChunkingClient{
		uri:    "http://localhost:1", // unreachable
		client: http.DefaultClient,
	}

	options := JobRequestOptions{
		Tasks: []string{"extraction"},
	}

	fileReader := bytes.NewReader([]byte("content"))
	resp, err := client.SubmitJob(context.Background(), "", "iso-1", fileReader, "test.pdf", options)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}
}

func TestSubmitJob_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := &smartChunkingClient{
		uri:    server.URL,
		client: server.Client(),
	}

	options := JobRequestOptions{
		Tasks: []string{"extraction"},
	}

	fileReader := bytes.NewReader([]byte("content"))
	resp, err := client.SubmitJob(context.Background(), "", "iso-1", fileReader, "test.pdf", options)

	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}
}
