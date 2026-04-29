# API Usage Examples

This directory contains comprehensive usage examples for all GenAI Hub Service API endpoints.

## Quick Links

### Core Endpoints

- **[Chat Completions](./chat-completions.md)** - Text generation and conversation across OpenAI, Bedrock (Claude, Llama), and Vertex AI (Gemini)
- **[Embeddings](./embeddings.md)** - Vector embeddings for semantic search, similarity, and clustering across all providers

### Provider-Specific Endpoints

#### AWS Bedrock
- **[Bedrock Converse API](./bedrock-converse.md)** - Standardized chat completions for Claude and Llama models
- **[Bedrock Invoke API](./bedrock-invoke.md)** - Direct model invocation and embeddings generation

#### Google Vertex AI
- **[Vertex AI Predict API](./vertex-predict.md)** - Native Vertex AI endpoints for Imagen and text embeddings

#### OpenAI
- **[DALL-E Image Generation](./dalle-image-generation.md)** - Text-to-image generation with DALL-E 3
- **[Gemini Image Generation](./gemini-image-generation.md)** - Gemini native image generation and editing

## Getting Started

### Prerequisites

All API requests require:
1. **JWT Authentication Token** - Obtained from your authentication provider
2. **Gateway URL** - Your GenAI Hub Service endpoint (e.g., `https://your-gateway.example.com`)

### Basic Example

```bash
# Set your JWT token
export JWT_TOKEN="your-jwt-token-here"

# Test chat completion
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Hello, how can you help me?"
      }
    ]
  }'
```

## Documentation Structure

Each guide includes:

1. **Overview** - Endpoint URLs, supported models, and capabilities
2. **Authentication** - JWT requirements and headers
3. **Basic Usage** - Simple curl examples to get started
4. **Advanced Examples** - Complex use cases with multiple parameters
5. **Code Examples** - Python and JavaScript/Node.js implementations
6. **Error Handling** - Common errors and retry strategies
7. **Best Practices** - Prompt engineering, performance optimization, model selection
8. **References** - Links to official provider documentation

## API Endpoints Summary

### Chat Completions

| Provider | Endpoint | Supported Models |
|----------|----------|-----------------|
| OpenAI | `/openai/deployments/{modelId}/chat/completions` | GPT-3.5, GPT-4, GPT-4o |
| Anthropic (Bedrock) | `/anthropic/deployments/{modelId}/chat/completions` | Claude 3 Haiku, Claude 3.5 Haiku/Sonnet |
| Meta (Bedrock) | `/meta/deployments/{modelId}/chat/completions` | Llama 3 8B Instruct |
| Google | `/google/deployments/{modelId}/chat/completions` | Gemini 1.5, 2.0, 2.5 (all variants) |

### Embeddings

| Provider | Endpoint | Supported Models |
|----------|----------|-----------------|
| OpenAI | `/openai/deployments/{modelId}/embeddings` | text-embedding-ada-002, text-embedding-3-small/large |
| Amazon (Bedrock) | `/amazon/deployments/{modelId}/embeddings` | Titan Text, Nova 2 Multimodal |
| Google | `/google/deployment/{modelId}/embeddings` | text-multilingual-embedding-002 |

### Image Generation

| Provider | Endpoint | Supported Models |
|----------|----------|-----------------|
| OpenAI | `/openai/deployments/dall-e-3/images/generations` | DALL-E 3 |
| Google (Gemini) | `/google/deployments/{modelId}/generateContent` | gemini-3.1-flash-image-preview, gemini-2.5-flash-image |
| Google (Imagen) | `/google/deployment/{modelId}/image/generation` | imagen-3, imagen-3-fast |

## Common Patterns

### Authentication

All requests require the same authentication pattern:

```bash
-H "Authorization: Bearer ${JWT_TOKEN}"
```

### Error Handling

Standard HTTP status codes:
- `200` - Success
- `400` - Bad request (invalid parameters)
- `401` - Unauthorized (invalid/missing JWT)
- `429` - Rate limit exceeded
- `500` - Internal server error

