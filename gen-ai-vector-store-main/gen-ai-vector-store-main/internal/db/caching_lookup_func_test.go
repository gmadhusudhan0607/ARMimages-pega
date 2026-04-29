/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package db

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// fakeResolver for testing
var fakeDNSCalls int

func fakeLookupHost(ctx context.Context, host string) ([]string, error) {
	fakeDNSCalls++
	if host == "fail.com" {
		return nil, errors.New("lookup failed")
	}
	return []string{"1.2.3.4"}, nil
}

func TestCachingLookupFunc_CachesResult(t *testing.T) {
	fakeDNSCalls = 0
	lookup := cachingLookupFunc(100*time.Millisecond, fakeLookupHost)
	ctx := context.Background()
	addrs1, err := lookup(ctx, "example.com")
	if err != nil || !reflect.DeepEqual(addrs1, []string{"1.2.3.4"}) {
		t.Fatalf("unexpected result: %v, err: %v", addrs1, err)
	}
	addrs2, err := lookup(ctx, "example.com")
	if err != nil || !reflect.DeepEqual(addrs2, []string{"1.2.3.4"}) {
		t.Fatalf("unexpected result: %v, err: %v", addrs2, err)
	}
	if fakeDNSCalls != 1 {
		t.Errorf("expected 1 DNS call, got %d", fakeDNSCalls)
	}
}

func TestCachingLookupFunc_Expires(t *testing.T) {
	fakeDNSCalls = 0
	lookup := cachingLookupFunc(50*time.Millisecond, fakeLookupHost)
	ctx := context.Background()
	_, err := lookup(ctx, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	_, err = lookup(ctx, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fakeDNSCalls != 2 {
		t.Errorf("expected 2 DNS calls after expiry, got %d", fakeDNSCalls)
	}
}

func TestCachingLookupFunc_Error(t *testing.T) {
	fakeDNSCalls = 0
	lookup := cachingLookupFunc(time.Second, fakeLookupHost)
	ctx := context.Background()
	_, err := lookup(ctx, "fail.com")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
