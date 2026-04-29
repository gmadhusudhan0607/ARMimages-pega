/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package helpers

import (
	"testing"
)

func TestCutOffFloatingPointPrecision(t *testing.T) {
	type args struct {
		value           float64
		numbersAfterDot int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		// TODO: Add test cases.
		{
			name: "Test 0",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 0,
			},
			want: 1,
		},
		{
			name: "Test 1",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 1,
			},
			want: 1.2,
		},
		{
			name: "Test 2",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 2,
			},
			want: 1.23,
		},
		{
			name: "Test 3",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 3,
			},
			want: 1.234,
		},
		{
			name: "Test 4",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 4,
			},
			want: 1.2345,
		},
		{
			name: "Test 5",
			args: args{
				value:           1.23456789,
				numbersAfterDot: 5,
			},
			want: 1.23456,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CutOffFloatingPointPrecision(tt.args.value, tt.args.numbersAfterDot); got != tt.want {
				t.Errorf("CutOffFloatingPointPrecision() = %v, want %v", got, tt.want)
			}
		})
	}
}
