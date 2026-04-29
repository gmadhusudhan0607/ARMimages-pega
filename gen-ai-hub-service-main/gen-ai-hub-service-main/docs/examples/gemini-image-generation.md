# Gemini Image Generation Examples

This guide demonstrates how to use Google Gemini models for image generation through the GenAI Hub Service `/generateContent` endpoint.

## Overview

The GenAI Hub Service exposes Gemini's native image generation API at:

```
POST /google/deployments/{modelId}/generateContent
```

**Supported Models**:
- `gemini-3.1-flash-image-preview` (global endpoint)
- `gemini-2.5-flash-image` (regional endpoint, default deployment region)

**Capabilities**:
- Text-to-image generation
- Image editing with reference images
- Multi-turn conversational editing

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Basic Text-to-Image Generation

Generate an image from a text prompt.

### Request

```bash
curl -X POST \
  https://your-gateway.example.com/google/deployments/gemini-3.1-flash-image-preview/generateContent \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          {
            "text": "A beautiful sunset over mountains with vibrant orange and purple colors"
          }
        ]
      }
    ],
    "generationConfig": {
      "responseModalities": ["IMAGE"]
    }
  }'
```

### Response

```json
{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "inline_data": {
              "mime_type": "image/png",
              "data": "iVBORw0KGgoAAAANSUhEUgAAA..."
            }
          }
        ]
      },
      "finishReason": "STOP"
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 12,
    "candidatesTokenCount": 0,
    "totalTokenCount": 12
  }
}
```

The `data` field contains the base64-encoded image. To save it:

```bash
echo "iVBORw0KGgoAAAANSUhEUgAAA..." | base64 -d > output.png
```

## Image Editing with Reference Images

Edit an existing image by providing it as a reference along with editing instructions.

### Request

```bash
curl -X POST \
  https://your-gateway.example.com/google/deployments/gemini-3.1-flash-image-preview/generateContent \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          {
            "text": "Add a rainbow arcing across the sky in this image"
          },
          {
            "inline_data": {
              "mime_type": "image/png",
              "data": "<base64-encoded-reference-image>"
            }
          }
        ]
      }
    ],
    "generationConfig": {
      "responseModalities": ["IMAGE"]
    }
  }'
```

**Note**: The reference image must be base64-encoded. Use this command to encode:

```bash
base64 -i input.png | tr -d '\n'
```

### Response

Same structure as text-to-image generation. The `inline_data.data` field contains the edited image.

## Multi-Turn Conversational Editing

Refine an image through multiple iterations by maintaining conversation history.

### First Turn: Generate Initial Image

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "A serene lake surrounded by pine trees"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseModalities": ["IMAGE"]
  }
}
```

### Second Turn: Refine the Image

Include the previous image in the conversation:

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "A serene lake surrounded by pine trees"
        }
      ]
    },
    {
      "role": "model",
      "parts": [
        {
          "inline_data": {
            "mime_type": "image/png",
            "data": "<base64-encoded-generated-image-from-first-turn>"
          }
        }
      ]
    },
    {
      "role": "user",
      "parts": [
        {
          "text": "Add a wooden dock extending into the water and some ducks swimming nearby"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseModalities": ["IMAGE"]
  }
}
```

## Generation Configuration Parameters

Fine-tune the generation behavior with `generationConfig`:

### Temperature

Controls randomness (0.0 - 2.0). Higher values = more creative/random outputs.

```json
{
  "generationConfig": {
    "responseModalities": ["IMAGE"],
    "temperature": 1.5
  }
}
```

### Top-P Sampling

Nucleus sampling parameter (0.0 - 1.0). Lower values = more focused/deterministic.

```json
{
  "generationConfig": {
    "responseModalities": ["IMAGE"],
    "topP": 0.8
  }
}
```

### Top-K Sampling

Limits token selection to top K options (1 - 40).

```json
{
  "generationConfig": {
    "responseModalities": ["IMAGE"],
    "topK": 20
  }
}
```

### Combined Configuration

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "A futuristic cityscape at night"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseModalities": ["IMAGE"],
    "temperature": 1.2,
    "topP": 0.9,
    "topK": 30
  }
}
```

## Error Handling

### Common Error Responses

#### 400 Bad Request

Invalid request format or unsupported model ID.

```json
{
  "error": {
    "code": "400",
    "message": "Invalid request body: generationConfig.responseModalities is required for image generation"
  }
}
```

#### 401 Unauthorized

Missing or invalid JWT token.

```json
{
  "error": {
    "code": "401",
    "message": "Unauthorized: Invalid or missing authentication token"
  }
}
```

#### 429 Too Many Requests

Rate limit exceeded.

```json
{
  "error": {
    "code": "429",
    "message": "Rate limit exceeded. Please retry after some time."
  }
}
```

#### 500 Internal Server Error

Backend service error.

```json
{
  "error": {
    "code": "500",
    "message": "Internal server error"
  }
}
```

## Best Practices

### 1. Prompt Engineering

**Be specific**: Include details about style, composition, lighting, and mood.

❌ **Vague**: "A house"

✅ **Specific**: "A cozy wooden cabin in a snowy forest, warm lights glowing from windows, evening twilight, photorealistic style"

### 2. Image Editing

**Include context**: Reference what should be preserved and what should change.

✅ **Good**: "Keep the mountain landscape but change the time from day to sunset with orange and pink sky"

### 3. Multi-Turn Refinement

**Incremental changes**: Make small adjustments in each turn rather than drastic changes.

✅ **Turn 1**: "A beach scene"
✅ **Turn 2**: "Add palm trees on the left side"
✅ **Turn 3**: "Add a small boat in the water"

### 4. Generation Parameters

- **High creativity**: Use temperature 1.5-2.0 for artistic/abstract images
- **High consistency**: Use temperature 0.7-1.0 and lower topP for predictable results
- **Balanced**: Default values (temperature 1.0, topP 0.95) work well for most cases

### 5. Image Encoding

**MIME type support**: `image/png`, `image/jpeg`, `image/webp`

**Size limits**: Check Vertex AI documentation for current limits (typically ~4MB for reference images)

## Testing

### Integration Tests

The service includes integration tests that verify model routing:

```bash
make integration-test
```

### Live Tests

Run live tests against deployed models:

```bash
make test-live-image CONFIG=gemini-image PROMPT=image-generation
```

See `make test-live help` for all available options.

## Additional Resources

- [Gemini Image Generation Guide](https://ai.google.dev/gemini-api/docs/image-generation) - Official Google documentation
- [Vertex AI GenerateContent API Reference](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/inference) - Complete API specification
- [ADR-0003](../adr/0003-native-gemini-generatecontent-endpoint.md) - Architectural decision for native endpoint
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema

## Support

For issues or questions:
- Check the [architecture documentation](../guides/architecture.md)
- Review [building and testing guide](../guides/building_and_testing.md)
- Open an issue in the repository
