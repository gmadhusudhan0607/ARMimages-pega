# Cloud Function Testing Guide

This directory contains tests for the GCP Cloud Function that handles Vertex AI requests.

## Test-Driven Development Approach

**IMPORTANT**: When making changes to this Cloud Function, follow a **Test-Driven Development (TDD)** approach to ensure:
- **Non-breaking changes** - Existing behavior is validated before modifications
- **Backward compatibility** - All existing tests continue to pass
- **Regression prevention** - New tests catch future issues
- **Zero-downtime deployments** - Safe to deploy at any time

### TDD Workflow (RED → GREEN → REFACTOR)

1. **RED**: Write a failing test for the new behavior
   ```bash
   # Add test to test_main.py
   # Run tests - they should fail
   ./run_tests.sh
   ```

2. **GREEN**: Implement minimal code to make the test pass
   ```bash
   # Update main.py.tpl
   # Run tests - they should pass
   ./run_tests.sh
   ```

3. **REFACTOR**: Improve code while keeping tests green
   ```bash
   # Clean up implementation
   # Run tests - ensure they still pass
   ./run_tests.sh
   ```

**Example**: See ADR-0003 for path-based routing implementation using TDD.

## Test Suite Overview

`test_main.py` contains comprehensive tests covering:

### Test Categories

**1. Helper Functions**
- Model detection (`is_gemini_model`, `is_image_generation_model`)
- Model ID extraction (`extract_model_id`)
- Global endpoint detection (`is_global_model`)
- URL construction (`get_vertex_native_api_url`)

**2. Request Routing**
- Path-based detection (`is_generate_content_request`)
- Request dispatch to appropriate handlers
- Backward compatibility with existing models

**3. Request Handlers**
- Gemini chat (OpenAI SDK)
- Gemini image generation (native Vertex AI API)
- Imagen (Vertex AI Vision SDK)
- Text embeddings

**4. Error Handling**
- Model not found
- API errors
- Timeout handling
- Malformed requests

## Running the Tests

### Quick Start

```bash
cd distribution/genai-gcpvertex-infra-terraform/src/main/terraform/gcp/tests/
./run_tests.sh
```

### Run Specific Tests

```bash
./run_tests.sh -k test_gemini_image_generation
./run_tests.sh -k test_path_based_routing
```

### Run with Coverage

```bash
./run_tests.sh --cov=main --cov-report=html
```

### Continuous Testing

```bash
# Watch mode (requires pytest-watch)
ptw -- test_main.py
```

## Key Testing Requirements

### Endpoint Format Validation

Tests verify correct endpoint transformations:
- Input from client: `/generateContent` (REST-style, slash-prefixed)
- Output to Vertex AI: `:generateContent` (gRPC-style, colon-prefixed)

This transformation is **critical** for Vertex AI API compatibility.

### Model Detection

Tests validate proper model identification:
- Gemini chat: `gemini-3.0-flash`, `gemini-2.5-pro`
- Gemini image: `gemini-3.1-flash-image-preview`, `gemini-3-pro-image-preview`
- Imagen: `imagen-3`, `imagen-4.0-*`
- Embeddings: `text-multilingual-embedding-*`, `gemini-embedding-*`

### Backward Compatibility

Regression tests ensure existing models continue working:
- Gemini chat → OpenAI SDK (unchanged)
- Imagen → Vertex AI Vision SDK (unchanged)
- Text embeddings → Vertex AI Embeddings API (unchanged)

## Adding New Tests

When adding new functionality:

1. **Write test first** (TDD approach)
   ```python
   def test_new_feature():
       # Arrange
       request = create_test_request(...)

       # Act
       result = function_under_test(request)

       # Assert
       assert result == expected_value
   ```

2. **Run test** - Should fail (RED)
3. **Implement** - Add code to `../templates/main.py.tpl`
4. **Run test** - Should pass (GREEN)
5. **Refactor** - Improve while tests stay green

## Deployment Checklist

Before deploying cloud function changes:

- [ ] All tests pass locally (`./run_tests.sh`)
- [ ] No regressions in existing tests
- [ ] New functionality covered by tests
- [ ] Code reviewed and approved
- [ ] Integration tests pass (from genai-hub-service)

## Reference

- **ADR-0003**: Path-based routing decision (example of TDD approach)
- **Architecture docs**: `../../../../../../docs/guides/infrastructure_coordination.md`
