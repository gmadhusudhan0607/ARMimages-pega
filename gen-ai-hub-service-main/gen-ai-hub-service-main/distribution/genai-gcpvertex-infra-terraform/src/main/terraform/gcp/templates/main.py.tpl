# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.

import functions_framework
import os
import google
from google.api_core import exceptions as google_api_exceptions
from google.auth.transport.requests import Request
import openai
from vertexai.preview.vision_models import ImageGenerationModel
import vertexai
import random
import string
import base64
import json
import requests
import time
from google.cloud import monitoring_v3
from google.protobuf.timestamp_pb2 import Timestamp
from vertexai.language_models import TextEmbeddingModel
from typing import AsyncGenerator, Tuple, Dict, Any
import asyncio
from werkzeug.wrappers import Response

# Model prefixes that require global endpoint (not regional)
# Some models like gemini-3.0-flash are only available globally
# Using prefixes allows matching model families (e.g., gemini-3.0-flash, gemini-3.0-flash-preview)
GLOBAL_ENDPOINT_MODEL_PREFIXES = [
    "gemini-3.0-flash",
    "gemini-3.0-flash-preview",
    "gemini-3.0-pro",
    "gemini-3.0-pro-preview",
    "gemini-3.1-flash-image",  # Gemini image generation models (Nano Banana)
    # Add more model prefixes as needed when they require global endpoints
]

def get_project_id():
    """
    Retrieves the GCP project ID from Cloud Run metadata service.

    Returns:
        str: The GCP project ID where this function is deployed
    """
    response = requests.get("http://metadata.google.internal/computeMetadata/v1/project/project-id", headers={"Metadata-Flavor": "Google"})
    return response.text.strip()

def get_location():
    """
    Determines the GCP region for Vertex AI API calls.

    Priority:
        1. VERTEX_INFERENCE_REGION environment variable (if set)
        2. Instance region from Cloud Run metadata service

    Returns:
        str: The GCP region (e.g., 'us-central1')
    """
    # Check for environment variable first
    vertex_region = os.environ.get('VERTEX_INFERENCE_REGION')
    if vertex_region and vertex_region.strip():
        return vertex_region.strip()

    # Fall back to existing behavior
    response = requests.get("http://metadata.google.internal/computeMetadata/v1/instance/region", headers={"Metadata-Flavor": "Google"})
    return response.text.split('/')[-1].strip()

project_id = get_project_id()
region = get_location()

def get_google_api_status_code(error: Exception) -> int:
    """
    Map Google API exceptions to HTTP status codes.

    Args:
        error: Exception raised by the Google API client.

    Returns:
        HTTP status code for the known exception type, or 500 when unmapped.
    """
    for exception_type, status_code in (
        (google_api_exceptions.InvalidArgument, 400),
        (google_api_exceptions.Unauthenticated, 401),
        (google_api_exceptions.PermissionDenied, 403),
        (google_api_exceptions.NotFound, 404),
        (google_api_exceptions.ResourceExhausted, 429),
        (google_api_exceptions.TooManyRequests, 429),
        (google_api_exceptions.InternalServerError, 500),
        (google_api_exceptions.ServiceUnavailable, 503),
    ):
        if isinstance(error, exception_type):
            return status_code
    return 500

def get_openai_error_type(error: openai.APIStatusError) -> str | None:
    """
    Extract the upstream error type from an OpenAI APIStatusError body.

    Args:
        error: APIStatusError returned by the OpenAI client.

    Returns:
        The upstream error type when present, otherwise None.
    """
    body = getattr(error, "body", None)
    if isinstance(body, dict):
        nested_error = body.get("error")
        if isinstance(nested_error, dict) and nested_error.get("type"):
            return nested_error["type"]
        if body.get("type"):
            return body["type"]
    return None

def get_function_timeout(default: int = 600) -> int:
    """
    Parse FUNCTION_TIMEOUT from the environment with a safe fallback.

    Returns:
        int: Parsed timeout in seconds, or the default when unset/invalid.
    """
    raw_value = os.environ.get('FUNCTION_TIMEOUT')
    if raw_value is None or not raw_value.strip():
        return default

    try:
        return int(raw_value)
    except (TypeError, ValueError):
        print(f"Invalid FUNCTION_TIMEOUT value {raw_value!r}; using default {default}.")
        return default

