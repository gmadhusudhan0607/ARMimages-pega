# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.

"""
pytest configuration and fixtures for cloud function tests.

This file sets up mocks for GCP metadata service calls that happen
during module import, before any tests run.
"""

import pytest
import sys
from unittest.mock import Mock, patch, MagicMock


# Mock GCP metadata service at import time
@pytest.fixture(scope="session", autouse=True)
def mock_gcp_metadata():
    """
    Mock GCP metadata service calls that happen during module import.

    The main.py module calls get_project_id() and get_location() at module level,
    which try to connect to metadata.google.internal. This fixture mocks those calls.
    """
    with patch('requests.get') as mock_get:
        # Create mock responses for metadata queries
        mock_response = Mock()

        # For project ID query: returns 'test-project-id'
        # For region query: returns 'projects/test-project/zones/us-central1-a'
        def mock_metadata_response(url, *args, **kwargs):
            response = Mock()
            if 'project-id' in url:
                response.text.strip.return_value = 'test-project-id'
            elif 'region' in url:
                response.text.split.return_value = ['projects', 'test-project', 'zones', 'us-central1-a']
                response.text.strip.return_value = 'us-central1'
            return response

        mock_get.side_effect = mock_metadata_response

        # Mock monitoring client to avoid initialization
        with patch('google.cloud.monitoring_v3.MetricServiceClient'):
            yield mock_get


@pytest.fixture(autouse=True)
def reset_module_state():
    """Reset any module-level state between tests"""
    # Clean up any cached modules if needed
    yield
    # Cleanup after test
    pass
