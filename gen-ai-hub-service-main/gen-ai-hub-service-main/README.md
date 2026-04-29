# GenAI Hub Service

[![Quality Gate Status](https://sonar.pega.com/api/project_badges/measure?project=PCLD%3Agen-ai-hub-service&metric=alert_status)](https://sonar.pega.com/dashboard?id=PCLD%3Agen-ai-hub-service)
[![Lines of Code](https://sonar.pega.com/api/project_badges/measure?project=PCLD%3Agen-ai-hub-service&metric=ncloc)](https://sonar.pega.com/dashboard?id=PCLD%3Agen-ai-hub-service)
[![Coverage](https://sonar.pega.com/api/project_badges/measure?project=PCLD%3Agen-ai-hub-service&metric=coverage)](https://sonar.pega.com/dashboard?id=PCLD%3Agen-ai-hub-service)
[![Security Rating](https://sonar.pega.com/api/project_badges/measure?project=PCLD%3Agen-ai-hub-service&metric=security_rating)](https://sonar.pega.com/dashboard?id=PCLD%3Agen-ai-hub-service)

The **GenAI Hub Service** is the central API gateway through which Pega Infinity instances and other Pega services connect to Large Language Models (LLMs). It abstracts provider-specific APIs behind a unified interface and handles routing, authentication, model mapping, metrics, and observability for all LLM traffic.

* [Agile Studio Information](#agile-studio-information)
* [Architecture Overview](#architecture-overview)
* [Supported LLM Providers](#supported-llm-providers)
* [Repository Structure](#repository-structure)
* [Configuration](#configuration)
* [Building the Code](#building-the-code)
* [Testing](#testing)
* [Local Development](#local-development)
* [Model Management](#model-management)
* [Git Usage](#git-usage)
* [Build & Release Notifications](#build--release-notifications)
* [Code Quality](#code-quality)

------------------------

## Agile Studio Information

Release Record: null
Product Record: PRD-7655
Backlog Record: BL-11359
Squad Record: SQUAD-326
Project Record: PROJ-10955

------------------------

## Architecture Overview

This repository contains **two cooperating services** deployed as a pair in Kubernetes:

### GenAI Hub Service (`cmd/service/main.go`) — Port 8080

The main API gateway for all LLM requests. It:

- Exposes OpenAI-compatible endpoints (`/openai/deployments/:modelId/...`) for Azure OpenAI and other providers
- Exposes AWS Bedrock endpoints (`/anthropic/`, `/amazon/`, `/meta/`, `/mistral/`) using the Bedrock Converse API
- Exposes GCP Vertex AI endpoints (`/google/deployments/:modelId/...`) for Gemini and Imagen models
- Resolves model mappings from a configuration file, dynamic mappings from the Ops service, or private model configs
- Enforces authentication via UAS (User Authorization Service) and SAX (Service Access Framework) token enrichment
- Collects Prometheus metrics and OpenTelemetry traces per request

### GenAI Gateway Ops (`cmd/ops/main.go`) — Port 8081

The operations sidecar service. It:

- Synchronizes AWS GenAI Infrastructure model mappings on a configurable schedule (default: every 5 minutes)
- Exposes `/v1/mappings` for the Hub Service to fetch current model-to-endpoint mappings
- Exposes `/v1/models/defaults` for default model configuration
- Exposes `/v1/isolations/:isolationId/metrics` for per-isolation usage metrics
- Accepts monitoring events via `POST /v1/events`

Both services expose health and metrics endpoints on **port 8082**:

- `GET /health/liveness` — always returns healthy
- `GET /health/readiness` — checks that required mappings are loaded
- `GET /metrics` — Prometheus metrics
- `GET /debug/pprof/*` — Go pprof profiling endpoints (Hub Service only)

For a detailed architecture description see [`docs/guides/architecture.md`](docs/guides/architecture.md).

------------------------

## Supported LLM Providers

| Provider | Route Prefix | Models |
|---|---|---|
| **Azure OpenAI** | `/openai/deployments/:modelId/` | GPT-4o, GPT-4, DALL-E, Whisper (real-time), embeddings, and any custom Azure deployment |
| **AWS Bedrock — Anthropic** | `/anthropic/deployments/:modelId/` | Claude 3 (Haiku, Sonnet, Opus) and Claude 3.5 family |
| **AWS Bedrock — Amazon** | `/amazon/deployments/:modelId/` | Amazon Nova (Pro, Lite, Micro) and Titan Embeddings |
| **AWS Bedrock — Meta** | `/meta/deployments/:modelId/` | Llama 3 family |
| **AWS Bedrock — Mistral** | `/mistral/deployments/:modelId/` | Mistral and Mixtral models |
| **GCP Vertex AI** | `/google/deployments/:modelId/` | Gemini 1.5 / 2.0 / 2.5 (chat, embeddings, generateContent), Imagen (image generation) |

All Bedrock calls use the AWS Bedrock **Converse API** for a unified request/response format.
Vertex AI requests are forwarded to a GCP Cloud Function that handles OpenAI SDK compatibility.

------------------------

## Repository Structure

```
cmd/
  service/          # GenAI Hub Service binary (port 8080)
    api/            # Request handlers: bedrock, buddies, models, realtime, vertex
    health/         # Liveness and readiness endpoints
    otel/           # OpenTelemetry tracer setup
  ops/              # GenAI Gateway Ops binary (port 8081)
    api/            # Ops handlers: mappings, defaults, metrics, events
internal/
  aws/              # AWS SDK utilities
  cntx/             # Service context and logger helpers
  ginctx/           # Gin context utilities
  helpers/          # Common helper functions and env-var access
  infra/            # Model mapping sync, AWS Bedrock client, credentials
  middleware/       # UAS auth, SAX enrichment, HTTP metrics, provider guard
  models/           # Model registry: config, loader, specs, types, validation
  monitoring/       # Monitoring event client and request reporter
  proxy/            # Generic HTTP proxy client
  repository/       # Monitoring event repository
  request/          # Request pipeline: processors, middleware, resolvers, JSON, cache
  saxclient/        # SAX (Service Access Framework) authentication client
  testutils/        # Shared test utilities
pkg/
  heimdallgzip/     # Heimdall HTTP client with gzip support
apidocs/            # OpenAPI spec (spec.yaml) and Swagger UI static assets
test/
  integration/      # Docker Compose-based integration tests (Ginkgo/Gomega)
  live/             # End-to-end tests against real LLM providers
  load/             # Performance and load tests
distribution/       # Infrastructure-as-Code: Docker, Helm, Terraform, SCE definitions
docs/
  guides/           # architecture.md, building_and_testing.md, code_conventions.md, infrastructure_coordination.md
  adr/              # Architecture Decision Records
```

See [`SCE_TO_PRODUCT_MAPPING.md`](SCE_TO_PRODUCT_MAPPING.md) for the mapping of `distribution/` components to Pega products.

------------------------

## Configuration

The Hub Service reads configuration from three sources:

| Source | Environment Variable | Description |
|---|---|---|
| Static mapping file | `CONFIGURATION_FILE` | YAML file with model endpoints and credentials; generated from Helm templates |
| Dynamic mappings | `MAPPING_ENDPOINT` | URL of the Ops service `/v1/mappings` endpoint; auto-refreshed when `USE_AUTO_MAPPING=true` |
| Default models | `MODELS_DEFAULTS_ENDPOINT` | URL of the Ops service `/v1/models/defaults` endpoint |
| Private model configs | `PRIVATE_MODEL_CONFIG_DIR` | Directory of per-model YAML configs (default: `/private-model-config`) |

Additional key environment variables:

| Variable | Description |
|---|---|
| `USE_AUTO_MAPPING` | `true` to use the mapping synchronizer; `false` to use ESO-injected mappings |
| `USE_GENAI_INFRA_MODELS` | Enables the AWS GenAI Infrastructure model routing path |
| `USE_SAX` | Enables SAX token injection for Launchpad deployments |
| `SAX_CONFIG_PATH` | Path to the SAX client credentials file (JSON, mounted by ESO) |
| `GENAI_INFRA_MAPPING_REFRESH_INTERVAL` | Mapping sync interval for the Ops service (default: `5m`) |
| `SERVICE_PORT` | Hub Service API port (default: `8080`) |
| `OPS_PORT` | Ops Service port (default: `8081`) |
| `SERVICE_HEALTHCHECK_PORT` | Health/metrics port (default: `8082`) |

For local development, run `make generateMappingFiles` to generate a local configuration file from the Helm templates.

------------------------

## Building the Code

Requires **Go 1.24** or later (see `go.mod` for the exact version).

```sh
make build          # fmt + vet + lint + staticcheck + compile both binaries to bin/
make fmt            # Format code with go fmt
make vet            # Run go vet
make lint           # Run golangci-lint
make staticcheck    # Run staticcheck
make goimports      # Format imports with goimports
```

------------------------

## Testing

### Unit Tests

```sh
make test           # Run all unit tests with race detection and coverage report
```

The CI pipeline enforces **≥ 80% code coverage on new code**.

### Integration Tests

Integration tests use Docker Compose to start the service and a mock LLM server (`genai-hub-mockserver`).

```sh
# Using Make
make integration-test-up    # Start containers
make integration-test-run   # Run Ginkgo integration test suite
make integration-test-down  # Stop and remove containers

# Or using Gradle (runs up, test, and down automatically)
./gradlew integrationTest
```

See [`test/integration/README.md`](test/integration/README.md) for full details.

### Request-Processing Tests

Per-scenario tests that run the service in an isolated configuration:

```sh
make test-request-processing                    # Run all request-processing scenarios
make test-request-processing TEST=<name>        # Run a specific scenario
make test-request-processing TEST=<name> KEEP=true  # Keep service running for debugging
```

### Live Tests

End-to-end tests against real LLM provider endpoints (require credentials):

```sh
make test-live RUN=list                         # List all test cases
make test-live RUN=all                          # Run all configs × all prompts
make test-live CONFIG=<config>                  # Run a specific config
make test-live PROMPT=<prompt>                  # Run a specific prompt against all configs
make test-live-chat                             # Chat completion tests only
make test-live-embeddings                       # Embedding tests only
make test-live-streaming                        # Streaming tests only
make test-live-memleak CONFIG=<config>          # Memory leak detection
```

See [`test/live/README.md`](test/live/README.md) for setup instructions.

------------------------

## Local Development

```sh
make run              # Start the service with local config
make run_pretty       # Start the service with formatted JSON logs (requires jl — install with: brew install jl or go install github.com/koenbollen/jl@latest)

make generateMappingFiles              # Generate config files from Helm templates
make generatePrivateModelConfigFiles   # Generate private model config files
```

The Swagger UI is available at `http://localhost:8080/` when the service is running.

------------------------

## Model Management

### Adding AWS Bedrock Models

```sh
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1
make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json  # Batch mode
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true
```

### Adding GCP Vertex AI Models

```sh
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion PREVIEW=true
make add-vertex-model CONFIG=scripts/vertex-model-sample.json  # Batch mode
```

Supported `MODEL_TYPE` values: `chat_completion`, `embedding`, `image`.

For the complete end-to-end model addition workflow including infrastructure changes, see [`docs/guides/infrastructure_coordination.md`](docs/guides/infrastructure_coordination.md).

------------------------

## Git Usage

### Branches

| Name | Description |
| --- | --- |
| `main` | Ongoing development — produces `[release]-dev-[build #]` artifacts |
| `release/#.#.#` | Release hardening and patches |
| `feature/[Story ID]-[description]` | Feature work branched from `main` |
| `bugfix/[Bug ID]-[description]` | Bug fixes branched from `main` |
| `bugfix/release-#.#.#/[Bug ID]-[description]` | Bug fixes against a release branch |
| `poc/[description]` | Proof-of-concept work (not merged) |

### Committing

Commit messages must include the Agile Studio work item ID as the prefix, e.g. `US-12345: Add support for new Bedrock model`.

### Pull Requests

All merges to `main` or `release/*` branches require a pull request with:

- At least one approved reviewer
- A passing CI build
- A work item ID in the PR title that is in an appropriate status

### CODEOWNERS

Default reviewers for repository sections are configured in [`CODEOWNERS`](CODEOWNERS).

------------------------

## Build & Release Notifications

In [`pipeline/configuration.properties`](pipeline/configuration.properties) you can configure build and release notifications:

| Property | Description |
|---|---|
| `RELEASE_ANNOUNCEMENT_WEBEX_SPACE_NOTIFICATION_LIST` | Comma-separated Webex space IDs for release announcements |
| `RELEASE_ANNOUNCEMENT_EMAIL_NOTIFICATION_LIST` | Comma-separated email addresses for release announcements |
| `BUILD_STATUS_WEBEX_SPACE_NOTIFICATION_LIST` | Comma-separated Webex space IDs for build status updates |
| `BUILD_STATUS_EMAIL_NOTIFICATION_LIST` | Comma-separated email addresses for build status updates |

------------------------

## Code Quality

| Tool | Command | Description |
|---|---|---|
| golangci-lint | `make lint` | Comprehensive Go linter suite |
| staticcheck | `make staticcheck` | Advanced static analysis |
| SonarQube | CI pipeline | Code coverage, security, and quality gate enforcement |
| Veracode | CI pipeline | Security vulnerability scanning |

See [`docs/guides/code_conventions.md`](docs/guides/code_conventions.md) for coding standards and conventions.

Swagger UI: [GitHub Pages](https://pega-cloudengineering.github.io/gen-ai-hub-service/)
