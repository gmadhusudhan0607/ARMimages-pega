#
# Copyright (c) 2024 Pegasystems Inc.
# All rights reserved.
#

OS_NAME ?= $(shell uname -s)
OS_ARCH ?= $(shell uname -m)

ifeq ($(OS_NAME), Darwin)
FIND_CMD=gfind
SED_CMD=gsed
else
FIND_CMD=find
SED_CMD=sed
endif

COMPOSE_PROJECT_NAME = genai-hub-service
DOCKER_COMPOSE_VER = v2.24.6
DOCKER_COMPOSE_URI = https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VER}/docker-compose-${OS_NAME}-${OS_ARCH}

CURRENT_DIR ?= $(shell pwd)

# Use locally installed docker-compose if found
ifeq (,$(shell which docker-compose))
SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION=false
DOCKER_COMPOSE_TOOL=./bin/docker-compose
else
SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION=true
DOCKER_COMPOSE_TOOL=$(shell which docker-compose)
endif

MOCK_SERVER_HOST_IN_DOCKER_COMPOSE = genai-hub-mockserver
CONFIGURATION_DIR_TEMPLATES = ./distribution/genai-hub-service-helm/src/main/helm/configuration
CONFIGURATION_DIR = ./build/config
CONFIGURATION_FILE_INTEG = $(CONFIGURATION_DIR)/mapping.yaml
CONFIGURATION_FILE_INTEG1 = $(CONFIGURATION_DIR)/mapping1.yaml
CONFIGURATION_FILE_LOCAL = $(CONFIGURATION_DIR)/mapping-local.yaml
CONFIGURATION_FILE_INTEG_GENAI_INFRA = $(CONFIGURATION_DIR)/mapping-genai-infra.yaml
CONFIGURATION_FILE_INTEG_PRIVATE_MODEL = $(CONFIGURATION_DIR)/mapping-private-model.yaml
CONFIGURATION_DIR_PRIVATE_MODEL_TEMPLATE = ./distribution/genai-private-model-config-terraform/src/main/terraform/aws/templates/model-secret-template.tftpl
CONFIGURATION_DIR_PRIVATE_MODEL = ./build/private-model-config
PRIVATE_MODEL_FILE_PREFIX = genai_private_model

DOCKER_ARTIFACTORY_URL = $(shell cat gradle.properties  | grep -E '^\s*dockerRegistryUrl\s*=' | awk -F = '{print $$2}' | tr -d ' ' | awk -F '//' '{print $$2}' )
DOCKER_REPO_PREFIX = $(shell cat gradle.properties  | grep -E '^\s*dockerRepoPrefix\s*=' | awk -F = '{print $$2}' | tr -d ' ' )
DOCKER_DEV_REPO = $(shell cat gradle.properties  | grep -E '^\s*dockerDevRegistry\s*=' | awk -F = '{print $$2}' | tr -d ' ' )


.EXPORT_ALL_VARIABLES:
# Image URL to use all running integration tests
SERVICE_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-hub-service:local"
SERVICE_DOCKER_OPS_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-gateway-ops:local"

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

SRC_MODULES_DIR = $(shell go list ./... | grep -vE "/test/integration|/test/load|/test/live")
SRC_LINT_DIRS = $(shell go list ./... | grep -vE "/test/integration|/test/load|/test/live" | sed "s|$$(go list -m)||" | sed 's|^|.|')
LABEL_FILTER ?=
##############################################################

##@ Environment
define setup_env
	$(eval ENV_FILE := $(1))
	@echo " - setup environment from $(ENV_FILE)"
	$(eval include $(1))
	$(eval export)
endef

