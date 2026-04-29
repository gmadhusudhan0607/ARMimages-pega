# Google Vertex AI Predict API Examples

This guide demonstrates how to use Google Vertex AI models through the Predict API pattern exposed by the GenAI Hub Service.

## Overview

The GenAI Hub Service provides Vertex AI endpoints that support both OpenAI-compatible APIs and native Vertex AI Predict API patterns:

### OpenAI-Compatible Endpoints

```
POST /google/deployments/{modelId}/chat/completions
POST /google/deployments/{modelId}/embeddings
```

### Native Vertex AI Predict Endpoints

```
POST /google/deployments/{modelId}/images/generations
POST /google/deployments/{modelId}/embeddings
```

**Supported Models**:
- **Gemini Chat**: `gemini-1.5-pro`, `gemini-1.5-flash`, `gemini-2.0-flash`, `gemini-2.5-flash`, `gemini-2.5-pro`, `gemini-2.5-flash-lite`
- **Gemini Image**: `gemini-3.1-flash-image-preview`, `gemini-2.5-flash-image`
- **Imagen**: `imagen-3`, `imagen-3-fast`
- **Text Embeddings**: `text-multilingual-embedding-002`

**Capabilities**:
- Text generation with large context windows (up to 2M tokens)
- Image generation (Gemini and Imagen)
- Text embeddings
- Multimodal understanding (text + images)

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Imagen Image Generation (Predict API)

### Basic Image Generation

Generate images using Imagen models via Vertex AI Predict API.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/imagen-3/image/generation" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "modelId": "imagen-3.0-generate-002",
    "payload": {
      "prompt": "A photorealistic image of a sunset over mountains",
      "number_of_images": 1,
      "aspect_ratio": "1:1",
      "safety_filter_level": "block_some",
      "person_generation": "allow_all"
    }
  }'
```

### Response

```json
{
  "predictions": [
    {
      "bytesBase64Encoded": "iVBORw0KGgoAAAANSUhEUgAAA...",
      "mimeType": "image/png"
    }
  ]
}
```

### Generate Multiple Images

Request multiple variations in a single call.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/imagen-3-fast/image/generation" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "modelId": "imagen-3.0-fast-generate-001",
    "payload": {
      "prompt": "A futuristic cityscape at night with neon lights",
      "number_of_images": 4,
      "aspect_ratio": "16:9",
      "safety_filter_level": "block_some",
      "person_generation": "allow_adult"
    }
  }'
```

### Aspect Ratio Options

Imagen supports various aspect ratios:

```bash
# Square
"aspect_ratio": "1:1"

# Portrait
"aspect_ratio": "3:4"
"aspect_ratio": "9:16"

# Landscape
"aspect_ratio": "4:3"
"aspect_ratio": "16:9"
```

### Safety Filter Levels

Control content safety filtering:

```json
{
  "safety_filter_level": "block_most",    // Most restrictive
  "safety_filter_level": "block_some",    // Balanced (recommended)
  "safety_filter_level": "block_few"      // Least restrictive
}
```

### Person Generation Options

Control generation of human-like figures:

```json
{
  "person_generation": "dont_allow",      // No people
  "person_generation": "allow_adult",     // Adults only
  "person_generation": "allow_all"        // All ages
}
```

## Text Embeddings (Predict API)

### Generate Text Embeddings

Create vector embeddings using Vertex AI text embedding models.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/text-multilingual-embedding-002/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-multilingual-embedding-002",
    "texts": [
      "What is the capital of France?"
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
              0.02345678901234567,
              ...
            ]
          }
        }
      ],
      {},
      {
        "billableCharacterCount": 28
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
      0.02345678901234567,
      ...
    ]
  }
]
```

### Batch Text Embeddings

Process multiple texts in a single request.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployment/text-multilingual-embedding-002/embeddings" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-multilingual-embedding-002",
    "texts": [
      "First text to embed",
      "Second text to embed",
      "Third text to embed"
    ]
  }'
```

## Gemini Chat (OpenAI-Compatible)

While Gemini models support the Predict API, the OpenAI-compatible endpoint is recommended for chat completions.

### Basic Chat Request

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployments/gemini-2.5-flash/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Explain how transformers work in machine learning"
      }
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

### Multimodal Request

Include images in Gemini requests for visual understanding.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployments/gemini-1.5-pro/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Describe what you see in this image"
          },
          {
            "type": "image_url",
            "image_url": {
              "url": "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
            }
          }
        ]
      }
    ]
  }'
