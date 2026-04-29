package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInsertAndRead(t *testing.T) {
	type args struct {
		isolationGuid string
		ts            int64
	}
	type expectations struct {
		isolation string
		count     int
	}
	oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()
	moreThan24HoursAgo := time.Now().Add(-25 * time.Hour).Unix()
	tests := []struct {
		name string
		args []args
		exp  []expectations
	}{
		{
			"read empty store",
			[]args{},
			[]expectations{
				{"1", 0}},
		},
		{
			"insert first record",
			[]args{
				{"1", oneHourAgo},
			},
			[]expectations{
				{"1", 1},
				{"2", 0},
			},
		},
		{
			"insert 2 records same isolation",
			[]args{
				{"1", oneHourAgo},
				{"1", oneHourAgo},
			},
			[]expectations{
				{"1", 2},
			},
		},
		{
			"insert 2 records 2 isolation",
			[]args{
				{"1", oneHourAgo},
				{"2", oneHourAgo},
			},
			[]expectations{
				{"1", 1},
				{"2", 1},
			},
		},
		{
			"do not count yesterday's record",
			[]args{
				{"1", oneHourAgo},
				{"1", moreThan24HoursAgo},
			},
			[]expectations{
				{"1", 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Reset()
			for _, a := range tt.args {
				Insert(Event{a.isolationGuid, a.ts})
			}
			for _, e := range tt.exp {
				r := Read(e.isolation, time.Now().Add(-24*time.Hour).Unix(), time.Now().Unix())
				assert.Equal(t, e.count, len(r))
			}
		})
	}
}
