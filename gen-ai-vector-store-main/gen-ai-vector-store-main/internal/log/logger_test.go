/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package log

import (
	"go.uber.org/zap"
	"testing"
)

func TestGetNamedLogger(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want func() *zap.Logger
	}{
		{
			name: "successfully initialize zap logger",
			args: args{
				name: "genai-vector-store",
			},
			want: func() *zap.Logger {
				log, _ := zap.NewProduction()
				return log.Named("genai-vector-store")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNamedLogger(tt.args.name); got.Name() != tt.want().Name() {
				t.Errorf("GetNamedLogger() = %v, want %v", got.Name(), tt.want().Name())
			}
		})
	}
}

func TestGetNamedLoggerWithParams(t *testing.T) {
	type args struct {
		name   string
		params []zap.Field
	}
	tests := []struct {
		name string
		args args
		want func() *zap.Logger
	}{
		{
			name: "successfully initialize zap logger",
			args: args{
				name: "genai-vector-store",
			},
			want: func() *zap.Logger {
				log, _ := zap.NewProduction()
				return log.Named("genai-vector-store")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNamedLoggerWithParams(tt.args.name, tt.args.params...); got.Name() != tt.want().Name() {

				t.Errorf("GetNamedLoggerWithParams() = %v, want %v", got.Name(), tt.want().Name())
			}
		})
	}
}
