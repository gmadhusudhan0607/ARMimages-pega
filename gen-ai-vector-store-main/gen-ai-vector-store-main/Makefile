# Cline plugin in Intellij for some reason does not import PATHs correctly when running commands in shell.
# It can not find `go` properly.  So that why we need workaround here to search `go`
# This problem does not exist in VisualCode , only in Intellij

# Try to find Go installation
GO := $(shell which go 2>/dev/null)

# If 'go' not found in PATH, search common locations based on OS
ifeq ($(GO),)
    # Detect OS first for path selection
    DETECTED_OS := $(shell uname -s)

    # Common Go installation paths for both Linux and macOS
    COMMON_GO_PATHS := /usr/local/go/bin/go /usr/local/bin/go $(HOME)/go/bin/go

    # OS-specific paths
    ifeq ($(DETECTED_OS),Darwin)
        # macOS specific paths (Homebrew, MacPorts, etc.)
        OS_SPECIFIC_PATHS := /opt/homebrew/bin/go /opt/homebrew/opt/go/bin/go /usr/local/opt/go/bin/go /Library/Go/bin/go
    else
        # Linux specific paths
        OS_SPECIFIC_PATHS := /usr/lib/go/bin/go /usr/lib/golang/bin/go /opt/go/bin/go /snap/bin/go
    endif

    # Combine all paths
    POSSIBLE_GO_PATHS := $(COMMON_GO_PATHS) $(OS_SPECIFIC_PATHS)

    # Find first existing Go binary
    GO := $(firstword $(wildcard $(POSSIBLE_GO_PATHS)))
endif

# If still not found, check GOROOT environment variable
ifeq ($(GO),)
    ifneq ($(GOROOT),)
        GO := $(GOROOT)/bin/go
    endif
endif

# If still not found, provide error message
ifeq ($(GO),)
    $(error Go not found in PATH or common locations. Please install Go and ensure it is available in PATH or a standard location)
else
    # Add Go's bin directory to PATH for this make session
    export PATH := $(dir $(GO)):$(PATH)
endif

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

OS_NAME ?= $(shell uname -s)
OS_ARCH ?= $(shell uname -m)

# Use locally installed docker-compose if found
ifeq (,$(shell which docker-compose))
SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION=false
DOCKER_COMPOSE_TOOL=./bin/docker-compose
else
SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION=true
DOCKER_COMPOSE_TOOL=$(shell which docker-compose)
endif

DOCKER_COMPOSE_VER = v2.24.6
DOCKER_COMPOSE_URI = https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VER}/docker-compose-${OS_NAME}-${OS_ARCH}

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

ENV_FILE_EXISTS=$(shell test -e '.env' && echo -n yes)

# Exclude non-unit tests from "go tests"
SRC_MODULES_DIR = $(shell $(GO) list ./... | grep -vE "/src/integTest/")

DOCKER_ARTIFACTORY_URL = $(shell cat gradle.properties  | grep -E '^\s*dockerRegistryUrl\s*=' | awk -F = '{print $$2}' | tr -d ' ' | awk -F '//' '{print $$2}' )
DOCKER_REPO_PREFIX = $(shell cat gradle.properties  | grep -E '^\s*dockerRepoPrefix\s*=' | awk -F = '{print $$2}' | tr -d ' ' )
DOCKER_DEV_REPO = $(shell cat gradle.properties  | grep -E '^\s*dockerDevRegistry\s*=' | awk -F = '{print $$2}' | tr -d ' ' )

.EXPORT_ALL_VARIABLES:
# Image URL to use all running integration tests
SERVICE_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-vector-store:local"
OPS_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-vector-store-ops:local"
BACKGROUND_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-vector-store-background:local"
SERVICE_RO_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-vector-store:local"
OPS_RO_DOCKER_IMAGE ?= "$(DOCKER_ARTIFACTORY_URL)/$(DOCKER_REPO_PREFIX)/genai-vector-store-ops:local"

REPORT_SECTION_NAME_UNIT_TESTS = "GenAI Vector Store Unit Tests"
REPORT_SECTION_NAME_INTEG_TESTS = "GenAI Vector Store Integration Tests"

