/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package aws

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

func TestGetSecret(t *testing.T) {
	type args struct {
		ctx context.Context
		log *zap.Logger
		sm  SecretsManagerClientFactory
	}
	tests := []struct {
		name    string
		args    args
		want    *DBSecret
		wantErr bool
	}{
		{
			name: "successfully return secret",
			args: args{
				ctx: context.Background(),
				log: log.GetNamedLogger("test"),
				sm: &MockedSecretsManagerClient{
					Secret: &DBSecret{
						Username: "test",
						Password: "test",
					},
				},
			},
			want: &DBSecret{
				Username: "test",
				Password: "test",
			},
		},
		{
			name: "error while returning secret",
			args: args{
				ctx: context.Background(),
				log: log.GetNamedLogger("test"),
				sm: &MockedSecretsManagerClient{
					Secret: &DBSecret{
						Username: "test",
						Password: "test",
					},
					Error: fmt.Errorf("some error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error while unmarshalling secret",
			args: args{
				ctx: context.Background(),
				log: log.GetNamedLogger("test"),
				sm: &MockedSecretsManagerClient{
					SecretString: "wrong json format",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDBSecret(tt.args.ctx, tt.args.log, tt.args.sm, "some-secret-arn")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDBSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDBSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}
