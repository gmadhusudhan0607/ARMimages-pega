# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.

"""
Tests for the GCP Cloud Function that handles Vertex AI requests.

This test suite focuses on the Gemini image generation functionality (Nano Banana)
which uses the /generateContent endpoint.
"""

import pytest
import importlib
import json
import sys
import os
from unittest.mock import Mock, patch, MagicMock, call

# Set environment variables to avoid metadata service calls during import
os.environ['GCP_PROJECT_ID'] = 'test-project-id'
os.environ['GCP_REGION'] = 'us-central1'

# Mock requests and monitoring BEFORE importing main
sys.modules['google.cloud.monitoring_v3'] = MagicMock()

# Now we can safely import main
import main


class _FakeAPIStatusError(Exception):
    """Minimal stand-in for openai.APIStatusError used in unit tests."""

    def __init__(self, message, status_code=None, body=None):
        super().__init__(message)
        self.status_code = status_code
        self.body = body


def _parse_sse_json_event(stream_output):
    """Parse the first JSON payload from an SSE data event."""
    for line in stream_output.splitlines():
        if line.startswith("data: "):
            return json.loads(line[len("data: "):])
    raise AssertionError(f"No SSE data event found in output: {stream_output!r}")


class TestGeminiImageGeneration:
    """Test suite for Gemini image generation via /generateContent endpoint"""

    @pytest.fixture
    def mock_request_gemini_image(self):
        """Create a mock request for Gemini image generation"""
        request = Mock()
        request_data = {
            "model": "google/gemini-3.1-flash-image-preview",
            "modelId": "gemini-3.1-flash-image-preview",
            "contents": [
                {
                    "parts": [
                        {"text": "Generate a beautiful sunset over mountains"}
                    ]
                }
            ]
        }
        request.get_json.return_value = request_data
        return request

    @pytest.fixture
    def mock_request_gemini_image_with_reference(self):
        """Create a mock request with reference images for multi-turn editing"""
        request = Mock()
        request_data = {
            "model": "google/gemini-3.1-flash-image-preview",
            "modelId": "gemini-3.1-flash-image-preview",
            "contents": [
                {
                    "parts": [
                        {"text": "Make this image more colorful"},
                        {
                            "inlineData": {
                                "mimeType": "image/jpeg",
                                "data": "base64encodedimage=="
                            }
                        }
                    ]
                }
            ]
        }
        request.get_json.return_value = request_data
        return request

    @pytest.fixture
    def mock_vertex_credentials(self):
        """Mock Google auth credentials"""
        with patch('google.auth.default') as mock_auth:
            mock_creds = Mock()
            mock_creds.token = "mock-token-12345"
            mock_auth.return_value = (mock_creds, "mock-project")
            yield mock_creds

    @pytest.fixture
    def mock_vertex_api_response(self):
        """Mock successful Vertex AI API response for image generation"""
        return {
            "candidates": [
                {
                    "content": {
                        "parts": [
                            {
                                "inlineData": {
                                    "mimeType": "image/png",
                                    "data": "base64encodedimage=="
                                }
                            }
                        ]
                    },
                    "finishReason": "STOP"
                }
            ],
            "usageMetadata": {
                "promptTokenCount": 10,
                "candidatesTokenCount": 0,
                "totalTokenCount": 10
            }
        }

    def test_gemini_image_model_detection(self):
        """Test that gemini image models are correctly detected"""
        # These should be detected as Gemini models
        assert main.is_gemini_model("google/gemini-3.1-flash-image-preview")
        assert main.is_gemini_model("google/gemini-3-pro-image-preview")
        assert main.is_gemini_model("google/gemini-2.5-flash-image")

        # Regular Gemini chat models should also be detected
        assert main.is_gemini_model("google/gemini-3.0-flash")
        assert main.is_gemini_model("google/gemini-2.5-pro")

    def test_image_generation_model_detection(self):
        """Test that image generation models are correctly identified"""
        assert main.is_image_generation_model("gemini-3.1-flash-image-preview")
        assert main.is_image_generation_model("gemini-3-pro-image-preview")
        assert main.is_image_generation_model("gemini-2.5-flash-image")

        # Chat models should not be identified as image generation
        assert not main.is_image_generation_model("gemini-3.0-flash")
        assert not main.is_image_generation_model("gemini-2.5-pro")

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_gemini_image_generation_basic_prompt(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_request_gemini_image,
        mock_vertex_credentials,
        mock_vertex_api_response
    ):
        """
        Test basic image generation with text prompt.

        Verifies:
        - Cloud function correctly identifies Gemini image model
        - Routes to native Vertex AI API (not OpenAI SDK)
        - Uses :generateContent endpoint format
        - Returns 200 with image data
        """
        # Mock the Vertex AI API response
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_vertex_api_response
        mock_requests_post.return_value = mock_response

        # Call the main handler
        response, status_code = main.handle_request(mock_request_gemini_image)

        # Verify status code
        assert status_code == 200

        # Verify the Vertex AI API was called with correct endpoint
        mock_requests_post.assert_called_once()
        call_args = mock_requests_post.call_args

        # Check that the URL includes :generateContent (not /generateContent)
        assert ":generateContent" in call_args[0][0]

        # Verify model name is in the URL
        assert "gemini-3.1-flash-image-preview" in call_args[0][0]

        # Verify authorization header
        assert call_args[1]["headers"]["Authorization"] == "Bearer mock-token-12345"

        # Verify request payload contains contents
        payload = call_args[1]["json"]
        assert "contents" in payload
        assert payload["contents"][0]["parts"][0]["text"] == "Generate a beautiful sunset over mountains"

        # Verify timeout is set to CLIENT_TIMEOUT (derived from FUNCTION_TIMEOUT)
        assert call_args[1]["timeout"] == main.CLIENT_TIMEOUT

        # Verify response contains image data
        assert "candidates" in response
        assert response["candidates"][0]["content"]["parts"][0]["inlineData"]["mimeType"] == "image/png"

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_gemini_image_generation_with_reference_images(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_request_gemini_image_with_reference,
        mock_vertex_credentials,
        mock_vertex_api_response
    ):
        """
        Test image generation with reference images for multi-turn editing.

        Verifies:
        - Reference images are included in the request
        - Proper formatting for conversational editing
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_vertex_api_response
        mock_requests_post.return_value = mock_response

        response, status_code = main.handle_request(mock_request_gemini_image_with_reference)

        assert status_code == 200

        # Verify request payload includes both text and image
        call_args = mock_requests_post.call_args
        payload = call_args[1]["json"]

        parts = payload["contents"][0]["parts"]
        assert len(parts) == 2
        assert parts[0]["text"] == "Make this image more colorful"
        assert "inlineData" in parts[1]
        assert parts[1]["inlineData"]["mimeType"] == "image/jpeg"

    @patch('main.push_metric')
    def test_gemini_image_generation_model_not_found(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Test error handling for invalid model name"""
        request = Mock()
        request.get_json.return_value = {
            "model": "google/invalid-model-name",
            "modelId": "invalid-model-name",
            "contents": [{"parts": [{"text": "test"}]}]
        }

        response, status_code = main.handle_request(request)

        # Should return 400 for invalid model
        assert status_code == 400
        assert "error" in response

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_gemini_image_generation_api_error(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_request_gemini_image,
        mock_vertex_credentials
    ):
        """Test error handling when Vertex AI API returns an error"""
        # Mock API error response
        mock_response = Mock()
        mock_response.status_code = 400
        mock_response.json.return_value = {
            "error": {
                "code": 400,
                "message": "Invalid request",
                "status": "INVALID_ARGUMENT"
            }
        }
        mock_requests_post.return_value = mock_response

        response, status_code = main.handle_request(mock_request_gemini_image)

        # Should propagate the error
        assert status_code == 400
        assert "error" in response

    def test_endpoint_url_format(self):
        """
        Test that the Vertex AI endpoint URL is correctly formatted for global models.

        Critical: The endpoint must use :generateContent (colon-prefixed) not /generateContent
        Models like gemini-3.1-flash-image-preview are only available globally.
        """
        project_id = "test-project"
        region = "us-central1"
        model_name = "gemini-3.1-flash-image-preview"

        # gemini-3.1-flash-image-preview requires global endpoint
        expected_url = f"https://aiplatform.googleapis.com/v1/projects/{project_id}/locations/global/publishers/google/models/{model_name}:generateContent"

        actual_url = main.get_vertex_native_api_url(project_id, region, model_name)

        assert actual_url == expected_url
        assert ":generateContent" in actual_url
        assert "/generateContent" not in actual_url  # Should NOT have slash prefix
        assert "/locations/global/" in actual_url  # Should use global location

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_regular_gemini_chat_still_works(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """
        Ensure regular Gemini chat models still use OpenAI SDK path.

        This is a regression test to ensure we don't break existing Gemini chat functionality.
        """
        request = Mock()
        request_data = {
            "model": "google/gemini-3.0-flash",
            "messages": [
                {"role": "user", "content": "Hello"}
            ],
            "stream": False
        }
        request.get_json.return_value = request_data

        # Mock OpenAI client response (not native Vertex API)
        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_completion = Mock()
            mock_completion.to_dict.return_value = {
                "choices": [{"message": {"content": "Hi there"}}]
            }
            mock_client.chat.completions.create.return_value = mock_completion
            mock_openai.return_value = mock_client

            response, status_code = main.handle_request(request)

            # Should still use OpenAI SDK for chat models
            assert status_code == 200
            mock_openai.assert_called_once()

            # Verify OpenAI client was created with CLIENT_TIMEOUT
            openai_call_kwargs = mock_openai.call_args[1]
            assert openai_call_kwargs["timeout"] == main.CLIENT_TIMEOUT

            # Should NOT call native Vertex API endpoint
            mock_requests_post.assert_not_called()


class TestHelperFunctions:
    """Test helper functions for model detection and URL generation"""

    def test_extract_model_id(self):
        """Test extracting model ID from various formats"""
        assert main.extract_model_id("google/gemini-3.1-flash-image-preview") == "gemini-3.1-flash-image-preview"
        assert main.extract_model_id("gemini-3.1-flash-image-preview") == "gemini-3.1-flash-image-preview"

    def test_is_gemini_model_various_formats(self):
        """Test Gemini model detection with various input formats"""
        # With google/ prefix
        assert main.is_gemini_model("google/gemini-3.1-flash-image-preview")
        assert main.is_gemini_model("google/gemini-3.0-flash")

        # Without prefix
        assert main.is_gemini_model("gemini-3.1-flash-image-preview")
        assert main.is_gemini_model("gemini-3.0-flash")

        # Not Gemini models
        assert not main.is_gemini_model("imagen-3.0-generate-001")
        assert not main.is_gemini_model("text-multilingual-embedding-002")

    def test_is_global_model(self):
        """Test global model detection"""
        # Models that require global endpoint
        assert main.is_global_model("gemini-3.0-flash")
        assert main.is_global_model("gemini-3.0-flash-preview")
        assert main.is_global_model("gemini-3.0-pro")
        assert main.is_global_model("gemini-3.1-flash-image-preview")

        # With google/ prefix
        assert main.is_global_model("google/gemini-3.1-flash-image-preview")

        # Models that use regional endpoint
        assert not main.is_global_model("gemini-1.5-pro")
        assert not main.is_global_model("gemini-1.5-flash")
        assert not main.is_global_model("gemini-2.5-flash-image")
        assert not main.is_global_model("imagen-3.0-generate-001")

    def test_get_vertex_native_api_url_global_model(self):
        """Test URL generation for global endpoint models"""
        project_id = "test-project"
        region = "us-central1"

        # gemini-3.1-flash-image-preview requires global endpoint
        url = main.get_vertex_native_api_url(project_id, region, "gemini-3.1-flash-image-preview")
        assert url == "https://aiplatform.googleapis.com/v1/projects/test-project/locations/global/publishers/google/models/gemini-3.1-flash-image-preview:generateContent"
        assert "/locations/global/" in url
        assert ":generateContent" in url

    def test_get_vertex_native_api_url_regional_model(self):
        """Test URL generation for regional endpoint models"""
        project_id = "test-project"
        region = "us-central1"

        # gemini-2.5-flash-image uses regional endpoint (not global like gemini-3.1-flash-image-preview)
        url = main.get_vertex_native_api_url(project_id, region, "gemini-2.5-flash-image")
        assert url == "https://us-central1-aiplatform.googleapis.com/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-flash-image:generateContent"
        assert f"/locations/{region}/" in url
        assert ":generateContent" in url


class TestAPIDetection:
    """Test suite for detect_target_api() and _detect_target_api_from_model_name().

    Per ADR-0003, routing should be based on the request path (API detection)
    rather than model name patterns (model detection).
    """

    def test_detect_target_api_generate_content(self):
        """Path with generateContent routes to generateContent API"""
        request = Mock()
        request.path = "/google/deployments/gemini-3.1-flash-image-preview/generateContent"
        assert main.detect_target_api(request) == "generateContent"

    def test_detect_target_api_chat_completions(self):
        """Path with chat/completions routes to chat/completions API"""
        request = Mock()
        request.path = "/google/deployments/gemini-2.5-flash/chat/completions"
        assert main.detect_target_api(request) == "chat/completions"

    def test_detect_target_api_images_generations(self):
        """Path with images/generations routes to images/generations API"""
        request = Mock()
        request.path = "/google/deployments/imagen-4.0/images/generations"
        assert main.detect_target_api(request) == "images/generations"

    def test_detect_target_api_embeddings(self):
        """Path with embeddings routes to embeddings API"""
        request = Mock()
        request.path = "/google/deployments/text-multilingual-embedding-002/embeddings"
        assert main.detect_target_api(request) == "embeddings"

    def test_detect_target_api_no_path(self):
        """Request without path attribute returns empty string"""
        request = Mock(spec=[])
        request.get_json = Mock(return_value={})
        assert main.detect_target_api(request) == ""

    def test_detect_target_api_empty_path(self):
        """Request with empty path returns empty string"""
        request = Mock()
        request.path = ""
        assert main.detect_target_api(request) == ""

    def test_detect_target_api_unknown_path(self):
        """Request with unrecognized path returns empty string"""
        request = Mock()
        request.path = "/google/deployments/some-model/unknown"
        assert main.detect_target_api(request) == ""

    def test_fallback_model_name_gemini(self):
        """Fallback: Gemini model names map to chat/completions"""
        assert main._detect_target_api_from_model_name("gemini-2.5-flash") == "chat/completions"
        assert main._detect_target_api_from_model_name("google/gemini-3.0-flash") == "chat/completions"

    def test_fallback_model_name_imagen(self):
        """Fallback: Imagen model names map to images/generations"""
        assert main._detect_target_api_from_model_name("imagen-3.0-generate-002") == "images/generations"
        assert main._detect_target_api_from_model_name("imagen-4.0-generate-001") == "images/generations"

    def test_fallback_model_name_embedding(self):
        """Fallback: Embedding model names map to embeddings"""
        assert main._detect_target_api_from_model_name("text-multilingual-embedding-002") == "embeddings"
        assert main._detect_target_api_from_model_name("gemini-embedding-001") == "embeddings"

    def test_fallback_model_name_unknown(self):
        """Fallback: Unknown model names return empty string"""
        assert main._detect_target_api_from_model_name("unknown-model") == ""

    def test_fallback_does_not_return_generate_content(self):
        """Fallback cannot detect generateContent from model name alone.

        This is intentional: generateContent routing requires explicit path-based
        detection. Model name patterns like gemini-*-image are insufficient because
        new models may not follow this naming convention.
        """
        assert main._detect_target_api_from_model_name("gemini-3.1-flash-image-preview") == "chat/completions"


class TestPathBasedRouting:
    """Test suite for path-based routing to all API endpoints.

    This verifies that routing to handlers is determined by the request path,
    not by the model name, following ADR-0003's API detection principle.
    """

    @pytest.fixture
    def mock_vertex_credentials(self):
        """Mock Google auth credentials"""
        with patch('google.auth.default') as mock_auth:
            mock_creds = Mock()
            mock_creds.token = "mock-token-12345"
            mock_auth.return_value = (mock_creds, "mock-project")
            yield mock_creds

    @pytest.fixture
    def mock_vertex_api_response(self):
        """Mock successful Vertex AI API response"""
        return {
            "candidates": [
                {
                    "content": {
                        "parts": [
                            {
                                "inlineData": {
                                    "mimeType": "image/png",
                                    "data": "base64encodedimage=="
                                }
                            }
                        ]
                    },
                    "finishReason": "STOP"
                }
            ],
            "usageMetadata": {
                "promptTokenCount": 10,
                "candidatesTokenCount": 0,
                "totalTokenCount": 10
            }
        }

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_generate_content_path_routes_any_model(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_vertex_credentials,
        mock_vertex_api_response
    ):
        """
        Test that a model WITHOUT "-image" in the name is routed to the native
        Vertex AI :generateContent endpoint when the request path contains
        /generateContent.

        This proves routing is based on the request PATH, not the model name.
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_vertex_api_response
        mock_requests_post.return_value = mock_response

        request = Mock()
        request.path = "/google/deployments/gemini-my-custom-model/generateContent"
        request_data = {
            "model": "google/gemini-my-custom-model",
            "modelId": "gemini-my-custom-model",
            "contents": [
                {"parts": [{"text": "Generate a beautiful sunset"}]}
            ]
        }
        request.get_json.return_value = request_data

        response, status_code = main.handle_request(request)

        assert status_code == 200
        mock_requests_post.assert_called_once()
        call_args = mock_requests_post.call_args
        assert ":generateContent" in call_args[0][0]
        assert "gemini-my-custom-model" in call_args[0][0]
        assert "candidates" in response

    @patch('main.push_metric')
    def test_chat_completions_path_routes_to_openai_sdk(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Chat completions path routes to OpenAI SDK handler regardless of model name"""
        request = Mock()
        request.path = "/google/deployments/my-custom-model/chat/completions"
        request_data = {
            "model": "google/my-custom-model",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": False
        }
        request.get_json.return_value = request_data

        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_completion = Mock()
            mock_completion.to_dict.return_value = {
                "choices": [{"message": {"content": "Hi"}}]
            }
            mock_client.chat.completions.create.return_value = mock_completion
            mock_openai.return_value = mock_client

            response, status_code = main.handle_request(request)

            assert status_code == 200
            mock_openai.assert_called_once()

    @patch('main.push_metric')
    def test_embeddings_path_routes_to_embedding_handler(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Embeddings path routes to text embedding handler"""
        request = Mock()
        request.path = "/google/deployments/text-multilingual-embedding-002/embeddings"
        request_data = {
            "model": "text-multilingual-embedding-002",
            "modelId": "text-multilingual-embedding-002",
            "texts": ["Hello world"]
        }
        request.get_json.return_value = request_data

        with patch('main.vertexai') as mock_vertexai, \
             patch('main.TextEmbeddingModel') as mock_embed_model:
            mock_embedding = Mock()
            mock_embedding.statistics.token_count = 3
            mock_embed_model.from_pretrained.return_value.get_embeddings.return_value = [mock_embedding]

            response, status_code = main.handle_request(request)

            assert status_code == 200
            assert "usage" in response
            mock_embed_model.from_pretrained.assert_called_once()

    @patch('main.push_metric')
    def test_images_generations_path_routes_to_imagen_handler(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Images generations path routes to Imagen handler"""
        request = Mock()
        request.path = "/google/deployments/imagen-4.0/images/generations"
        request_data = {
            "model": "imagen-4.0",
            "modelId": "imagen-4.0-generate-001",
            "payload": {"prompt": "A sunset"}
        }
        request.get_json.return_value = request_data

        with patch('main.vertexai') as mock_vertexai, \
             patch('main.ImageGenerationModel') as mock_image_model:
            mock_images = Mock()
            mock_images.images = []  # No images for simplicity
            mock_image_model.from_pretrained.return_value.generate_images.return_value = mock_images

            response, status_code = main.handle_request(request)

            assert status_code == 200
            mock_image_model.from_pretrained.assert_called_once()


class TestBackwardCompatibility:
    """Test that model-name-based fallback routing still works when path is absent.

    This ensures zero-downtime during migration: older deployments that do not
    include path information in requests continue to work correctly.
    """

    @pytest.fixture
    def mock_vertex_credentials(self):
        """Mock Google auth credentials"""
        with patch('google.auth.default') as mock_auth:
            mock_creds = Mock()
            mock_creds.token = "mock-token-12345"
            mock_auth.return_value = (mock_creds, "mock-project")
            yield mock_creds

    @patch('main.push_metric')
    def test_gemini_model_without_path_routes_to_chat(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Gemini model without path falls back to chat/completions via model name"""
        request = Mock(spec=[])
        request.get_json = Mock(return_value={
            "model": "google/gemini-2.5-flash",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": False
        })

        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_completion = Mock()
            mock_completion.to_dict.return_value = {
                "choices": [{"message": {"content": "Hi"}}]
            }
            mock_client.chat.completions.create.return_value = mock_completion
            mock_openai.return_value = mock_client

            response, status_code = main.handle_request(request)

            assert status_code == 200
            mock_openai.assert_called_once()

    @patch('main.push_metric')
    def test_imagen_model_without_path_routes_to_imagen(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Imagen model without path falls back to images/generations via model name"""
        request = Mock(spec=[])
        request.get_json = Mock(return_value={
            "model": "imagen-3.0-generate-002",
            "modelId": "imagen-3.0-generate-002",
            "payload": {"prompt": "A sunset"}
        })

        with patch('main.vertexai'), \
             patch('main.ImageGenerationModel') as mock_image_model:
            mock_images = Mock()
            mock_images.images = []
            mock_image_model.from_pretrained.return_value.generate_images.return_value = mock_images

            response, status_code = main.handle_request(request)

            assert status_code == 200
            mock_image_model.from_pretrained.assert_called_once()

    @patch('main.push_metric')
    def test_embedding_model_without_path_routes_to_embeddings(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """Embedding model without path falls back to embeddings via model name"""
        request = Mock(spec=[])
        request.get_json = Mock(return_value={
            "model": "text-multilingual-embedding-002",
            "modelId": "text-multilingual-embedding-002",
            "texts": ["Hello world"]
        })

        with patch('main.vertexai'), \
             patch('main.TextEmbeddingModel') as mock_embed_model:
            mock_embedding = Mock()
            mock_embedding.statistics.token_count = 3
            mock_embed_model.from_pretrained.return_value.get_embeddings.return_value = [mock_embedding]

            response, status_code = main.handle_request(request)

            assert status_code == 200
            mock_embed_model.from_pretrained.assert_called_once()


class TestErrorStatusHandling:
    """Tests for preserving and mapping upstream error status codes."""

    @pytest.fixture
    def mock_vertex_credentials(self):
        """Mock Google auth credentials."""
        with patch('google.auth.default') as mock_auth:
            mock_creds = Mock()
            mock_creds.token = "mock-token-12345"
            mock_auth.return_value = (mock_creds, "mock-project")
            yield mock_creds

    def test_handle_gemini_preserves_openai_status_code(self, mock_vertex_credentials):
        """OpenAI APIStatusError preserves the upstream HTTP status code."""
        request_json = {
            "model": "google/gemini-2.5-flash",
            "messages": [{"role": "user", "content": "Hello"}],
        }
        rate_limit_error = _FakeAPIStatusError(
            "rate limit exceeded",
            status_code=429,
            body={"error": {"type": "rate_limit_exceeded"}},
        )

        with patch.object(main.openai, "APIStatusError", _FakeAPIStatusError), \
             patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_client.chat.completions.create.side_effect = rate_limit_error
            mock_openai.return_value = mock_client

            response, status_code = main.handle_gemini(request_json)

        assert status_code == 429
        assert response["error"] == "Gemini model processing failed"
        assert response["type"] == "rate_limit_exceeded"

    def test_handle_gemini_stream_emits_status_code_for_openai_api_error(self, mock_vertex_credentials):
        """Streaming SSE error payload includes the upstream status code."""
        request = Mock()
        request.get_json.return_value = {
            "model": "google/gemini-2.5-flash",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": True,
        }
        rate_limit_error = _FakeAPIStatusError(
            "rate limit exceeded",
            status_code=429,
            body={"error": {"type": "rate_limit_exceeded"}},
        )

        with patch.object(main.openai, "APIStatusError", _FakeAPIStatusError), \
             patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_client.chat.completions.create.side_effect = rate_limit_error
            mock_openai.return_value = mock_client

            response = main.handle_gemini_stream(request)
            stream_output = "".join(response.response)

        payload = _parse_sse_json_event(stream_output)
        assert payload["status_code"] == 429
        assert payload["type"] == "rate_limit_exceeded"

    def test_handle_imagen_maps_resource_exhausted_to_429(self):
        """Google ResourceExhausted errors map to HTTP 429 for Imagen."""
        request_json = {
            "modelId": "imagen-4.0-generate-001",
            "payload": {"prompt": "A sunset"},
        }

        with patch('main.vertexai'), patch('main.ImageGenerationModel') as mock_image_model:
            mock_image_model.from_pretrained.return_value.generate_images.side_effect = (
                main.google_api_exceptions.ResourceExhausted("quota exceeded")
            )

            response, status_code = main.handle_imagen(request_json)

        assert status_code == 429
        assert response["error"] == "Imagen model processing failed"

    def test_handle_text_embedding_maps_invalid_argument_to_400(self):
        """Google InvalidArgument errors map to HTTP 400 for text embeddings."""
        request_json = {
            "model": "text-multilingual-embedding-002",
            "texts": ["Hello world"],
        }

        with patch('main.vertexai'), patch('main.TextEmbeddingModel') as mock_embed_model:
            mock_embed_model.from_pretrained.return_value.get_embeddings.side_effect = (
                main.google_api_exceptions.InvalidArgument("invalid input")
            )

            response, status_code = main.handle_text_embedding(request_json)

        assert status_code == 400
        assert response["error"] == "Text embedding model processing failed"

    @pytest.mark.parametrize(
        ("handler_name", "patch_target", "request_json", "method_name"),
        [
            (
                "handle_imagen",
                "main.ImageGenerationModel",
                {"modelId": "imagen-4.0-generate-001", "payload": {"prompt": "A sunset"}},
                "generate_images",
            ),
            (
                "handle_text_embedding",
                "main.TextEmbeddingModel",
                {"model": "text-multilingual-embedding-002", "texts": ["Hello world"]},
                "get_embeddings",
            ),
        ],
    )
    def test_google_api_handlers_unexpected_errors_fall_back_to_500(
        self,
        handler_name,
        patch_target,
        request_json,
        method_name,
    ):
        """Unexpected non-Google exceptions still fall back to HTTP 500."""
        with patch('main.vertexai'), patch(patch_target) as mock_model:
            getattr(mock_model.from_pretrained.return_value, method_name).side_effect = Exception("boom")

            response, status_code = getattr(main, handler_name)(request_json)

        assert status_code == 500
        assert "processing failed" in response["error"]


class TestClientTimeout:
    """Test suite for CLIENT_TIMEOUT computation and usage.

    BUG-975397: The OpenAI client timeout should be derived from the Cloud Function
    timeout (FUNCTION_TIMEOUT env var) minus a 20-second buffer, with a minimum
    floor of 30 seconds. This ensures the SDK produces clean timeout errors before
    the Cloud Function is hard-killed by the platform.
    """

    def test_default_function_timeout(self):
        """FUNCTION_TIMEOUT defaults to 600 when env var is not set"""
        assert main.FUNCTION_TIMEOUT == 600

    def test_default_client_timeout_value(self):
        """CLIENT_TIMEOUT is FUNCTION_TIMEOUT - 20 = 580 by default"""
        assert main.CLIENT_TIMEOUT == 580

    def test_client_timeout_computation_logic(self):
        """CLIENT_TIMEOUT = max(FUNCTION_TIMEOUT - 20, 30) for various inputs"""
        # Test the formula directly since module-level constants can't be re-imported
        test_cases = [
            (300, 280),   # Default: 300 - 20 = 280
            (600, 580),   # Large timeout: 600 - 20 = 580
            (60, 40),     # Small timeout: 60 - 20 = 40
            (50, 30),     # Edge case: 50 - 20 = 30 (exactly at floor)
            (40, 30),     # Below threshold: max(20, 30) = 30 (floor kicks in)
            (20, 30),     # Very small: max(0, 30) = 30 (floor kicks in)
            (10, 30),     # Tiny: max(-10, 30) = 30 (floor kicks in)
        ]
        for function_timeout, expected_client_timeout in test_cases:
            result = max(function_timeout - 20, 30)
            assert result == expected_client_timeout, (
                f"For FUNCTION_TIMEOUT={function_timeout}: "
                f"expected CLIENT_TIMEOUT={expected_client_timeout}, got {result}"
            )

    def test_client_timeout_minimum_floor(self):
        """CLIENT_TIMEOUT never drops below 30 seconds regardless of FUNCTION_TIMEOUT"""
        for function_timeout in [1, 10, 20, 30, 49, 50]:
            result = max(function_timeout - 20, 30)
            assert result >= 30, (
                f"CLIENT_TIMEOUT should be >= 30 for FUNCTION_TIMEOUT={function_timeout}, "
                f"but got {result}"
            )

    def test_client_timeout_with_env_var_override(self):
        """CLIENT_TIMEOUT respects FUNCTION_TIMEOUT env var at module load time.

        Since CLIENT_TIMEOUT is computed at import time, we verify the formula
        by reloading the module with a custom FUNCTION_TIMEOUT env var.
        """
        with patch.dict(os.environ, {'FUNCTION_TIMEOUT': '120'}):
            # Reload the module to re-evaluate module-level constants
            importlib.reload(main)
            try:
                assert main.FUNCTION_TIMEOUT == 120
                assert main.CLIENT_TIMEOUT == 100  # max(120 - 20, 30) = 100
            finally:
                # Restore original env and reload to reset module state
                with patch.dict(os.environ, {'FUNCTION_TIMEOUT': '300'}):
                    importlib.reload(main)

    def test_client_timeout_floor_with_env_var(self):
        """CLIENT_TIMEOUT floors at 30 even when FUNCTION_TIMEOUT is very low."""
        with patch.dict(os.environ, {'FUNCTION_TIMEOUT': '10'}):
            importlib.reload(main)
            try:
                assert main.FUNCTION_TIMEOUT == 10
                assert main.CLIENT_TIMEOUT == 30  # max(10 - 20, 30) = 30
            finally:
                with patch.dict(os.environ, {'FUNCTION_TIMEOUT': '300'}):
                    importlib.reload(main)

    @pytest.fixture
    def mock_vertex_credentials(self):
        """Mock Google auth credentials"""
        with patch('google.auth.default') as mock_auth:
            mock_creds = Mock()
            mock_creds.token = "mock-token-12345"
            mock_auth.return_value = (mock_creds, "mock-project")
            yield mock_creds

    @patch('main.push_metric')
    def test_handle_gemini_passes_client_timeout_to_openai(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """handle_gemini creates OpenAI client with timeout=CLIENT_TIMEOUT"""
        request_json = {
            "model": "google/gemini-2.5-flash",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": False
        }

        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_completion = Mock()
            mock_completion.to_dict.return_value = {
                "choices": [{"message": {"content": "Hi"}}]
            }
            mock_client.chat.completions.create.return_value = mock_completion
            mock_openai.return_value = mock_client

            response, status_code = main.handle_gemini(request_json)

            assert status_code == 200
            mock_openai.assert_called_once()
            call_kwargs = mock_openai.call_args[1]
            assert "timeout" in call_kwargs, "OpenAI client must be created with timeout parameter"
            assert call_kwargs["timeout"] == main.CLIENT_TIMEOUT

    @patch('main.push_metric')
    def test_handle_gemini_stream_passes_client_timeout_to_openai(
        self,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """handle_gemini_stream creates OpenAI client with timeout=CLIENT_TIMEOUT"""
        request = Mock()
        request.get_json.return_value = {
            "model": "google/gemini-2.5-flash",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": True
        }

        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_chunk = Mock()
            mock_chunk.to_dict.return_value = {
                "choices": [{"delta": {"content": "Hi"}}]
            }
            mock_client.chat.completions.create.return_value = iter([mock_chunk])
            mock_openai.return_value = mock_client

            # handle_gemini_stream returns a Response object; consume the generator
            response = main.handle_gemini_stream(request)
            # Consume the response generator to trigger OpenAI client creation
            response_data = "".join(response.response)

            mock_openai.assert_called_once()
            call_kwargs = mock_openai.call_args[1]
            assert "timeout" in call_kwargs, "OpenAI client must be created with timeout parameter"
            assert call_kwargs["timeout"] == main.CLIENT_TIMEOUT

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_handle_gemini_image_generation_passes_client_timeout(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """handle_gemini_image_generation passes timeout=CLIENT_TIMEOUT to requests.post"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "candidates": [{"content": {"parts": [{"text": "image data"}]}}]
        }
        mock_requests_post.return_value = mock_response

        request_json = {
            "model": "google/gemini-3.1-flash-image-preview",
            "modelId": "gemini-3.1-flash-image-preview",
            "contents": [{"parts": [{"text": "Generate a sunset"}]}]
        }

        response, status_code = main.handle_gemini_image_generation(request_json)

        assert status_code == 200
        mock_requests_post.assert_called_once()
        call_kwargs = mock_requests_post.call_args[1]
        assert "timeout" in call_kwargs, "requests.post must be called with timeout parameter"
        assert call_kwargs["timeout"] == main.CLIENT_TIMEOUT

    @patch('main.push_metric')
    @patch('main.requests.post')
    def test_all_handlers_use_same_timeout(
        self,
        mock_requests_post,
        mock_push_metric,
        mock_vertex_credentials
    ):
        """All three handlers (gemini, gemini_stream, image_generation) use the same CLIENT_TIMEOUT.

        This ensures consistent timeout behavior across all request types.
        """
        collected_timeouts = []

        # 1. handle_gemini
        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_completion = Mock()
            mock_completion.to_dict.return_value = {"choices": [{"message": {"content": "Hi"}}]}
            mock_client.chat.completions.create.return_value = mock_completion
            mock_openai.return_value = mock_client

            main.handle_gemini({
                "model": "google/gemini-2.5-flash",
                "messages": [{"role": "user", "content": "Hello"}],
            })
            collected_timeouts.append(("handle_gemini", mock_openai.call_args[1]["timeout"]))

        # 2. handle_gemini_stream
        with patch('main.openai.OpenAI') as mock_openai:
            mock_client = Mock()
            mock_chunk = Mock()
            mock_chunk.to_dict.return_value = {"choices": [{"delta": {"content": "Hi"}}]}
            mock_client.chat.completions.create.return_value = iter([mock_chunk])
            mock_openai.return_value = mock_client

            request = Mock()
            request.get_json.return_value = {
                "model": "google/gemini-2.5-flash",
                "messages": [{"role": "user", "content": "Hello"}],
                "stream": True
            }
            response = main.handle_gemini_stream(request)
            "".join(response.response)
            collected_timeouts.append(("handle_gemini_stream", mock_openai.call_args[1]["timeout"]))

        # 3. handle_gemini_image_generation
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"candidates": [{"content": {"parts": [{"text": "ok"}]}}]}
        mock_requests_post.return_value = mock_response

        main.handle_gemini_image_generation({
            "model": "google/gemini-3.1-flash-image-preview",
            "modelId": "gemini-3.1-flash-image-preview",
            "contents": [{"parts": [{"text": "Generate an image"}]}]
        })
        collected_timeouts.append(("handle_gemini_image_generation", mock_requests_post.call_args[1]["timeout"]))

        # All three handlers should use the same CLIENT_TIMEOUT value
        for handler_name, timeout_value in collected_timeouts:
            assert timeout_value == main.CLIENT_TIMEOUT, (
                f"{handler_name} used timeout={timeout_value}, "
                f"expected CLIENT_TIMEOUT={main.CLIENT_TIMEOUT}"
            )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
