/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gin-gonic/gin"
)

var helperSuite = helpers.HelperSuite

// isIntegrationTest is cached at startup to avoid repeated os.Getenv calls on every request
var isIntegrationTest bool

func init() {
	isIntegrationTest = os.Getenv("IS_INTEGRATION_TEST") == "true"
}

type ConverseModelInference struct {
	Ctx        context.Context
	InfraModel infra.ModelConfig
	SaxToken   string
	RawInput   []byte
	GinContext *gin.Context //add the GinContext so we can stream the response
}

type ConverseSdkInvoke func(*ConverseModelInference, AwsProvider) error

// ErrCredentialsAcquisition is returned when AWS credentials cannot be acquired.
// This typically indicates an authentication or authorization issue (e.g. invalid or expired OIDC token).
type ErrCredentialsAcquisition struct {
	Cause error
}

func (e *ErrCredentialsAcquisition) Error() string {
	if e == nil || e.Cause == nil {
		return "failed to acquire model credentials"
	}
	return e.Cause.Error()
}

func (e *ErrCredentialsAcquisition) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func DoConverse(modelCall *ConverseModelInference, awsProvider AwsProvider) error {
	// do not need aws authentication for integration tests
	// the integration test setup provides a mock endpoint that does not require authentication
	if isIntegrationTest {
		return doConverseForIntegrationTest(modelCall, awsProvider)
	}

	sessionName := helperSuite.RandStringRunes(10)
	ctx := modelCall.Ctx
	l := cntx.LoggerFromContext(ctx).Sugar().With("AwsSession", sessionName)
	defer l.Sync() //nolint:errcheck

	awsCredentials, err := getAwsCredentials(ctx, modelCall.InfraModel, modelCall.SaxToken, sessionName, awsProvider)
	if err != nil {
		credErr := &ErrCredentialsAcquisition{Cause: fmt.Errorf("failed to acquire model credentials: %w", err)}
		l.Error(credErr)
		return credErr
	}
	return handleHttpRequest(ctx, modelCall, awsProvider, awsCredentials)
}

func getAwsCredentials(ctx context.Context, infraModel infra.ModelConfig, saxToken string, sessionName string, awsProvider AwsProvider) (aws.Credentials, error) {

	l := cntx.LoggerFromContext(ctx)

	cache := getCredentialsCache()
	cacheKey := generateCacheKey(saxToken, infraModel)

	if credentials, found := cache.Get(cacheKey); found {
		l.Sugar().Debugf("Credential cache hit valid until %s", credentials.Expires.Format("20060102150405"))
		return credentials, nil
	}
	l.Sugar().Debug("Credential cache miss. Issuing new credential.")

	// cache miss
	assumeRoleInput := sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &infraModel.OIDCRole,
		RoleSessionName:  &sessionName,
		WebIdentityToken: &saxToken,
	}

	cfg, err := awsProvider.NewAwsConfig(infraModel.Region)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	svc := awsProvider.NewStsClient(cfg)
	identity, err := svc.AssumeRoleWithWebIdentity(ctx, &assumeRoleInput)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to assume role with web identity: %w", err)
	}

	credentials := aws.Credentials{
		AccessKeyID:     *identity.Credentials.AccessKeyId,
		SecretAccessKey: *identity.Credentials.SecretAccessKey,
		SessionToken:    *identity.Credentials.SessionToken,
		Expires:         *identity.Credentials.Expiration,
	}

	if ok := cache.Set(cacheKey, credentials); !ok {
		l.Sugar().Warnf("Credential with expiration %s was not cached", credentials.Expires.Format("20060102150405"))
	}
	l.Sugar().Debugf("Credential cached. Expires at %s", credentials.Expires.Format("20060102150405"))
	return credentials, nil
}

func handleHttpRequest(ctx context.Context, modelCall *ConverseModelInference, awsProvider AwsProvider, credentials aws.Credentials) error {
	l := cntx.LoggerFromContext(ctx).Sugar()

	req, err := createSignedRequest(ctx, modelCall, credentials)
	if err != nil {
		return err
	}

	path := modelCall.InfraModel.Path
	l.Infof("Redirecting [%s %s] to [%s]", req.Method, path, req.URL)
	httpTime := time.Now()
	err = executeRequest(modelCall, awsProvider, req)
	if err != nil {
		l.Errorf("Error executing request: %v", err)
		return err
	}
	l.Debugf("Received response in %dms from: %s", time.Since(httpTime), req.URL)
	return nil
}

