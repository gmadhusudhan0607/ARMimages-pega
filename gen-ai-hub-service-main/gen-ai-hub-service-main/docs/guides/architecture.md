# Architecture Overview

## Two-Service Architecture

This repository contains **two main services** that work together:

### 1. GenAI Hub Service (`cmd/service/main.go`) - Port 8080

- Main API gateway for LLM requests
- Handles OpenAI, Bedrock, and Vertex AI endpoints
- Routes requests to appropriate LLM providers
- Manages model mappings, metrics, and monitoring

### 2. GenAI Gateway Ops (`cmd/ops/main.go`) - Port 8081

- Operations and management service
- Syncs model mappings from AWS GenAI Infrastructure
- Provides `/v1/mappings` endpoint for dynamic configuration
- Exposes isolation-specific metrics
- Runs scheduled tasks (mapping synchronizer)

Both services expose health endpoints on port 8082:

- `/health/liveness` - Always returns healthy
- `/health/readiness` - Checks dependencies (mappings, etc.)

## Key Components

### Model Registry (`internal/models/`)

- **Registry**: Central model registry with versioning and capabilities
- **Loader**: Loads model definitions from YAML files
- **Types**: Model, Provider, FunctionalCapability, Endpoint definitions
- **Validation**: Validates model configurations and naming conventions

### Request Processing (`internal/request/`)

- **Processors**: Per-provider request/response processors (Bedrock, Vertex, OpenAI)
- **Middleware**: API version validation, model resolution, metadata injection
- **Metrics**: Prometheus metrics collection and reporting
- **Resolvers**: Target endpoint resolution (static config vs dynamic mapping)

### Infrastructure Integration (`internal/infra/`)

- **Client**: AWS Bedrock API client for Converse API
- **Mapping**: GenAI Infrastructure model mapping sync (credentials, secrets)
- Supports two modes:
  - External Secrets Operator (ESO) injection
  - Auto-mapping via Ops service synchronizer

### Middleware (`internal/middleware/`)

- **UAS Validator**: User Authorization Service validation
- **SAX Enrichment**: SAX token injection for Launchpad deployments
- **Provider Validation**: Ensures enabled providers (Azure/Bedrock/Vertex)
- **HTTP Metrics**: Request/response logging and metrics

## Configuration System

The service uses **multi-source configuration**:

### 1. Static Mapping File (`CONFIGURATION_FILE` env var)

- YAML file with model endpoints and credentials
- Generated from Helm templates for integration tests
- Located at `distribution/genai-hub-service-helm/src/main/helm/configuration/`

### 2. Dynamic Mappings (via Ops service)

- `MAPPING_ENDPOINT`: Ops service `/v1/mappings` endpoint
- `MODELS_DEFAULTS_ENDPOINT`: Ops service `/v1/models/defaults` endpoint
- Refreshed periodically by mapping synchronizer

### 3. Private Model Configs (`PRIVATE_MODEL_CONFIG_DIR`)

- Directory with per-model configuration files
- Generated from `genai-private-model-config-terraform` templates

## Distribution Structure

**IMPORTANT**: Always consult `SCE_TO_PRODUCT_MAPPING.md` before making changes to understand:

- Which product a component belongs to (GenAIGatewayServiceProduct, GenAIInfrastructure, etc.)
- Whether changes affect runtime code (service subcomponents) or infrastructure (SCE/Terraform/Helm)
- The resource type (backing-services vs controlplane-services) which determines upgrade order

**When adding new components**: Update `SCE_TO_PRODUCT_MAPPING.md` with the new component's product mapping and subcomponents.

The `distribution/` directory contains Service Catalog Entries (SCEs) organized by component:

- **-sce**: Service Catalog Entry definitions
- **-terraform**: Terraform infrastructure modules
- **-helm**: Helm charts for Kubernetes deployment
- **-docker**: Docker image build definitions
- **-product-catalog**: The definition of high level products that organize SCEs as single deployments

Key product mappings (see mapping file for complete details):

- **GenAIGatewayServiceProduct** (backing-service): genai-hub-service, role
- **GenAIInfrastructure** (controlplane-service): AWS Bedrock infra, defaults, SAX OIDC
- **GenAIInfrastructureGCP** (controlplane-services): GCP Vertex host/infra
- **GenAIPrivateModels** (backing-service): private model configs and external secrets

## Test Organization

- **Unit tests**: `*_test.go` files alongside source code
- **Integration tests**: `test/integration/` - Docker Compose-based tests
- **Request processing tests**: `test/integration/request-processing/*/` - Per-scenario test suites
- **Live tests**: `test/live/` - Tests against real services with real LLM backends
- **Load tests**: `test/load/` - Performance and load testing

For test commands, see `building_and_testing.md`.
