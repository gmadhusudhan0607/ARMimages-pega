# DALL-E Image Generation Examples

This guide demonstrates how to use DALL-E 3 for image generation through the GenAI Hub Service.

## Overview

The GenAI Hub Service exposes OpenAI's DALL-E image generation API at:

```
POST /openai/deployments/{modelId}/images/generations?api-version={api-version}
```

**Supported Models**:
- `dall-e-3` - Latest DALL-E model with improved quality and prompt following

**Supported API Versions**:
- `2024-02-01` (recommended)
- `2023-05-15`

**Capabilities**:
- High-quality image generation from text prompts
- Multiple size options
- Style control (vivid vs natural)
- Quality settings
- Automatic prompt enhancement

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Basic Image Generation

### Simple Text-to-Image

Generate an image from a text prompt.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A serene mountain landscape at sunset with a lake in the foreground",
    "n": 1,
    "size": "1024x1024"
  }'
```

### Response

```json
{
  "created": 1698437856,
  "data": [
    {
      "url": "https://example.com/generated-image.png",
      "revised_prompt": "A serene mountain landscape at sunset with a calm lake reflecting the colorful sky in the foreground, surrounded by pine trees and rocky peaks"
    }
  ]
}
```

**Note**: DALL-E 3 automatically enhances prompts for better results. The `revised_prompt` field shows the enhanced version used for generation.

## Image Sizes

DALL-E 3 supports three size options:

### Square (1024x1024)

Default size, good for most use cases.

```json
{
  "prompt": "A futuristic robot assistant",
  "size": "1024x1024"
}
```

### Landscape (1792x1024)

Ideal for wide scenes and panoramas.

```json
{
  "prompt": "A panoramic view of a cyberpunk city at night",
  "size": "1792x1024"
}
```

### Portrait (1024x1792)

Best for vertical compositions and portraits.

```json
{
  "prompt": "A tall ancient oak tree reaching toward the sky",
  "size": "1024x1792"
}
```

## Style Control

### Vivid Style (Default)

Creates hyper-realistic and dramatic images with rich colors.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A dragon flying over a medieval castle",
    "size": "1024x1024",
    "style": "vivid"
  }'
```

### Natural Style

Creates images with more natural, less dramatic colors and compositions.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A cozy coffee shop interior with customers reading",
    "size": "1024x1024",
    "style": "natural"
  }'
```

## Quality Settings

### Standard Quality

Faster generation, good for most use cases.

```json
{
  "prompt": "A modern office workspace",
  "quality": "standard"
}
```

### HD Quality

Higher detail and consistency, slower generation.

```json
{
  "prompt": "A detailed architectural rendering of a sustainable building",
  "quality": "hd",
  "size": "1792x1024"
}
```

## Complete Request Examples

### High-Quality Landscape

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "An epic sci-fi battle scene in space with detailed starships and explosions",
    "n": 1,
    "size": "1792x1024",
    "quality": "hd",
    "style": "vivid"
  }'
```

### Natural Portrait

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A portrait of a wise elderly person with gentle expression",
    "n": 1,
    "size": "1024x1792",
    "quality": "hd",
    "style": "natural"
  }'
```

## Error Handling

### Common Error Responses

#### 400 Bad Request

Invalid parameters or unsupported size.

```json
{
  "error": {
    "code": "400",
    "message": "Invalid size: size must be one of 1024x1024, 1792x1024, 1024x1792"
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

#### 400 Content Policy Violation

Prompt violates content policy.

```json
{
  "error": {
    "code": "content_policy_violation",
    "message": "Your request was rejected as a result of our safety system. Your prompt may contain text that is not allowed by our safety system."
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

## Best Practices

### 1. Prompt Engineering

**Be specific and detailed**:

Bad: "A cat"

Good: "A fluffy orange tabby cat sitting on a windowsill, looking outside at a garden, soft natural lighting, photorealistic style"

**Include style guidance**:
- "Photorealistic"
- "Oil painting style"
- "Digital art"
- "Watercolor illustration"
- "3D render"

**Specify composition**:
- "Close-up portrait"
- "Wide-angle shot"
- "Bird's eye view"
- "From below looking up"

### 2. Prompt Enhancement

DALL-E 3 automatically enhances prompts. To leverage this:


### 3. Size Selection

Choose size based on intended use:

| Size | Aspect Ratio | Best For |
|------|-------------|----------|
| 1024x1024 | 1:1 | Social media posts, avatars, general purpose |
| 1792x1024 | 16:9 | Banners, headers, landscape photos |
| 1024x1792 | 9:16 | Mobile wallpapers, vertical displays, portraits |

### 4. Quality vs Cost

**Standard Quality**:
- Faster generation (typically 10-20s)
- Lower cost
- Good for prototyping and iteration
- Suitable for most use cases

**HD Quality**:
- Slower generation (typically 30-60s)
- Higher cost
- More detail and consistency
- Best for final production images

### 5. Style Selection

**Vivid Style**:
- Rich, saturated colors
- Dramatic lighting
- Hyper-realistic
- Best for: Fantasy, sci-fi, dramatic scenes

**Natural Style**:
- Subtle, realistic colors
- Natural lighting
- Less dramatic
- Best for: Everyday scenes, portraits, documentation

### 6. Content Policy Compliance

To avoid content policy violations:

- Avoid requesting images of identifiable people
- Don't request violent or disturbing content
- Avoid explicit or adult content
- Don't request copyrighted characters or logos
- Keep prompts appropriate for general audiences

### 7. Iteration Strategy


## Performance Optimization

### 1. Caching

Cache generated images to avoid regeneration:


### 2. Batch Processing

Process multiple prompts efficiently:


## References

- [OpenAI DALL-E 3 Guide](https://platform.openai.com/docs/guides/images) - Official OpenAI documentation
- [DALL-E API Reference](https://platform.openai.com/docs/api-reference/images) - Complete API specification
- [OpenAI Content Policy](https://openai.com/policies/usage-policies) - Usage policies and guidelines
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
