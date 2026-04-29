/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ContextChecker provides methods for checking context configuration
type ContextChecker interface {
	IsUseGenAiInfraModels(ctx context.Context) bool
	IsUseGCPVertex(ctx context.Context) bool
	IsUseAzureGenAIURL(ctx context.Context) bool
	AzureGenAIURL(ctx context.Context) string
	IsLLMProviderConfigured(ctx context.Context, provider string) bool
	LoggerFromContext(ctx context.Context) *zap.Logger
	ContextWithGinContext(ctx context.Context, gc *gin.Context) context.Context
}

// DefaultContextChecker implements ContextChecker using real cntx package functions
type DefaultContextChecker struct{}

func (d *DefaultContextChecker) IsUseGenAiInfraModels(ctx context.Context) bool {
	return cntx.IsUseGenAiInfraModels(ctx)
}

func (d *DefaultContextChecker) IsUseGCPVertex(ctx context.Context) bool {
	return cntx.IsUseGCPVertex(ctx)
}

func (d *DefaultContextChecker) IsUseAzureGenAIURL(ctx context.Context) bool {
	return cntx.IsUseAzureGenAIURL(ctx)
}

func (d *DefaultContextChecker) AzureGenAIURL(ctx context.Context) string {
	return cntx.AzureGenAIURL(ctx)
}

func (d *DefaultContextChecker) IsLLMProviderConfigured(ctx context.Context, provider string) bool {
	return cntx.IsLLMProviderConfigured(ctx, provider)
}

func (d *DefaultContextChecker) LoggerFromContext(ctx context.Context) *zap.Logger {
	return cntx.LoggerFromContext(ctx)
}

func (d *DefaultContextChecker) ContextWithGinContext(ctx context.Context, gc *gin.Context) context.Context {
	return cntx.ContextWithGinContext(ctx, gc)
}

// For testing purposes, providing exported variables for mocking in tests
var (
	GetInfraModelsForContext = infra.GetInfraModelsForContext
	getEnvOrPanic            = helpers.GetEnvOrPanic
	retrieveMapping          = RetrieveMappingImpl
	fetchAWSModels           = fetchAWSModelsImpl
	fetchGCPModels           = fetchGCPModelsImpl
	fetchAzureModels         = fetchAzureModelsImpl
	deduplicateModels        = deduplicateModelsImpl
	enrichModels             = enrichModelsImpl
	extractDefaultModels     = extractDefaultModelsImpl
	getDefaults              = infra.GetDefaultModelsForContext
	loggerFromContext        = cntx.LoggerFromContext
)

func (m *ModelUrlParams) String() string {
	return fmt.Sprintf("modelId=%s", m.ModelName)
}

func GetModel(cd *Mapping, modelName string) (*Model, *AppError) {
	for _, m := range cd.Models {
		if m.Name == modelName {
			return &m, nil
		}
	}
	errMsg := fmt.Sprintf("unrecognized model name: %s", modelName)
	return nil, &AppError{Message: errMsg, Error: fmt.Errorf("%s", errMsg)}
}

func getModelWithModelId(cd *Mapping, modelName string, modelId string) (*Model, *AppError) {
	for _, m := range cd.Models {
		if m.Name == modelName && strings.HasSuffix(m.ModelId, modelId) {
			return &m, nil
		}
	}
	errMsg := fmt.Sprintf("unrecognized model with name: %s and modelId: %s", modelName, modelId)
	return nil, &AppError{Message: errMsg, Error: fmt.Errorf("%s", errMsg)}
}

// GetModelsMappingForCurrentIsolation returns a copy of models with `{{ .IsolationId }}` placeholder in modelUrl set
// to the isolationId of the modelUrlParams.IsolationId
func GetModelsMappingForCurrentIsolation(ctx context.Context, mapping *Mapping, modelUrlParams *ModelUrlParams) (*Mapping, *AppError) {
	// copy the models as before we replace the IsolationId placeholder with the actual value
	ret := &Mapping{
		Models: make([]Model, len(mapping.Models)),
	}
	copy(ret.Models, mapping.Models)

	// fill the isolationId in the modelUrl
	tmpl := template.New("modelUrl")
	for i, m := range ret.Models {
		urlTemplate, _ := tmpl.Parse(m.ModelUrl)

		var buf bytes.Buffer

		if err := urlTemplate.Execute(&buf, modelUrlParams); err != nil {
			return ret, &AppError{
				Message: fmt.Sprintf("unable to update modelUrl [%s] with isolation [%s]", m.ModelUrl, modelUrlParams.IsolationId),
				Error:   err}
		}
		m.ModelUrl = buf.String()
		ret.Models[i] = m
	}

	return ret, nil
}

func GetModelRequestParams(c *gin.Context) *ModelUrlParams {
	return &ModelUrlParams{
		ModelName:   c.Param(ModelIdParamName),
		IsolationId: c.Param(IsolationIdParamName),
	}
}

// HandleImageGenerationRequest generic method to handle the API requests
func HandleImageGenerationRequest(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		modelUrlParams := GetModelRequestParams(c)

		if modelUrlParams.ModelName == "" {
			l.Error("model name is empty in the Request path")
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "model name as url param is required",
			})
			return
		}
		// check if the model is recognized
		m, err := GetModel(mapping, modelUrlParams.ModelName)
		if err != nil {
			l.Error(err.Message)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		// construct prefix strings
		PrefixPath := fmt.Sprintf("/openai/deployments/%s", modelUrlParams.ModelName)
		operationPath := ""
		if strings.HasPrefix(c.Request.URL.Path, PrefixPath) {
			// current endpoint
			operationPath = strings.TrimPrefix(c.Request.URL.Path, PrefixPath)

		} else {
			// the request does not fit to pattern
			msg := fmt.Sprintf("Error while parsing the request: Unrecognized request URI %s", c.Request.URL.Path)
			l.Error(msg)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		modelUrl := setApiVersionParam(GetEntityEndpointUrl(m.RedirectURL, operationPath))

		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

		CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))

		l.Infof("Received response from: %s", modelUrl)
	}

	return fn

}

// HandleExperimentalModelChatCompletionRequest handler for Experimental Models (Gemini-1.5-Pro-Preview) requests
// - It does not implement features like manipulation of System prompt for Copyright infringements
func HandleExperimentalModelChatCompletionRequest(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		modelUrlParams := GetModelRequestParams(c)

		if modelUrlParams.ModelName == "" {
			l.Error("modelId is empty")
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "modelId param is required",
			})
			return
		}

		// check if the model is recognized
		m, err := GetModel(mapping, modelUrlParams.ModelName)
		if err != nil {
			l.Error(err.Message)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		PrefixPath := fmt.Sprintf("/google/deployments/%s", modelUrlParams.ModelName)

		// construct prefix strings
		operationPath := ""
		if strings.HasPrefix(c.Request.URL.Path, PrefixPath) {
			// current endpoint
			operationPath = strings.TrimPrefix(c.Request.URL.Path, PrefixPath)

		} else {
			// the request does not fit to pattern
			msg := fmt.Sprintf("Error while parsing the request: Unrecognized request URI %s", c.Request.URL.Path)
			l.Error(msg)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		modelUrl := GetEntityEndpointUrl(m.RedirectURL, operationPath)

		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

		CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))

		l.Infof("Received response from: %s", modelUrl)

	}

	return fn
}

