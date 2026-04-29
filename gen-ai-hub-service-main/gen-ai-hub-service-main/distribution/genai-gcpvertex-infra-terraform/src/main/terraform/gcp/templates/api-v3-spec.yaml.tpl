swagger: '2.0'
host: ${apihost}
info:
  title: GenAI GCP Vertex API ${suffix}
  version: 1.0.0
security:
  - sax_us: []
paths:
  /google/deployments/{vertexModelId}/chat/completions:
    post:
      description: Generate chat completions using Gemini models
      operationId: "Geminichatcompletion"
      x-google-backend:
        address: ${function_url}
        path_translation: CONSTANT_ADDRESS
        deadline: ${function_timeout_seconds}.0
      consumes:
        - application/json
      produces:
        - application/json
        - text/event-stream
      parameters:
        - in: path
          name: vertexModelId
          description: The model to use (e.g, gemini-1.5-pro, gemini-1.5-flash)
          required: true
          type: string
        - in: body
          name: body
          description: Chat Completion Request
          required: true
          schema:
            $ref: '#/definitions/ChatCompletionRequest'
      responses:
        '200':
          description: Successful chat completion generation
          schema:
            $ref: '#/definitions/ChatCompletionResponse'

  /google/deployments/{vertexModelId}/embeddings:
    post:
      description: Generate text Embedding using Gemini models
      operationId: "TextEmbedding"
      x-google-backend:
        address: ${function_url}
        path_translation: CONSTANT_ADDRESS
        deadline: ${function_timeout_seconds}.0
      consumes:
        - application/json
      produces:
        - application/json
      parameters:
        - in: path
          name: vertexModelId
          description: The model to use (e.g, text-multilingual-embedding-002)
          required: true
          type: string
        - in: body
          name: body
          description:  Embedding Generation Request
          required: true
          schema:
            $ref: '#/definitions/EmbeddingRequest'
      responses:
        '200':
          description: Successful embedding generation
          schema:
            $ref: '#/definitions/EmbeddingResponse'

  /google/deployments/{vertexModelId}/images/generations:
    post:
      description: Generate images for the given prompt using Imagen model
      operationId: "Imagen"
      x-google-backend:
        address: ${function_url}
        path_translation: CONSTANT_ADDRESS
        deadline: ${function_timeout_seconds}.0
      consumes:
        - application/json
      produces:
        - application/json
      parameters:
        - in: path
          name: vertexModelId
          description: The model to use (e.g, imagen-3.0-generate-002)
          required: true
          type: string
        - in: body
          name: body
          description:  Image Generation Request
          required: true
          schema:
            $ref: '#/definitions/ImageGenerationRequest'
      responses:
        '200':
          description: Successful image generation
          schema:
            $ref: '#/definitions/ImageGenerationResponse'

  /google/deployments/{vertexModelId}/generateContent:
    post:
      description: Generate content using Gemini models (including image generation)
      operationId: "GeminiGenerateContent"
      x-google-backend:
        address: ${function_url}
        path_translation: CONSTANT_ADDRESS
        deadline: ${function_timeout_seconds}.0
      consumes:
        - application/json
      produces:
        - application/json
      parameters:
        - in: path
          name: vertexModelId
          description: The model to use (e.g, gemini-3.1-flash-image-preview, gemini-2.5-flash-image)
          required: true
          type: string
        - in: body
          name: body
          description: Generate Content Request
          required: true
          schema:
            $ref: '#/definitions/GenerateContentRequest'
      responses:
        '200':
          description: Successful content generation
          schema:
            $ref: '#/definitions/GenerateContentResponse'

definitions:
  ChatCompletionRequest:
    type: object
  ChatCompletionResponse:
    type: object
  ImageGenerationRequest:
    type: object
  ImageGenerationResponse:
    type: object
  EmbeddingRequest:
    type: object
  EmbeddingResponse:
    type: object
  GenerateContentRequest:
    type: object
  GenerateContentResponse:
    type: object
securityDefinitions:
  sax_us:
    authorizationUrl: ""
    flow: "implicit"
    type: "oauth2"
    x-google-issuer: "${oidc_issuer}"
    x-google-jwks_uri: "${oidc_issuer}/v1/keys"
    x-google-audiences: "backing-services"
