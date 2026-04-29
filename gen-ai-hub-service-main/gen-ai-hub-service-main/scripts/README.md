# Model Management Scripts

Cross-platform Go scripts for scaffolding new AI models in the gen-ai-hub-service project.

## Overview

When adding new AI models to the service, multiple configuration files need to be updated in a coordinated manner. These scripts automate the scaffolding process for different cloud providers:

| Script | Provider | Command |
|--------|----------|---------|
| `add-bedrock-model.go` | AWS Bedrock | `make add-bedrock-model` |
| `add-vertex-model.go` | GCP Vertex AI | `make add-vertex-model` |

## Prerequisites

- Go 1.21 or later (the project's Go version)

---

# Add Vertex AI Model Script

A script for scaffolding new GCP Vertex AI models (Google Gemini, Imagen, etc.).

## What It Does

The script automates adding a new Vertex AI model by modifying:

- `apidocs/spec.yaml` - Adds model to `Models-VertexAIOpenAI` enum
- `cmd/service/main_test.go` - Adds test scenario and model entry
- `distribution/.../configuration/models/<name>.yaml` - Creates config with base, `-next`, and `-deprecated` variants
- `distribution/.../templates/model-metadata.yaml` - Adds model metadata
- `test/integration/service/mappings_test.go` - Adds integration tests for all 3 variants
- `internal/models/specs/gcp/vertex/google/<version>.yaml` - Adds model specification with `maxOutputTokens` for max_tokens framework
- `internal/request/processors/registry/registry.go` - Registers model processor

## Usage

### Via Make (Recommended)

```bash
# Single model with template (recommended)
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash

# Single model without template (creates TODO scaffolds)
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion

# Preview model (appends -preview suffix, e.g., gemini-3.0-flash-preview)
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion PREVIEW=true

# Batch mode (multiple models from JSON config)
make add-vertex-model CONFIG=scripts/vertex-model-sample.json

# Dry run (preview changes without modifying files)
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion DRY_RUN=true
```

### Direct Go Execution

```bash
# Single model with template
go run scripts/add-vertex-model.go -model-name gemini-3.0-flash -model-type chat_completion -template gemini-2.5-flash

# Single model without template
go run scripts/add-vertex-model.go -model-name gemini-3.0-flash -model-type chat_completion

# Batch mode
go run scripts/add-vertex-model.go -config scripts/vertex-model-sample.json

# Dry run
go run scripts/add-vertex-model.go -model-name gemini-3.0-flash -model-type chat_completion -dry-run
```

## Command-Line Parameters

| Parameter | Description | Required |
|-----------|-------------|----------|
| `-model-name` | The model name (e.g., `gemini-3.0-flash`) | Yes (unless using `-config`) |
| `-model-type` | Type of model: `chat_completion`, `embedding`, `image` | Yes (unless using `-config`) |
| `-template` | Existing model to use as template for metadata | No |
| `-preview` | Add model as preview (appends `-preview` suffix to model name) | No |
| `-config` | Path to JSON config file for batch processing | No |
| `-dry-run` | Preview changes without modifying files | No |

## Model Types

| Type | Description | Example Models |
|------|-------------|----------------|
| `chat_completion` | Text generation / chat models | gemini-2.5-flash, gemini-2.5-pro |
| `embedding` | Text embedding models | text-multilingual-embedding-002 |
| `image` | Image generation models | imagen-3, imagen-3-fast |

## Template Feature

The `-template` flag copies configuration from an existing model in `model-metadata.yaml`:

```bash
# Add a new Gemini model using gemini-2.5-flash as template
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash
```

When using a template:
- Copies the metadata structure from the template model
- Replaces identifiers (model_name, model_mapping_id, model_id, etc.)
- Preserves capability definitions, parameters, and other settings

If no template is specified, the script generates a minimal scaffold with TODO markers.

## Batch Mode

For adding multiple models at once, create a JSON configuration file:

```json
{
  "models": [
    {
      "model_name": "gemini-3.0-flash",
      "model_type": "chat_completion",
      "template": "gemini-2.5-flash"
    },
    {
      "model_name": "gemini-3.0-pro",
      "model_type": "chat_completion",
      "template": "gemini-2.5-pro"
    },
    {
      "model_name": "text-embedding-005",
      "model_type": "embedding",
      "template": "text-multilingual-embedding-002"
    }
  ]
}
```

Then run:

```bash
make add-vertex-model CONFIG=path/to/models.json
```

See `scripts/vertex-model-sample.json` for a complete example.

## Files Modified

| File | Description |
|------|-------------|
| `apidocs/spec.yaml` | OpenAPI spec with model enums |
| `cmd/service/main_test.go` | Unit test scenarios |
| `distribution/.../configuration/models/<name>.yaml` | Model routing configuration |
| `distribution/.../templates/model-metadata.yaml` | Model capabilities and parameters |
| `test/integration/service/mappings_test.go` | Integration test scenarios |

## Post-Processing

After running the script, review and update the generated entries:

1. **Search for TODO markers** in the modified files (if not using a template)
2. **Update model capabilities**: Verify features, input/output modalities, mime_types
3. **Set correct token limits**: Update `input_tokens`, `max_tokens` maximums
4. **Add description**: Replace placeholder descriptions
5. **Verify parameters**: Ensure parameter ranges match the model's specifications
6. **Run tests**: Execute `make test` to verify the changes

## Examples

### Add a New Chat Completion Model

```bash
make add-vertex-model MODEL_NAME=gemini-3.0-ultra MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-pro
```

### Add a New Embedding Model

```bash
make add-vertex-model MODEL_NAME=text-embedding-005 MODEL_TYPE=embedding TEMPLATE=text-multilingual-embedding-002
```

### Add a New Image Generation Model

```bash
make add-vertex-model MODEL_NAME=imagen-4 MODEL_TYPE=image TEMPLATE=imagen-3
```

### Preview Changes for Multiple Models

```bash
make add-vertex-model CONFIG=scripts/vertex-model-sample.json DRY_RUN=true
```

---

# Add Bedrock Model Script

A script for scaffolding new AWS Bedrock models (Amazon Nova, Anthropic Claude, Meta Llama, etc.).

## What It Does

The script automates adding a new AWS Bedrock model by modifying:

- `distribution/genai-awsbedrock-infra-sce/src/main/resources/metadata.json` - Adds model ID to allowed values
- `distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml` - Adds model metadata
- `internal/models/specs/aws/bedrock/<creator>/<spec>.yaml` - Adds model specification
- `internal/request/processors/registry/registry.go` - Registers model processor

## Usage

### Via Make (Recommended)

```bash
# Single model
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0

# With template (copies configuration from existing model)
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1

# Batch mode (multiple models from JSON config)
make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json

# Dry run (preview changes without modifying files)
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true
```

### Direct Go Execution

```bash
# Single model
go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0

# With template
go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0 -template nova-lite-v1

# Batch mode
go run scripts/add-bedrock-model.go -config scripts/bedrock-model-sample.json

# Dry run
go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0 -dry-run
```

## Command-Line Parameters

| Parameter | Description | Required |
|-----------|-------------|----------|
| `-model-id` | The full model ID (e.g., `amazon.nova-new-v1:0`) | Yes (unless using `-config`) |
| `-template` | Existing model to use as template for configuration | No |
| `-config` | Path to JSON config file for batch processing | No |
| `-dry-run` | Preview changes without modifying files | No |

## Model ID Format

Model IDs must follow the format: `<creator>.<model-name>:<version>`

Examples:
- `amazon.nova-lite-v1:0`
- `anthropic.claude-3-5-sonnet-20241022-v2:0`
- `meta.llama3-2-90b-instruct-v1:0`

The script parses this format to extract:
- **Creator**: The model provider (`amazon`, `anthropic`, or `meta`)
- **Model Mapping ID**: The model name without creator prefix (e.g., `nova-lite-v1`)
- **Version**: The version suffix (defaults to `v1` if not specified)

## Supported Creators

| Creator | Spec File | Registry Function | Extension |
|---------|-----------|-------------------|-----------|
| `amazon` | `amazon/nova.yaml` or `amazon/embeddings.yaml` | `registerBedrockAmazonProcessors` | `NewBedrockAmazon20230601Extension` |
| `anthropic` | `anthropic/claude.yaml` | `registerBedrockAnthropicProcessors` | `NewBedrockAnthropic20230601Extension` |
| `meta` | `meta/llama.yaml` | `registerBedrockMetaProcessors` | `NewBedrockMeta20230601Extension` |

## Template Feature

The `-template` flag allows you to copy configuration from an existing model in `model-metadata.yaml`. This is useful when adding a new variant of an existing model family.

```bash
# Add a new Nova model using nova-lite-v1 as template
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1
```

When using a template:
- The script finds the template's configuration block in `model-metadata.yaml`
- Copies the structure and replaces identifiers with the new model's values
- Preserves capability definitions, parameters, and other settings

If no template is specified, the script generates a minimal scaffold with TODO markers.

## Batch Mode

For adding multiple models at once, create a JSON configuration file:

```json
{
  "models": [
    {
      "model_id": "amazon.nova-example-v1:0",
      "template": "nova-lite-v1"
    },
    {
      "model_id": "anthropic.claude-example-v1:0",
      "template": "claude-3-haiku"
    }
  ]
}
```

Then run:

```bash
make add-bedrock-model CONFIG=path/to/models.json
```

See `scripts/bedrock-model-sample.json` for a complete example.

## Dry Run Mode

Use dry run mode to preview what changes would be made without actually modifying any files:

```bash
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true
```

This is useful for:
- Verifying the model ID is parsed correctly
- Checking which files would be modified
- Testing batch configurations before applying

## Files Modified

| File | Description |
|------|-------------|
| `distribution/genai-awsbedrock-infra-sce/src/main/resources/metadata.json` | SCE metadata with allowed model IDs |
| `distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml` | Helm chart model metadata configuration |
| `internal/models/specs/aws/bedrock/<creator>/<spec>.yaml` | Model specification files |
| `internal/request/processors/registry/registry.go` | Processor registry for request handling |

## Post-Processing

After running the script, review and update the generated entries:

1. **Search for TODO markers** in the modified files
2. **Update model capabilities**: Verify features, input/output modalities
3. **Set correct token limits**: Update `maxInputTokens`, `maxOutputTokens`
4. **Add description**: Replace placeholder descriptions
5. **Verify parameters**: Ensure parameter ranges match the model's specifications
6. **Run tests**: Execute `make test-processors-registry-coverage` to verify registration

## Examples

### Add a New Amazon Nova Model

```bash
make add-bedrock-model MODEL_ID=amazon.nova-premier-v1:0 TEMPLATE=nova-lite-v1
```

### Add a New Claude Model

```bash
make add-bedrock-model MODEL_ID=anthropic.claude-4-opus:0 TEMPLATE=claude-3-5-sonnet-20241022-v2
```

### Add a New Llama Model

```bash
make add-bedrock-model MODEL_ID=meta.llama4-70b-instruct-v1:0
```

### Preview Changes for Multiple Models

```bash
make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json DRY_RUN=true
```

---

## Troubleshooting

### "Model already exists" Warning

If the script reports that a model already exists, it will skip that model. Check the listed files to see where the model is already defined.

### "Unknown creator" Error (Bedrock)

The creator portion of the model ID must be one of: `amazon`, `anthropic`, or `meta`. Other creators are not currently supported.

### "Invalid model_type" Error (Vertex)

The model type must be one of: `chat_completion`, `embedding`, or `image`.

### Changes Not Applied

If running via Make and changes aren't appearing:
1. Ensure you're in the repository root directory
2. Check that the file paths in the script match your repository structure
3. Try running with `DRY_RUN=true` to verify the script runs correctly

## Contributing

When modifying these scripts:
1. Update this README if adding new features or parameters
2. Test with dry run mode before committing changes
3. Ensure cross-platform compatibility (Linux, macOS, Windows)