// HandleChatCompletionRequest generic method to handle the API requests
func HandleChatCompletionRequest(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		modelUrlParams := GetModelRequestParams(c)

		l.Debugf("HandleChatCompletionRequest: Extracted modelName=%s from URL", modelUrlParams.ModelName)

		if modelUrlParams.ModelName == "" {
			l.Error("modelId is empty")
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "modelId param is required",
			})
			return
		}
		// check if the model is recognized
		l.Debugf("HandleChatCompletionRequest: Looking up model '%s' in mapping with %d models", modelUrlParams.ModelName, len(mapping.Models))
		m, err := GetModel(mapping, modelUrlParams.ModelName)
		if err != nil {
			l.Errorf("HandleChatCompletionRequest: Model lookup failed for '%s': %s", modelUrlParams.ModelName, err.Message)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}
		l.Debugf("HandleChatCompletionRequest: Found model '%s' with redirectURL=%s", m.Name, m.RedirectURL)

		// construct prefix strings
		PrefixPath := fmt.Sprintf("/openai/deployments/%s", modelUrlParams.ModelName)

		operationPath := ""
		if strings.HasPrefix(c.Request.URL.Path, PrefixPath) {
			// current endpoint
			operationPath = strings.TrimPrefix(c.Request.URL.Path, PrefixPath)

		} else {
			// the request does not fit to pattern
			msg := fmt.Sprintf("Error while parsing the request: Unrecognized request URI %s", c.Request.URL.Path)
			l.Error(msg)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		privateModelExists, privateModel, err := privateModelCheck(modelUrlParams.ModelName, ctx)

		if err != nil {
			l.Errorf("%s, Error: %s", err.Message, err.Error.Error())
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		var modelUrl string

		if privateModelExists {
			l.Debug("The Private Model will be used")
			modelUrl = GetEntityEndpointUrl(privateModel.RedirectURL, operationPath)
			c.Request.Header.Set("api-key", privateModel.ApiKey)
		} else {
			modelUrl = GetEntityEndpointUrl(m.RedirectURL, operationPath)
		}

		modelUrl = setApiVersionParam(modelUrl)

		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

		CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))

		l.Infof("Received response from: %s", modelUrl)
	}

	return fn

}

// HandleEmbeddingsRequest generic method to handle the API requests
func HandleEmbeddingsRequest(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		modelUrlParams := GetModelRequestParams(c)

		if modelUrlParams.ModelName == "" {
			l.Error("modelId is empty")
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "modelId param is required",
			})
			return
		}
		// check if the model is recognized
		m, err := GetModel(mapping, modelUrlParams.ModelName)
		if err != nil {
			l.Error(err.Message)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		// construct prefix
		PrefixPath := fmt.Sprintf("/openai/deployments/%s", modelUrlParams.ModelName)
		operationPath := ""
		if strings.HasPrefix(c.Request.URL.Path, PrefixPath) {
			// current endpoint
			operationPath = strings.TrimPrefix(c.Request.URL.Path, PrefixPath)
		} else {
			// the request does not fit pattern
			msg := fmt.Sprintf("Error while parsing the request: Unrecognized request URI %s", c.Request.URL.Path)
			l.Error(msg)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    msg,
			})
			return
		}

		privateModelExists, privateModel, err := privateModelCheck(modelUrlParams.ModelName, ctx)

		if err != nil {
			l.Errorf("%s, Error: %s", err.Message, err.Error.Error())
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    err.Message,
			})
			return
		}

		var modelUrl string

		if privateModelExists {
			l.Debug("The Private Model will be used")
			modelUrl = GetEntityEndpointUrl(privateModel.RedirectURL, operationPath)
			c.Request.Header.Set("api-key", privateModel.ApiKey)
		} else {
			modelUrl = GetEntityEndpointUrl(m.RedirectURL, operationPath)
		}

		modelUrl = setApiVersionParam(modelUrl)

		l.Infof("Redirecting [%s %s] to [%s]", c.Request.Method, c.Request.RequestURI, modelUrl)

		CallTarget(c, ctx, modelUrl, cntx.IsUseSax(ctx))
		// TODO: Add a response & check status code which should be used
	}

	return fn

}

func privateModelCheck(modelId string, ctx context.Context) (bool, *Model, *AppError) {

	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	// defining some private model pertinent variables
	var privateModel *Model
	var privateModelFilesExists bool
	var privateModelExists bool

	//check for private mode files within the /private-model-config mount
	privateModelFiles, privateModelFilesExists := checkPrivateModelFiles()

	if privateModelFilesExists {

		l.Debugf("Private Model config files are available on the %s mount", PrivateModelFilePath)

		privateModelMapping, err := getPrivateModelMapping(privateModelFiles)

		if err != nil {
			return false, &Model{}, err
		}

		//check if the model is present in the private model mapping
		privateModelExists, privateModel = doesPrivateModelExist(privateModelMapping, modelId, ctx)

		return privateModelExists, privateModel, nil
	} else {
		l.Debugf("Private Model config files are NOT available on the %s mount", PrivateModelFilePath)
		return false, &Model{}, nil
	}
}

func checkPrivateModelFiles() (*[]string, bool) {

	// declare a file-list slice
	var fileList []string

	// read files within a directory
	files, err := os.ReadDir(PrivateModelFilePath)

	if err != nil {
		// when no files are found under /private-model-config
		return nil, false
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), PrivateModelFilePrefix) {
			fileList = append(fileList, file.Name())
		}
	}

	if len(fileList) > 0 {
		// Sort the file list to ensure consistent ordering
		sort.Strings(fileList)
		return &fileList, true
	}

	return nil, false
}

func getPrivateModelMapping(fileList *[]string) (*Mapping, *AppError) {

	// initializing variables
	privateModelMapping := Mapping{}

	for _, file := range *fileList {

		absFilePath := fmt.Sprintf("%s/%s", PrivateModelFilePath, file)

		content, err := os.ReadFile(absFilePath)
		if err != nil {
			errMsg := fmt.Sprintf("error encountered while reading the file %s", absFilePath)
			return nil, &AppError{Message: errMsg, Error: err}
		}

		var modelList []Model

		err = yaml.Unmarshal(content, &modelList)

		if err != nil {
			errMsg := fmt.Sprintf("error encountered while un-marshalling the file %s", absFilePath)
			return nil, &AppError{Message: errMsg, Error: err}
		}

		privateModelMapping.Models = append(privateModelMapping.Models, modelList...)
	}

	return &privateModelMapping, nil
}

func doesPrivateModelExist(privateModelMapping *Mapping, model string, ctx context.Context) (bool, *Model) {
	l := cntx.LoggerFromContext(ctx).Sugar()

	for _, m := range privateModelMapping.Models {

		if m.Name == model {
			if m.Active {
				return true, &m
			} else {
				l.Debugf("The model %s is present in private model mapping, but the 'active' flag is set to false in the configuration file", m.Name)
				return false, &m
			}
		}
	}
	return false, &Model{}
}