##@ Mapping Config from templates
define create_mapping_config
	$(eval TEMPLATE_DIR := $(firstword $(subst :, ,$(1))))
	$(eval OUTPUT_FILE  := $(firstword $(subst :, ,$(2))))
	$(eval MOCK_SERVER  := $(firstword $(subst :, ,$(3))))
	$(eval SVC_PORT     := $(firstword $(subst :, ,$(4))))
	$(eval OUTPUT_DIR := $(dir $(OUTPUT_FILE)))

	@echo " - creating mapping file: $(OUTPUT_FILE)"
	@echo " - SERVICE_HOST_PORT: ${SVC_PORT}"
	mkdir -p $(OUTPUT_DIR)
	rm -fr   $(OUTPUT_FILE)

	$(eval SUB_DIRS=$(shell $(FIND_CMD) $(TEMPLATE_DIR) -mindepth 1 -maxdepth 1 -type d -exec  basename {} \;))
	for module in $(SUB_DIRS); do \
		echo "$${module}:" >> $(OUTPUT_FILE) ; \
		for file in $(TEMPLATE_DIR)/$${module}/*.yaml; do \
			cat "$${file}" |  $(SED_CMD) -e "s/^/  /"  >> $(OUTPUT_FILE) ;\
			echo "" |  $(SED_CMD) -e "s/^/  /"  >> $(OUTPUT_FILE) ;\
		done ;\
	done
	@- $(SED_CMD) "/^[[:space:]]*$$/d" -i  $(OUTPUT_FILE)
	@- echo  "" >> $(OUTPUT_FILE)

	$(SED_CMD) -i "s/{{\s*\.Release\.Namespace\s*}}\.genai-hub-service\.svc\.cluster\.local:443/localhost:${SVC_PORT}/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/{{\s*\.Values\.GenAIURL\s*}}/http:\/\/${MOCK_SERVER}:11080/g" $(OUTPUT_FILE)
#	$(SED_CMD) -i "s/{{\s*\.Values\.GenAIURL\s*}}/https:\/\/dev-pcld-openai-apim-eastus2-itigj.azure-api.net/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/{{\s*\.Values\.DemoAwsBedrockURL\s*}}/http:\/\/${MOCK_SERVER}:11080/g" $(OUTPUT_FILE)
#	$(SED_CMD) -i "s/{{\s*\.Values\.DemoGcpVertexURL\s*}}/https:\/\/genai-vertexai-gateway-d0lz0m1x.uc.gateway.dev/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/{{\s*\.Values\.DemoGcpVertexURL\s*}}/http:\/\/${MOCK_SERVER}:11080/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/{{\s*\.Values\.SelfStudyBuddyURLv1\s*}}/http:\/\/${MOCK_SERVER}:11080/g" $(OUTPUT_FILE)

endef

# creating Private Model config files from template
define create_private_model_config

	$(eval PRIVATE_MODEL_TEMPLATE := $(1))
	$(eval OUTPUT_DIR := $(2))
	$(eval MOCK_SERVER := $(3))
	$(eval SVC_PORT := $(4))
	$(eval MODEL := $(5))
	$(eval MODEL_PROVIDER := $(6))
	$(eval OUTPUT_FILE := $(OUTPUT_DIR)/$(PRIVATE_MODEL_FILE_PREFIX)_$(MODEL))

	mkdir -p $(OUTPUT_DIR)
	rm -fr   $(OUTPUT_FILE)

	@echo " - creating private model config file: $(OUTPUT_FILE)"
	cat $(PRIVATE_MODEL_TEMPLATE) > $(OUTPUT_FILE)

	$(SED_CMD) -i "s/\$${Model}/$(MODEL)/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/\$${ModelProvider}/$(MODEL_PROVIDER)/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/{{\s*\.Release\.Namespace\s*}}\.genai-hub-service\.svc\.cluster\.local:443/localhost:${SVC_PORT}/g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s|\$${ModelEndpoint}|http://${MOCK_SERVER}:11080/${MODEL_PROVIDER}/deployments/${MODEL}|g" $(OUTPUT_FILE)
	$(SED_CMD) -i "s/\$${Active}/true/g" $(OUTPUT_FILE)

endef

.PHONY: test-processors-registry-coverage
test-processors-registry-coverage:
	@echo "🔍 Checking processor registry coverage..."
	go test -v ./internal/request/processors/registry -run TestAllModelsHaveProcessors

.PHONY: test-registry-integrity  
test-registry-integrity:
	@echo "🔍 Checking processor registry integrity..."
	go test -v ./internal/request/processors/registry -run TestProcessorRegistryIntegrity


.PHONY: test
test: fmt vet lint staticcheck test-processors-registry-coverage 
	mkdir -p build/test-results/unitTest
	CGO_ENABLED=1 go tool gotestsum --format standard-verbose --jsonfile ./build/test-results/unitTest/unitTestReport.json --junitfile ./build/test-results/unitTest/unitTestReport.xml -- -failfast -race -coverprofile=coverage.out -v $(SRC_MODULES_DIR) -coverpkg=./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet $(SRC_MODULES_DIR)

.PHONY: lint
lint: ## Download golangci-lint locally if necessary.
	 go tool golangci-lint run $(SRC_LINT_DIRS)

.PHONY: staticcheck
staticcheck:
	go tool staticcheck  $(SRC_MODULES_DIR)

.PHONY: goimports
goimports:
	goimports -w -local "github.com/Pega-CloudEngineering/" cmd internal pkg

.PHONY: build
build: fmt vet lint staticcheck ## Build service binary.
	go build -o bin/ ./cmd/service/...
	go build -o bin/ ./cmd/ops/...

.PHONY: run
run:  ## Run a service from your host.
	CONFIGURATION_FILE=$(CONFIGURATION_FILE_LOCAL) go run cmd/service/main.go

.PHONY: run_pretty
run_pretty:  ## Run a service from your host.
	CONFIGURATION_FILE=$(CONFIGURATION_FILE_LOCAL) go run cmd/service/main.go 2>&1 | jl

integrationTestEnv: ## load env/integration-test.env environment vars
	$(call setup_env, env/integration-test.env )

.PHONY: docker-compose-tool
docker-compose-tool:
	mkdir -p $(LOCALBIN)
	$(SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION) || test -s $(LOCALBIN)/docker-compose || ( curl -SL "${DOCKER_COMPOSE_URI}" -o $(LOCALBIN)/docker-compose && chmod a+x $(LOCALBIN)/docker-compose )

.PHONY: integration-test-up
integration-test-up: docker-compose-tool integrationTestEnv generateMappingFiles generatePrivateModelConfigFiles # generateIntegrationMappingFile generateLocalMappingFile
	SERVICE_DOCKER_IMAGE=${SERVICE_DOCKER_IMAGE} SERVICE_DOCKER_OPS_IMAGE=${SERVICE_DOCKER_OPS_IMAGE} CONFIGURATION_DIR=$(CONFIGURATION_DIR) $(DOCKER_COMPOSE_TOOL) --profile app -f ./test/integration/docker-compose.yaml up -d --force-recreate -V
	sleep 3
	docker ps | grep -E "genai-hub-mockserver|genai-hub-service|genai-gateway-ops"
	echo "-----------------------------------------"
	docker container logs genai-hub-service

.PHONY: integration-test-down
integration-test-down: docker-compose-tool
	$(DOCKER_COMPOSE_TOOL) --profile app -f ./test/integration/docker-compose.yaml down

.PHONY: integration-test-run
integration-test-run: integrationTestEnv
	mkdir -p ./build/test-results/integrationTest/
	go clean -cache
	if [ -n $(LABEL_FILTER) ]; then \
		go tool ginkgo --label-filter=$(LABEL_FILTER) --junit-report ./build/test-results/integrationTest/junit_report.xml -v ./test/integration/... ; \
	else \
		go tool ginkgo --junit-report ./build/test-results/integrationTest/junit_report.xml -v ./test/integration/... ; \
	fi
# Create configuration files with replaced helm variables
.PHONY: generateMappingFiles
generateMappingFiles:
	# Generate mapping.yaml used by locally running app
	$(call create_mapping_config,  $(CONFIGURATION_DIR_TEMPLATES)  , $(CONFIGURATION_FILE_LOCAL) , localhost , 8080)
	# Generate mapping.yaml used by containers if integration tests
	$(call create_mapping_config,  $(CONFIGURATION_DIR_TEMPLATES)  , $(CONFIGURATION_FILE_INTEG) , $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 18080 )
	# Generate mapping1.yaml used by containers if integration tests (with  REQUEST_PROCESSING_COPYRIGHT_PROTECTION=true)
	$(call create_mapping_config,  $(CONFIGURATION_DIR_TEMPLATES)  , $(CONFIGURATION_FILE_INTEG1) , $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 48080 )
	# Generate mapping-private-model.yaml used by containers for testing Private Model scenarios
	$(call create_mapping_config,  $(CONFIGURATION_DIR_TEMPLATES)  , $(CONFIGURATION_FILE_INTEG_PRIVATE_MODEL) , $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 58080)
	# Generate mapping-genai-infra.yaml for GenAI Infra models test
	$(call create_mapping_config,  $(CONFIGURATION_DIR_TEMPLATES)  , $(CONFIGURATION_FILE_INTEG_GENAI_INFRA) , $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 9080 )

	echo ---------------------------------------------------
	cat  ./build/config/mapping.yaml
	echo ---------------------------------------------------

# Create private model configuration files
.PHONY: generatePrivateModelConfigFiles
generatePrivateModelConfigFiles:
	$(call create_private_model_config, $(CONFIGURATION_DIR_PRIVATE_MODEL_TEMPLATE), $(CONFIGURATION_DIR_PRIVATE_MODEL), $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 58080, gpt-35-turbo, openai)
	$(call create_private_model_config, $(CONFIGURATION_DIR_PRIVATE_MODEL_TEMPLATE), $(CONFIGURATION_DIR_PRIVATE_MODEL), $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 58080, gpt-4o-mini, openai)
	$(call create_private_model_config, $(CONFIGURATION_DIR_PRIVATE_MODEL_TEMPLATE), $(CONFIGURATION_DIR_PRIVATE_MODEL), $(MOCK_SERVER_HOST_IN_DOCKER_COMPOSE), 58080, text-embedding-ada-002, openai)

.PHONY: wiremock-up
wiremock-up: docker-compose-tool
	$(DOCKER_COMPOSE_TOOL) -f ./test/integration/request-processing/docker-compose.yaml up -d --force-recreate -V

.PHONY: wiremock-down
wiremock-down: docker-compose-tool
	$(DOCKER_COMPOSE_TOOL)  -f ./test/integration/request-processing/docker-compose.yaml down


# Just na alias to run all request processing tests
request-processing-tests: test-request-processing

# ANSI color codes
RED := \033[0;31m
NC := \033[0m # No Color

##@ Model Management
.PHONY: add-bedrock-model
# Cross-platform script using Go (works on Linux, macOS, and Windows)
# Usage:
#   Single model:  make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0
#   With template: make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1
#   Batch mode:    make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json
#   Dry run:       make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true
add-bedrock-model:
	@if [ -z "$(MODEL_ID)" ] && [ -z "$(CONFIG)" ]; then \
		echo "❌ Error: MODEL_ID or CONFIG parameter is required"; \
		echo ""; \
		echo "Usage:"; \
		echo "  Single model:  make add-bedrock-model MODEL_ID=<model_id> [TEMPLATE=<template>] [DRY_RUN=true]"; \
		echo "  Batch mode:    make add-bedrock-model CONFIG=<config.json> [DRY_RUN=true]"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0"; \
		echo "  make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1"; \
		echo "  make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json"; \
		echo "  make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true"; \
		exit 1; \
	fi
	@if [ -n "$(CONFIG)" ]; then \
		if [ "$(DRY_RUN)" = "true" ]; then \
			go run scripts/add-bedrock-model.go -config $(CONFIG) -dry-run; \
		else \
			go run scripts/add-bedrock-model.go -config $(CONFIG); \
		fi; \
	else \
		if [ "$(DRY_RUN)" = "true" ]; then \
			if [ -n "$(TEMPLATE)" ]; then \
				go run scripts/add-bedrock-model.go -model-id $(MODEL_ID) -template $(TEMPLATE) -dry-run; \
			else \
				go run scripts/add-bedrock-model.go -model-id $(MODEL_ID) -dry-run; \
			fi; \
		else \
			if [ -n "$(TEMPLATE)" ]; then \
				go run scripts/add-bedrock-model.go -model-id $(MODEL_ID) -template $(TEMPLATE); \
			else \
				go run scripts/add-bedrock-model.go -model-id $(MODEL_ID); \
			fi; \
		fi; \
	fi

.PHONY: add-vertex-model
# Cross-platform script using Go (works on Linux, macOS, and Windows)
# Usage:
#   Single model:  make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion
#   With template: make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash
#   Preview model: make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion PREVIEW=true
#   Batch mode:    make add-vertex-model CONFIG=scripts/vertex-model-sample.json
#   Dry run:       make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion DRY_RUN=true
add-vertex-model:
	@if [ -z "$(MODEL_NAME)" ] && [ -z "$(CONFIG)" ]; then \
		echo "❌ Error: MODEL_NAME and MODEL_TYPE, or CONFIG parameter is required"; \
		echo ""; \
		echo "Usage:"; \
		echo "  Single model:  make add-vertex-model MODEL_NAME=<name> MODEL_TYPE=<type> [TEMPLATE=<template>] [PREVIEW=true] [DRY_RUN=true]"; \
		echo "  Batch mode:    make add-vertex-model CONFIG=<config.json> [DRY_RUN=true]"; \
		echo ""; \
		echo "Model types: chat_completion, embedding, image"; \
		echo ""; \
		echo "Options:"; \
		echo "  PREVIEW=true   Add model as preview (appends -preview suffix to model name)"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion"; \
		echo "  make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash"; \
		echo "  make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion PREVIEW=true"; \
		echo "  make add-vertex-model CONFIG=scripts/vertex-model-sample.json"; \
		echo "  make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion DRY_RUN=true"; \
		exit 1; \
	fi
	@if [ -n "$(CONFIG)" ]; then \
		if [ "$(DRY_RUN)" = "true" ]; then \
			go run scripts/add-vertex-model.go -config $(CONFIG) -dry-run; \
		else \
			go run scripts/add-vertex-model.go -config $(CONFIG); \
		fi; \
	else \
		if [ -z "$(MODEL_TYPE)" ]; then \
			echo "❌ Error: MODEL_TYPE is required when using MODEL_NAME"; \
			echo "   Valid types: chat_completion, embedding, image"; \
			exit 1; \
		fi; \
		PREVIEW_FLAG=""; \
		if [ "$(PREVIEW)" = "true" ]; then \
			PREVIEW_FLAG="-preview"; \
		fi; \
		if [ "$(DRY_RUN)" = "true" ]; then \
			if [ -n "$(TEMPLATE)" ]; then \
				go run scripts/add-vertex-model.go -model-name $(MODEL_NAME) -model-type $(MODEL_TYPE) -template $(TEMPLATE) $$PREVIEW_FLAG -dry-run; \
			else \
				go run scripts/add-vertex-model.go -model-name $(MODEL_NAME) -model-type $(MODEL_TYPE) $$PREVIEW_FLAG -dry-run; \
			fi; \
		else \
			if [ -n "$(TEMPLATE)" ]; then \
				go run scripts/add-vertex-model.go -model-name $(MODEL_NAME) -model-type $(MODEL_TYPE) -template $(TEMPLATE) $$PREVIEW_FLAG; \
			else \
				go run scripts/add-vertex-model.go -model-name $(MODEL_NAME) -model-type $(MODEL_TYPE) $$PREVIEW_FLAG; \
			fi; \
		fi; \
	fi

.PHONY: test-request-processing
# Usage:
#   make test-request-processing                           # runs all tests
#   make test-request-processing TEST=name                 # runs only the test in test/integration/request-processing/name
#   make test-request-processing TEST=name KEEP=true # runs specific test without stopping service

test-request-processing:
	@# Validate KEEP parameter usage
	@if [ -n "$$KEEP" ] && [ -z "$$TEST" ]; then \
		echo "❌ Error: KEEP parameter can only be used when TEST parameter is provided"; \
		echo "   Usage: make test-request-processing TEST=<test_name> KEEP=true"; \
		exit 1; \
	fi
	@echo "🔍 Running tests for request-processing..."
	mkdir -p ./build/test-results/requestProcessingTest/
	@failed_tests=0; \
	total_tests=0; \
	failed_test_names=""; \
	if [ -n "$$TEST" ]; then \
		test_dirs="test/integration/request-processing/$$TEST/"; \
	else \
		test_dirs="test/integration/request-processing/*/"; \
	fi; \
	for test_dir in $$test_dirs; do \
		if [ -d "$$test_dir" ] && [ -f "$$test_dir/suite_test.go" ]; then \
			test_name=$$(basename "$$test_dir"); \
			echo ""; \
			echo "▶️  Running tests in: $$test_name"; \
			if [ "$$KEEP" = "true" ]; then \
				echo "   🔧 KEEP=true: Service will not be stopped after test"; \
			fi; \
			echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
			total_tests=$$((total_tests + 1)); \
			mkdir -p "./build/test-results/requestProcessingTest/$$test_name"; \
			cd "$$test_dir" && \
			if KEEP=$$KEEP go tool ginkgo -v .; then \
				echo "✅ $$test_name: PASSED"; \
			else \
				echo "❌ $$test_name: FAILED"; \
				failed_tests=$$((failed_tests + 1)); \
				if [ -z "$$failed_test_names" ]; then \
					failed_test_names="$$test_name"; \
				else \
					failed_test_names="$$failed_test_names, $$test_name"; \
				fi; \
			fi; \
			cd - >/dev/null; \
		fi; \
	done; \
	echo ""; \
	echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
	echo "📊 Test Summary:"; \
	echo "   Total test suites: $$total_tests"; \
	echo "   Passed: $$((total_tests - failed_tests))"; \
	echo "   Failed: $$failed_tests"; \
	if [ $$failed_tests -eq 0 ]; then \
		echo "🎉 All request-processing tests passed!"; \
	else \
		echo "💥 $$failed_tests test suite(s) failed!"; \
		printf "   Failed tests: $(RED)%s$(NC)\n" "$$failed_test_names"; \
		exit 1; \
	fi