func createSignedRequest(ctx context.Context, modelCall *ConverseModelInference, creds aws.Credentials) (*http.Request, error) {
	endpoint := modelCall.InfraModel.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", modelCall.InfraModel.Region)
	}
	l := cntx.LoggerFromContext(ctx).Sugar()
	l.Debugf("AWS Bedrock Runtime target endpoint: %s", endpoint)

	path := modelCall.InfraModel.Path
	if path == "" {
		path = fmt.Sprintf("/model/%s/converse", url.PathEscape(modelCall.InfraModel.ModelId))
	}

	l.Debugf("AWS Bedrock Runtime target path: %s", path)
	modelUrl := fmt.Sprintf("%s%s", endpoint, path)

	payloadHash := sha256.Sum256(modelCall.RawInput)
	payloadHashStr := fmt.Sprintf("%x", payloadHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, modelUrl, bytes.NewReader(modelCall.RawInput))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}
	host := parsedUrl.Host
	req.Host = host

	signer := v4.NewSigner()
	if err = signer.SignHTTP(ctx, creds, req, payloadHashStr, "bedrock", modelCall.InfraModel.Region, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to sign HTTP request: %w", err)
	}

	return req, nil
}

func executeRequest(modelCall *ConverseModelInference, awsProvider AwsProvider, req *http.Request) error {
	// Find a way to injet the client
	c := awsProvider.GetAwsClient(req.URL.String())
	// Measure model call duration
	modelCallStart := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	responseReceivedAt := time.Now()
	modelCallDuration := int(time.Since(modelCallStart).Milliseconds())
	defer resp.Body.Close()

	middleware.SetModelCallDuration(modelCall.GinContext, modelCallDuration)

	if resp.StatusCode >= http.StatusBadRequest {
		logBedrockErrorResponse(modelCall.Ctx, resp, modelCallStart, responseReceivedAt)
	}

	modelCall.GinContext.Status(resp.StatusCode)
	for name, values := range resp.Header {
		for _, value := range values {
			modelCall.GinContext.Header(name, value)
		}
	}

	if _, err = io.Copy(modelCall.GinContext.Writer, resp.Body); err != nil {
		return fmt.Errorf("error during copy of response body: %w", err)
	}

	return nil
}

// logBedrockErrorResponse logs the error details when AWS Bedrock returns a 4xx or 5xx response.
// It reads the response body for logging and replaces it so it can still be forwarded to the caller.
func logBedrockErrorResponse(ctx context.Context, resp *http.Response, callStartTime time.Time, callResponseTime time.Time) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	amzRequestID := resp.Header.Get("x-amz-request-id")
	amzID2 := resp.Header.Get("x-amz-id-2")

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	payload := escapePayload(string(bodyBytes))
	if err != nil {
		payload = escapePayload(string(bodyBytes)) + " (truncated: " + err.Error() + ")"
	}

	l.Errorf("Bedrock upstream error status=%d x_amz_request_id=%s x_amz_id_2=%s call_start_time=%s call_response_time=%s payload=%s",
		resp.StatusCode, amzRequestID, amzID2, callStartTime.UTC().Format(time.RFC3339Nano), callResponseTime.UTC().Format(time.RFC3339Nano), payload)
}

func escapePayload(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\\r\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

type AwsProvider interface {
	NewStsClient(cfg aws.Config) StsClientInterface
	NewAwsConfig(region string) (aws.Config, error)
	NewBedrockEndpointHttpClient() HttpClientInterface
	GetAwsClient(url string) HttpClientInterface
}

func NewAwsProxy() *AwsProxy {
	return &AwsProxy{}
}

type AwsProxy struct{}

type StsClientInterface interface {
	AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

type HttpClientInterface interface {
	Do(r *http.Request) (*http.Response, error)
}

func (a *AwsProxy) NewStsClient(cfg aws.Config) StsClientInterface {
	return sts.NewFromConfig(cfg)
}

func (a *AwsProxy) NewAwsConfig(region string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	return cfg, err
}

func (a *AwsProxy) NewBedrockEndpointHttpClient() HttpClientInterface {
	return &http.Client{}
}

func (a *AwsProxy) GetAwsClient(url string) HttpClientInterface {
	return proxy.NewClient(url)
}