# Client timeout derived from Cloud Function timeout.
# Set lower than CF timeout to produce clean SDK errors before CF hard-kills.
FUNCTION_TIMEOUT = get_function_timeout()
CLIENT_TIMEOUT = max(FUNCTION_TIMEOUT - 20, 30)

if not project_id:
    raise Exception('Failed to load Google ProjectId from Cloud Run Function Runtime Metadata. Review Google Cloud Project configuration.')

if not region:
    raise Exception('Failed to load Google Region from Cloud Run Function Metadata. Review Google Cloud Project configuration.')

def extract_model_id(model_name: str) -> str:
    """
    Extract model ID from full model name by removing provider prefix.

    Examples:
        "google/gemini-3.1-flash-image-preview" -> "gemini-3.1-flash-image-preview"
        "gemini-3.1-flash-image-preview" -> "gemini-3.1-flash-image-preview"
    """
    return model_name.replace("google/", "") if model_name.startswith("google/") else model_name

def is_global_model(model_name: str) -> bool:
    """
    Checks if a model requires global endpoint based on prefix matching.
    """
    model_id = extract_model_id(model_name)

    for prefix in GLOBAL_ENDPOINT_MODEL_PREFIXES:
        if model_id.startswith(prefix):
            return True
    return False

def is_gemini_model(model_name: str) -> bool:
    """
    Check if model is a Gemini model (chat or image generation).

    Returns True for any model starting with "gemini" after removing provider prefix.
    Examples: gemini-3.0-flash, gemini-3.1-flash-image-preview, gemini-2.5-pro
    """
    model_id = extract_model_id(model_name)
    return model_id.startswith("gemini")

def is_image_generation_model(model_name: str) -> bool:
    """
    Check if model is a Gemini image generation model by name.

    Image generation models (Nano Banana) have "-image" in their name and are Gemini models.

    Note: This function checks model NAME only. For routing decisions, prefer
    is_generate_content_request() which checks the request PATH and is more
    flexible (works with any Gemini model, not just those with "-image" in name).

    Examples:
        - gemini-3.1-flash-image-preview (Nano Banana 2)
        - gemini-3-pro-image-preview (Nano Banana Pro)
        - gemini-2.5-flash-image (Nano Banana)
    """
    model_id = extract_model_id(model_name)
    return "-image" in model_id and is_gemini_model(model_name)

def detect_target_api(request) -> str:
    """
    Detect the target API from the request path.

    Routes requests based on explicit API path segments rather than model name patterns.
    This is the primary routing mechanism per ADR-0003's "API detection over model detection"
    principle. The request path is set by the ESP proxy based on the model's configured
    targetAPI in the Helm model configuration.

    Path patterns (from api-v3-spec.yaml.tpl):
        /google/deployments/{model}/chat/completions  -> "chat/completions"
        /google/deployments/{model}/generateContent    -> "generateContent"
        /google/deployments/{model}/images/generations -> "images/generations"
        /google/deployments/{model}/embeddings         -> "embeddings"

    Args:
        request: Flask request object with path attribute

    Returns:
        One of: "generateContent", "chat/completions", "images/generations",
        "embeddings", or "" if the API cannot be detected from the path.
    """
    try:
        request_path = getattr(request, 'path', None)
        if not isinstance(request_path, str):
            return ""
    except (TypeError, AttributeError):
        return ""

    if 'generateContent' in request_path:
        return "generateContent"
    if 'chat/completions' in request_path:
        return "chat/completions"
    if 'images/generations' in request_path:
        return "images/generations"
    if 'embeddings' in request_path:
        return "embeddings"

    return ""


def _detect_target_api_from_model_name(model_name: str) -> str:
    """
    Fallback: detect target API from model name patterns.

    Used only when the request path does not contain a recognizable API segment.
    This maintains backward compatibility during the transition period while
    older deployments may not yet include path information.

    Args:
        model_name: Model name (with or without "google/" prefix)

    Returns:
        One of: "chat/completions", "images/generations", "embeddings", or ""
        Note: Does NOT return "generateContent" because model-name-based detection
        cannot reliably distinguish generateContent models from chat models.
    """
    model_id = extract_model_id(model_name)

    if model_id.startswith("text-multilingual-embedding") or model_id.startswith("gemini-embedding"):
        return "embeddings"
    if model_id.startswith("imagen"):
        return "images/generations"
    if model_id.startswith("gemini"):
        return "chat/completions"

    return ""