LIVE_CONFIGS_DIR = ./test/live/configs
LIVE_PROMPTS_DIR = ./test/live/prompts

.PHONY: test-live test-live-memleak test-live-memleak-get test-live-embeddings test-live-chat test-live-streaming test-live-image test-live-webrtc-text test-live-webrtc
# Usage:
#   make test-live                                                             # show help
#   make test-live RUN=list                                                    # list all test cases
#   make test-live RUN=all                                                     # all configs × all prompts
#   make test-live RUN=all VERBOSE=1                                           # + runner progress messages
#   make test-live RUN=all VERBOSE=2                                           # + runner progress + service logs (debug)
#   make test-live CONFIG=llm-retry                                            # specific config × all prompts
#   make test-live PROMPT=long-response                                        # all configs × specific prompt
#   make test-live CONFIG=llm-retry PROMPT=long-response                       # specific config × specific prompt
#   make test-live CONFIG=llm-retry MODEL=gpt-4o-mini                          # specific config × specific model
#   make test-live TEST_CASE=TestLive/llm-retry/long-response/ChatCompletion   # specific test case
#   make test-live RUN=all OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080  # use external services (both required)
test-live:
	@if [ -z "$(CONFIG)" ] && [ -z "$(PROMPT)" ] && [ -z "$(TEST_CASE)" ] && [ -z "$(RUN)" ]; then \
		echo "Live Test Runner"; \
		echo ""; \
		printf "  Available configs: "; ls -1 $(LIVE_CONFIGS_DIR) | tr '\n' ', ' | sed 's/,$$//'; echo; \
		printf "  Available prompts: "; ls -1 $(LIVE_PROMPTS_DIR) | tr '\n' ', ' | sed 's/,$$//'; echo; \
		echo ""; \
		echo "Usage:"; \
		echo "  make test-live RUN=list                                        # list all test cases"; \
		echo "  make test-live RUN=all                                         # all configs × all prompts"; \
		echo "  make test-live RUN=all VERBOSE=1                               # + runner progress messages"; \
		echo "  make test-live RUN=all VERBOSE=2                               # + runner progress + service logs (debug)"; \
		echo "  make test-live CONFIG=llm-retry                                # specific config × all prompts"; \
		echo "  make test-live PROMPT=long-response                            # all configs × specific prompt"; \
		echo "  make test-live CONFIG=llm-retry PROMPT=long-response           # specific config × specific prompt"; \
		echo "  make test-live RUN=all MODEL=gpt-4o-mini                       # filter by model name"; \
		echo "  make test-live RUN=all MODEL=openai/gpt-4o-mini                # filter by provider/model"; \
		echo "  make test-live TEST_CASE=TestLive/llm-retry/long-response/...  # specific test case"; \
		echo "  make test-live RUN=all OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080  # use external services (both required)"; \
		echo ""; \
		echo "Memory leak detection:"; \
		echo "  Options: MEMLEAK_ROUNDS=5 LEAK_HEAP_THRESHOLD_PCT=20 LEAK_GOROUTINE_THRESHOLD=5 LEAK_THREAD_THRESHOLD=2"; \
		echo ""; \
		echo "  Local services (started by the test runner):"; \
		echo "    make test-live-memleak CONFIG=llm-retry                        # detect leaks (1 config, all prompts)"; \
		echo "    make test-live-memleak CONFIG=llm-retry PROMPT=short-response  # detect leaks (1 config, 1 prompt)"; \
		echo "    make test-live-memleak CONFIG=llm-retry VERBOSE=1              # with runner progress"; \
		echo ""; \
		echo "  GET endpoints only (no inference):"; \
		echo "    make test-live-memleak-get CONFIG=llm-retry                      # detect leaks in /models, /models/defaults, /health"; \
		echo "    make test-live-memleak-get CONFIG=llm-retry MEMLEAK_GET_REPETITIONS=100  # more requests per round"; \
		echo ""; \
		echo "  External services (e.g. port-forwarded from cluster):"; \
		echo "    Forward hub-service (8080), ops (8081), and healthcheck/pprof (8082) ports to localhost, then:"; \
		echo ""; \
		echo "    make test-live-memleak OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 HEALTHCHECK_URL=http://localhost:8082"; \
		echo ""; \
		echo "Convenience targets (run specific test types):"; \
		echo "  make test-live-embeddings CONFIG=llm-retry PROMPT=long-response"; \
		echo "  make test-live-embeddings OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 PROMPT=long-response"; \
		echo "  make test-live-chat                                            # run only chat completion tests"; \
		echo "  make test-live-chat CONFIG=adaptive-strategy-disabled PROMPT=short-response"; \
		echo "  make test-live-streaming                                       # run only streaming tests"; \
		echo "  make test-live-streaming CONFIG=llm-retry PROMPT=long-response"; \
		echo "  make test-live-image                                           # run only image generation tests"; \
		echo "  make test-live-image CONFIG=gemini-image PROMPT=image-generation"; \
		echo "  make test-live-webrtc-text PROMPT=voice                       # run WebRTC realtime text tests"; \
		echo "  make test-live-webrtc-text PROMPT=voice MODEL=gpt-realtime-mini  # filter to specific model"; \
		echo "  make test-live-webrtc PROMPT=voice                              # run WebRTC realtime audio tests"; \
		echo "  make test-live-webrtc PROMPT=voice MODEL=gpt-realtime VERBOSE=1  # with runner progress"; \
	elif [ "$(RUN)" = "list" ]; then \
		echo "Live Test Cases:"; \
		echo ""; \
		count=0; \
		for config in $$(ls -1 $(LIVE_CONFIGS_DIR)); do \
			for prompt in $$(ls -1 $(LIVE_PROMPTS_DIR)); do \
				for test_type in ChatCompletion ChatCompletionStreaming Embeddings ImageGeneration; do \
					echo "  TestLive/$$config/$$prompt/$$test_type"; \
					count=$$((count + 1)); \
				done; \
			done; \
		done; \
		echo ""; \
		echo "Total: $$count test cases"; \
		echo ""; \
		echo "Run a specific test:"; \
		echo "  make test-live TEST_CASE=TestLive/<config>/<prompt>/<test_type>"; \
	else \
		RUN_FLAG=""; \
		LOG_OUTPUT_VAL="false"; \
		LOG_LEVEL_VAL="critical"; \
		VERBOSE_RUNNER_VAL="false"; \
		if [ -n "$(TEST_CASE)" ]; then \
			RUN_FLAG="-run $(TEST_CASE)"; \
		fi; \
		if [ "$(VERBOSE)" = "1" ]; then \
			VERBOSE_RUNNER_VAL="true"; \
			LOG_LEVEL_VAL="info"; \
		elif [ "$(VERBOSE)" = "2" ]; then \
			VERBOSE_RUNNER_VAL="true"; \
			LOG_OUTPUT_VAL="true"; \
			LOG_LEVEL_VAL="debug"; \
		fi; \
		echo "Running live tests $${RUN_FLAG:+(test case: $(TEST_CASE))} $${CONFIG:+(config: $(CONFIG))} $${PROMPT:+(prompt: $(PROMPT))} $${MODEL:+(model: $(MODEL))}"; \
		CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 $$RUN_FLAG ./test/live/runner/...; \
	fi

