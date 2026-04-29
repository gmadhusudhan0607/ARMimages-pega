/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package repository

var store []Event

func Insert(m Event) {
	store = append(store, m)
}

func Read(isolation string, from int64, to int64) []Event {
	var r []Event
	for _, m := range store {
		if m.Isolation == isolation && m.Timestamp >= from && m.Timestamp < to {
			r = append(r, m)
		}
	}
	return r
}

// exported function to support unit testing
func Reset() {
	store = []Event{}
}

func Count() int {
	return len(store)
}

type Event struct {
	Isolation string `json:"isolation"`
	Timestamp int64  `json:"timestamp"`
}

func NewEvent(isolation string, timestamp int64) *Event {
	return &Event{
		Isolation: isolation,
		Timestamp: timestamp,
	}
}
