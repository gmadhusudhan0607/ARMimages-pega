# AWS Bedrock Invoke API Examples

This guide demonstrates how to use AWS Bedrock models through the lower-level Invoke API for direct model access via the GenAI Hub Service.

## Overview

The GenAI Hub Service provides wildcard endpoints that support both the Converse API and the Invoke API pattern:

```
POST /{provider}/deployments/{modelId}/*targetApi
```

Where `targetApi` can be:
- `/chat/completions` - Automatically routes to Converse API
- `/embeddings` - Automatically routes to Invoke API for embeddings
- `/converse` - Explicit Converse API
- `/invoke` - Explicit Invoke API (low-level model access)

**Supported Providers**:
- `anthropic` - Claude models
- `meta` - Llama models
- `amazon` - Titan and Nova models

**Use Cases for Invoke API**:
- Direct model access with provider-specific parameters
- Custom request/response formats
- Advanced model configurations not available in Converse API
- Embedding generation (automatically uses Invoke)
- Legacy integrations requiring specific API formats

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Invoke vs Converse API

### Converse API (Recommended)

**Advantages**:
- Standardized request/response format across all models
- Simplified message structure
- Automatic handling of model-specific quirks
- Better for most use cases

**Endpoint**:
```
POST /anthropic/deployments/{modelId}/converse
```

### Invoke API (Advanced)

**Advantages**:
- Full control over model-specific parameters
- Access to all model capabilities
- Required for some specialized features
- Direct pass-through to Bedrock

**Endpoint**:
```
POST /anthropic/deployments/{modelId}/invoke
```

## Embeddings via Invoke API

The GenAI Hub Service automatically routes embedding requests to the Invoke API.

### Amazon Titan Text Embeddings

Generate text embeddings using the Invoke API pattern.

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/titan-text-embedding/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "inputText": "Example text for embedding generation"
  }'
```

**Alternative explicit invoke endpoint**:

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/titan-text-embedding/invoke" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "inputText": "Example text for embedding generation"
  }'
```

### Response

```json
{
  "embedding": [
    0.0234375,
    -0.015625,
    0.041015625,
    ...
  ],
  "embeddingsByType": {
    "float": [
      0.0234375,
      -0.015625,
      0.041015625,
      ...
    ]
  },
  "inputTextTokenCount": 6
}
```

### Batch Text Embeddings

Process multiple texts in a single request (if supported by model).

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/titan-text-embedding/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "texts": [
      "First text to embed",
      "Second text to embed",
      "Third text to embed"
    ]
  }'
```

## Amazon Nova Multimodal Invoke

### Single Embedding Request

Generate embeddings for text, images, or multimodal content.

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/invoke" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "taskType": "SINGLE_EMBEDDING",
    "singleEmbeddingParams": {
      "embeddingPurpose": "GENERIC_INDEX",
      "embeddingDimension": 3072,
      "text": {
        "truncationMode": "END",
        "value": "A detailed product description for search indexing"
      }
    }
  }'
```

### Segmented Video Embedding

Process video content with automatic segmentation.

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/invoke" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "taskType": "SEGMENTED_EMBEDDING",
    "segmentedEmbeddingParams": {
      "embeddingPurpose": "GENERIC_INDEX",
      "embeddingDimension": 3072,
      "video": {
        "format": "mp4",
        "embeddingMode": "AUDIO_VIDEO_COMBINED",
        "source": {
          "s3Location": {
            "uri": "s3://my-bucket/videos/sample.mp4"
          }
        },
        "segmentationConfig": {
          "type": "FIXED_LENGTH",
          "fixedLengthSegmentationConfig": {
            "segmentLengthSeconds": 30,
            "overlapSeconds": 5
          }
        }
      }
    }
  }'
```

### Response Format

```json
{
  "embeddings": [
    {
      "embedding": [0.123, -0.456, 0.789, ...],
      "startTime": 0,
      "endTime": 30
    },
    {
      "embedding": [0.234, -0.567, 0.890, ...],
      "startTime": 25,
      "endTime": 55
    }
  ],
  "inputTokenCount": 0
}
```

## Direct Model Invocation

### Claude with Invoke API

For advanced Claude use cases requiring direct model access.

```bash
curl -X POST \
  "https://your-gateway.example.com/anthropic/deployments/claude-3-5-sonnet/invoke" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "anthropic_version": "bedrock-2023-05-31",
    "max_tokens": 1024,
    "messages": [
      {
        "role": "user",
        "content": "Explain the invoke API pattern"
      }
    ]
  }'
