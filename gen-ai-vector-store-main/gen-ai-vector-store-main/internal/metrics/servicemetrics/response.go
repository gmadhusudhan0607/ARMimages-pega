/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import "sync/atomic"

type Response struct {
	itemsReturned atomic.Int32
}

func (r *Response) SetItemsReturned(items int) {
	r.itemsReturned.Store(int32(items))
}

func (r *Response) ItemsReturned() int {
	return int(r.itemsReturned.Load())
}