# Memory leak detection target
# Runs the test suite multiple rounds, captures pprof snapshots, and compares for leaks.
# When using external services (OPS_URL + SERVICE_URL), you must also set HEALTHCHECK_URL
# to point at the service's healthcheck/pprof port (where /debug/pprof/* is served).
# Usage:
#   make test-live-memleak OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 HEALTHCHECK_URL=http://localhost:8082  # external services
#   make test-live-memleak CONFIG=llm-retry                                            # local services, all prompts
#   make test-live-memleak CONFIG=llm-retry PROMPT=short-response                      # local services, specific prompt
#   make test-live-memleak CONFIG=llm-retry VERBOSE=1                                  # with runner progress
#   make test-live-memleak CONFIG=llm-retry MEMLEAK_ROUNDS=10                          # custom rounds
test-live-memleak:
	@echo "Running memory leak detection..."
	@RUN_FLAG=""; \
	LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) HEALTHCHECK_URL=$(HEALTHCHECK_URL) \
	MEMLEAK=true MEMLEAK_ROUNDS=$(MEMLEAK_ROUNDS) LEAK_HEAP_THRESHOLD_PCT=$(LEAK_HEAP_THRESHOLD_PCT) LEAK_GOROUTINE_THRESHOLD=$(LEAK_GOROUTINE_THRESHOLD) LEAK_THREAD_THRESHOLD=$(LEAK_THREAD_THRESHOLD) \
	VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL \
	go test -v -count=1 -timeout 60m -run TestMemLeak$$ ./test/live/runner/...