def is_generate_content_request(request) -> bool:
    """
    Check if the request should be routed to the native Vertex AI :generateContent endpoint.

    Delegates to detect_target_api() for path-based detection per ADR-0003.

    Falls back to model-name-based detection (is_image_generation_model) for backward
    compatibility when path information is not available.

    Args:
        request: Flask request object with path and JSON body

    Returns:
        True if the request should use native Vertex AI :generateContent endpoint
    """
    if detect_target_api(request) == "generateContent":
        return True

    # Fallback: check if request contains 'contents' field (generateContent payload signature)
    # combined with model-name-based detection for backward compatibility.
    # This handles cases where the path is not available (e.g., internal routing).
    try:
        request_json = request.get_json(silent=True)
    except (TypeError, AttributeError):
        request_json = None
    if request_json and 'contents' in request_json and 'messages' not in request_json:
        model_name = request_json.get("model") or request_json.get("modelId", "")
        return is_image_generation_model(model_name)

    return False

def get_vertex_native_api_url(project_id: str, region: str, model_name: str) -> str:
    """
    Generate Vertex AI native API URL for image generation models.

    CRITICAL: Uses :generateContent (colon-prefixed) not /generateContent.
    This is the native Vertex AI API format, different from the OpenAI SDK endpoint.

    Some models (like gemini-3.1-flash-image-preview) are only available globally,
    not in regional endpoints. The function checks is_global_model() and uses the
    appropriate endpoint.

    Format (regional):
        https://{region}-aiplatform.googleapis.com/v1/projects/{project}/locations/{region}/publishers/google/models/{model}:generateContent

    Format (global):
        https://aiplatform.googleapis.com/v1/projects/{project}/locations/global/publishers/google/models/{model}:generateContent

    Args:
        project_id: GCP project ID
        region: GCP region (e.g., us-central1) - used for regional models
        model_name: Model name (with or without "google/" prefix)

    Returns:
        Full Vertex AI native API URL with :generateContent endpoint
    """
    model_id = extract_model_id(model_name)

    if is_global_model(model_name):
        # Global endpoint for models only available globally
        return f"https://aiplatform.googleapis.com/v1/projects/{project_id}/locations/global/publishers/google/models/{model_id}:generateContent"
    else:
        # Regional endpoint for standard models
        return f"https://{region}-aiplatform.googleapis.com/v1/projects/{project_id}/locations/{region}/publishers/google/models/{model_id}:generateContent"

def get_vertex_base_url(model_name: str) -> str:
    """
    Returns the appropriate Vertex AI base URL based on model requirements.
    Some models (like gemini-3.0-flash) are only available globally.
    """
    if is_global_model(model_name):
        return f"https://aiplatform.googleapis.com/v1beta1/projects/{project_id}/locations/global/endpoints/openapi"
    else:
        return f"https://{region}-aiplatform.googleapis.com/v1beta1/projects/{project_id}/locations/{region}/endpoints/openapi"

# Initialize Cloud Monitoring client
client = monitoring_v3.MetricServiceClient()
project_name = f"projects/{project_id}"