// GetModels returns list of available models
func GetModels(ctx context.Context, mapping *Mapping) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck
		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		modelUrlParams := GetModelRequestParams(c)

		modelsWithCurrentIsolation, err := GetModelsMappingForCurrentIsolation(c, mapping, modelUrlParams)
		if err != nil {
			l.Error(err.Error)
			c.JSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Message,
			})
			return
		}

		// handle get all models
		if modelUrlParams.ModelName == "" {
			c.JSON(http.StatusOK, modelsWithCurrentIsolation)
			return
		}

		// handle get model by id
		if m, err := GetModel(modelsWithCurrentIsolation, modelUrlParams.ModelName); err != nil {
			l.Error(err.Error)
			c.JSON(http.StatusNotFound, RespErr{
				StatusCode: http.StatusNotFound,
				Message:    err.Message,
			})
		} else {
			c.JSON(http.StatusOK, m)
		}
	}
	return fn
}

// providerFetchResult holds the result of fetching models from a single provider
type providerFetchResult struct {
	models     []ModelInfo
	err        error
	statusCode int
}

// fetchBedrockModelsIfConfigured fetches models from AWS Bedrock if configured
func fetchBedrockModelsIfConfigured(ctx context.Context, checker ContextChecker) providerFetchResult {
	l := checker.LoggerFromContext(ctx).Sugar()

	if !checker.IsLLMProviderConfigured(ctx, "Bedrock") || !checker.IsUseGenAiInfraModels(ctx) {
		return providerFetchResult{models: []ModelInfo{}}
	}

	models, err := fetchAWSModels(ctx)
	if err != nil {
		msg := fmt.Sprintf("AWS: Failed to fetch models due to: %v", err)
		l.Warn(msg)
		return providerFetchResult{err: fmt.Errorf("%s", msg)}
	}

	return providerFetchResult{models: models}
}

// fetchVertexModelsIfConfigured fetches models from GCP Vertex if configured
func fetchVertexModelsIfConfigured(ctx context.Context, checker ContextChecker) providerFetchResult {
	l := checker.LoggerFromContext(ctx).Sugar()

	if !checker.IsLLMProviderConfigured(ctx, "Vertex") || !checker.IsUseGCPVertex(ctx) {
		return providerFetchResult{models: []ModelInfo{}}
	}

	models, err := fetchGCPModels(ctx)
	if err != nil {
		msg := fmt.Sprintf("GCP: Failed to fetch models due to: %v", err)
		l.Warn(msg)
		return providerFetchResult{err: fmt.Errorf("%s", msg)}
	}

	return providerFetchResult{models: models}
}

// fetchAzureModelsIfConfigured fetches models from Azure OpenAI if configured
func fetchAzureModelsIfConfigured(ctx context.Context, checker ContextChecker) providerFetchResult {
	l := checker.LoggerFromContext(ctx).Sugar()

	if !checker.IsLLMProviderConfigured(ctx, "Azure") || !checker.IsUseAzureGenAIURL(ctx) {
		l.Warn("Azure GenAIURL not set")
		return providerFetchResult{models: []ModelInfo{}}
	}

	azureURL := checker.AzureGenAIURL(ctx)
	l.Debugf("Getting azure models data")

	models, statusCode, err := fetchAzureModels(ctx, azureURL)
	if err != nil {
		msg := fmt.Sprintf("Azure: Failed to fetch models due to: %v", err)
		l.Warn(msg)
		return providerFetchResult{err: fmt.Errorf("%s", msg), statusCode: statusCode}
	}

	return providerFetchResult{models: models, statusCode: statusCode}
}

// collectModelsFromProviders fetches models from all configured providers
// concurrently using errgroup. Each provider fetch runs in its own goroutine
// and the results are merged once all have completed. Individual provider
// failures are recorded as warnings; the remaining providers' models are
// still returned.
// providerMergeOrder defines the deterministic order in which provider results
// are merged into the final model list.
var providerMergeOrder = []string{"Bedrock", "Vertex", "Azure"}

func collectModelsFromProviders(ctx context.Context, checker ContextChecker) ([]ModelInfo, []string) {
	l := checker.LoggerFromContext(ctx).Sugar()

	var (
		mu      sync.Mutex
		results = make(map[string]providerFetchResult, len(providerMergeOrder))
	)

	g, gctx := errgroup.WithContext(ctx)

	// Fetch from AWS Bedrock
	g.Go(func() error {
		r := fetchBedrockModelsIfConfigured(gctx, checker)
		mu.Lock()
		results["Bedrock"] = r
		mu.Unlock()
		return nil // never fail the group; errors are collected as warnings
	})

	// Fetch from GCP Vertex
	g.Go(func() error {
		r := fetchVertexModelsIfConfigured(gctx, checker)
		mu.Lock()
		results["Vertex"] = r
		mu.Unlock()
		return nil
	})

	// Fetch from Azure OpenAI
	g.Go(func() error {
		r := fetchAzureModelsIfConfigured(gctx, checker)
		mu.Lock()
		results["Azure"] = r
		mu.Unlock()
		return nil
	})

	if err := g.Wait(); err != nil {
		// This should never happen — all goroutines return nil.
		l.Errorf("unexpected errgroup error: %v", err)
	}

	// Merge results in deterministic order.
	var (
		allModels []ModelInfo
		warnings  []string
	)
	for _, provider := range providerMergeOrder {
		r := results[provider]
		if r.err != nil {
			warnings = append(warnings, r.err.Error())
		} else {
			allModels = append(allModels, r.models...)
		}
	}

	return allModels, warnings
}

// buildModelsResponse builds and sends the final JSON response
func buildModelsResponse(c *gin.Context, models []ModelInfo, warnings []string) {
	if len(warnings) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"models":   models,
			"warnings": warnings,
		})
	} else {
		c.JSON(http.StatusOK, models)
	}
}

// HandleCachedGetModelsRequest serves the model list from a lazily-populated
// cache. The first request triggers cache population using the caller's
// request context (which carries SAX credentials via the gin context).
func HandleCachedGetModelsRequest(ctx context.Context, checker ContextChecker, cache *ModelListCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := checker.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck
		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		reqCtx := checker.ContextWithGinContext(ctx, c)
		models, warnings := cache.GetModels(reqCtx)
		buildModelsResponse(c, models, warnings)
	}
}

func fetchDefaultModelsConfiguration(ctx context.Context) (*infra.DefaultModelConfig, error) {
	var defaults infra.DefaultModelConfig

	// Apply environment overrides if they exist
	defaults.Smart = os.Getenv("SMART_MODEL_OVERRIDE")
	defaults.Fast = os.Getenv("FAST_MODEL_OVERRIDE")
	defaults.Pro = os.Getenv("PRO_MODEL_OVERRIDE")

	// Get default model mappings and apply any overrides
	if defaults.Smart == "" || defaults.Fast == "" || defaults.Pro == "" {
		infraDefaults, err := getDefaults(ctx)
		if err != nil {
			return &defaults, err
		}
		if defaults.Smart == "" {
			defaults.Smart = infraDefaults.Smart
		}
		if defaults.Fast == "" {
			defaults.Fast = infraDefaults.Fast
		}
		if defaults.Pro == "" {
			defaults.Pro = infraDefaults.Pro
		}
	}

	return &defaults, nil
}

