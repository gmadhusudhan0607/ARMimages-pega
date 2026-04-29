/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetBuddy(t *testing.T) {
	mapping := &Mapping{
		Buddies: []Buddy{
			{Name: "buddy1", RedirectURL: "http://localhost:8080"},
			{Name: "buddy2", RedirectURL: "http://localhost:9090"},
			{Name: "buddy-no-url", RedirectURL: ""},
		},
	}

	tests := []struct {
		name          string
		buddyId       string
		expectError   bool
		expectBuddy   *Buddy
		errorContains string
	}{
		{
			name:        "Success: Find existing buddy",
			buddyId:     "buddy1",
			expectError: false,
			expectBuddy: &Buddy{Name: "buddy1", RedirectURL: "http://localhost:8080"},
		},
		{
			name:        "Success: Find buddy with empty URL",
			buddyId:     "buddy-no-url",
			expectError: false,
			expectBuddy: &Buddy{Name: "buddy-no-url", RedirectURL: ""},
		},
		{
			name:          "Error: Unrecognized buddy",
			buddyId:       "unknown-buddy",
			expectError:   true,
			errorContains: "unrecognized buddyId: unknown-buddy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buddy, err := GetBuddy(mapping, tt.buddyId)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, buddy)
				assert.Contains(t, err.Message, tt.errorContains)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, buddy)
				assert.Equal(t, tt.expectBuddy.Name, buddy.Name)
				assert.Equal(t, tt.expectBuddy.RedirectURL, buddy.RedirectURL)
			}
		})
	}
}

func TestHandleBuddyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mapping        *Mapping
		isolationId    string
		buddyId        string
		path           string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "Error: Missing isolationId",
			mapping: &Mapping{
				Buddies: []Buddy{{Name: "test-buddy", RedirectURL: "http://localhost:8080"}},
			},
			isolationId:    "",
			buddyId:        "test-buddy",
			path:           "/v1//buddies/test-buddy/question",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "isolationId and buddyId params are required",
		},
		{
			name: "Error: Unrecognized buddy",
			mapping: &Mapping{
				Buddies: []Buddy{{Name: "test-buddy", RedirectURL: "http://localhost:8080"}},
			},
			isolationId:    "isolation-123",
			buddyId:        "unknown-buddy",
			path:           "/v1/isolation-123/buddies/unknown-buddy/question",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "unrecognized buddyId: unknown-buddy",
		},
		{
			name: "Error: Buddy with empty RedirectURL returns 404",
			mapping: &Mapping{
				Buddies: []Buddy{{Name: "selfstudybuddy", RedirectURL: ""}},
			},
			isolationId:    "isolation-123",
			buddyId:        "selfstudybuddy",
			path:           "/v1/isolation-123/buddies/selfstudybuddy/question",
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "buddy 'selfstudybuddy' is not mapped to any provider URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, engine := gin.CreateTestContext(recorder)

			// Register the route with the handler
			engine.POST("/v1/:isolationId/buddies/:buddyId/*operation", HandleBuddyRequest(context.Background(), tt.mapping))

			// Create request
			req, _ := http.NewRequest(http.MethodPost, tt.path, nil)
			c.Request = req

			// Serve the request
			engine.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tt.expectedMsg)
		})
	}
}

func TestHandleBuddyRequest_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock server to receive the redirected request
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer mockServer.Close()

	mapping := &Mapping{
		Buddies: []Buddy{{Name: "test-buddy", RedirectURL: mockServer.URL}},
	}

	recorder := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(recorder)

	// Register the route with the handler
	engine.POST("/v1/:isolationId/buddies/:buddyId/*operation", HandleBuddyRequest(context.Background(), mapping))

	// Create request
	req, _ := http.NewRequest(http.MethodPost, "/v1/isolation-123/buddies/test-buddy/question", nil)

	// Serve the request
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}