def handle_gemini_image_generation(request_json: Dict[str, Any]) -> Tuple[Dict[str, Any], int]:
    """
    Handle Gemini image generation requests using native Vertex AI API.

    Unlike chat models which use OpenAI SDK, image generation models (Nano Banana)
    use the native Vertex AI API because:
    - Different endpoint format (:generateContent with colon-prefix)
    - Supports multi-turn conversational editing
    - Accepts reference images (up to 14)
    - Has different request/response structure

    Args:
        request_json: Request payload with model, contents, and optional parameters

    Returns:
        Tuple of (response_dict, status_code)
    """
    try:
        # Get GCP credentials
        creds, project = google.auth.default()
        auth_req = Request()
        creds.refresh(auth_req)

        # Extract model name
        model_name = request_json.get("model") or request_json.get("modelId")
        if not model_name:
            return {"error": "Model name must be specified in 'model' or 'modelId' field"}, 400

        # Build native Vertex AI URL with :generateContent endpoint
        url = get_vertex_native_api_url(project_id, region, model_name)

        # Prepare request payload - forward contents as-is
        payload = {
            "contents": request_json.get("contents", [])
        }

        # Add optional parameters if present
        if "generationConfig" in request_json:
            payload["generationConfig"] = request_json["generationConfig"]
        if "safetySettings" in request_json:
            payload["safetySettings"] = request_json["safetySettings"]

        # Make request to Vertex AI native API
        response = requests.post(
            url,
            json=payload,
            headers={
                "Authorization": f"Bearer {creds.token}",
                "Content-Type": "application/json"
            },
            timeout=CLIENT_TIMEOUT
        )

        # Return response as-is with original status code
        if response.status_code != 200:
            # Try to parse error response, fallback to text if JSON fails
            try:
                error_body = response.json()
            except:
                error_body = {"error": "Vertex AI API error", "details": response.text}
            return error_body, response.status_code

        return response.json(), 200

    except requests.exceptions.RequestException as e:
        return {"error": "Vertex AI API request failed", "details": str(e)}, 500
    except Exception as e:
        return {"error": "Gemini image generation failed", "details": str(e)}, 500

@functions_framework.http
def handle_request(request):
    """
    Unified function to handle requests for Gemini, Imagen, and Text Embedding models.

    Routing is based on the target API detected from the request path (primary)
    or model name patterns (fallback for backward compatibility). This follows
    ADR-0003's "API detection over model detection" principle.

    Target API routing:
        generateContent    -> handle_gemini_image_generation (native Vertex AI API)
        chat/completions   -> handle_gemini / handle_gemini_stream (OpenAI SDK)
        images/generations -> handle_imagen (Imagen SDK)
        embeddings         -> handle_text_embedding (TextEmbeddingModel)
    """
    response_code = None
    try:
        request_json = request.get_json(silent=True)
        if not request_json:
            response_code = 400
            return {"error": "Request JSON is missing."}, response_code

        model_name = request_json.get("model") or request_json.get("modelId")

        if not model_name:
            response_code = 400
            return {"error": "Model name must be specified in the request JSON under 'model' or 'modelId' key."}, response_code

        # Primary: detect target API from request path (ADR-0003 compliant)
        target_api = detect_target_api(request)

        # Fallback: detect from model name patterns for backward compatibility
        if not target_api:
            target_api = _detect_target_api_from_model_name(model_name)

        # Route to the appropriate handler based on target API
        if target_api == "generateContent":
            response, response_code = handle_gemini_image_generation(request_json)
        elif target_api == "chat/completions":
            if request_json.get("stream", False):
                return handle_gemini_stream(request)
            else:
                response, status_code = handle_gemini(request_json)
                return response, status_code
        elif target_api == "images/generations":
            response, response_code = handle_imagen(request_json)
        elif target_api == "embeddings":
            response, response_code = handle_text_embedding(request_json)
        else:
            response_code = 400
            response = {"error": f"Not supported '{model_name}'"}

        push_metric(
            metric_type="custom.googleapis.com/total_request_count_new",
            value=1,
            labels={
                "model_name": model_name,
                "guide_id": "",
                "response_code": str(response_code)
            }
        )

        return response, response_code

    except Exception as e:
        response_code = 500
        return {"error": "Internal server error", "details": str(e)}, response_code

def handle_gemini(request_json: Dict[str, Any]) -> Tuple[Dict[str, Any], int]:
    """
    Handles requests for the Gemini model.
    """
    try:
        creds, project = google.auth.default()
        auth_req = Request()
        creds.refresh(auth_req)

        model_name = request_json.get("model", "")
        client = openai.OpenAI(
            base_url=get_vertex_base_url(model_name),
            api_key=creds.token,
            timeout=CLIENT_TIMEOUT
        )

        response = client.chat.completions.create(**request_json)
        return response.to_dict(), 200
    except openai.APIStatusError as e:
        error_response = {"error": "Gemini model processing failed", "details": str(e)}
        error_type = get_openai_error_type(e)
        if error_type:
            error_response["type"] = error_type
        return error_response, e.status_code or 500
    except Exception as e:
        return {"error": "Gemini model processing failed", "details": str(e)}, 500