# Memory leak detection for GET endpoints only (/models, /models/defaults, /health/*)
# Isolates non-inference endpoint leak detection from model inference workload.
# Usage:
#   make test-live-memleak-get CONFIG=llm-retry                                            # local services
#   make test-live-memleak-get CONFIG=llm-retry MEMLEAK_GET_REPETITIONS=100                # more requests per round
#   make test-live-memleak-get OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 HEALTHCHECK_URL=http://localhost:8082
test-live-memleak-get:
	@echo "Running memory leak detection (GET endpoints only)..."
	@RUN_FLAG=""; \
	LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) HEALTHCHECK_URL=$(HEALTHCHECK_URL) \
	MEMLEAK=true MEMLEAK_ROUNDS=$(MEMLEAK_ROUNDS) MEMLEAK_GET_REPETITIONS=$(MEMLEAK_GET_REPETITIONS) \
	LEAK_HEAP_THRESHOLD_PCT=$(LEAK_HEAP_THRESHOLD_PCT) LEAK_GOROUTINE_THRESHOLD=$(LEAK_GOROUTINE_THRESHOLD) LEAK_THREAD_THRESHOLD=$(LEAK_THREAD_THRESHOLD) \
	VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL \
	go test -v -count=1 -timeout 60m -run TestMemLeakGET$$ ./test/live/runner/...

