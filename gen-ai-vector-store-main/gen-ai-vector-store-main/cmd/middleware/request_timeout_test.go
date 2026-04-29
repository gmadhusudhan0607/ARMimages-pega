// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestTimeoutSimple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should complete request within timeout", func(t *testing.T) {
		// Set timeout to 2 seconds
		os.Setenv("HTTP_REQUEST_TIMEOUT", "2s")
		defer os.Unsetenv("HTTP_REQUEST_TIMEOUT")

		router := gin.New()
		router.Use(RequestTimeoutMiddleware())
		router.GET("/test", func(c *gin.Context) {
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("should not leak goroutines", func(t *testing.T) {
		// Get baseline goroutine count
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		// Set timeout to 100ms
		os.Setenv("HTTP_REQUEST_TIMEOUT", "100ms")
		defer os.Unsetenv("HTTP_REQUEST_TIMEOUT")

		router := gin.New()
		router.Use(RequestTimeoutMiddleware())
		router.GET("/test", func(c *gin.Context) {
			// Simulate work that respects context cancellation
			select {
			case <-c.Request.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})

		// Send multiple requests
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}()
		}
		wg.Wait()

		// Wait for goroutines to clean up
		time.Sleep(500 * time.Millisecond)
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		// Check goroutine count
		finalGoroutines := runtime.NumGoroutine()
		goroutineIncrease := finalGoroutines - initialGoroutines

		// Should not have significant goroutine increase
		assert.Less(t, goroutineIncrease, 10,
			"Goroutine leak detected: initial=%d, final=%d, increase=%d",
			initialGoroutines, finalGoroutines, goroutineIncrease)
	})
}

// These tests were removed because they tested deleted middleware functions
// Only keeping the essential tests that work with RequestTimeoutSimple

func BenchmarkRequestTimeoutSimple(b *testing.B) {
	gin.SetMode(gin.TestMode)
	os.Setenv("HTTP_REQUEST_TIMEOUT", "5s")
	defer os.Unsetenv("HTTP_REQUEST_TIMEOUT")

	router := gin.New()
	router.Use(RequestTimeoutMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// TestNoGoroutineLeakUnderLoad simulates the production issue
func TestNoGoroutineLeakUnderLoad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	// Set short timeout
	os.Setenv("HTTP_REQUEST_TIMEOUT", "100ms")
	defer os.Unsetenv("HTTP_REQUEST_TIMEOUT")

	router := gin.New()
	router.Use(RequestTimeoutMiddleware()) // Using the fixed version

	// Simulate database operation that might hang
	router.GET("/test", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Simulate database work with retries
		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				// Context cancelled, stop retrying
				return
			case <-time.After(50 * time.Millisecond):
				// Simulate database operation
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send many concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()

		// Small delay to spread load
		if i%50 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Wait for all requests to complete
	wg.Wait()

	// Let goroutines clean up
	time.Sleep(2 * time.Second)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	// Check final goroutine count
	finalGoroutines := runtime.NumGoroutine()
	goroutineIncrease := finalGoroutines - baselineGoroutines

	t.Logf("Final goroutines: %d (increase: %d)", finalGoroutines, goroutineIncrease)

	// In the buggy version, this would show hundreds of leaked goroutines
	// With the fix, should be minimal
	assert.Less(t, goroutineIncrease, 50,
		"Significant goroutine leak detected under load: baseline=%d, final=%d, increase=%d",
		baselineGoroutines, finalGoroutines, goroutineIncrease)
}
