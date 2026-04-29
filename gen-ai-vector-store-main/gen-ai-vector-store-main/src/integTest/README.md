# Integration Test Standards

This document defines the standards and conventions for all integration tests in this directory.

## Ginkgo/BDD Test Naming Conventions

### Top-Level Describe Block
- **MUST** include HTTP method and full endpoint path
- Format: `Describe("METHOD /endpoint/path", func()`
- Example: `Describe("POST /v1/{isolationID}/collections/{collectionName}/query/chunks", func()`

### Context Blocks for Organization
Use Context blocks to group related test scenarios:
- `Context("with strong consistency", func()`
- `Context("with eventual consistency", func()`
- `Context("timeout handling", func()`
- `Context("response headers", func()`
- `Context("error scenarios", func()`
- `Context("retry behavior", func()`

### It Blocks (Test Cases)
- Focus on **behavior**, not technical implementation
- Keep descriptions **concise and clear**
- **NO numeric prefixes** (avoid "test1:", "test2:", etc.)
- **NO endpoint/method repetition** (that's already in Describe)
- **NO HTTP status codes** in descriptions unless testing status codes specifically

#### ✅ Good Examples:
```go
It("creates document successfully", func()
It("returns error when embedding service fails", func()
It("retries and succeeds after initial timeout", func()
It("prevents connection leaks after timeout", func()
It("accepts document for async processing", func()
It("overwrites existing document on second PUT", func()
```

#### ❌ Bad Examples (DO NOT USE):
```go
It("test1: Returns 200 when successful", func()           // ❌ Numeric prefix, status code
It("v1_query_chunks: Returns chunks", func()               // ❌ Endpoint prefix
It("POST returns 504 on timeout", func()                   // ❌ Method/status focus
It("Test 1: Document insertion times out", func()          // ❌ Test numbering
```

## Test File Organization

### File Naming Convention
- Pattern: `{resource}_{operation}_{version}_test.go`
- Examples:
  - `documents_put_v2_test.go` - PUT operations on documents v2 API
  - `query_chunks_v2_test.go` - Query chunks operations v2 API
  - `v1_documents_put_timeout_test.go` - Timeout tests for v1 documents PUT
  - `background_processing_timeout_test.go` - Background processing timeout tests

### Test Data Organization
```
data/
├── test01/           # Use zero-padded numbering
├── test02/           # NOT test1/, test2/
├── test03/
└── timeout_scenario/ # Descriptive names are also acceptable
```

## WireMock Integration Pattern

### Setup Pattern
```go
BeforeEach(func() {
    // 1. Initialize test variables
    isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
    testExpectations = []string{}
    
    // 2. Create isolation
    CreateIsolation(opsBaseURI, isolationID, "1GB")
})
```

### Mock Management
```go
// Track expectation IDs for cleanup
expID := CreateMockServerExpectation(jsonData)
testExpectations = append(testExpectations, expID)
```

### Cleanup Pattern
```go
AfterEach(func() {
    // Cleanup expectations
    for _, expID := range testExpectations {
        DeleteExpectationIfExist(wiremockManager, expID)
    }
    
    // Don't cleanup if test failed (for debugging)
    if !CurrentSpecReport().Failed() {
        DeleteIsolation(opsBaseURI, isolationID)
    }
})
```

## Complete Example

```go
var _ = Describe("PUT /v1/{isolationID}/collections/{collectionName}/documents", func() {
    var (
        isolationID string
        collectionID string
        testExpectations []string
    )
    
    BeforeEach(func() {
        // Setup code
    })
    
    AfterEach(func() {
        // Cleanup code
    })
    
    Context("with strong consistency", func() {
        It("creates document successfully", func() {
            By("Setting up mock expectation")
            // Mock setup
            
            By("Submitting document")
            // API call
            
            By("Verifying database state")
            // Assertions
        })
        
        It("returns error when embedding service fails", func() {
            // Test implementation
        })
    })
    
    Context("with eventual consistency", func() {
        It("accepts document for async processing", func() {
            // Test implementation
        })
    })
})
```

## Migration from Old Patterns

When refactoring existing tests:
1. Remove "test" number prefixes from It blocks
2. Add Context blocks to group related tests  
3. Simplify It descriptions to focus on behavior
4. Keep Method + endpoint in Describe block

## Suite Organization

### Suite File (suite_test.go)
- Contains test suite setup and teardown
- Starts all required services (main, ops, background)
- Manages database and WireMock containers
- Should have descriptive suite name: `RunSpecs(t, "Descriptive Suite Name")`

### Test Files
- One file per major endpoint or feature area
- Related tests grouped in same file
- Use clear, descriptive file names

## Best Practices

1. **Use By() for test steps** - Makes test flow clear in output
2. **Wait for async operations** - Use WaitForDocumentStatusInDB, etc.
3. **Clean up resources** - Always clean up in AfterEach
4. **Preserve failed test state** - Don't cleanup if test failed (for debugging)
5. **Track mock expectations** - Store IDs for cleanup
6. **Verify mock interactions** - Check call counts when testing retries
7. **Use helper functions** - Leverage functions from src/integTest/functions/
8. **Test isolation** - Each test should be independent

## Running Tests

```bash
# Run all integration tests in a directory
make integtest-service

# Run specific test file
go test -v ./src/integTest/service/timeout/...

# Run focused tests (use FOCUS environment variable)
FOCUS="timeout handling" make integtest-service

# Keep containers running after test (for debugging)
KEEP=60sec make integtest-service