func findMatchingDefaultModels(allModels []ModelInfo, defaults *infra.DefaultModelConfig) []ModelInfo {
	var matched []ModelInfo

	for _, model := range allModels {
		// Match by ModelMappingId
		if model.ModelMappingId == defaults.Fast || model.ModelMappingId == defaults.Smart || model.ModelMappingId == defaults.Pro {
			matched = append(matched, model)
		}
	}

	return matched
}

// logDefaultConfiguration logs the default model configuration
func logDefaultConfiguration(l *zap.SugaredLogger, defaults *infra.DefaultModelConfig) {
	l.Debugf("Default models from configuration: fast=%s, smart=%s, pro=%s",
		defaults.Fast, defaults.Smart, defaults.Pro)
}

// logEnrichedModels logs enriched model details
func logEnrichedModels(l *zap.SugaredLogger, enriched []ModelInfo) {
	l.Debugf("Total models after deduplication and enrichment: %d", len(enriched))

	for i, model := range enriched {
		l.Debugf("Model %d: Name=%s, MappingId=%s, Creator=%s, Provider=%s",
			i, model.ModelName, model.ModelMappingId, model.Creator, model.Provider)
	}
}

// logMatchedModels logs models that match the defaults
func logMatchedModels(l *zap.SugaredLogger, matched []ModelInfo, defaults *infra.DefaultModelConfig) {
	l.Debugf("Models matching defaults: %d (looking for fast=%s, smart=%s, pro=%s)",
		len(matched), defaults.Fast, defaults.Smart, defaults.Pro)

	for i, model := range matched {
		l.Debugf("Matched model %d: Name=%s, MappingId=%s", i, model.ModelName, model.ModelMappingId)
	}
}

// logExtractedDefaultModels logs the final extracted default models
func logExtractedDefaultModels(l *zap.SugaredLogger, defaultModels *DefaultModels, defaults *infra.DefaultModelConfig) {
	l.Debugf("Final default models extraction result: Fast=%v, Smart=%v, Pro=%v",
		defaultModels.Fast != nil, defaultModels.Smart != nil, defaultModels.Pro != nil)

	logModelStatus(l, "Fast", defaultModels.Fast, defaults.Fast)
	logModelStatus(l, "Smart", defaultModels.Smart, defaults.Smart)

	if defaults.Pro != "" {
		logModelStatus(l, "Pro", defaultModels.Pro, defaults.Pro)
	}
}

// logModelStatus logs the status of a specific model type
func logModelStatus(l *zap.SugaredLogger, modelType string, model *ModelInfo, defaultName string) {
	if model != nil {
		l.Debugf("%s model found: %s (mapping_id: %s)", modelType, model.ModelName, model.ModelMappingId)
	} else {
		l.Warnf("No %s model found matching: %s", strings.ToLower(modelType), defaultName)
	}
}

