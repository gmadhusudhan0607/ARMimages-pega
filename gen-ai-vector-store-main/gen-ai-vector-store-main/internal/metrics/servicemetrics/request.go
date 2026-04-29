/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"sync"
	"time"
)

type Request struct {
	start, stop time.Time
	m           sync.RWMutex
}

func (r *Request) StartProcessing() {
	r.m.Lock()
	defer r.m.Unlock()
	r.start = time.Now()
}

func (r *Request) StopProcessing() {
	r.m.Lock()
	defer r.m.Unlock()
	r.stop = time.Now()
}

func (r *Request) Duration() time.Duration {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.start.IsZero() || r.stop.IsZero() || r.stop.Before(r.start) {
		return 0
	}

	return r.stop.Sub(r.start)
}
