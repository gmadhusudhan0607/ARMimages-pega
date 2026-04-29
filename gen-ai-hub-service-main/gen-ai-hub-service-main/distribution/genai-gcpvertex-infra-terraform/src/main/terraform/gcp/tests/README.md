# Cloud Function Tests

This directory contains tests for the GCP Cloud Function that handles Vertex AI requests.

## Directory Structure

```
tests/
├── conftest.py              # pytest fixtures and configuration
├── pytest.ini               # pytest settings
├── requirements-dev.txt     # test dependencies
├── run_tests.sh            # test runner script
├── test_main.py            # test suite
└── TESTING.md              # detailed testing documentation
```

## Development Approach

**Test-Driven Development (TDD) is strongly recommended** when making changes to this Cloud Function.

Following TDD helps ensure:
- **Non-breaking changes** - Tests validate existing behavior before modifications
- **Backward compatibility** - Existing tests continue to pass with new changes
- **Regression prevention** - New tests catch future breaking changes
- **Design clarity** - Writing tests first helps clarify requirements

**TDD Workflow:**
1. **RED**: Write a failing test for the new behavior
2. **GREEN**: Implement minimal code to make the test pass
3. **REFACTOR**: Improve code while keeping tests green

See ADR-0003 for an example of TDD approach used in path-based routing implementation.

## Running Tests

```bash
# From this directory
./run_tests.sh

# Or directly with pytest (after setup)
pytest test_main.py -v
```

## Test Coverage

The test suite validates:
- Helper functions (model identification, URL construction)
- Path-based request routing
- Request handler behavior
- Error handling
- Global vs regional endpoint selection

See `TESTING.md` for detailed documentation.