```

### Llama with Invoke API

Direct invocation of Llama models with custom parameters.

```bash
curl -X POST \
  "https://your-gateway.example.com/meta/deployments/llama3-8b-instruct/invoke" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "<|begin_of_text|><|start_header_id|>user<|end_header_id|>\n\nExplain recursion<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n",
    "max_gen_len": 512,
    "temperature": 0.7,
    "top_p": 0.9
  }'
```

## API Routing Reference

The GenAI Hub Service automatically routes requests based on the target API path:

### Automatic Routing

| Endpoint Path | Routes To | Use Case |
|--------------|-----------|----------|
| `/chat/completions` | Converse API | Standard chat completions |
| `/embeddings` | Invoke API | Embedding generation |
| `/converse` | Converse API | Explicit converse call |
| `/invoke` | Invoke API | Direct model invocation |

### Example URLs

```bash
# These two are equivalent for chat:
POST /anthropic/deployments/claude-3-haiku/chat/completions
POST /anthropic/deployments/claude-3-haiku/converse

# These two are equivalent for embeddings:
POST /amazon/deployments/titan-text-embedding/embeddings
POST /amazon/deployments/titan-text-embedding/invoke
```

## Error Handling

### Common Error Responses

#### 400 Bad Request

Invalid payload or unsupported parameters.

```json
{
  "error": {
    "code": "400",
    "message": "Invalid request body: inputText field is required"
  }
}
```

#### 401 Unauthorized

```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized: Invalid or missing authentication token"
  }
}
```

#### 404 Model Not Found

```json
{
  "error": {
    "code": "404",
    "message": "Model titan-text-embedding is with API invoke not available in this GenAI Gateway Service deployment. Contact CloudOps."
  }
}
```

#### 500 Internal Server Error

```json
{
  "error": {
    "code": "500",
    "message": "Internal server error"
  }
}
```

### Python Error Handling with Retry


## Best Practices

### 1. Choose the Right API

**Use Converse API when**:
- Building standard chat applications
- You want simplified message handling
- Cross-model compatibility is important
- You don't need model-specific features

**Use Invoke API when**:
- Generating embeddings
- You need direct model access
- Using model-specific advanced features
- Migrating from direct Bedrock integration

### 2. Embedding Generation

**Text embeddings best practices**:
- Normalize and preprocess text consistently
- Use appropriate embedding dimensions for your use case
- Batch requests when possible for efficiency
- Cache embeddings for frequently used text

**Multimodal embeddings**:
- Match text description to visual content
- Choose embedding dimension based on modality (1024 for image-only, 3072 for text/multimodal)
- Consider compute cost vs accuracy trade-offs

### 3. Video Processing

**Segmentation strategies**:
- **Fixed-length**: Consistent segment size, good for uniform content
- **Shot-based**: Semantic boundaries, better for edited video
- **Overlap**: Add overlap between segments to avoid boundary issues

**Configuration recommendations**:
```json
{
  "segmentLengthSeconds": 30,  // Balance granularity vs performance
  "overlapSeconds": 5           // Smooth transitions between segments
}
```

### 4. Error Handling

- Implement exponential backoff for rate limits
- Log detailed error information for debugging
- Don't retry authentication errors
- Handle model availability errors gracefully

### 5. Performance Optimization

- Use connection pooling for multiple requests
- Implement request batching where supported
- Cache embeddings to reduce API calls
- Monitor token usage and costs

## Migration Guide

### From Direct Bedrock to GenAI Hub

**Before (Direct Bedrock)**:

**After (GenAI Hub Service)**:

**Benefits**:
- Centralized authentication via JWT
- Built-in monitoring and logging
- Rate limiting and quota management
- No AWS credentials needed in application code

## References

- [AWS Bedrock InvokeModel API](https://docs.aws.amazon.com/bedrock/latest/APIReference/API_runtime_InvokeModel.html) - Official AWS documentation
- [AWS Bedrock Converse API](https://docs.aws.amazon.com/bedrock/latest/APIReference/API_runtime_Converse.html) - Converse API reference
- [Amazon Titan Embeddings](https://docs.aws.amazon.com/bedrock/latest/userguide/titan-embedding-models.html) - Titan embedding specifications
- [Amazon Nova Multimodal](https://docs.aws.amazon.com/bedrock/latest/userguide/nova-multimodal-embed.html) - Nova embedding guide
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
