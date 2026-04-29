# JSON Path Package

A high-performance JSON path manipulation package designed for fast proxy operations with minimal delay.

## Features

- **GetValueByPath**: Extract values from JSON using dot-separated paths
- **SetValueByPath**: Set or override values in JSON using dot-separated paths
- **Streaming Support**: Optimized streaming versions for large JSON payloads
- **High Performance**: Microsecond-level operations with minimal memory allocations
- **Array Support**: Access array elements using numeric indices
- **Path Creation**: Automatically creates intermediate objects when setting nested paths

## Usage

### Basic Operations

```go
import "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/json"

// Extract a value
requestBody := []byte(`{"generationConfig": {"maxOutputTokens": 1024}}`)
value, err := json.GetValueByPath(requestBody, "generationConfig.maxOutputTokens")
// value: 1024

// Set a value
modifiedJSON, err := json.SetValueByPath(requestBody, "generationConfig.maxOutputTokens", 2048)
// Returns modified JSON with updated value
```

### Streaming Operations (Recommended for Large JSON)

```go
// Streaming get - more efficient for large payloads
reader := strings.NewReader(jsonData)
value, err := json.StreamingGetValueByPath(reader, "path.to.value")

// Streaming set - more efficient for large payloads
reader := strings.NewReader(jsonData)
var writer bytes.Buffer
err := json.StreamingSetValueByPath(reader, &writer, "path.to.value", newValue)
```

### Path Examples

- Simple: `"model"`
- Nested: `"generationConfig.maxOutputTokens"`
- Array access: `"messages.0.role"`
- Deep nesting: `"config.advanced.settings.timeout"`

## Performance

Benchmark results on 11th Gen Intel i7:

- **GetValueByPath**: ~4.7μs per operation
- **SetValueByPath**: ~3.7μs per operation  
- **StreamingGetValueByPath**: ~2.2μs per operation (fastest)
- **StreamingSetValueByPath**: ~3.8μs per operation

## Implementation Details

- Uses Go's `json.NewDecoder` with `UseNumber()` for precise numeric handling
- Streaming operations minimize memory allocations
- No external dependencies beyond Go standard library
- Designed specifically for proxy service performance requirements
- Avoids over-engineering while maintaining clean, logical structure

## Error Handling

The package provides clear error messages for:
- Invalid JSON input
- Non-existent paths
- Array index out of bounds
- Type mismatches
- Empty inputs

## Testing

Comprehensive test suite includes:
- Unit tests for all functions
- Edge case testing
- Performance benchmarks
- Usage examples
- Error condition validation

Run tests: `go test -v`
Run benchmarks: `go test -bench=. -benchmem`
