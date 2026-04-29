/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var inference = testCoverseModelInference()

func testCoverseModelInference() ConverseModelInference {
	ctx := cntx.ServiceContext("bedrock-client-test")
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	inference := ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Region: "us-east-1",
		},
		RawInput:   []byte("{}}"),
		SaxToken:   "DUMMYTOKEN",
		GinContext: ginCtx,
	}
	return inference
}

type mockError struct {
	NewAwsConfig              error
	NewStsClient              error
	AssumeRoleWithWebIdentity error
	Do                        error
}

func setupMockWith(e mockError) *MockAwsProvider {
	var awsMock MockAwsProvider
	awsMock.On("NewAwsConfig", mock.AnythingOfType("string")).Return(aws.Config{}, e.NewAwsConfig)
	awsMock.On("NewStsClient", mock.AnythingOfType("aws.Config")).Return(&awsMock, e.NewStsClient)
	awsMock.On("NewBedrockEndpointHttpClient").Return(&awsMock)
	awsMock.On("GetAwsClient", mock.AnythingOfType("string")).Return(&awsMock)

	dummyValue := helpers.RandStringRunes(10)
	expirationTime := time.Now().Add(time.Millisecond * 10)
	c := types.Credentials{
		AccessKeyId:     &dummyValue,
		SecretAccessKey: &dummyValue,
		SessionToken:    &dummyValue,
		Expiration:      &expirationTime,
	}
	awsMock.On("AssumeRoleWithWebIdentity",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(&sts.AssumeRoleWithWebIdentityOutput{Credentials: &c}, e.AssumeRoleWithWebIdentity)

	h := http.Header{}
	h.Add("Content-Type", "application/json")
	awsMock.On("Do",
		mock.Anything,
	).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		Header:     h,
	}, e.Do)

	return &awsMock
}

func TestDoConverse(t *testing.T) {
	tests := []struct {
		name        string
		mockErr     mockError
		wantErr     assert.ErrorAssertionFunc
		wantCredErr bool
	}{
		{
			name:    "HappyPath",
			wantErr: assert.NoError,
		},
		{
			name:    "FailOnNewAwsConfig",
			mockErr: mockError{NewAwsConfig: errors.New("fail")},
			wantErr: assert.Error,
		},
		{
			name:        "FailOnAssumeRoleWithWebIdentity_WrapsAsCredentialsAcquisitionErr",
			mockErr:     mockError{AssumeRoleWithWebIdentity: errors.New("invalid oidc token")},
			wantErr:     assert.Error,
			wantCredErr: true,
		},
		{
			name:    "FailOnDo",
			mockErr: mockError{Do: errors.New("fail")},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := getCredentialsCache()
			defer cache.clear()

			err := DoConverse(&inference, setupMockWith(tt.mockErr))
			tt.wantErr(t, err, fmt.Sprintf("DoConverse(%v, %v)", &inference, tt.mockErr))
			if tt.wantCredErr {
				var credErr *ErrCredentialsAcquisition
				assert.True(t, errors.As(err, &credErr), "expected error to be wrapped as ErrCredentialsAcquisition")
			}
		})
	}
}

