# Embeddings API Examples

This guide demonstrates how to use embedding models across different providers supported by the GenAI Hub Service.

## Overview

The GenAI Hub Service provides embedding endpoints for various providers:

### OpenAI-Compatible Endpoint

```
POST /openai/deployments/{modelId}/embeddings?api-version={api-version}
```

**Supported Models**:
- `text-embedding-ada-002` - OpenAI Ada v2 (1536 dimensions)
- `text-embedding-3-small` - OpenAI v3 Small (512-1536 dimensions)
- `text-embedding-3-large` - OpenAI v3 Large (256-3072 dimensions)

### AWS Bedrock Endpoints

```
POST /amazon/deployments/titan-text-embedding/embeddings
POST /amazon/deployments/nova-2-multimodal-embeddings/embeddings
```

**Supported Models**:
- `titan-text-embedding` - Amazon Titan Text Embeddings (1024 dimensions)
- `nova-2-multimodal-embeddings` - Amazon Nova Multimodal (1024-3072 dimensions)

### Google Vertex AI Endpoint

```
POST /google/deployments/{modelId}/embeddings
```

**Supported Models**:
- `text-multilingual-embedding-002` - Google multilingual embeddings (768 dimensions)

**Use Cases**:
- Semantic search and retrieval
- Document similarity and clustering
- Recommendation systems
- Question answering systems
- Text classification

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## OpenAI Embeddings

### Basic Text Embedding

Generate embeddings for a single text string.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/text-embedding-3-small/embeddings?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "The quick brown fox jumps over the lazy dog"
  }'
```

### Response

```json
{
  "object": "list",
  "model": "text-embedding-3-small",
  "data": [
    {
      "index": 0,
      "object": "embedding",
      "embedding": [
        0.0023064255,
        -0.009327292,
        -0.0028842222,
        ...
      ]
    }
  ],
  "usage": {
    "prompt_tokens": 8,
    "total_tokens": 8
  }
}
```

### Batch Text Embeddings

Process multiple texts in a single request.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/text-embedding-3-small/embeddings?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "input": [
      "First document about machine learning",
      "Second document about artificial intelligence",
      "Third document about neural networks"
    ]
  }'
```

### Response (Batch)

```json
{
  "object": "list",
  "model": "text-embedding-3-small",
  "data": [
    {
      "index": 0,
      "object": "embedding",
      "embedding": [0.002, -0.009, ...]
    },
    {
      "index": 1,
      "object": "embedding",
      "embedding": [0.003, -0.008, ...]
    },
    {
      "index": 2,
      "object": "embedding",
      "embedding": [0.001, -0.007, ...]
    }
  ],
  "usage": {
    "prompt_tokens": 24,
    "total_tokens": 24
  }
}
```

### OpenAI v3 with Custom Dimensions

OpenAI v3 models support dimension reduction for efficiency.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/text-embedding-3-large/embeddings?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Example text for embedding",
    "dimensions": 1024
  }'
```

## Amazon Titan Embeddings

### Basic Titan Embedding

Generate embeddings using Amazon Titan.

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/titan-text-embedding/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "inputText": "The Text Titan Embedding v2 model is provided by Amazon"
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
  "inputTextTokenCount": 11
}
```

## Amazon Nova Multimodal Embeddings

### Text-Only Embedding

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "taskType": "SINGLE_EMBEDDING",
    "singleEmbeddingParams": {
      "embeddingPurpose": "GENERIC_INDEX",
      "embeddingDimension": 3072,
      "text": {
        "truncationMode": "END",
        "value": "Product description for search indexing"
      }
    }
  }'
```

### Image Embedding

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "taskType": "SINGLE_EMBEDDING",
    "singleEmbeddingParams": {
      "embeddingPurpose": "GENERIC_INDEX",
      "embeddingDimension": 1024,
      "image": {
        "format": "png",
        "source": {
          "bytes": "<base64-encoded-image-data>"
        }
      }
    }
  }'
```

### Multimodal Embedding (Text + Image)

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "taskType": "SINGLE_EMBEDDING",
    "singleEmbeddingParams": {
      "embeddingPurpose": "GENERIC_INDEX",
      "embeddingDimension": 3072,
      "text": {
        "truncationMode": "END",
        "value": "A scenic mountain landscape"
      },
      "image": {
        "format": "png",
        "source": {
          "bytes": "<base64-encoded-image-data>"
        }
      }
    }
  }'
```

## Google Vertex AI Embeddings

### Text Embedding

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/text-multilingual-embedding-002/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-multilingual-embedding-002",
    "texts": [
      "Why is Poland cold"
    ]
  }'
```

### Response

```json
[
  {
    "_prediction_response": [
      [
        {
          "embeddings": {
            "statistics": {
              "token_count": 6,
              "truncated": false
            },
            "values": [
              -0.06757664680480957,
              0.0578920841217041,
              ...
            ]
          }
        }
      ],
      {},
      {
        "billableCharacterCount": 15
      },
      "",
      "",
      null
    ],
    "statistics": {
      "token_count": 6,
      "truncated": false
    },
    "values": [
      -0.06757664680480957,
      0.0578920841217041,
      ...
    ]
  }
]
```

### Batch Vertex Embeddings

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/text-multilingual-embedding-002/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-multilingual-embedding-002",
    "texts": [
      "First text in multiple languages",
      "Second text for embedding",
      "Third text for comparison"
    ]
  }'
```

## Common Use Cases

### Semantic Search


### Document Clustering


### Text Classification


## Error Handling

### Common Error Responses

#### 400 Bad Request

```json
{
  "error": {
    "code": "400",
    "message": "Invalid request: input field is required"
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

#### 429 Rate Limited

```json
{
  "error": {
    "code": "429",
    "message": "Rate limit exceeded. Please retry after some time."
  }
}
```

## Best Practices

### 1. Model Selection

| Model | Dimensions | Best For | Cost |
|-------|-----------|----------|------|
| text-embedding-ada-002 | 1536 | General purpose | Low |
| text-embedding-3-small | 512-1536 | Fast, efficient | Very Low |
| text-embedding-3-large | 256-3072 | High accuracy | Medium |
| titan-text-embedding | 1024 | AWS ecosystem | Low |
| nova-2-multimodal | 1024-3072 | Multimodal search | Medium |
| text-multilingual-embedding | 768 | Multilingual | Low |

### 2. Dimension Selection


### 3. Text Preprocessing


### 4. Batching


### 5. Caching


### 6. Normalization


## Performance Optimization

### Parallel Processing


## References

- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings) - Official OpenAI documentation
- [OpenAI Embeddings API](https://platform.openai.com/docs/api-reference/embeddings) - Complete API specification
- [Amazon Titan Embeddings](https://docs.aws.amazon.com/bedrock/latest/userguide/titan-embedding-models.html) - Titan model documentation
- [Amazon Nova Embeddings](https://docs.aws.amazon.com/bedrock/latest/userguide/nova-multimodal-embed.html) - Nova multimodal guide
- [Vertex AI Text Embeddings](https://cloud.google.com/vertex-ai/generative-ai/docs/embeddings/get-text-embeddings) - Google embeddings documentation
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
