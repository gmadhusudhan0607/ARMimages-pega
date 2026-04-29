# Infrastructure Coordination

**CRITICAL**: This repository contains both runtime service code and infrastructure-as-code (IaC) components that must be coordinated carefully.

## Zero-Downtime Upgrade Constraints

**CRITICAL**: All changes MUST be forward and backward compatible. No rollbacks are allowed.

### Upgrade Sequence and Impact

Products upgrade in this order with a time gap: **GenAIInfrastructure** (control plane) upgrades FIRST, then **GenAIGatewayServiceProduct** (backing services) upgrades AFTER. During this period:

- Control plane is on version N+1 while backing services remain on version N
- GenAIInfrastructure serves ALL instances across regions (Americas, Europe, Asia)
- Breaking changes affect all connected instances region-wide
- Cached IAM roles and mappings must remain valid during transition

### Design Principles

When making infrastructure changes:

1. **Additive only** - Add new resources, keep old ones; remove in next major version
2. **Deprecation period** - Mark resources as deprecated before removal
3. **Graceful migration** - New code uses new resources, old code still works with old
4. **Test upgrade path** - Verify control plane upgrade first, then backing services

**Key Question**: "What happens when control plane is upgraded but backing services are still on the old version?" If the answer is "it breaks," redesign the change.

**Examples**: Don't remove IAM roles or change data formats in breaking ways. Instead, add new resources while keeping old ones, support both formats during transition, then remove old resources in the next major version.

## Runtime vs Infrastructure

### genai-hub-service (runtime)

Deployed as a microservice in Kubernetes:

- Go code in `cmd/service/`, `cmd/ops/`, `internal/`, `pkg/`
- Changes affect the running service behavior
- Docker images: `genai-hub-service-docker`, `genai-gateway-ops-docker`

### Infrastructure SCEs (IaC)

Cloud provisioning wrappers:

- Terraform modules (`*-terraform`) provision AWS/GCP resources
- Helm charts (`*-helm`) deploy Kubernetes resources
- SCE definitions (`*-sce`) define deployment parameters
- These set up required resources so the system works end-to-end

## Adding New Environment Variables

When the runtime code needs a new environment variable, you must update **all three layers** to prevent deployment failures:

### 1. Runtime Code

Use the environment variable in Go code:

```go
myValue := helpers.GetEnvOrPanic("MY_NEW_VAR")
```

### 2. Helm Chart

Add the variable to `distribution/genai-hub-service-helm/`:

- Update `values.yaml` with default value
- Add to deployment template to inject as environment variable
- Document in chart README

### 3. SCE Definition

Update `distribution/genai-hub-service-sce/`:

- Add new prompt parameter (fixed or dynamic)
- Map parameter to Helm chart variable
- Define default value and validation constraints

**Example flow**:

```
SCE prompt → Helm chart value → Container env var → Go code
```

Failure to update all three layers will cause IaC deployment failures when the service expects an environment variable that wasn't provided.

## Adding New AWS Bedrock Models

Adding a new AWS model requires **coordinated changes across both runtime and infrastructure**:

### 1. Infrastructure Changes (GenAIInfrastructure product)

Must deploy AWS resources first:

#### a. Update GenAIAWSBedrockInfra SCE

Location: `distribution/genai-awsbedrock-infra-sce/`

- Add the new model ID to `allowedValues` for the ModelID parameter
- This validates that the model ID is recognized and supported

#### b. Update Terraform module

Location: `distribution/genai-awsbedrock-infra-terraform/`

- Add new model mapping and configuration

#### c. Update product definition

Location: `distribution/product-catalog/`

- Edit `genai-infrastructure.yml` to add a new service block
- Each model gets its own block that provisions the GenAIAWSBedrockInfra SCE
- Critical parameters in the block:
  - `ModelID`: Must match the allowed value added to the SCE (e.g., `anthropic.claude-3-haiku-20240307-v1:0`)
  - `ModelMapping`: User-facing model name (e.g., `claude-3-haiku`)
  - `Region`, `AccountID`, `TargetApi`, inference profile settings
- Terraform will provision:
  - IAM roles and policies for model access
  - Secrets for credentials
  - Cross-region inference profiles (if needed)

### 2. Runtime Changes (genai-hub-service)

Then add model metadata to the service:

#### a. Add model metadata (REQUIRED for all model types)

- Update `distribution/genai-hub-service-helm/src/main/helm/model-metadata.yaml`
- Every new model (AWS Bedrock, OpenAI, GCP Vertex) MUST have metadata defined here
- Includes: model name, provider, capabilities, version information

#### b. Generate model specs

- Use `make add-bedrock-model MODEL_ID=<model-id>` to generate specs in `internal/models/specs/`
- Or manually create for other provider types

#### c. Register processor

- Add processor registration in `internal/request/processors/registry/`

#### d. Update Helm configuration

- Add model-specific configuration in `distribution/genai-hub-service-helm/src/main/helm/configuration/`

#### e. Add integration test coverage

**Both changes must occur together** for the system to work end-to-end. The infrastructure provides the AWS resources and credentials, while the runtime knows how to route requests to those resources.

Similar coordination is required for GCP Vertex models (GenAIInfrastructureGCP) and private models (GenAIPrivateModels).

## Component Management

- **CRITICAL**: Before making changes, consult `SCE_TO_PRODUCT_MAPPING.md` (see "Distribution Structure" in `architecture.md` for details)
- This determines the upgrade order (control plane first, backing services second) and blast radius of changes
- **When adding new components**: Update the mapping file with component name, product, resource type, and subcomponents