.PHONY: all
all: build

##@ Environment
define setup_env
	$(eval ENV_FILE := $(1))
	@echo " - setup environment from $(ENV_FILE)"
	$(eval include $(1))
	$(eval export)
endef

localAwsTestEnv: ## load env/localaws--test.env environment
	$(call setup_env, env/local-aws-test.env )
# if .env exist load it as well (will replace previously loaded variables
ifeq ($(ENV_FILE_EXISTS), yes)
	$(call setup_env, .env )
endif

localTestEnv: ## load env/local-test.env environment
	$(call setup_env, env/local-test.env )
# if .env exist load it as well (will replace previously loaded variables from env/local-test.env)
# It allows to use of vars from env/local-test.env but redefine any of them locally  without worry about committing them
ifeq ($(ENV_FILE_EXISTS), yes)
	$(call setup_env, .env )
endif

integrationTestEnv: ## load env/integration-test.env environment vars
	$(call setup_env, env/integration-test.env )
ifeq ($(ENV_FILE_EXISTS), yes)
	$(call setup_env, .env )
endif

integrationTestLocallyEnv: ## load env/integration-test.env environment vars
	$(call setup_env, env/integration-test-locally.env )
ifeq ($(ENV_FILE_EXISTS), yes)
	$(call setup_env, .env )
endif

##@ Development
.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	$(GO) vet ./...

.PHONY: mockery
mockery:
	$(GO) tool mockery

.PHONY: gopogh
gopogh: ## Without this unit-test-report throws warning

.PHONY: lint
lint:
	$(GO) tool golangci-lint run

.PHONY: test
test: localTestEnv mockery fmt vet lint ## Run tests.
	mkdir -p ./build/test-results/
	CGO_ENABLED=1 $(GO) tool gotestsum --format standard-verbose --jsonfile ./build/test-results/unitTestReport.json -- -failfast -race -coverprofile=coverage.out -v $(SRC_MODULES_DIR)

.PHONY: unit-test-report
unit-test-report: gopogh
	$(GO) tool gopogh -in ./build/test-results/unitTestReport.json -out_html ./build/test-results/unitTestReport.html -name $(REPORT_SECTION_NAME_UNIT_TESTS)

.PHONY: integration-test-report
	$(GO) tool gopogh -in ./build/test-results/integTestReport.json -out_html ./build/test-results/integTestReport.html -name $(REPORT_SECTION_NAME_INTEG_TESTS)

.PHONY: cover
cover: mockery fmt vet test ## Run tests and show coverage using go tool cover.
	$(GO) tool cover -html=coverage.out

##@ Build

.PHONY: build
build: fmt vet lint staticcheck ## Build service binary.
	$(GO) build -o bin/ ./cmd/service/...
	$(GO) build -o bin/ ./cmd/ops/...

.PHONY: run-service
run-service: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/service/main.go

.PHONY: run-service-local-aws
run-service-local-aws: localAwsTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/service/main.go

.PHONY: run-ops
run-ops: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/ops/main.go

.PHONY: run-background
run-background: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/background/main.go

.PHONY: run-service_pretty
run-service_pretty: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/service/main.go 2>&1 | jq -c "del(.ts, .caller, .logger, .latency, .\"user-agent\", .ip, .time) "

.PHONY: run-ops_pretty
run-ops_pretty: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/ops/main.go 2>&1 | jq -c '(.ts|=if . then (. | strftime("%Y-%m-%d %H:%M:%S")) else null end) | del(.caller, .logger, .latency, ."user-agent", .ip, .time)'

.PHONY: run-background_pretty
run-background_pretty: localTestEnv fmt vet ## Run a service from your host.
	$(GO) run cmd/background/main.go 2>&1 | jq -c "del(.ts, .caller, .logger, .latency, .\"user-agent\", .ip, .time) "


.PHONY: clean
clean:
	rm -fr  distribution/go/build/
	rm -fr  distribution/helm/build
	rm -fr  distribution/sce/build
	rm -fr  distribution/docker/build
	rm -fr  distribution/ops-docker/build
	rm -fr  distribution/isolation-terraform/src/main/terraform/aws/terraform.*
	rm -fr  distribution/isolation-terraform/src/main/terraform/aws/.terraform*

