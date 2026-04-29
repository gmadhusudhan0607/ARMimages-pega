# Chat Completions API Examples

This guide demonstrates how to use chat completions endpoints across different providers supported by the GenAI Hub Service.

## Overview

The GenAI Hub Service provides multiple chat completion endpoints supporting various LLM providers:

### OpenAI-Compatible Endpoint

```
POST /openai/deployments/{modelId}/chat/completions?api-version={api-version}
```

**Supported Models**:
- `gpt-35-turbo` - GPT-3.5 Turbo (4K context)
- `gpt-35-turbo-16k` - GPT-3.5 Turbo (16K context)
- `gpt-4-preview` - GPT-4 Preview
- `gpt-4-vision-preview` - GPT-4 with vision capabilities
- `gpt-4o` - GPT-4 Optimized
- `gpt-4o-mini` - GPT-4 Optimized Mini
- `gpt-5` - GPT-5 (400K context, vision support)
- `gpt-5-mini` - GPT-5 Mini (400K context, vision support)
- `gpt-5-nano` - GPT-5 Nano (400K context, vision support)
- `gpt-5-chat` - GPT-5 Chat (400K context, vision support)
- `gpt-5.1` - GPT-5.1 (400K context, vision support)
- `gpt-5.2` - GPT-5.2 (400K context, vision support)

**Capabilities**:
- Text generation and conversation
- System prompts for context
- Streaming responses
- Function calling
- Vision analysis (GPT-4 Vision and all GPT-5 models)

### Google Vertex AI Endpoint

```
POST /google/deployments/{modelId}/chat/completions
```

**Supported Models**:
- `gemini-1.5-pro` - Gemini 1.5 Pro
- `gemini-1.5-flash` - Gemini 1.5 Flash
- `gemini-2.0-flash` - Gemini 2.0 Flash
- `gemini-2.5-flash` - Gemini 2.5 Flash
- `gemini-2.5-pro` - Gemini 2.5 Pro
- `gemini-2.5-flash-lite` - Gemini 2.5 Flash Lite

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## OpenAI Chat Completions

### Basic Chat Request

Generate a simple chat completion with GPT-4.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "What is the capital of France?"
      }
    ],
    "temperature": 0.7,
    "max_tokens": 150
  }'
```

### Response

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The capital of France is Paris."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 8,
    "total_tokens": 33
  }
}
```

### Streaming Response

Enable streaming to receive tokens as they are generated.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Write a short poem about the ocean."
      }
    ],
    "stream": true,
    "max_tokens": 100
  }'
```

Streaming response format (Server-Sent Events):

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"The"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" ocean"},"finish_reason":null}]}

data: [DONE]
```

### Multi-Turn Conversation

Maintain conversation history for context-aware responses.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "You are a knowledgeable history teacher."
      },
      {
        "role": "user",
        "content": "Who was Julius Caesar?"
      },
      {
        "role": "assistant",
        "content": "Julius Caesar was a Roman general and statesman who played a critical role in the events that led to the demise of the Roman Republic and the rise of the Roman Empire."
      },
      {
        "role": "user",
        "content": "What year was he assassinated?"
      }
    ]
  }'
```

### Vision Capabilities (GPT-4 Vision)

Analyze images with GPT-4 Vision models.

```bash
curl -X POST \
  "https://your-gateway.example.com/openai/deployments/gpt-4-vision-preview/chat/completions?api-version=2024-02-01" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "What is in this image?"
          },
          {
            "type": "image_url",
            "image_url": {
              "url": "https://example.com/image.jpg"
            }
          }
        ]
      }
    ],
    "max_tokens": 300
  }'
```

## Google Vertex AI Chat Completions

### Basic Gemini Request

Use Gemini models via OpenAI-compatible endpoint.

```bash
curl -X POST \
  "https://your-gateway.example.com/google/deployments/gemini-2.5-flash/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Explain the theory of relativity."
      }
    ],
    "temperature": 0.7,
    "max_tokens": 600
  }'
```

### Multimodal Request (Gemini)

Include images in your Gemini requests.

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
            "text": "What is shown in this image?"
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

### Temperature Control

Control randomness in responses (0.0 - 2.0).

```json
{
  "messages": [...],
  "temperature": 0.2  // More focused and deterministic
}
```

```json
{
  "messages": [...],
  "temperature": 1.5  // More creative and diverse
}
```

### Top-P (Nucleus Sampling)

Alternative to temperature, controls diversity (0.0 - 1.0).

```json
{
  "messages": [...],
  "top_p": 0.9
}
```

### Presence and Frequency Penalties

Reduce repetition in responses.

```json
{
  "messages": [...],
  "presence_penalty": 0.6,   // Penalize new tokens based on presence
  "frequency_penalty": 0.8   // Penalize tokens based on frequency
}
```

### Stop Sequences

Define custom stop sequences to end generation.

```json
{
  "messages": [...],
  "stop": ["\n", "END", "###"]
}
```

## Error Handling

### Common Error Responses

#### 400 Bad Request

Invalid request format or missing required fields.

```json
{
  "error": {
    "code": "400",
    "message": "Invalid request: messages field is required"
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

**Use clear, specific instructions**:

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are an expert Python developer. Provide clean, well-documented code with error handling."
    },
    {
      "role": "user",
      "content": "Write a function to validate email addresses using regex."
    }
  ]
}
```

**Break complex tasks into steps**:

```json
{
  "messages": [
    {
      "role": "user",
      "content": "Analyze this code for bugs:\n1. Check syntax errors\n2. Identify logical errors\n3. Suggest improvements\n\n```python\n[code here]\n```"
    }
  ]
}
```

### 2. Token Management

- **Set appropriate max_tokens**: Balance response length with cost
- **Monitor token usage**: Track `usage` field in responses
- **Optimize prompts**: Remove unnecessary verbosity

### 3. Streaming Best Practices

- **Use for long responses**: Improve perceived latency
- **Handle connection drops**: Implement retry logic
- **Buffer partial responses**: Ensure complete sentences

### 4. Model Selection

- **GPT-3.5 Turbo**: Fast, cost-effective for simple tasks
- **GPT-4**: Complex reasoning, detailed analysis
- **GPT-4o**: Optimized balance of speed and capability
- **Claude 3.5 Sonnet**: Long context, complex analysis
- **Gemini 2.5**: Multimodal capabilities, large context

### 5. Context Window Management

Different models have different context limits:
- GPT-3.5 Turbo: 4K-16K tokens
- GPT-4: 8K-32K tokens
- GPT-5: 400K tokens
- Gemini 1.5/2.5: Up to 2M tokens

Monitor token counts and implement truncation strategies for long conversations.

## References

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat) - Official OpenAI documentation
- [Vertex AI OpenAI Library](https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/call-vertex-using-openai-library/) - Google Vertex AI with OpenAI SDK
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
