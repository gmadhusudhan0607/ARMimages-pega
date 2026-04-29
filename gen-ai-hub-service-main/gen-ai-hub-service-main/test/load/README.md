## Load Test: trigger_test.go

This Go test file (`trigger_test.go`) is used for basic load testing of the Gen AI Hub Service endpoints. It sends randomized or sequential POST requests to various model endpoints to simulate traffic and measure response characteristics.

### Usage

- Run the tests with:
  ```
  go test -v ./test/load
  ```

### Configuration

- **Tokens:**  
  Replace `"add_generated_tokens_here"` in the `tokens` slice with valid Bearer tokens issued with SAX for authentication. This simulate different users interacting with the application to provide richer data for the design of monitoring for multitenant application.
  ```go
  var tokens = []string{
      "your_token_1",
      "your_token_2",
      // ...
  }
  ```

- **Endpoints:**  
  The `models` slice defines the target endpoints, request bodies, and labels. Update URLs and request bodies as needed for your environment.

### Extending

- **Add a new model:**  
  Add a new entry to the `models` slice:
  ```go
  {
      "http://localhost:8080/new/endpoint",
      `{"your": "json body %d"}`,
      []string{"label1", "label2"},
  }
  ```
- **Filter by label:**  
  Set the `filter` variable in `TestSequentialPostRequests` to run tests only for models with a specific label.

### Notes

- The test randomizes payloads and delays to simulate real-world usage.
- Response status, latency, and a snippet of the response body are logged for each request.
