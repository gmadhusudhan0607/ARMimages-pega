# AWS Bedrock Converse API Examples

This guide demonstrates how to use AWS Bedrock models through the Converse API exposed by the GenAI Hub Service.

## Overview

The GenAI Hub Service provides standardized endpoints for AWS Bedrock models using the Converse API:

```
POST /anthropic/deployments/{modelId}/chat/completions
POST /meta/deployments/{modelId}/chat/completions
POST /amazon/deployments/{modelId}/embeddings
```

**Supported Anthropic Models**:
- `claude-3-haiku` - Fast, compact model for simple tasks
- `claude-3-5-haiku` - Enhanced Haiku with improved performance
- `claude-3-5-sonnet` - Most capable model for complex reasoning

**Supported Meta Models**:
- `llama3-8b-instruct` - Instruction-tuned Llama 3 8B

**Supported Amazon Models**:
- `titan-text-embedding` - Amazon Titan text embeddings
- `nova-2-multimodal-embeddings` - Multimodal embeddings (text, image, video)

**Capabilities**:
- Text generation and conversation
- Long context processing (up to 200K tokens for Claude)
- Tool/function calling
- Multi-turn conversations
- Embeddings generation

## Authentication

All requests require a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Anthropic Claude Models

### Basic Chat Request

Send a simple chat completion request to Claude.

```bash
curl -X POST \
  "https://your-gateway.example.com/anthropic/deployments/claude-3-5-sonnet/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Explain the difference between machine learning and deep learning."
      }
    ],
    "max_tokens": 500,
    "temperature": 0.7
  }'
```

### Response

```json
{
  "output": {
    "message": {
      "role": "assistant",
      "content": [
        {
          "text": "Machine learning and deep learning are related but distinct concepts:\n\nMachine Learning (ML):\n- Broader field of AI where algorithms learn from data\n- Can use various techniques (decision trees, SVMs, neural networks)\n- Often requires manual feature engineering\n- Works well with smaller datasets\n\nDeep Learning (DL):\n- Subset of machine learning using artificial neural networks\n- Uses multiple layers (hence \"deep\") to learn hierarchical representations\n- Automatically learns features from raw data\n- Requires larger datasets and more computational power\n- Excels at tasks like image recognition, NLP, and speech processing\n\nIn essence, all deep learning is machine learning, but not all machine learning is deep learning."
        }
      ]
    }
  },
  "stopReason": "end_turn",
  "usage": {
    "inputTokens": 18,
    "outputTokens": 158,
    "totalTokens": 176
  }
}
```

### System Prompts

Claude supports system prompts for better context control.

```bash
curl -X POST \
  "https://your-gateway.example.com/anthropic/deployments/claude-3-haiku/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "system": [
      {
        "text": "You are a senior software architect specializing in microservices. Provide detailed technical explanations with code examples."
      }
    ],
    "messages": [
      {
        "role": "user",
        "content": "How should I handle distributed transactions in microservices?"
      }
    ],
    "max_tokens": 1000,
    "temperature": 0.5
  }'
```

### Multi-Turn Conversation

Maintain conversation context across multiple exchanges.

```bash
curl -X POST \
  "https://your-gateway.example.com/anthropic/deployments/claude-3-5-sonnet/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "What is the capital of Japan?"
      },
      {
        "role": "assistant",
        "content": [
          {
            "text": "The capital of Japan is Tokyo."
          }
        ]
      },
      {
        "role": "user",
        "content": "What is its population?"
      }
    ],
    "max_tokens": 300
  }'
```

### Long Context Processing

Claude models support very long contexts (up to 200K tokens).

```bash
curl -X POST \
  "https://your-gateway.example.com/anthropic/deployments/claude-3-5-sonnet/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Here is a long document to analyze:\n\n[50,000+ word document here]\n\nSummarize the key points and identify any inconsistencies."
      }
    ],
    "max_tokens": 2000,
    "temperature": 0.3
  }'
```

## Meta Llama Models

### Basic Llama Request

Use Llama 3 for chat completions.

```bash
curl -X POST \
  "https://your-gateway.example.com/meta/deployments/llama3-8b-instruct/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Write a Python function to calculate the factorial of a number."
      }
    ],
    "max_tokens": 500,
    "temperature": 0.7
  }'
```

### Response Format

```json
{
  "output": {
    "message": {
      "role": "assistant",
      "content": [
        {
          "text": "Here'\''s a Python function to calculate factorial:\n\n```python\ndef factorial(n):\n    \"\"\"Calculate factorial of n.\"\"\"\n    if n < 0:\n        raise ValueError(\"Factorial not defined for negative numbers\")\n    if n == 0 or n == 1:\n        return 1\n    return n * factorial(n - 1)\n\n# Example usage\nprint(factorial(5))  # Output: 120\n```\n\nThis recursive implementation handles edge cases like 0 and 1, and raises an error for negative inputs."
        }
      ]
    }
  },
  "stopReason": "end_turn",
  "usage": {
    "inputTokens": 15,
    "outputTokens": 142,
    "totalTokens": 157
  }
}
```

### System Context for Llama

```bash
curl -X POST \
  "https://your-gateway.example.com/meta/deployments/llama3-8b-instruct/chat/completions" \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "system": [
      {
        "text": "You are a helpful coding assistant. Always include comments and explain your code."
      }
    ],
    "messages": [
      {
        "role": "user",
        "content": "Create a JavaScript function to debounce user input."
      }
    ],
    "max_tokens": 600
  }'
