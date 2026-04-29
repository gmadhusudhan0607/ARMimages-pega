/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package parallelization

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Group represents a parallelization group with limited concurrent workers
type Group struct {
	semaphore chan struct{}
}

// Limited creates a new Group with a maximum of n concurrent workers
func Limited(n int) *Group {
	return &Group{
		semaphore: make(chan struct{}, n),
	}
}

// Subgroup represents a subgroup with its own context and error group
type Subgroup struct {
	ctx       context.Context
	group     *errgroup.Group
	semaphore chan struct{}
}

// WithContext creates a new Subgroup with the given context
func (g *Group) WithContext(ctx context.Context) (*Subgroup, context.Context) {
	group, ctx := errgroup.WithContext(ctx)
	return &Subgroup{
		ctx:       ctx,
		group:     group,
		semaphore: g.semaphore,
	}, ctx
}

// Go runs the function in a goroutine, respecting the global concurrency limit
func (sg *Subgroup) Go(f func(ctx context.Context) error) {
	sg.group.Go(func() error {
		// Acquire semaphore slot with context cancellation support
		select {
		case sg.semaphore <- struct{}{}:
			// Successfully acquired semaphore
			defer func() {
				// Release semaphore slot
				<-sg.semaphore
			}()

			// Execute the function with the subgroup's context
			return f(sg.ctx)
		case <-sg.ctx.Done():
			// Context cancelled while waiting for semaphore
			return sg.ctx.Err()
		}
	})
}

// Wait waits for all goroutines in the subgroup to complete
func (sg *Subgroup) Wait() error {
	return sg.group.Wait()
}
