# Processor Registry

## Overview
This package manages the registration and creation of request processors for different AI model providers.

## Adding New Models

When adding new models to `internal/models/specs/`, you MUST also register corresponding processors:

### 1. Add Model Specification
Add your model YAML file under `internal/models/specs/{infrastructure}/{provider}/{creator}/`

### 2. Create or Identify Processor Extension
- Check if an existing extension in `internal/request/processors/extensions/` can handle your model
- If not, create a new extension implementing the required interfaces

### 3. Register the Processor
Add registration in `registerAllProcessors()` function in `registry.go`:

```go
_ = registry.Register(ProcessorKey{
    Provider:       "your-provider",
    Infrastructure: "your-infrastructure", 
    Creator:        "your-creator",
    Version:        "your-version",
}, func() interface{} {
    return extensions.NewYourExtension()
})
```

### 4. Validate Registration
Run the validation test to ensure your registration is correct:

```bash
make test-registry-coverage
```

## Troubleshooting

### "Missing Processor Registration" Error
This means you added a model but forgot to register a processor. Follow the steps above.

### "Orphaned Processor" Warning  
This means you have a registered processor but no corresponding model. This might be intentional (e.g., for API version mapping) or indicate cleanup is needed.

### "Failed to Create Processor" Error
This means your processor factory function is broken. Check that:
- The extension exists and is importable
- The factory function returns a valid instance
- All dependencies are available

## Test Commands

```bash
# Check that all models have processor registrations
make test-registry-coverage

# Check processor registry integrity
make test-registry-integrity

# Run all tests (includes registry validation)
make test
```

## Architecture

The processor registry uses a key-based system where each model is mapped to a processor using:
- Provider (e.g., "azure", "openai", "bedrock")
- Infrastructure (e.g., "azure", "aws", "gcp")
- Creator (e.g., "openai", "anthropic", "google")
- Version (e.g., "2024-02-01", "2023-06-01")

This allows for fine-grained control over which processor handles each specific model configuration.