# Convenience targets for running specific test types
# When OPS_URL and SERVICE_URL are set, uses external services
# Otherwise, starts local services using available configs
test-live-embeddings:
	@echo "Running embeddings tests..."
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "^TestLive$$/.+/.+/Embeddings" ./test/live/runner/...

test-live-chat:
	@echo "Running chat completion tests (non-streaming)..."
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "^TestLive$$/.+/.+/ChatCompletion/should_complete" ./test/live/runner/...

test-live-streaming:
	@echo "Running streaming chat completion tests..."
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "^TestLive$$/.+/.+/ChatCompletionStreaming" ./test/live/runner/...

test-live-image:
	@echo "Running image generation tests..."
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=$(PROMPT) MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "^TestLive$$/.+/.+/ImageGeneration" ./test/live/runner/...

test-live-image-text:
	@echo "Running image generation with text output tests..."
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CONFIG=$(CONFIG) PROMPT=image-generation-with-text MODEL=$(MODEL) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "^TestLive$$/.+/.+/ImageGeneration" ./test/live/runner/...

test-live-webrtc-text:
	@echo "Running WebRTC realtime text tests..."
	@command -v pkg-config >/dev/null 2>&1 || { echo "ERROR: pkg-config not found. Install with: brew install pkgconf (macOS) or apt install pkg-config (Linux)"; exit 1; }
	@pkg-config --exists opus 2>/dev/null || { echo "ERROR: libopus not found. Install with: brew install opus (macOS) or apt install libopus-dev (Linux)"; exit 1; }
	@pkg-config --exists opusfile 2>/dev/null || { echo "ERROR: libopusfile not found. Install with: brew install opusfile (macOS) or apt install libopusfile-dev (Linux)"; exit 1; }
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CGO_ENABLED=1 CONFIG=$(CONFIG) MODEL=$(MODEL) PROMPT=$(PROMPT) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "TestLiveWebRTC$$" ./test/live/runner/...