```

## Advanced Configuration

### Imagen Advanced Parameters

#### Safety Filtering

Fine-tune content moderation:

```json
{
  "modelId": "imagen-3.0-generate-002",
  "payload": {
    "prompt": "Your prompt here",
    "safety_filter_level": "block_most",  // Strictest filtering
    "person_generation": "dont_allow"     // No human figures
  }
}
```

#### Image Quality

```json
{
  "payload": {
    "prompt": "High-quality architectural photography",
    "number_of_images": 1,
    "aspect_ratio": "16:9"
  }
}
```

### Text Embedding Parameters

#### Task-Specific Embeddings

While the Predict API doesn't explicitly support task types, consider preprocessing text for specific use cases:


## Error Handling

### Common Error Responses

#### 400 Bad Request

Invalid parameters or malformed request.

```json
{
  "error": {
    "code": "400",
    "message": "Invalid aspect_ratio: must be one of 1:1, 4:3, 3:4, 16:9, 9:16"
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

#### 500 Internal Server Error

```json
{
  "error": {
    "code": "500",
    "message": "Internal server error"
  }
}
```

## Best Practices

### 1. Model Selection

**Imagen 3**:
- High-quality image generation
- Best for professional/commercial use
- Slower but higher fidelity

**Imagen 3 Fast**:
- Faster generation
- Good quality for most use cases
- Cost-effective for prototyping

**Gemini**:
- Use for image generation with conversational editing
- Best for iterative refinement
- Native support via generateContent endpoint

### 2. Prompt Engineering for Images

**Be specific and detailed**:

Bad: "A house"

Good: "A two-story Victorian house with blue paint, white trim, wraparound porch, and garden in front, photographed at golden hour"

**Specify style and quality**:
- "Photorealistic portrait"
- "Digital art in impressionist style"
- "High-resolution architectural photography"

### 3. Text Embedding Best Practices

**Preprocessing**:
- Remove excess whitespace
- Normalize text encoding
- Consider language-specific handling

**Batch Processing**:
- Group texts by similar length
- Process in batches of 5-10 for efficiency
- Cache embeddings for frequently used text

### 4. Aspect Ratio Selection

Choose aspect ratios based on use case:
- **1:1**: Social media posts, profile images
- **4:3 / 3:4**: Traditional photography
- **16:9 / 9:16**: Widescreen, mobile screens

### 5. Safety and Content Filtering

- Start with `block_some` for balanced filtering
- Use `block_most` for public-facing applications
- Test safety filters with edge cases
- Monitor rejected requests for false positives

### 6. Performance Optimization

**Image Generation**:
- Request multiple images in single call when needed
- Use Imagen 3 Fast for non-critical applications
- Cache generated images to reduce API calls

**Embeddings**:
- Batch multiple texts together
- Implement connection pooling
- Cache embeddings with TTL strategy

## API Comparison

### Imagen (Predict API) vs Gemini (generateContent)

| Feature | Imagen Predict API | Gemini generateContent |
|---------|-------------------|----------------------|
| Endpoint | `/image/generation` | `/generateContent` |
| Input Format | Vertex AI native | Gemini native |
| Editing Support | No | Yes (multi-turn) |
| Aspect Ratios | Predefined options | Not specified |
| Best For | Standalone generation | Conversational editing |

### When to Use Each

**Use Imagen Predict API**:
- Single-shot image generation
- Need specific aspect ratios
- Vertex AI native integration
- Maximum image quality

**Use Gemini generateContent**:
- Iterative image refinement
- Conversational editing workflow
- Multi-turn image editing
- See [Gemini Image Generation guide](./gemini-image-generation.md)

## References

- [Vertex AI Imagen Documentation](https://cloud.google.com/vertex-ai/generative-ai/docs/image/overview) - Official Imagen guide
- [Vertex AI Text Embeddings](https://cloud.google.com/vertex-ai/generative-ai/docs/embeddings/get-text-embeddings) - Text embedding documentation
- [Gemini Models on Vertex AI](https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models) - Gemini model specifications
- [Vertex AI Predict API](https://cloud.google.com/vertex-ai/docs/predictions/get-predictions) - Predict API reference
- [Gemini Image Generation](./gemini-image-generation.md) - Gemini-specific image generation guide
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
