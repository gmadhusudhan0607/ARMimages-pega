/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/client"
)

const (
	MODEL_CONFIG_KEY = "modelConfig"
	MODEL_ID_KEY     = "modelId"
)

// DoBedrockConverseCall
// Generic method to handle the API requests for Bedrock backed models
func handleBedrockCall(ctx context.Context, c *gin.Context, isConverse bool) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

	var m interface{}
	var exists bool
	if m, exists = c.Get(MODEL_CONFIG_KEY); !exists {
		msg := "infrastructure mapping not found for requested model"
		c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
			StatusCode: http.StatusBadRequest,
			Message:    msg,
		})
		return
	}
	// using the current mappings from cp-settings
	modelConfig := m.(*Model)

	if isConverse {
		var requestModelId string
		if v, found := c.Get(MODEL_ID_KEY); found {
			requestModelId = v.(string)
		}

		if modelConfig.ModelId != requestModelId {
			// adjust body modelId to reflect the value present in model configuration
			// this is needed to comply with models that uses Inference Profile in AWS Bedrock
			if err := writePayloadField(c, "modelId", modelConfig.ModelId); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, RespErr{
					StatusCode: http.StatusInternalServerError,
					Message:    err.Error(),
				})
				return
			}
		}
	}
	// Extract the rest of the API Gateway route to be invoked after the model name that is being handled:
	// Example:
	//    /anthropic/deployments/claude-3-haiku/chat/completions
	// will create an operation path as
	//    /chat/completions
	PrefixPath := fmt.Sprintf("/%s/deployments/%s", modelConfig.Provider, modelConfig.Name)
	operationPath := strings.TrimPrefix(c.Request.URL.RequestURI(), PrefixPath)

	modelUrl := GetEntityEndpointUrl(modelConfig.RedirectURL, operationPath)
	l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

	CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))
	l.Infof("Received response from: %s", modelUrl)
}

func DoBedrockConverseCall(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleBedrockCall(ctx, c, true)
	}
}

// Titan text embed text model using invoke api, modelId is not needed
func DoBedrockRedirectCall(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleBedrockCall(ctx, c, false)
	}
}

func writePayloadField(c *gin.Context, field string, value interface{}) error {
	body, _ := io.ReadAll(c.Request.Body)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	payload[field] = value
	newBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(newBody))
	return nil
}

func ValidateBedrockConverseRequest(ctx context.Context) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		var msg string
		var ok bool
		if ok, msg = verifyModelInParams(c); !ok {
			l.Error(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		// Get the Model ID passed in the request Body
		requestModelId, err := modelIdPresentInRequestbody(c)
		if err != nil {
			msg := fmt.Sprintf("failed to parse request body to JSON with error %s", err.Error())
			l.Error(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		if len(requestModelId) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "missing mandatory field modelId for AWS Bedrock model",
			})
			return
		}

		c.Set(MODEL_ID_KEY, requestModelId)
		l.Debugf("Model ID mentioned in the Request body is: %s", requestModelId)
		c.Next()
	}
	return fn
}

func SelectModelMapping(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		// try to find mapping for the model name from the Request param (/:provider/deployments/:modelId
		modelUrlParams := GetModelRequestParams(c)

		var requestModelId string
		v, found := c.Get(MODEL_ID_KEY)
		if found {
			requestModelId = v.(string)
		}

		modelConfig, err := getModelWithModelId(mapping, modelUrlParams.ModelName, requestModelId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		c.Set(MODEL_CONFIG_KEY, modelConfig)
	}

	return fn
}

func verifyModelInParams(c *gin.Context) (bool, string) {

	modelUrlParams := GetModelRequestParams(c)
	if modelUrlParams.ModelName == "" {
		idNotFoundMsg := "modelId param is required"
		c.JSON(http.StatusBadRequest, RespErr{
			StatusCode: http.StatusBadRequest,
			Message:    idNotFoundMsg,
		})
		return false, idNotFoundMsg
	}

	return true, ""
}

func modelIdPresentInRequestbody(c *gin.Context) (string, error) {

	var reqBody []uint8

	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	defer c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	body := ReqBodyType{}

	err = json.Unmarshal(reqBody, &body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(body.ModelId), nil
}

func HandleBedrockModelCall(ctx context.Context, configLoader infra.ConfigLoader, invoker client.ConverseSdkInvoke) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			l.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		jsonMap := make(map[string]interface{})
		errParse := json.Unmarshal(body, &jsonMap)
		if errParse != nil {
			l.Error(errParse)
			c.AbortWithStatusJSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid JSON payload",
			})
			return
		}

		// route: find mapping that matches modelId
		modelsAvailable, err := configLoader(ctx)
		if err != nil {
			err = fmt.Errorf("internal error loading GenAI Infrastructure configuration: %w", err)
			l.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			})
			return
		}

		if len(modelsAvailable) == 0 {
			l.Warn("No AWS Bedrock models configured for this GenAI Gateway Service deployment")
		}

		targetModelId, targetApi := calculateTargetModelAndApiFromRequest(c)
		l.Debugf("Target modelId: %s, targetApi: %s", targetModelId, targetApi)

		var infraModelConfig *infra.ModelConfig
		var found bool
		if found, infraModelConfig = infra.FindBestMatch(ctx, modelsAvailable, targetModelId, targetApi); !found {
			msg := fmt.Sprintf("model %s is with API %s not available in this GenAI Gateway Service deployment. Contact CloudOps.", targetModelId, targetApi)
			c.AbortWithStatusJSON(http.StatusNotFound, RespErr{
				StatusCode: http.StatusNotFound,
				Message:    msg,
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		// Do not enforce if Token is present at this point. Just collect and pass it along.
		saxToken, _ := helpers.ExtractTokenValue(authHeader)
		modelCall := client.ConverseModelInference{
			Ctx:        ctx,
			InfraModel: *infraModelConfig,
			SaxToken:   saxToken,
			RawInput:   body,
			GinContext: c,
		}

		awsProvider := client.NewAwsProxy()
		if err = invoker(&modelCall, awsProvider); err != nil {
			l.Error(err)
			var credErr *client.ErrCredentialsAcquisition
			if errors.As(err, &credErr) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, RespErr{
					StatusCode: http.StatusUnauthorized,
					Message:    err.Error(),
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			})
			return
		}
	}
	return fn
}

func calculateTargetModelAndApiFromRequest(c *gin.Context) (string, string) {
	targetModelId := c.Param("modelId")
	targetApi := strings.Trim(c.Param("targetApi"), "/")
	if targetApi == "chat/completions" {
		targetApi = "converse" // backward compatibility with calling chat/completions API
	}
	if targetApi == "embeddings" {
		targetApi = "invoke" // backward compatibility with calling embeddings API
	}
	return targetModelId, targetApi
}