test-live-webrtc:
	@echo "Running WebRTC realtime audio tests..."
	@echo "Checking build dependencies for Opus encoding (CGO)..."
	@command -v pkg-config >/dev/null 2>&1 || { echo "ERROR: pkg-config not found. Install with: brew install pkgconf (macOS) or apt install pkg-config (Linux)"; exit 1; }
	@pkg-config --exists opus 2>/dev/null || { echo "ERROR: libopus not found. Install with: brew install opus (macOS) or apt install libopus-dev (Linux)"; exit 1; }
	@pkg-config --exists opusfile 2>/dev/null || { echo "ERROR: libopusfile not found. Install with: brew install opusfile (macOS) or apt install libopusfile-dev (Linux)"; exit 1; }
	@LOG_OUTPUT_VAL="false"; \
	LOG_LEVEL_VAL="critical"; \
	VERBOSE_RUNNER_VAL="false"; \
	if [ "$(VERBOSE)" = "1" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_LEVEL_VAL="info"; \
	elif [ "$(VERBOSE)" = "2" ]; then \
		VERBOSE_RUNNER_VAL="true"; \
		LOG_OUTPUT_VAL="true"; \
		LOG_LEVEL_VAL="debug"; \
	fi; \
	CGO_ENABLED=1 CONFIG=$(CONFIG) MODEL=$(MODEL) PROMPT=$(PROMPT) OPS_URL=$(OPS_URL) SERVICE_URL=$(SERVICE_URL) VERBOSE_RUNNER=$$VERBOSE_RUNNER_VAL LOG_OUTPUT=$$LOG_OUTPUT_VAL LOG_LEVEL=$$LOG_LEVEL_VAL go test -v -count=1 -run "TestLiveWebRTCAudio" ./test/live/runner/...
