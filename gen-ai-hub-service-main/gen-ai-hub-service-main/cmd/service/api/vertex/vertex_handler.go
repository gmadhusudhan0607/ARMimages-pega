/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package vertex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
)

const (
	MODEL_CONFIG_KEY = "modelConfig"
	MODEL_ID_KEY     = "modelId"
)

func verifyModelInParams(c *gin.Context) (bool, string) {

	modelUrlParams := api.GetModelRequestParams(c)
	if modelUrlParams.ModelName == "" {
		idNotFoundMsg := "modelId param is required"
		c.JSON(http.StatusBadRequest, api.RespErr{
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

	body := api.ReqBodyType{}

	err = json.Unmarshal(reqBody, &body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(body.ModelId), nil
}

func getModelWithModelId(cd *api.Mapping, modelName string, modelId string) (*api.Model, *api.AppError) {
	for _, m := range cd.Models {
		if m.Name == modelName && strings.HasSuffix(m.ModelId, modelId) {
			return &m, nil
		}
	}
	errMsg := fmt.Sprintf("unrecognized model with name: %s and modelId: %s", modelName, modelId)
	return nil, &api.AppError{Message: errMsg, Error: fmt.Errorf("%s", errMsg)}
}

func CheckVertexImagenRequest(ctx context.Context) gin.HandlerFunc {

	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		var msg string
		var ok bool
		if ok, msg = verifyModelInParams(c); !ok {
			l.Error(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
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
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		if len(requestModelId) == 0 {
			msg = "missing mandatory field modelId for Imagen model"
			l.Error(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		c.Set(MODEL_ID_KEY, requestModelId)
		l.Debugf("Model ID mentioned in the Request body is: %s", requestModelId)
	}

	return fn

}

func SelectImagenModelMapping(ctx context.Context, mapping *api.Mapping) gin.HandlerFunc {

	fn := func(c *gin.Context) {

		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		//find mapping for the model name from the Request param (/:provider/deployments/:modelId
		modelUrlParams := api.GetModelRequestParams(c)

		var requestModelId string
		v, found := c.Get(MODEL_ID_KEY)
		if found {
			requestModelId = v.(string)
		}

		modelConfig, err := getModelWithModelId(mapping, modelUrlParams.ModelName, requestModelId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		c.Set(MODEL_CONFIG_KEY, modelConfig)
	}

	return fn
}

func CallImagenApi(ctx context.Context) gin.HandlerFunc {
	//
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		var m interface{}
		var exists bool
		if m, exists = c.Get(MODEL_CONFIG_KEY); !exists {
			msg := "infrastructure mapping not found for requested model"
			l.Error(msg)
			c.AbortWithStatusJSON(http.StatusBadRequest, api.RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		modelConfig := m.(*api.Model)

		// Extract the rest of the API Gateway route to be invoked after the model name that is being handled:
		// Example:
		//    /google/deployments/imagen-3/images/generations
		// will create an operation path as
		//    /images/generations
		PrefixPath := fmt.Sprintf("/%s/deployments/%s", modelConfig.Provider, modelConfig.Name)
		operationPath := strings.TrimPrefix(c.Request.URL.RequestURI(), PrefixPath)
		modelUrl := api.GetEntityEndpointUrl(modelConfig.RedirectURL, operationPath)
		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

		api.CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))

	}

	return fn

}