##@ Deployment
.PHONY: postgres-up
postgres-up: ## Run local postgres for development.
	docker-compose -f test/postgres/docker-compose.yml up -d

.PHONY: postgres-down
postgres-down: ## Shut down local postgres.
	docker-compose -f test/postgres/docker-compose.yml down

.PHONY: terraform-setup
terraform-setup:
	./gradlew ":distribution:isolation-terraform:terraformInit"
	mkdir -p distribution/isolation-terraform/src/main/terraform/aws/terraform.d/plugins
	mkdir -p distribution/isolation-terraform/src/main/terraform/aws/.terraform/providers/pega.com
	cp -fr distribution/isolation-terraform/build/terraformRun/aws/terraform.d/plugins/pega.com  distribution/isolation-terraform/src/main/terraform/aws/terraform.d/plugins/pega.com
	cp -fr distribution/isolation-terraform/build/terraformRun/aws/.terraform/providers/pega.com distribution/isolation-terraform/src/main/terraform/aws/.terraform/providers/pega.com
	mkdir -p distribution/isolation-terraform/src/main/terraform/aws/.terraform/providers/pega.com
	cd distribution/isolation-terraform/src/main/terraform/aws/; terraform init
	echo "Local setup COMPLETED"
	echo "run: "
	echo "   cd ./distribution/isolation-terraform/src/main/terraform/aws/"
	echo "   terraform apply -var-file='testing.tfvars' -auto-approve"
	echo ""
	echo "To cleanup run:"
	echo "  make terraform-clean"

.PHONY: terraform-clean
terraform-clean:
	rm -fr  distribution/isolation-terraform/src/main/terraform/aws/terraform.*
	rm -fr  distribution/isolation-terraform/src/main/terraform/aws/.terraform*

.PHONY: docker-compose-tool
docker-compose-tool:
	mkdir -p $(LOCALBIN)
	$(SKIP_DOCKER_COMPOSE_TOOL_INSTALLATION) || test -s $(LOCALBIN)/docker-compose || ( curl -SL "${DOCKER_COMPOSE_URI}" -o $(LOCALBIN)/docker-compose && chmod a+x $(LOCALBIN)/docker-compose )

.PHONY: integration-test-up
integration-test-up: docker-compose-tool integrationTestEnv
   	ifeq (${CI}, true)
		docker login ${DOCKER_DEV_REPO} -u ${ORG_GRADLE_PROJECT_dockerRegistryUser} -p ${ORG_GRADLE_PROJECT_dockerRegistryPassword}
    endif
	SERVICE_DOCKER_IMAGE=${SERVICE_DOCKER_IMAGE} OPS_DOCKER_IMAGE=${OPS_DOCKER_IMAGE} BACKGROUND_DOCKER_IMAGE=${BACKGROUND_DOCKER_IMAGE} SERVICE_RO_DOCKER_IMAGE=${SERVICE_RO_DOCKER_IMAGE} OPS_RO_DOCKER_IMAGE=${OPS_RO_DOCKER_IMAGE} $(DOCKER_COMPOSE_TOOL) --profile app -f ./src/integTest/docker-compose.yaml  up -d --force-recreate -V 
	sleep 4
	docker ps | grep -E "vector-store|pgvector|vs-ada-mockserver"
	echo "-----------------------------------------"
	docker container logs vector-store-background
	echo "-----------------------------------------"
	docker container logs vector-store-ops
	echo "-----------------------------------------"
	docker container logs vector-store
	echo "-----------------------------------------"

.PHONY: integration-test-print-docker-logs
integration-test-print-docker-logs:
   	ifeq (${CI}, true)
		echo "-----------------------------------------"
		docker container logs vector-store-background
		echo "-----------------------------------------"
		docker container logs vector-store-ops
		echo "-----------------------------------------"
		docker container logs vector-store
		echo "-----------------------------------------"
    endif

.PHONY: integration-test-down
integration-test-down: docker-compose-tool
ifeq ($(CI),"true")
	echo "-----------------------------------------"
	docker container logs vector-store
	echo "-----------------------------------------"