func setupMockWithStatusCode(statusCode int) *MockAwsProvider {
	var awsMock MockAwsProvider
	awsMock.On("NewAwsConfig", mock.AnythingOfType("string")).Return(aws.Config{}, nil)
	awsMock.On("NewStsClient", mock.AnythingOfType("aws.Config")).Return(&awsMock, nil)
	awsMock.On("NewBedrockEndpointHttpClient").Return(&awsMock)
	awsMock.On("GetAwsClient", mock.AnythingOfType("string")).Return(&awsMock)

	dummyValue := helpers.RandStringRunes(10)
	expirationTime := time.Now().Add(time.Minute)
	c := types.Credentials{
		AccessKeyId:     &dummyValue,
		SecretAccessKey: &dummyValue,
		SessionToken:    &dummyValue,
		Expiration:      &expirationTime,
	}
	awsMock.On("AssumeRoleWithWebIdentity",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(&sts.AssumeRoleWithWebIdentityOutput{Credentials: &c}, nil)

	h := http.Header{}
	h.Add("Content-Type", "application/json")
	h.Add("x-amzn-requestid", "test-request-id-123")
	awsMock.On("Do",
		mock.Anything,
	).Return(&http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"validation error"}`))),
		Header:     h,
	}, nil)

	return &awsMock
}

func TestDoConverse_BedrockErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "4xxResponse", statusCode: http.StatusBadRequest},
		{name: "5xxResponse", statusCode: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := getCredentialsCache()
			defer cache.clear()

			err := DoConverse(&inference, setupMockWithStatusCode(tt.statusCode))
			assert.NoError(t, err, "DoConverse should not return an error for %d responses (response is proxied to client)", tt.statusCode)
		})
	}
}

// errReader is a reader that returns partial data followed by an error.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	return n, r.err
}

func TestLogBedrockErrorResponse(t *testing.T) {
	tests := []struct {
		name          string
		requestID     string
		statusCode    int
		body          io.Reader
		wantBodyInLog string
	}{
		{
			name:          "SuccessfulRead_LogsStatusAndRequestId",
			requestID:     "test-amzn-request-id-abc123",
			statusCode:    http.StatusBadRequest,
			body:          bytes.NewReader([]byte(`{"message":"ValidationException"}`)),
			wantBodyInLog: `{"message":"ValidationException"}`,
		},
		{
			name:          "PartialReadError_LogsRequestIdAndPartialBody",
			requestID:     "test-amzn-request-id-partial",
			statusCode:    http.StatusInternalServerError,
			body:          &errReader{data: []byte(`{"partial":`), err: errors.New("read error")},
			wantBodyInLog: `{"partial":`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			ctx := cntx.ContextWithLogger(cntx.ServiceContext("bedrock-client-test"), zap.New(core))

			h := http.Header{}
			h.Set("x-amz-request-id", tt.requestID)
			h.Set("x-amz-id-2", "amz-id-2-token")
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     h,
				Body:       io.NopCloser(tt.body),
			}

			callStartTime := time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC)
			callResponseTime := callStartTime.Add(150 * time.Millisecond)
			logBedrockErrorResponse(ctx, resp, callStartTime, callResponseTime)

			entries := logs.All()
			assert.Len(t, entries, 1, "expected exactly one log entry")
			msg := entries[0].Message
			assert.Contains(t, msg, "Bedrock upstream error")
			assert.Contains(t, msg, fmt.Sprintf("x_amz_request_id=%s", tt.requestID), "log message should contain x-amz-request-id")
			assert.Contains(t, msg, "x_amz_id_2=amz-id-2-token", "log message should contain x-amz-id-2")
			assert.Contains(t, msg, tt.wantBodyInLog, "log message should contain body")
			assert.Contains(t, msg, "call_start_time=2026-04-08T10:00:00Z")
			assert.Contains(t, msg, "call_response_time=2026-04-08T10:00:00.15Z")
		})
	}
}

func TestErrCredentialsAcquisition(t *testing.T) {
	cause := errors.New("original error")
	tests := []struct {
		name       string
		err        *ErrCredentialsAcquisition
		wantErrMsg string
		wantUnwrap error
	}{
		{
			name:       "NilCause_ReturnsDefaultMessage",
			err:        &ErrCredentialsAcquisition{},
			wantErrMsg: "failed to acquire model credentials",
			wantUnwrap: nil,
		},
		{
			name:       "WithCause_ReturnsCauseMessageAndUnwraps",
			err:        &ErrCredentialsAcquisition{Cause: cause},
			wantErrMsg: cause.Error(),
			wantUnwrap: cause,
		},
		{
			name:       "NilReceiver_Unwrap_ReturnsNil",
			err:        nil,
			wantUnwrap: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErrMsg != "" {
				assert.Equal(t, tt.wantErrMsg, tt.err.Error())
			}
			assert.Equal(t, tt.wantUnwrap, tt.err.Unwrap())
		})
	}
}

type MockAwsProvider struct {
	mock.Mock
}

func (m *MockAwsProvider) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m *MockAwsProvider) GetAwsClient(url string) HttpClientInterface {
	args := m.Called(url)
	return args.Get(0).(HttpClientInterface)
}

func (m *MockAwsProvider) NewBedrockEndpointHttpClient() HttpClientInterface {
	return m
}

func (m *MockAwsProvider) NewStsClient(cfg aws.Config) StsClientInterface {
	args := m.Called(cfg)
	return args.Get(0).(*MockAwsProvider)
}

func (m *MockAwsProvider) NewAwsConfig(region string) (aws.Config, error) {
	args := m.Called(region)
	return args.Get(0).(aws.Config), args.Error(1)
}

func (m *MockAwsProvider) AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*sts.AssumeRoleWithWebIdentityOutput), args.Error(1)
}

func TestAwsProxy_Constructors(t *testing.T) {

	a := NewAwsProxy()
	assert.IsType(t, &AwsProxy{}, a)

	config, _ := a.NewAwsConfig("us-east-1")
	assert.IsType(t, aws.Config{}, config)

	stsClient := a.NewStsClient(config)
	assert.IsType(t, &sts.Client{}, stsClient)

	httpClient := a.NewBedrockEndpointHttpClient()
	assert.IsType(t, &http.Client{}, httpClient)
}