### Streaming Responses

Chat endpoints support streaming via `stream: true`:

```json
{
  "messages": [...],
  "stream": true
}
```

### Rate Limiting

Implement exponential backoff for 429 errors:


## Use Case Guides

### Semantic Search
See [Embeddings Guide](./embeddings.md#semantic-search) for complete implementation.

### Multi-Turn Conversations
See [Chat Completions Guide](./chat-completions.md#multi-turn-conversation) for conversation history management.

### Image Generation
- **DALL-E**: See [DALL-E Guide](./dalle-image-generation.md) for text-to-image
- **Gemini**: See [Gemini Image Guide](./gemini-image-generation.md) for iterative editing
- **Imagen**: See [Vertex Predict Guide](./vertex-predict.md#imagen-image-generation-predict-api) for production-quality images

### Multimodal AI
- **Vision**: See [Chat Completions Guide](./chat-completions.md#vision-capabilities-gpt-4-vision) for image analysis
- **Embeddings**: See [Bedrock Converse Guide](./bedrock-converse.md#amazon-nova-2-multimodal-embeddings) for multimodal embeddings

## Model Selection Guide

### Chat Completions

| Use Case | Recommended Model | Reason |
|----------|------------------|---------|
| Fast responses | GPT-3.5 Turbo, Claude 3 Haiku | Speed optimized |
| Complex reasoning | GPT-4, Claude 3.5 Sonnet | Higher capability |
| Long context | Claude 3.5 Sonnet, Gemini 1.5 Pro | 200K-2M tokens |
| Cost optimization | GPT-4o Mini, Llama 3 8B | Lower cost |
| Multimodal | GPT-4 Vision, Gemini 1.5 Pro | Image understanding |

### Embeddings

| Use Case | Recommended Model | Reason |
|----------|------------------|---------|
| General purpose | text-embedding-3-small | Balance of speed/quality |
| High accuracy | text-embedding-3-large | Best retrieval performance |
| Multimodal | Nova 2 Multimodal | Text + image search |
| Multilingual | text-multilingual-embedding-002 | 100+ languages |
| AWS ecosystem | Titan Text Embeddings | Native AWS integration |

### Image Generation

| Use Case | Recommended Model | Reason |
|----------|------------------|---------|
| High quality | DALL-E 3 HD, Imagen 3 | Best fidelity |
| Fast iteration | Imagen 3 Fast | Speed optimized |
| Editing workflow | Gemini Image | Multi-turn refinement |
| Specific aspect ratios | Imagen | Multiple ratio options |

## Testing

### Integration Tests

The service includes integration tests:

```bash
make integration-test
```

### Live Tests

Test against deployed models:

```bash
make test-live-image CONFIG=gemini-image PROMPT=image-generation
```

See `make test-live help` for all options.

## Additional Resources

### Documentation
- [Architecture Guide](../guides/architecture.md) - Service architecture overview
- [Building and Testing Guide](../guides/building_and_testing.md) - Development setup
- [Code Conventions](../guides/code_conventions.md) - Coding standards
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema

### External References
- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference)
- [AWS Bedrock Documentation](https://docs.aws.amazon.com/bedrock/)
- [Google Vertex AI Documentation](https://cloud.google.com/vertex-ai/docs)

## Support

For issues or questions:
1. Check the relevant example guide for your use case
2. Review the [Architecture Documentation](../guides/architecture.md)
3. Consult the [Building and Testing Guide](../guides/building_and_testing.md)
4. Open an issue in the repository with:
   - The endpoint you're using
   - Sample request/response (sanitized)
   - Error messages received
   - Steps to reproduce

## Contributing

When adding new examples:
1. Follow the existing structure and format
2. Include curl examples, Python, and JavaScript code
3. Add error handling examples
4. Document best practices
5. Link to official provider documentation
6. Update this README.md index