// HandleCachedGetDefaultModelsRequest serves default models using the lazily-
// populated model cache. The first request triggers cache population using
// the caller's request context (which carries SAX credentials). The defaults
// configuration (fast/smart/pro mapping IDs) is still resolved per-request from
// environment variables and the ops defaults endpoint.
func HandleCachedGetDefaultModelsRequest(ctx context.Context, checker ContextChecker, cache *ModelListCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infof("Serving [%s %s]", c.Request.Method, c.Request.RequestURI)

		// Get default model configuration (env vars + ops endpoint)
		defaults, err := fetchDefaultModelsConfiguration(ctx)
		if err != nil {
			l.Errorf("Failed to retrieve default model mappings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logDefaultConfiguration(l, defaults)

		// Use cached enriched models (populated on first request)
		reqCtx := checker.ContextWithGinContext(ctx, c)
		cachedModels, cacheWarnings := cache.GetModels(reqCtx)
		l.Debugf("Using cached models for defaults: %d models", len(cachedModels))
		if len(cacheWarnings) > 0 {
			l.Warnf("Model cache warnings (may affect default model resolution): %v", cacheWarnings)
		}

		logEnrichedModels(l, cachedModels)

		// Find and extract default models from cached list
		matched := findMatchingDefaultModels(cachedModels, defaults)
		logMatchedModels(l, matched, defaults)

		defaultModels := extractDefaultModels(matched, defaults)
		logExtractedDefaultModels(l, defaultModels, defaults)

		// Apply feature flag for Pro model
		if !helpers.GetEnvOrFalse("ENABLE_PRO_MODEL_DEFAULT") {
			defaultModels.Pro = nil
		}

		c.JSON(http.StatusOK, defaultModels)
	}
}

func fetchAWSModelsImpl(ctx context.Context) ([]ModelInfo, error) {

	l := cntx.LoggerFromContext(ctx).Sugar()
	models := make([]ModelInfo, 0)

	// Skip fetching AWS models if Bedrock is not enabled
	if !cntx.IsLLMProviderConfigured(ctx, "Bedrock") {
		l.Debug("Bedrock provider not enabled, returning empty model list")
		return models, nil
	}

	configs, err := GetInfraModelsForContext(ctx)
	if err != nil {
		return nil, err
	}

	for _, cfg := range configs {
		api := []string{cfg.TargetApi}
		creator := inferCreatorFromPath(cfg.Path)
		paths := calculateModelPaths(ctx, creator, cfg.ModelMapping, "", api)
		models = append(models, ModelInfo{
			Provider:  "bedrock",
			ModelPath: paths,
			Creator:   creator,
			ModelName: cfg.ModelMapping,
			ModelID:   cfg.ModelId,
		})
	}
	return models, nil
}

func fetchGCPModelsImpl(ctx context.Context) ([]ModelInfo, error) {

	var models = []ModelInfo{}
	l := cntx.LoggerFromContext(ctx).Sugar()

	// Skip fetching GCP models if Vertex is not enabled
	if !cntx.IsLLMProviderConfigured(ctx, "Vertex") {
		l.Debug("Vertex provider not enabled, returning empty model list")
		return models, nil
	}
	// Check if environment variable exists
	configFile, exists := os.LookupEnv("CONFIGURATION_FILE")
	if !exists {
		l.Debug("CONFIGURATION_FILE environment variable not set, skipping GCP model fetch")
		return []ModelInfo{}, nil
	}

	mapping, err := retrieveMapping(ctx, configFile)
	if err != nil {
		return nil, err
	}

	// Load model metadata once for the entire loop instead of per-model.
	metadata, metaErr := LoadModelMetadataFromFile(getModelMetadataPath())
	if metaErr != nil {
		l.Warnf("Could not load model metadata for GCP models: %v", metaErr)
	}

	for _, m := range mapping.Models {
		if m.Infrastructure == "gcp" {

			var modelID string
			if metadata != nil {
				modelID, err = resolveModelIDFromMetadata(m.Name, metadata)
			} else {
				err = metaErr
			}
			if err != nil {
				// Log warning but continue processing other models
				cntx.LoggerFromContext(ctx).Sugar().Warnf("Could not resolve modelID for GCP model %s: %v", m.Name, err)
			}
			paths := []string{m.Path}

			models = append(models, ModelInfo{
				Provider:  "vertex",
				ModelPath: paths,
				Creator:   m.Creator,
				ModelName: m.Name,
				ModelID:   modelID,
			})
		}
	}
	return models, nil
}

// AzureModelResponse represents the structure of a single Azure model from the API response
type AzureModelResponse struct {
	DeploymentID string `json:"deployment-id"`
	ModelName    string `json:"model-name"`
	ModelVersion string `json:"model-version"`
	Type         string `json:"type"`
	Endpoint     string `json:"endpoint"`
}

// fetchAzureAPI calls the Azure API and returns the response or an error with status code
func fetchAzureAPI(ctx context.Context, baseURL string) ([]AzureModelResponse, int, error) {
	l := cntx.LoggerFromContext(ctx).Sugar()

	// Check if baseURL is empty
	if baseURL == "" {
		l.Warn("Azure baseURL is empty")
		return nil, http.StatusBadRequest, fmt.Errorf("empty Azure baseURL")
	}

	url := fmt.Sprintf("%s/openai/models", baseURL)
	l.Debugf("Constructed Azure models URL: %s", url)

	resp, err := CallTargetWithResponse(ctx, url, "GET", http.Header{}, nil, cntx.IsUseSax(ctx))
	if err != nil {
		return nil, 0, fmt.Errorf("fetching Azure models failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		l.Warnf("Azure returned 401 Unauthorized")
		return nil, http.StatusUnauthorized, fmt.Errorf("unauthorized")
	}

	l.Debugf("Azure response status: %s", resp.Status)
	var result struct {
		Models []AzureModelResponse `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		l.Errorf("Failed to decode Azure models response: %v", err)
		return nil, resp.StatusCode, err
	}

	return result.Models, http.StatusOK, nil
}

// findModelConfigByName looks up a model configuration by name
func findModelConfigByName(mapping *Mapping, modelName string) *Model {
	if mapping != nil && len(mapping.Models) > 0 {
		for _, config := range mapping.Models {
			if config.Name == modelName {
				return &config
			}
		}
	}
	return nil
}

// createAzureModelInfo converts an Azure model response to ModelInfo.
// The metadata parameter should be pre-loaded to avoid repeated file I/O.
func createAzureModelInfo(ctx context.Context, azureModel AzureModelResponse, modelConfig *Model, metadata map[string]ModelMetadata) ModelInfo {
	l := cntx.LoggerFromContext(ctx).Sugar()

	fullModelName := fmt.Sprintf("%s-%s", azureModel.ModelName, azureModel.ModelVersion)
	var modelID string
	var err error
	if metadata != nil {
		modelID, err = resolveModelIDFromMetadata(fullModelName, metadata)
	} else {
		err = fmt.Errorf("model metadata not available")
	}
	if err != nil {
		l.Warnf("Could not resolve modelID for Azure model %s: %v", fullModelName, err)
	}

	// Determine API paths
	var api []string
	if modelConfig != nil && modelConfig.TargetAPI != "" {
		api = []string{modelConfig.TargetAPI}
	} else {
		api = inferAPIFromPath(azureModel.Type)
	}

	// Determine creator
	var creator string
	if modelConfig != nil && modelConfig.Creator != "" {
		creator = modelConfig.Creator
	} else {
		creator = inferCreatorFromPath(azureModel.Type)
	}

	// Determine model paths
	var paths []string
	if modelConfig != nil && modelConfig.Path != "" {
		paths = []string{modelConfig.Path}
	} else {
		paths = calculateModelPaths(ctx, "azure", modelID, azureModel.Endpoint, api)
	}

	return ModelInfo{
		Provider:  "azure",
		ModelPath: paths,
		Creator:   creator,
		ModelName: fullModelName,
		ModelID:   modelID,
	}
}

func fetchAzureModelsImpl(ctx context.Context, baseURL string) ([]ModelInfo, int, error) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	l.Debugf("Starting fetchAzureModels with baseURL: %s", baseURL)
	if !cntx.IsLLMProviderConfigured(ctx, "Azure") {
		l.Debug("Azure provider not enabled, returning empty model list")
		return []ModelInfo{}, http.StatusOK, nil
	}

	// Check if baseURL is empty
	if baseURL == "" {
		l.Debug("Azure baseURL is empty, returning empty model list")
		return []ModelInfo{}, http.StatusOK, nil
	}

	// Step 1: Fetch models from Azure API
	azureModels, statusCode, err := fetchAzureAPI(ctx, baseURL)
	if err != nil {
		return nil, statusCode, err
	}

	// Step 2: Get the model configuration mapping
	configFile := getEnvOrPanic("CONFIGURATION_FILE")
	mapping, err := retrieveMapping(ctx, configFile)
	if err != nil {
		return nil, 0, err
	}

	// Step 2b: Load model metadata once for the entire loop.
	metadata, metaErr := LoadModelMetadataFromFile(getModelMetadataPath())
	if metaErr != nil {
		l.Warnf("Could not load model metadata for Azure models: %v", metaErr)
	}

	// Step 3: Process each model and convert to ModelInfo
	models := make([]ModelInfo, 0, len(azureModels))
	for _, azureModel := range azureModels {
		modelConfig := findModelConfigByName(mapping, azureModel.ModelName)
		modelInfo := createAzureModelInfo(ctx, azureModel, modelConfig, metadata)
		models = append(models, modelInfo)
	}

	l.Debugf("Completed fetchAzureModels. Total models: %d", len(models))
	return models, http.StatusOK, nil
}

func deduplicateModelsImpl(models []ModelInfo) []ModelInfo {
	merged := make(map[string]ModelInfo)

	for _, model := range models {
		key := model.ModelName

		if existing, exists := merged[key]; exists {
			// Merge ModelPaths
			existing.ModelPath = mergeUniqueStrings(existing.ModelPath, model.ModelPath)
			merged[key] = existing
		} else {
			merged[key] = model
		}
	}

	result := make([]ModelInfo, 0, len(merged))
	for _, model := range merged {
		result = append(result, model)
	}
	return result
}

func mergeUniqueStrings(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, val := range append(a, b...) {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// shouldDisplayPreviewModels checks if Preview models should be displayed
// based on the DISPLAY_PREVIEW_MODELS environment variable
func shouldDisplayPreviewModels() bool {
	val := os.Getenv("DISPLAY_PREVIEW_MODELS")
	return strings.EqualFold(val, "true")
}

// isPreviewModel checks if a model has a Preview lifecycle (case-insensitive)
func isPreviewModel(lifecycle string) bool {
	return strings.EqualFold(lifecycle, "Preview")
}

// getModelMetadataPath returns the path to the model metadata file from the environment variable
// or defaults to /models-metadata/model-metadata.yaml if not set
func getModelMetadataPath() string {
	path := os.Getenv("MODEL_METADATA_PATH")
	if path == "" {
		return "/models-metadata/model-metadata.yaml"
	}
	return path
}

var LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]ModelMetadata
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// hasValidMetadata validates if a model has a corresponding metadata entry
func hasValidMetadata(model *ModelInfo, metadata map[string]ModelMetadata) bool {
	_, exists := metadata[model.ModelName]
	return exists
}

// validateAndEnrichModel validates if a model has metadata and enriches it if valid
func validateAndEnrichModel(ctx context.Context, model *ModelInfo, metadata map[string]ModelMetadata) (bool, error) {
	if !hasValidMetadata(model, metadata) {
		return false, nil
	}

	// Find the metadata key for this model
	var metadataKey string
	for key := range metadata {
		if key == model.ModelName {
			metadataKey = key
			break
		}
	}

	// Enrich the model with metadata since we know it exists
	enrichWithMetadata(ctx, model, metadata, metadataKey)

	log := loggerFromContext(ctx).Sugar()
	log.Debugf("Model [%s] passed metadata validation and was enriched", model.ModelName)

	return true, nil
}

// getModelIdentifier returns a human-readable identifier for a model
func getModelIdentifier(model *ModelInfo) string {
	if model.ModelName != "" {
		return model.ModelName
	}
	return fmt.Sprintf("Provider:%s Creator:%s ModelID:%s", model.Provider, model.Creator, model.ModelID)
}

// shouldFilterPreviewModel returns true if a model should be filtered out based on preview status
func shouldFilterPreviewModel(lifecycle string, displayPreview bool) bool {
	return isPreviewModel(lifecycle) && !displayPreview
}

// filterReason represents why a model was filtered out
type filterReason int

const (
	filterReasonNone filterReason = iota
	filterReasonNoMetadata
	filterReasonPreview
	filterReasonError
)

// processModelForEnrichment validates and enriches a single model
// Returns the filter reason (filterReasonNone means include the model)
func processModelForEnrichment(ctx context.Context, model *ModelInfo, metadata map[string]ModelMetadata, displayPreview bool) filterReason {
	log := loggerFromContext(ctx).Sugar()

	isValid, err := validateAndEnrichModel(ctx, model, metadata)
	if err != nil {
		log.Errorf("Error validating/enriching model [%s]: %v", model.ModelName, err)
		return filterReasonError
	}

	if !isValid {
		return filterReasonNoMetadata
	}

	if shouldFilterPreviewModel(model.Lifecycle, displayPreview) {
		return filterReasonPreview
	}

	return filterReasonNone
}

// filterAndEnrichModels processes all models and returns valid models plus filtered model identifiers
func filterAndEnrichModels(ctx context.Context, models []ModelInfo, metadata map[string]ModelMetadata, displayPreview bool) ([]ModelInfo, []string, []string) {
	log := loggerFromContext(ctx).Sugar()
	log.Debugf("Starting metadata validation for %d models", len(models))

	validModels := make([]ModelInfo, 0, len(models))
	var filteredModels []string
	var previewFilteredModels []string

	for _, model := range models {
		reason := processModelForEnrichment(ctx, &model, metadata, displayPreview)
		identifier := getModelIdentifier(&model)

		switch reason {
		case filterReasonNone:
			validModels = append(validModels, model)
		case filterReasonNoMetadata:
			filteredModels = append(filteredModels, identifier)
			log.Warnf("Model [%s] filtered out due to missing metadata entry in metadata file", identifier)
		case filterReasonPreview:
			previewFilteredModels = append(previewFilteredModels, identifier)
			log.Debugf("Model [%s] filtered out due to Preview lifecycle (DISPLAY_PREVIEW_MODELS=false)", identifier)
		case filterReasonError:
			// Error already logged in processModelForEnrichment
		default:
			// Unknown filter reason - log warning and skip model for safety
			log.Warnf("Model [%s] skipped due to unknown filter reason: %d", identifier, reason)
		}
	}

	return validModels, filteredModels, previewFilteredModels
}

// addDeprecatedModelsFromMetadata adds deprecated models from metadata that have no active infra mapping.
// This ensures clients can discover deprecated models and their fallback information even when
// the model's infrastructure mapping has been deactivated.
func addDeprecatedModelsFromMetadata(ctx context.Context, models []ModelInfo, metadata map[string]ModelMetadata) []ModelInfo {
	log := loggerFromContext(ctx).Sugar()

	// Build set of model names already present
	existing := make(map[string]struct{}, len(models))
	for _, m := range models {
		existing[m.Name] = struct{}{}
	}

	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		meta := metadata[key]

		lifecycle := meta.Lifecycle
		if lifecycle == "" {
			lifecycle = inferLifecycleFromDate(meta.DeprecationDate)
		}

		name := strings.ToLower(key)
		if lifecycle != "Deprecated" {
			continue
		}
		if _, found := existing[name]; found {
			continue
		}

		params := meta.Parameters
		if params == nil {
			params = map[string]ParameterSpec{}
		}

		model := ModelInfo{
			Name:               name,
			ModelName:          meta.ModelName,
			ModelMappingId:     meta.ModelMappingId,
			ModelID:            meta.ModelID,
			Provider:           meta.Provider,
			Creator:            meta.Creator,
			Description:        meta.ModelDescription,
			ModelLabel:         meta.ModelLabel,
			Version:            meta.Version,
			Type:               meta.Type,
			InputTokens:        meta.InputTokens,
			Lifecycle:          "Deprecated",
			DeprecationDate:    meta.DeprecationDate,
			AlternateModelInfo: meta.AlternateModelInfo,
			DeprecationInfo: DeprecationInfo{
				IsDeprecated:             true,
				ScheduledDeprecationDate: meta.DeprecationDate,
			},
			SupportedCapabilities: convertToSupportedCapabilities(meta.ModelCapabilities, meta.Provider, nil),
			Parameters:            params,
			ModelPath:             []string{},
			Examples:              meta.Examples,
		}

		models = append(models, model)
		log.Infof("Added deprecated model [%s] from metadata (no active mapping)", key)
	}

	return models
}

// logFilteringResults logs the summary of filtering results
func logFilteringResults(log *zap.SugaredLogger, validCount int, filteredModels, previewFilteredModels []string) {
	if len(previewFilteredModels) > 0 {
		log.Debugf("Preview models filtered out: %v", previewFilteredModels)
	}

	log.Debugf("Metadata filtering complete. Valid models: %d, Filtered models: %d", validCount, len(filteredModels))

	if len(filteredModels) > 0 {
		log.Warnf("Filtered models list: %v", filteredModels)
	}
}

// logParsedMetadata logs debug information about parsed metadata
func logParsedMetadata(log *zap.SugaredLogger, metadata map[string]ModelMetadata) {
	for k, v := range metadata {
		log.Debugf("Parsed metadata for [%s]: %+v", k, v.ModelCapabilities)
	}
}

func enrichModelsImpl(ctx context.Context, models []ModelInfo) []ModelInfo {
	log := loggerFromContext(ctx).Sugar()
	log.Debugf("Enriching the response with model metadata")

	metadata, err := LoadModelMetadataFromFile(getModelMetadataPath())
	if err != nil {
		log.Errorf("Failed to load model metadata: %v", err)
		return []ModelInfo{}
	}

	logParsedMetadata(log, metadata)

	displayPreview := shouldDisplayPreviewModels()
	validModels, filteredModels, previewFilteredModels := filterAndEnrichModels(ctx, models, metadata, displayPreview)
	validModels = addDeprecatedModelsFromMetadata(ctx, validModels, metadata)

	logFilteringResults(log, len(validModels), filteredModels, previewFilteredModels)

	return validModels
}

func inferAPIFromPath(path string) []string {
	var apis []string
	log := cntx.LoggerFromContext(context.Background()).Sugar()
	log.Debugf("Inferring API from path: %s", path)
	switch {
	case strings.Contains(path, "converse-stream"):
		apis = append(apis, "/converse-stream")
	case strings.Contains(path, "chat"):
		apis = append(apis, "/chat/completions")
	case strings.Contains(path, "gemini"):
		apis = append(apis, "/chat/completions")
	case strings.Contains(path, "converse"):
		apis = append(apis, "/converse")
	case strings.Contains(path, "embedding"):
		apis = append(apis, "/embeddings")
	case strings.Contains(path, "image") || strings.Contains(path, "vision"):
		apis = append(apis, "/images/generations")
	case strings.Contains(path, "invoke"):
		apis = append(apis, "/invoke")
	case strings.Contains(path, "realtime"):
		apis = append(apis, "/v1/realtime/client_secrets")
	}

	if len(apis) == 0 {
		apis = append(apis, "/unknown")

	}
	return apis
}

func inferCreatorFromPath(path string) string {
	switch {
	case strings.Contains(path, "google"):
		return "google"
	case strings.Contains(path, "meta"):
		return "meta"
	case strings.Contains(path, "amazon"):
		return "amazon"
	case strings.Contains(path, "anthropic"):
		return "anthropic"
	default:
		return "openai"
	}
}

func enrichWithMetadata(ctx context.Context, model *ModelInfo, metadata map[string]ModelMetadata, metadataKey string) {
	log := loggerFromContext(ctx).Sugar()
	log.Debugf("Trying to enrich model [%s]", model.ModelName)

	if meta, ok := metadata[model.ModelName]; ok {
		// Map legacy fields for backward compatibility
		if meta.Lifecycle == "" {
			meta.Lifecycle = inferLifecycleFromDate(meta.DeprecationDate)
		}

		model.Lifecycle = meta.Lifecycle
		model.DeprecationDate = meta.DeprecationDate
		model.ModelLabel = meta.ModelLabel
		model.Version = meta.Version
		model.Type = meta.Type
		model.InputTokens = meta.InputTokens
		model.Description = meta.ModelDescription

		// Copy AlternateModelInfo directly from metadata
		model.AlternateModelInfo = meta.AlternateModelInfo

		// Use ParameterSpec format directly
		if meta.Parameters != nil {
			model.Parameters = meta.Parameters
		} else {
			model.Parameters = map[string]ParameterSpec{}
		}

		// Set Autopilot-aligned fields
		model.ModelName = meta.ModelName
		model.Name = strings.ToLower(metadataKey) // Use metadata key directly as name

		// Set ModelMappingId for reference
		model.ModelMappingId = meta.ModelMappingId
		// Set deprecation info
		model.DeprecationInfo = DeprecationInfo{
			IsDeprecated:             meta.Lifecycle == "Deprecated",
			ScheduledDeprecationDate: meta.DeprecationDate,
		}

		// Set capabilities
		model.SupportedCapabilities = convertToSupportedCapabilities(meta.ModelCapabilities, model.Provider, model.ModelPath)
		model.Examples = meta.Examples

		// Set token limits from parameters if available
		setInputTokensFromParameters(ctx, model, meta.Parameters)
		setOutputTokensFromParameters(ctx, model, meta.Parameters)

		log.Debugf("Enriched model [%s] using metadata", model.ModelName)
	} else {
		log.Warnf("No metadata found for model name [%s]", model.ModelName)
		// Initialize fields to avoid nulls in JSON
		model.ModelPath = []string{}
		model.Parameters = map[string]ParameterSpec{}
		model.SupportedCapabilities = SupportedCapabilities{
			Streaming:               false,
			Multimodal:              []string{},
			Functions:               false,
			ParallelFunctionCalling: false,
			JSONMode:                false,
			IsMultimodal:            false,
		}
		model.DeprecationInfo = DeprecationInfo{
			IsDeprecated:             false,
			ScheduledDeprecationDate: "",
		}
	}
}

func inferLifecycleFromDate(deprecationDate string) string {
	if deprecationDate == "" || deprecationDate == "NA" {
		return "Generally Available"
	}
	parsed, err := time.Parse("2006-01-02", deprecationDate)
	if err != nil {
		return "Generally Available"
	}
	now := time.Now()
	if parsed.Before(now) {
		return "Deprecated"
	}
	if parsed.Before(now.AddDate(0, 3, 0)) {
		return "Nearing Deprecation"
	}
	return "Generally Available"
}

// extractDefaultModelsImpl first tries to find a ModelInfo where the ModelMappingId matches exactly.
func extractDefaultModelsImpl(models []ModelInfo, defaults *infra.DefaultModelConfig) *DefaultModels {
	result := &DefaultModels{}
	for _, model := range models {
		if result.Smart == nil && model.ModelMappingId == defaults.Smart {
			result.Smart = &model
		}
		if result.Fast == nil && model.ModelMappingId == defaults.Fast {
			result.Fast = &model
		}
		if result.Pro == nil && model.ModelMappingId == defaults.Pro {
			result.Pro = &model
		}
		// Check if all models are found (note: Pro might be empty if not configured)
		if result.Smart != nil && result.Fast != nil && (defaults.Pro == "" || result.Pro != nil) {
			return result
		}
	}
	return result
}

func resolveModelID(modelName string) (string, error) {
	// GCP & Azure: modelID must be fetched from static metadata
	metadata, err := LoadModelMetadataFromFile(getModelMetadataPath())
	if err != nil {
		return "", fmt.Errorf("failed to load model metadata: %w", err)
	}
	return resolveModelIDFromMetadata(modelName, metadata)
}

// resolveModelIDFromMetadata looks up the modelID in pre-loaded metadata,
// avoiding repeated file I/O when processing multiple models in a loop.
func resolveModelIDFromMetadata(modelName string, metadata map[string]ModelMetadata) (string, error) {
	if meta, ok := metadata[modelName]; ok {
		return meta.ModelID, nil
	}
	return "", fmt.Errorf("modelID not found in metadata for model: %s", modelName)
}

func formatPathWithAPI(prefix string, modelID string, api string) string {
	// Ensure API starts with a slash for proper path formatting
	if !strings.HasPrefix(api, "/") {
		api = "/" + api
	}
	return fmt.Sprintf("%s/%s%s", prefix, modelID, api)
}

func getPathPrefix(ctx context.Context, creator string) string {
	l := cntx.LoggerFromContext(ctx).Sugar()
	switch creator {
	case "bedrock", "anthropic", "meta", "amazon":
		return fmt.Sprintf("/%s/deployments", creator)
	case "vertex", "google":
		return "/google/deployments"
	case "azure", "azureopenai":
		return "/openai/deployments"
	default:
		l.Warnf("Unknown creator '%s' encountered, no deployment path available", creator)
		return ""
	}
}

func calculateModelPaths(ctx context.Context, creator, modelID, endpoint string, apis []string) []string {
	var paths []string

	// Special case for Azure with endpoint
	if (creator == "azureopenai" || creator == "azure") && endpoint != "" {
		return []string{fmt.Sprintf("/openai%s", endpoint)}
	}

	// For all other cases, use the standard path format
	prefix := getPathPrefix(ctx, creator)

	// If prefix is empty (unknown creator), return empty paths
	if prefix == "" {
		return []string{}
	}

	for _, api := range apis {
		path := formatPathWithAPI(prefix, modelID, api)
		paths = append(paths, path)
	}

	return paths
}

// Helper functions for Autopilot format conversion

// generateModelName creates a name field from modelName and version
func generateModelName(modelName, version string) string {
	if version != "" && version != "v1" && version != "1" {
		// Handle different version formats
		if strings.Contains(modelName, version) {
			return strings.ToLower(modelName)
		}
		return strings.ToLower(modelName + "-" + version)
	}
	return strings.ToLower(modelName)
}

// generateModelDescription creates a description based on provider and type
func generateModelDescription(provider, modelType string) string {
	var providerName string
	switch provider {
	case "bedrock":
		providerName = "AWS Bedrock"
	case "vertex":
		providerName = "Google Vertex AI"
	case "azure", "azureopenai":
		providerName = "Azure OpenAI"
	default:
		caser := cases.Title(language.English)
		providerName = caser.String(provider)
	}

	var typeDescription string
	switch modelType {
	case "chat_completion":
		typeDescription = "Chat Completions model"
	case "embedding":
		typeDescription = "Embedding model"
	case "image":
		typeDescription = "Image model"
	case "completion":
		typeDescription = "Completions model"
	case "realtime":
		typeDescription = "Realtime speech and audio model"
	default:
		typeDescription = "Unclassified model type"
	}

	return fmt.Sprintf("%s %s", providerName, typeDescription)
}

// hasStreamingTargetAPI checks if any target API path contains "converse-stream"
func hasStreamingTargetAPI(modelPaths []string) bool {
	for _, path := range modelPaths {
		if strings.Contains(path, "converse-stream") {
			return true
		}
	}
	return false
}

// convertToSupportedCapabilities converts ModelCapabilities to SupportedCapabilities
// For Bedrock providers, streaming capability is determined by checking target API paths
// For other providers, it uses metadata-based streaming detection for backward compatibility
func convertToSupportedCapabilities(capabilities ModelCapabilities, provider string, modelPaths []string) SupportedCapabilities {
	// Determine streaming capability
	var streaming bool
	if provider == "bedrock" {
		// For Bedrock, check if ANY target API path contains "converse-stream"
		streaming = hasStreamingTargetAPI(modelPaths)
	} else if provider == "vertex" {
		streaming = helpers.GetEnvOrFalse("USE_VERTEXAI_INFRA")
	} else {
		// For non-Bedrock providers, use metadata-based streaming detection
		streaming = contains(capabilities.Features, "streaming")
	}

	supported := SupportedCapabilities{
		Streaming:               streaming,
		Functions:               contains(capabilities.Features, "functionCalling") || contains(capabilities.Features, "toolCalling"),
		ParallelFunctionCalling: contains(capabilities.Features, "parallelFunctionCalling") || contains(capabilities.Features, "parallelToolCalling"),
		JSONMode:                contains(capabilities.Features, "jsonMode") || contains(capabilities.Features, "structuredOutput"),
		Multimodal:              capabilities.MimeTypes,
		IsMultimodal:            len(capabilities.MimeTypes) > 0 || len(capabilities.InputModalities) > 1,
	}
	return supported
}

// setInputTokensFromParameters extracts input token limits from parameters and sets them on the model
func setInputTokensFromParameters(ctx context.Context, model *ModelInfo, params map[string]ParameterSpec) {
	var inputTokensSet bool
	for key, param := range params {
		switch strings.ToLower(key) {
		case "maxinputtokens", "max_input_tokens":
			setInputTokensFromParam(model, param)
			inputTokensSet = true
		}
	}
	if !inputTokensSet {
		l := loggerFromContext(ctx).Sugar()
		l.Infof("input_tokens not set for model %s", model.ModelName)
	}
}

// setOutputTokensFromParameters handles output token limit configuration with precedence rules.
// max_completion_tokens takes precedence over max_tokens when both are present.
func setOutputTokensFromParameters(ctx context.Context, model *ModelInfo, params map[string]ParameterSpec) {
	var maxTokensParam *ParameterSpec
	var maxCompletionTokensParam *ParameterSpec

	for key, param := range params {
		switch strings.ToLower(key) {
		case "max_tokens":
			p := param
			maxTokensParam = &p
		case "max_completion_tokens":
			p := param
			maxCompletionTokensParam = &p
		default:
		}
	}

	// Log info when both are present
	if maxTokensParam != nil && maxCompletionTokensParam != nil {
		l := loggerFromContext(ctx).Sugar()
		l.Infof("Both max_tokens and max_completion_tokens present for model %s; max_completion_tokens takes precedence", model.ModelName)
	}

	// max_completion_tokens takes precedence, max_tokens is fallback
	switch {
	case maxCompletionTokensParam != nil:
		setOutputTokensFromParam(model, *maxCompletionTokensParam)
	case maxTokensParam != nil:
		setOutputTokensFromParam(model, *maxTokensParam)
	default:
		l := loggerFromContext(ctx).Sugar()
		l.Infof("Neither max_tokens nor max_completion_tokens populated for model %s", model.ModelName)
	}
}

func setInputTokensFromParam(model *ModelInfo, param ParameterSpec) {
	if param.Maximum != nil {
		val := int(*param.Maximum)
		model.InputTokens = &val
	}
}

func setOutputTokensFromParam(model *ModelInfo, param ParameterSpec) {
	// Priority 1: Use Maximum if available
	if param.Maximum != nil {
		val := int(*param.Maximum)
		model.OutputTokens = &val
		return
	}

	// Priority 2: Use Default if available
	if param.Default == nil {
		return
	}

	// Handle different types of default values using type switch
	switch defaultVal := param.Default.(type) {
	case float64:
		val := int(defaultVal)
		model.OutputTokens = &val
	case int:
		model.OutputTokens = &defaultVal
	default:
		// Unsupported default value type - no action taken
	}
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