```

## Amazon Titan Embeddings

### Generate Text Embeddings

Create vector embeddings for text using Amazon Titan.

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
    0.123,
    -0.456,
    0.789,
    ...
  ],
  "embeddingsByType": {
    "float": [
      0.123,
      -0.456,
      0.789,
      ...
    ]
  },
  "inputTextTokenCount": 11
}
```

## Amazon Nova 2 Multimodal Embeddings

### Text-Only Embedding

Generate embeddings from text content.

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
        "value": "Your text content to embed here"
      }
    }
  }'
```

### Image-Only Embedding

Generate embeddings from image data.

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

Combine text and image for unified embedding.

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
        "value": "A scenic mountain landscape at sunset"
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

### Video Embedding (Segmented)

Generate embeddings from video content with automatic segmentation.

```bash
curl -X POST \
  "https://your-gateway.example.com/amazon/deployments/nova-2-multimodal-embeddings/embeddings" \
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
            "uri": "s3://my-bucket/path/to/video.mp4"
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

## Advanced Configuration

### Temperature Control

- **Low (0.0-0.3)**: Focused, deterministic responses for factual tasks
- **Medium (0.4-0.7)**: Balanced creativity and consistency
- **High (0.8-1.0)**: Creative, diverse responses for brainstorming

```json
{
  "messages": [...],
  "temperature": 0.2,
  "max_tokens": 500
}
```

### Token Limits

Set appropriate token limits based on use case:

```json
{
  "messages": [...],
  "max_tokens": 2000  // Allow longer responses
}
```

### Stop Sequences

Define custom stop sequences to control generation:

```json
{
  "messages": [...],
  "stopSequences": ["\n\nHuman:", "END"]
}
```

## Error Handling

### Common Error Responses

#### 400 Bad Request

```json
{
  "error": {
    "code": "400",
    "message": "Invalid request: max_tokens must be a positive integer"
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

#### 404 Not Found

```json
{
  "error": {
    "code": "404",
    "message": "Model not available in this deployment"
  }
}
```

#### 429 Rate Limit

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

- **Claude 3 Haiku**: Fast responses, simple tasks, cost-effective
- **Claude 3.5 Sonnet**: Complex reasoning, long documents, detailed analysis
- **Llama 3 8B**: Open source option, code generation, general chat
- **Titan Embeddings**: Simple text embeddings
- **Nova 2 Multimodal**: Cross-modal search, video analysis

### 2. Prompt Engineering

**Be explicit with instructions**:

```json
{
  "messages": [
    {
      "role": "user",
      "content": "Provide a detailed analysis of the following code. Format your response as:\n1. Overview\n2. Issues found\n3. Recommendations\n\n[code here]"
    }
  ]
}
```

**Use system prompts for context**:

```json
{
  "system": [
    {
      "text": "You are a senior DevOps engineer specializing in Kubernetes. Provide production-ready solutions."
    }
  ],
  "messages": [...]
}
```

### 3. Token Management

- Monitor `usage` field in responses
- Set appropriate `max_tokens` limits
- Implement token counting for long contexts
- Consider model context windows (Claude: 200K, Llama: 8K)

### 4. Long Context Handling

For documents >10K tokens:
- Summarize progressively for very long documents
- Use Claude models for best long-context performance
- Structure prompts with clear sections

### 5. Embedding Best Practices

**Text embeddings**:
- Normalize input text
- Use consistent text preprocessing
- Choose appropriate embedding dimension

**Multimodal embeddings**:
- Match text description to image content
- Use appropriate dimensions (1024 for images, 3072 for text/multimodal)
- Consider video segmentation strategy for long videos

## References

- [AWS Bedrock Converse API](https://docs.aws.amazon.com/bedrock/latest/APIReference/API_runtime_Converse.html) - Official AWS documentation
- [Claude API Documentation](https://docs.anthropic.com/claude/reference/messages_post) - Anthropic Claude reference
- [Amazon Titan Models](https://docs.aws.amazon.com/bedrock/latest/userguide/titan-models.html) - Titan model specifications
- [Amazon Nova Embeddings](https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-embed.html) - Nova multimodal embeddings guide
- [OpenAPI Specification](../../apidocs/spec.yaml) - Complete API schema
- [Architecture Documentation](../guides/architecture.md) - Service architecture overview