def handle_gemini_stream(request):
    """Handles streaming requests for the Gemini model."""
    def generate():
        try:
            request_json = request.get_json(silent=True)
            creds, project = google.auth.default()
            auth_req = Request()
            creds.refresh(auth_req)

            model_name = request_json.get("model", "")
            client = openai.OpenAI(
                base_url=get_vertex_base_url(model_name),
                api_key=creds.token,
                timeout=CLIENT_TIMEOUT
            )

            response = client.chat.completions.create(**request_json)

            for chunk in response:
                chunk_dict = chunk.to_dict()
                yield "data: " + json.dumps(chunk_dict) + "\n\n"
            yield "data: [DONE]\n\n"

        except openai.APIStatusError as e:
            error_payload = {
                "error": "Gemini model processing failed",
                "details": str(e),
                "status_code": e.status_code or 500
            }
            error_type = get_openai_error_type(e)
            if error_type:
                error_payload["type"] = error_type
            yield "data: " + json.dumps(error_payload) + "\n\n"
        except Exception as e:
            error_payload = {
                "error": "Gemini model processing failed",
                "details": str(e),
                "status_code": 500
            }
            yield "data: " + json.dumps(error_payload) + "\n\n"

    return Response(generate(), mimetype='text/event-stream')

def handle_imagen(request_json: Dict[str, Any]) -> Tuple[Dict[str, Any], int]:
    """
    Handles requests for the Imagen model.
    """
    try:
        vertexai.init(project=project_id, location=region)

        generation_model = ImageGenerationModel.from_pretrained(request_json.get("modelId"))
        images = generation_model.generate_images(**request_json["payload"])

        UUID = ''.join(random.choices(string.ascii_uppercase + string.digits, k=8))

        response = {"predictions": []}
        for idx, image in enumerate(images.images, start=1):
            image_path = f"{UUID}-image-{idx}.png"
            image.save(location=image_path, include_generation_parameters=False)

            with open(image_path, "rb") as image_file:
                img_data = base64.b64encode(image_file.read()).decode('utf-8')

            response["predictions"].append({
                "bytesBase64Encoded": img_data,
                "mimeType": "image/png"
            })

        return response, 200
    except Exception as e:
        return {"error": "Imagen model processing failed", "details": str(e)}, get_google_api_status_code(e)

def handle_text_embedding(request_json: Dict[str, Any]) -> Tuple[Dict[str, Any], int]:
    """
    Handles requests for the Text Embedding model and includes token count in the response.
    """
    try:
        vertexai.init(project=project_id, location=region)

        model_name = request_json.get("model") or request_json.get("modelId")
        if not model_name:
            return {"error": "Model name must be specified in the request JSON under 'model' or 'modelId' key."}, 400

        embedding_model = TextEmbeddingModel.from_pretrained(model_name)
        texts = request_json.get("texts", [])

        if not texts:
            return {"error": "Request must include 'texts' array"}, 400

        # Get raw embeddings response
        response = embedding_model.get_embeddings(texts)

        # Extract total token count
        total_token_count = sum(embedding.statistics.token_count for embedding in response)

        # Append usage data to the raw response
        response_with_usage = {
            "embedding": response,  # Raw embedding response
            "usage": {"prompt_tokens": total_token_count}
        }

        return response_with_usage, 200

    except Exception as e:
        return {"error": "Text embedding model processing failed", "details": str(e)}, get_google_api_status_code(e)

def push_metric(metric_type: str, value: int, labels: Dict[str, str]):
    """
    Pushes a custom metric to Google Cloud Monitoring.
    """
    series = monitoring_v3.TimeSeries()
    series.metric.type = metric_type
    series.resource.type = "global"
    series.resource.labels["project_id"] = project_id

    for key, val in labels.items():
        series.metric.labels[key] = val

    now = time.time()

    end_time = Timestamp()
    end_time.FromSeconds(int(now))

    start_time = Timestamp()
    start_time.FromSeconds(int(now) - 60)

    point = monitoring_v3.Point()
    point.interval.end_time = start_time
    point.interval.start_time = start_time
    point.value.int64_value = value if isinstance(value, int) else 0

    series.points.append(point)

    try:
        client.create_time_series(name=project_name, time_series=[series])
        print(f"Pushed value {value} to {metric_type} with labels {labels}")
    except Exception as e:
        print(f"Error pushing metric {metric_type}: {e}")