endif
	$(DOCKER_COMPOSE_TOOL) --profile app -f ./src/integTest/docker-compose.yaml down 

.PHONY: essential-local-infra-up
essential-local-infra-up: docker-compose-tool integrationTestEnv
	$(DOCKER_COMPOSE_TOOL) -f ./src/integTest/docker-compose.yaml up -d --force-recreate -V

.PHONY: essential-local-infra-down
essential-local-infra-down: docker-compose-tool integrationTestEnv
	$(DOCKER_COMPOSE_TOOL) -f ./src/integTest/docker-compose.yaml down

.PHONY: integration-test-run-locally
integration-test-run-locally: integrationTestLocallyEnv
	mkdir -p ./build/test-results/integrationTest/
	$(GO) clean -cache
	$(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/...
	#$(GO) tool ginkgo -v ./src/integTest/service/readonly_mode/...
	#$(GO) tool ginkgo -v /src/integTest/ops/readonly_mode/

.PHONY: integration-test-run-locally_readonly_mode
integration-test-run-locally_readonly_mode: integrationTestLocallyEnv
	mkdir -p ./build/test-results/integrationTest/
	$(GO) clean -cache
	$(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/ops/readonly_mode/...  ./src/integTest/service/readonly_mode/...

.PHONY: integration-test-run-locally_emulation_mode
integration-test-run-locally_emulation_mode: integrationTestLocallyEnv
	mkdir -p ./build/test-results/integrationTest/
	$(GO) clean -cache
	$(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/ops/emulation_mode/...  ./src/integTest/service/emulation_mode/...

.PHONY: integration-test-run-request_timeout
integration-test-run-request_timeout: integrationTestEnv
	mkdir -p ./build/test-results/integrationTest/
	go clean -cache
	go tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/service/request_timeout/...

.PHONY: integration-test-run-locally_headers
integration-test-run-locally_headers: integrationTestLocallyEnv
	mkdir -p ./build/test-results/integrationTest/
	$(GO) clean -cache
	$(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/service/headers/...


.PHONY: integration-test-run
integration-test-run: integrationTestEnv
	mkdir -p ./build/test-results/integrationTest/
	$(GO) clean -cache
	$(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/...
	# TODO:BUG-858913 investigate why it's not working
	# ginkgo --json-report=build/test-results/integTestReport.json -v ./src/integTest/...
    ifeq (${CI}, true)
		echo "-----------------------------------------"
		docker container logs vector-store-background
		echo "-----------------------------------------"
		docker container logs vector-store-ops
		echo "-----------------------------------------"
		docker container logs vector-store
		echo "-----------------------------------------"
    endif

.PHONY: integration-run-service
integration-run-service: integrationTestEnv fmt vet ## Run a service from your host.
	SERVICE_PORT=28080 SERVICE_HEALTHCHECK_PORT=28082 $(GO) run cmd/service/main.go

.PHONY: integration-run-service_pretty
integration-run-service_pretty: integrationTestEnv fmt vet ## Run a service from your host.
	SERVICE_PORT=28080 SERVICE_HEALTHCHECK_PORT=28082 $(GO) run cmd/service/main.go 2>&1 | jq -c '(.ts|=if . then (. | strflocaltime("%Y-%m-%d %H:%M:%S")) else null end) | del(.caller, .logger, .latency, ."user-agent", .ip, .time)'

.PHONY: integration-run-ops
integration-run-ops: integrationTestEnv fmt vet ## Run a service from your host.
	OPS_PORT=28090 OPS_HEALTHCHECK_PORT=28092 $(GO) run cmd/ops/main.go

.PHONY: integration-run-ops_pretty
integration-run-ops_pretty: integrationTestEnv fmt vet ## Run a service from your host.
	OPS_PORT=28090 OPS_HEALTHCHECK_PORT=28092 $(GO) run cmd/ops/main.go  2>&1 | jq -c '(.ts|=if . then (. | strflocaltime("%Y-%m-%d %H:%M:%S")) else null end) | del(.caller, .logger, .latency, ."user-agent", .ip, .time)'

.PHONY: integration-run-background
integration-run-background: integrationTestEnv fmt vet ## Run a service from your host.
	BKG_HEALTHCHECK_PORT=28072 $(GO) run cmd/background/main.go

.PHONY: integration-run-background_pretty
integration-run-background_pretty: integrationTestEnv fmt vet ## Run a service from your host.
	BKG_HEALTHCHECK_PORT=28072 $(GO) run cmd/background/main.go  2>&1 | jq -c '(.ts|=if . then (. | strflocaltime("%Y-%m-%d %H:%M:%S")) else null end) | del(.caller, .logger, .latency, ."user-agent", .ip, .time)'

##@ Background Integration Tests (Testcontainers)

.PHONY: integtest-build-binaries
integtest-build-binaries: ## Pre-build all test service binaries (background, service, ops)
	@echo "Pre-building test service binaries..."
	@mkdir -p ./bin
	@echo "Building background-test..."
	@$(GO) build -o bin/background-test ./cmd/background
	@echo "Building service-test..."
	@$(GO) build -o bin/service-test ./cmd/service
	@echo "Building ops-test..."
	@$(GO) build -o bin/ops-test ./cmd/ops
	@echo "All test binaries built successfully"

.PHONY: integtest-background
integtest-background: ## Run background integration tests with testcontainers (use KEEP=5m to keep containers, FOCUS='test1|test2' for test focus)
	@echo "Running background integration tests..."
	@mkdir -p ./build/test-results/integrationTest/
	@$(GO) clean -cache
	@KEEP=$(KEEP) $(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/background/...

.PHONY: integtest-timeout
integtest-timeout: ## Run throttling integration tests with testcontainers (use KEEP=5m to keep containers, FOCUS='test1|test2' for test focus)
	@echo "Running 'timeout' integration tests..."
	@mkdir -p ./build/test-results/integrationTest/
	@$(GO) clean -cache
	@KEEP=$(KEEP) $(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/service/timeout/...

.PHONY: integtest-pool-exhaustion
integtest-pool-exhaustion: ## Reproduce ck-290: DB pool exhaustion when embedding service is slow (use KEEP=5m to keep containers, FOCUS='pattern' for focus)
	@echo "Running 'pool-exhaustion' integration tests..."
	@mkdir -p ./build/test-results/integrationTest/
	@$(GO) clean -cache
	@KEEP=$(KEEP) $(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/service/pool_exhaustion/...

.PHONY: integtest-reembedding
integtest-reembedding: ## Run throttling integration tests with testcontainers (use KEEP=5m to keep containers, FOCUS='test1|test2' for test focus)
	@echo "Running 'reembedding' integration tests..."
	@mkdir -p ./build/test-results/integrationTest/
	@$(GO) clean -cache
	@KEEP=$(KEEP) $(GO) tool ginkgo -v $(if $(FOCUS),--focus='$(FOCUS)') ./src/integTest/service/reembedding/...

.PHONY: integtest-background-with-build
integtest-background-with-build: integtest-build-binaries integtest-background ## Pre-build binaries then run background integration tests

.PHONY: integtest-ps
integtest-ps: ## Show running test processes and containers
	@echo "==================== Test Processes ===================="
	@pgrep -a -f "bin/.*-test" || echo "No test processes found"
	@echo ""
	@echo "==================== Test Containers ===================="
	@docker ps --filter "label=test-container=genai-vector-store-test" --format "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}" || echo "No test containers found"
	@echo ""
	@echo "==================== Test Ports ===================="
	@echo "Checking ports used by background tests (18080, 18082, 18090, 18092, 18099)..."
	@lsof -i :18080 -i :18082 -i :18090 -i :18092 -i :18099 2>/dev/null || echo "No processes using test ports"

.PHONY: integtest-clean
integtest-clean: ## Clean test artifacts (binaries and logs only, does not stop running processes)
	@echo "Cleaning test artifacts..."
	@rm -f bin/background-test bin/service-test bin/ops-test bin/main-test
	@rm -f bin/background-test.log logs/service-test.log logs/ops-test.log logs/service-test.log
	@echo "Test artifacts cleaned"

.PHONY: integtest-clean-all
integtest-clean-all: ## Force cleanup of ALL test resources (processes, containers, and artifacts)
	@echo "========================================"
	@echo "  Force Cleanup: ALL Test Resources"
	@echo "========================================"
	@echo ""
	@echo "Step 1: Killing test processes..."
	@-pkill -TERM -f "bin/background-test" && echo "  ✓ Terminated background-test processes" || echo "  - No background-test processes"
	@-pkill -TERM -f "bin/service-test" && echo "  ✓ Terminated service-test processes" || echo "  - No service-test processes"
	@-pkill -TERM -f "bin/ops-test" && echo "  ✓ Terminated ops-test processes" || echo "  - No ops-test processes"
	@-pkill -TERM -f "bin/main-test" && echo "  ✓ Terminated main-test processes" || echo "  - No main-test processes"
	@sleep 2
	@echo ""
	@echo "Step 2: Force killing any remaining test processes..."
	@-pkill -KILL -f "bin/.*-test" && echo "  ✓ Force killed remaining processes" || echo "  - No remaining processes"
	@echo ""
	@echo "Step 3: Stopping test containers..."
	@-docker ps -q --filter "label=test-container=genai-vector-store-test" | xargs -r docker stop && echo "  ✓ Stopped test containers" || echo "  - No running test containers"
	@echo ""
	@echo "Step 4: Removing test containers..."
	@-docker ps -aq --filter "label=test-container=genai-vector-store-test" | xargs -r docker rm -f && echo "  ✓ Removed test containers" || echo "  - No test containers to remove"
	@echo ""
	@echo "Step 5: Cleaning test artifacts..."
	@$(MAKE) integtest-clean
	@echo ""
	@echo "========================================"
	@echo "  Cleanup Complete!"
	@echo "========================================"
	@echo ""
	@echo "Verifying cleanup..."
	@echo ""
	@$(MAKE) integtest-ps

.PHONY: integtest-clean-processes
integtest-clean-processes: ## Kill only test processes (leaves containers running)
	@echo "Killing test processes..."
	@-pkill -TERM -f "bin/background-test" && echo "  ✓ Terminated background-test" || true
	@-pkill -TERM -f "bin/service-test" && echo "  ✓ Terminated service-test" || true
	@-pkill -TERM -f "bin/ops-test" && echo "  ✓ Terminated ops-test" || true
	@-pkill -TERM -f "bin/main-test" && echo "  ✓ Terminated main-test" || true
	@sleep 2
	@-pkill -KILL -f "bin/.*-test" && echo "  ✓ Force killed remaining processes" || true
	@echo "Done"

.PHONY: integtest-clean-containers
integtest-clean-containers: ## Stop and remove only test containers (leaves processes running)
	@echo "Stopping and removing test containers..."
	@-docker ps -q --filter "label=test-container=genai-vector-store-test" | xargs -r docker stop
	@-docker ps -aq --filter "label=test-container=genai-vector-store-test" | xargs -r docker rm -f
	@echo "Done"

##@ Pact Consumer Contract Tests

PACT_DIR = internal/embedders/pact
PACT_OUTPUT_DIR = $(PACT_DIR)/pact

.PHONY: pact-test
pact-test: ## Run Pact consumer contract tests to generate contract files
	@echo "Running Pact consumer contract tests..."
	@mkdir -p $(PACT_OUTPUT_DIR)
	cd $(PACT_DIR) && $(GO) test -tags=pact -v ./...
	@echo "Pact contract files generated in $(PACT_OUTPUT_DIR)"

.PHONY: pact-clean
pact-clean: ## Clean Pact generated files
	@echo "Cleaning Pact artifacts..."
	@rm -rf $(PACT_OUTPUT_DIR)
	@rm -rf distribution/service-go/build/pacts
	@rm -rf src/contractTest/build/pacts
	@echo "Pact artifacts cleaned"

.PHONY: pact-verify
pact-verify: pact-test ## Run Pact tests and verify contracts were generated
	@echo "Verifying Pact contract files..."
	@ls -la $(PACT_OUTPUT_DIR)/*.json 2>/dev/null || (echo "ERROR: No contract files found" && exit 1)
	@echo "Contract files verified successfully"
