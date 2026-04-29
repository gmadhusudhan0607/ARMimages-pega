# Building and Testing

**Requirements**: Go 1.24.11 or later (see `go.mod`)

## Building

```bash
make build          # Format, vet, lint, staticcheck, then build binaries to bin/
make fmt            # Format code with go fmt
make vet            # Run go vet
make lint           # Run golangci-lint
make staticcheck    # Run staticcheck
make goimports      # Format imports with goimports
```

## Testing

### Unit Tests

```bash
make test           # Run all unit tests with coverage

# Specific registry tests
make test-processors-registry-coverage  # Check processor registry coverage
make test-registry-integrity           # Check processor registry integrity
```

### Integration Tests

```bash
make integration-test-up    # Start Docker containers (service + mockserver)
make integration-test-run   # Run integration tests
make integration-test-down  # Clean up containers

# Or use Gradle
./gradlew integrationTest   # Run full integration test suite
```

Integration tests use a mockserver (`genai-hub-mockserver`) to simulate LLM provider responses.

### Request Processing Tests

Per-directory test suites with isolated service instances:

```bash
make test-request-processing              # Run all request-processing tests
make test-request-processing TEST=name    # Run specific test directory
make test-request-processing TEST=name KEEP=true  # Keep service running after test
```

Request processing tests each run in their own isolated service instance with specific configurations.

### Live Tests

Tests against real services with real LLM backends:

```bash
make test-live RUN=list                   # List all test cases
make test-live RUN=all                    # Run all configs × all prompts
make test-live CONFIG=llm-retry           # Specific config × all prompts
make test-live PROMPT=long-response       # All configs × specific prompt
make test-live CONFIG=llm-retry PROMPT=long-response  # Specific combo
make test-live-memleak CONFIG=llm-retry   # Memory leak detection
make test-live-embeddings                 # Run only embedding tests
make test-live-chat                       # Run only chat completion tests
make test-live-streaming                  # Run only streaming tests
```

Live tests require real service endpoints (set via `OPS_URL` and `SERVICE_URL`) or start local services automatically.

## Running Locally

```bash
make run              # Run service with local config
make run_pretty       # Run service with JSON log formatting (requires jl)

# Generate mapping files for local development
make generateMappingFiles              # Create config files from Helm templates
make generatePrivateModelConfigFiles   # Create private model configs
```

## Model Management

### Adding AWS Bedrock Models

```bash
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 TEMPLATE=nova-lite-v1
make add-bedrock-model CONFIG=scripts/bedrock-model-sample.json  # Batch mode
make add-bedrock-model MODEL_ID=amazon.nova-new-v1:0 DRY_RUN=true
```

### Adding GCP Vertex Models

```bash
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion TEMPLATE=gemini-2.5-flash
make add-vertex-model MODEL_NAME=gemini-3.0-flash MODEL_TYPE=chat_completion PREVIEW=true
make add-vertex-model CONFIG=scripts/vertex-model-sample.json  # Batch mode
```

Model types for Vertex: `chat_completion`, `embedding`, `image`

For the complete workflow of adding models including infrastructure changes, see `infrastructure_coordination.md`.
