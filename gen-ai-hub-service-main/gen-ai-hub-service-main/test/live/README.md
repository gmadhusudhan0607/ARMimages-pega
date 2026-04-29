# Live Tests

Live tests validate the GenAI Hub Service against real running instances with actual LLM provider backends. Tests are organized as a matrix of **configurations × prompts × test types**.

## Directory Structure

```
test/live/
├── README.md
├── configs/                          # Service configurations
│   ├── adaptive-strategy-disabled/
│   │   ├── env.genai-hub-service     # Service environment variables
│   │   └── env.genai-gateway-ops     # Ops environment variables
│   └── llm-retry/
│       ├── env.genai-hub-service
│       └── env.genai-gateway-ops
├── prompts/                          # Test prompts
│   ├── image-generation/
│   │   ├── system-prompt
│   │   ├── user-prompt
│   │   └── embeddings-input
│   ├── long-response/
│   │   ├── system-prompt
│   │   ├── user-prompt
│   │   └── embeddings-input
│   ├── short-response/
│   │   ├── system-prompt
│   │   ├── user-prompt
│   │   └── embeddings-input
│   └── voice/                        # Voice-optimized prompt for WebRTC audio tests
│       ├── system-prompt
│       ├── user-prompt
│       └── embeddings-input
└── runner/                           # Test framework
    ├── run_test.go                   # Test entry point (TestLive)
    ├── memleak_test.go               # Memory leak detection (TestMemLeak)
    ├── profiling.go                  # pprof snapshot capture and comparison
    ├── http.go                       # Test runners (suites + individual tests)
    ├── payload.go                    # Payload builders from templates
    ├── assertions.go                 # Response validators
    ├── model_target.go               # Model discovery and URL routing
    ├── environment.go                # Environment setup/teardown
    ├── discovery.go                  # Config/prompt directory discovery
    ├── response.go                   # Response file saving
    ├── webrtc.go                     # WebRTC realtime test runner
    ├── curl.go                       # Curl command generation
    ├── sax.go                        # JWT token resolution
    └── templates/                    # JSON payload templates
        ├── chat-completion.json.tmpl
        ├── chat-completion-stream.json.tmpl
        ├── chat-completion-google.json.tmpl
        ├── chat-completion-google-stream.json.tmpl
        ├── converse.json.tmpl
        ├── embeddings-openai.json.tmpl
        ├── embeddings-amazon.json.tmpl
        ├── embeddings-amazon-nova.json.tmpl
        └── curl.sh.tmpl
```

## Test Types

Each config × prompt combination runs the following test types:

| Test Type | Description |
|-----------|-------------|
| **ChatCompletion** | Non-streaming chat completion. Validates HTTP 200, valid JSON, `finish_reason: "stop"`. |
| **ChatCompletionStreaming** | Streaming chat completion via SSE. Validates incremental chunk delivery, proper `finish_reason` semantics, and temporal streaming. |
| **Embeddings** | Embedding generation. Validates HTTP 200, non-empty embedding vector in provider-specific format. Skipped automatically when `MODEL` is a chat-only model. |
| **ImageGeneration** | Image generation via provider-specific endpoints. Validates HTTP 200 and valid response. Only runs for the `image-generation` prompt. |
| **ModelsDiscovery** | Verifies realtime models appear in the `/models` endpoint response. Asserts all expected model families (`gpt-realtime`, `gpt-realtime-mini`, `gpt-realtime-1.5`) are present. |
| **WebRTC Realtime (Text)** | WebRTC realtime session with text input: fetches ephemeral token via `client_secrets`, establishes WebRTC peer connection with SDP offer/answer, verifies ICE connection and data channel, sends a text message on the data channel, and validates model audio/text response. Models are discovered dynamically from `/models`. |
| **WebRTC Realtime (Audio)** | WebRTC realtime session with synthesized speech input: same setup as the text variant, but synthesizes a spoken prompt via platform TTS (`say` on macOS, `espeak-ng` on Linux), Opus-encodes the PCM, and streams it on the WebRTC audio track at real-time pacing. Server-side VAD is disabled (`turn_detection: null`) to prevent mid-sentence triggers on natural speech pauses; the test explicitly commits the audio buffer (`input_audio_buffer.commit`) and requests a response (`response.create`). Request and response audio are saved alongside session logs. |

### Streaming Validation Details

The `ChatCompletionStreaming` test validates:

- **HTTP 200** status
- **Content-Type** contains `text/event-stream`
- **At least 1 chunk** received
- **All non-final chunks** have `finish_reason: null`
- **Last chunk** has `finish_reason: "stop"`
- **Stream terminates** with `data: [DONE]`
- **Temporal verification** — first-to-last chunk duration exceeds 50ms when more than 5 chunks are received (proves chunks are delivered over time, not buffered)

### Provider-Specific Behavior

| Provider | Non-Streaming | Streaming |
|----------|--------------|-----------|
| **OpenAI** | `POST /openai/deployments/{model}/chat/completions` | Same URL, `"stream": true` in request body |
| **Google** | `POST /google/deployments/{model}/chat/completions` | Same URL, `"stream": true` in request body |
| **Anthropic** | `POST /anthropic/deployments/{model}/converse` | `POST /anthropic/deployments/{model}/converse-stream` |
| **OpenAI (Realtime)** | `POST /openai/deployments/{model}/v1/realtime/client_secrets` | `POST /openai/deployments/{model}/v1/realtime/calls` |

## Running Tests

### Prerequisites

- `JWT` environment variable set, or `sax` CLI configured for automatic token resolution
- AWS credentials configured (profile `dev-ai`) for automatic secret discovery
- Services configured via `configs/` environment files
- Model mappings are generated automatically from Helm configuration at test launch

### SAX Authentication and Caching

When `JWT` is not set, the test runner automatically:

1. **Discovers the SAX secret ARN** — queries AWS Secrets Manager via the SDK
   for the first `sax/backing-services/<UUID>` secret
2. **Issues a JWT token** — runs `sax issue` with the discovered ARN
3. **Caches both values** in `/tmp` for reuse across parallel test processes

| Cache file | Contents | Expiry |
|------------|----------|--------|
| `/tmp/sax-secret-arn-{cell}.cache` | SAX secret ARN per cell | No expiry (invalidated on `sax issue` failure) |
| `/tmp/sax-jwt-token-{cell}.cache` | JWT token per cell | Derived from the token's `exp` claim minus 1 min margin |
| `/tmp/sax-config-{cell}.cache` | SAX client config path per cell | No expiry (invalidated when file is missing) |

Cache files use `flock`-based exclusive locking and atomic rename, so they are
safe for high-volume concurrent access from independent processes. Only one
process performs the AWS discovery or `sax issue` call; others block on the
lock and read the cached result.

**Invalidation** happens automatically in two cases:

- **`sax issue` failure** — if the token command fails (e.g. because the secret
  rotated), the ARN cache is invalidated and re-discovered from AWS. The JWT
  cache is also invalidated so a fresh token is issued with the new ARN.
- **HTTP 401 from `GET /models`** — when the service rejects the token during
  startup verification, both caches are invalidated, a new ARN is discovered,
  a fresh token is issued, and the `/models` check is retried once. If the
  retry also returns 401 (e.g. the service rejects tokens for a reason
  unrelated to caching), test initialization fails immediately.

To force a fresh discovery manually, delete the cache files:

```bash
rm -f /tmp/sax-secret-arn-*.cache* /tmp/sax-jwt-token-*.cache* /tmp/sax-config-*.cache*
```

### Initial Setup

No manual setup required. The test suite automatically:
- Generates `mapping.yaml` from Helm configuration model files
- Extracts `model-metadata.yaml` from Helm templates
- Resolves SAX configuration dynamically from AWS Secrets Manager

### Using External Services

You can run tests against already-deployed services instead of starting local instances:

| Environment Variable | Description |
|---------------------|-------------|
| `OPS_URL` | URL of already-deployed ops service (e.g., `http://localhost:8081`) |
| `SERVICE_URL` | URL of already-deployed hub-service (e.g., `http://localhost:8080`) |
| `HEALTHCHECK_URL` | URL of the service's healthcheck/pprof port (e.g., `http://localhost:8082`). **Required for `test-live-memleak` with external services.** |

**Important:** Both `OPS_URL` and `SERVICE_URL` must be provided together. If only one is set, the test will exit with an error. When both are set, configs are completely ignored and tests run once against the external environment.

To test against a Gateway deployed in a cluster, forward the hub-service (8080), ops (8081), and healthcheck/pprof (8082) ports to localhost.

For memory leak detection (`test-live-memleak`), you must also set `HEALTHCHECK_URL` — this is the port where `/debug/pprof/*` endpoints are served (the healthcheck port, not the main service port).

```bash
# Use external services (both required)
make test-live RUN=all OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080

# Memory leak detection against external services (all three URLs required)
make test-live-memleak OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 HEALTHCHECK_URL=http://localhost:8082

# Error: only one provided
make test-live RUN=all OPS_URL=http://localhost:8081
# → Error: OPS_URL is set but SERVICE_URL is missing. Both must be provided together.
```

### Commands

```bash
# Show help and available configs/prompts
make test-live

# List all test cases
make test-live RUN=list

# Run all tests (all configs × all prompts × all test types)
make test-live RUN=all

# Run with service logs (info level)
make test-live RUN=all VERBOSE=1

# Run with service logs (debug level)
make test-live RUN=all VERBOSE=2

# Filter by config
make test-live CONFIG=llm-retry

# Filter by prompt
make test-live PROMPT=short-response

# Filter by both
make test-live CONFIG=llm-retry PROMPT=short-response

# Filter by model name (exact match)
make test-live RUN=all MODEL=gpt-4o-mini

# Filter by provider/model
make test-live RUN=all MODEL=openai/gpt-4o-mini

# Combine filters
make test-live CONFIG=llm-retry MODEL=gpt-4o-mini

# Run a specific test case
make test-live TEST_CASE=TestLive/llm-retry/short-response/ChatCompletionStreaming
```

### Convenience Targets

Run specific test types with shorter commands:

```bash
# Embeddings tests
make test-live-embeddings                                              # all configs × all prompts
make test-live-embeddings CONFIG=llm-retry                             # specific config × all prompts  
make test-live-embeddings CONFIG=llm-retry PROMPT=long-response        # specific config × specific prompt
make test-live-embeddings OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080  # external services

# Chat completion tests (non-streaming)
make test-live-chat                                                    # all configs × all prompts
make test-live-chat CONFIG=adaptive-strategy-disabled                  # specific config
make test-live-chat CONFIG=llm-retry PROMPT=short-response             # specific config × prompt

# Streaming tests
make test-live-streaming                                               # all configs × all prompts
make test-live-streaming CONFIG=llm-retry                              # specific config
make test-live-streaming CONFIG=adaptive-strategy-disabled PROMPT=long-response

# Image generation tests
make test-live-image                                                   # all configs × all prompts
make test-live-image CONFIG=bajsg PROMPT=image-generation              # specific config × prompt

# WebRTC realtime tests (discovers models dynamically from /models)
make test-live-webrtc-text CONFIG=bajsg                                 # text input, all realtime models
make test-live-webrtc-text CONFIG=bajsg MODEL=gpt-realtime-mini         # text input, specific model
make test-live-webrtc CONFIG=bajsg                                      # audio input, all realtime models
make test-live-webrtc CONFIG=bajsg MODEL=gpt-realtime-mini              # audio input, specific model
```

### Using go test directly

```bash
cd test/live/runner

# Run all tests
go test -v -count=1 -timeout 600s

# Run only streaming tests
go test -v -run "TestLive/.*/ChatCompletionStreaming" -timeout 600s

# Run with config/prompt filters
CONFIG=llm-retry PROMPT=short-response go test -v -count=1 -timeout 600s
```

## WebRTC Realtime Tests

Two test functions validate the WebRTC realtime flow through the gateway. Both share the same session setup (ephemeral token, peer connection, ICE, data channel, `session.created`) but differ in how they provide input to the model:

- **`TestLiveWebRTCAudio`** (`test-live-webrtc`) — synthesizes speech from the `voice/` prompt via platform TTS (`say` on macOS, `espeak-ng` on Linux), encodes the PCM to Opus, and streams it on the WebRTC audio track. Server-side VAD is disabled; the test explicitly commits the audio buffer and requests a response.
- **`TestLiveWebRTC`** (`test-live-webrtc-text`) — sends a text message on the data channel.

Unlike HTTP-based tests, realtime models are discovered dynamically from the `/models` endpoint (type `realtime`), so new APIM deployments are tested automatically.

### Prerequisites

The WebRTC audio test has **build-time** and **runtime** dependencies beyond what the text test requires.

#### Build-time (CGO + Opus)

The audio test uses `gopkg.in/hraban/opus.v2` for PCM→Opus encoding, which requires CGO and system libraries:

| Dependency | macOS install | Linux install | Why |
|------------|--------------|---------------|-----|
| `pkg-config` | `brew install pkgconf` | `apt install pkg-config` | Locates Opus C headers/libs |
| `libopus` | `brew install opus` | `apt install libopus-dev` | Opus audio codec library |
| `libopusfile` | `brew install opusfile` | `apt install libopusfile-dev` | Opus file I/O |

CGO must be enabled: `CGO_ENABLED=1` (the Makefile targets set this automatically).

Without these, the build fails with `cgo: C compiler ... cannot find -lopus` or similar errors.

#### Runtime (TTS + optional tools)

| Dependency | macOS | Linux | Required? | Purpose |
|------------|-------|-------|-----------|---------|
| `say` | Built-in | N/A | **Yes** (macOS) | Speech synthesis (voice: Samantha) |
| `espeak-ng` | N/A | `apt install espeak-ng` | **Yes** (Linux) | Speech synthesis |
| `ffmpeg` | `brew install ffmpeg` | `apt install ffmpeg` | Optional | Converts response Ogg/Opus → WAV for playback |

The test checks for TTS availability at startup and fails with an actionable error message if the required tool is missing.

### Test Flow (shared setup)

1. **Discovery** — `GET /models` to find all realtime models (falls back to hardcoded list if unavailable)
2. **Ephemeral token** — `POST /openai/deployments/{model}/v1/realtime/client_secrets` with session config
3. **WebRTC setup** — creates peer connection with Opus audio track and data channel
4. **SDP negotiation** — `POST /openai/deployments/{model}/v1/realtime/calls` with SDP offer
5. **Connection verification** — waits for ICE connection and data channel to open
6. **Session verification** — waits for `session.created` event on the data channel

### Input (differs per test)

| Test | Step 7 |
|------|--------|
| `TestLiveWebRTCAudio` (audio) | Disable VAD via `session.update` → wait for `session.updated` → synthesize speech → Opus-encode PCM → stream on audio track → `input_audio_buffer.commit` → `response.create` |
| `TestLiveWebRTC` (text) | Send text message on data channel |

Both tests then wait for `response.done` and assert a non-empty transcript.

### Version Testing

Each APIM deployment is a separate test target. To test multiple versions of the same model (e.g., `gpt-realtime-mini`), create separate APIM deployments with distinct deployment names. Version selection happens at deployment level — there is no API parameter to select a version at request time.

### Output Files

Both tests save session artifacts to `/tmp/`:

```
/tmp/live-test-{uuid}-{test_name}.webrtc.json            # All data channel events
/tmp/live-test-{uuid}-{test_name}.webrtc.transcript       # Model's text response
/tmp/live-test-{uuid}-{test_name}.webrtc.request.wav      # Request audio (24kHz/16-bit/mono PCM) — audio test only
/tmp/live-test-{uuid}-{test_name}.webrtc.response.ogg     # Response audio (Ogg/Opus container)
/tmp/live-test-{uuid}-{test_name}.webrtc.response.rtp     # Response audio (raw RTP packets)
/tmp/live-test-{uuid}-{test_name}.webrtc.response.wav     # Response audio (24kHz WAV) — requires ffmpeg
```

| File | Always produced? | How to play / inspect |
|------|------------------|-----------------------|
| `.webrtc.json` | Yes | Any text editor / `jq` |
| `.webrtc.transcript` | Yes | Any text editor |
| `.webrtc.request.wav` | Audio test only | Any audio player |
| `.webrtc.response.ogg` | Yes (if model responds) | VLC, `ffplay`, browser |
| `.webrtc.response.rtp` | Yes (if model responds) | Wireshark, custom decoders |
| `.webrtc.response.wav` | Only if `ffmpeg` is installed | Any audio player |

**Optional dependency:** Install `ffmpeg` (`brew install ffmpeg` on macOS) to enable automatic Ogg→WAV conversion of response audio. Without it, the `.ogg` and `.rtp` files are still saved.

## Memory Leak Detection

The `test-live-memleak` target detects memory leaks by running the test suite multiple rounds and comparing pprof snapshots.

### How It Works

1. **Warm-up** — runs the full suite once to populate caches and trigger lazy initialization (no snapshot)
2. **Measurement rounds** — runs the suite N times, capturing a pprof snapshot (heap, goroutines, threads, allocs) after each round
3. **Comparison** — compares the first and last snapshots; any growth exceeding thresholds is flagged as a leak

All chat completion, streaming, and embedding failures are tolerated during measurement — the goal is to exercise code paths, not validate responses.

In addition to inference calls, the suite exercises non-inference endpoints to detect leaks in model listing, default resolution, and health check code paths:

| GET Endpoint | Port | Auth | Purpose |
|-------------|------|------|---------|
| `/` | Service (8080) | None | Swagger/root handler and full middleware stack |
| `/models` | Service (8080) | JWT | Model listing and provider aggregation |
| `/models/defaults` | Service (8080) | JWT | Default model resolution |
| `/health/liveness` | Healthcheck (8082) | None | Liveness probe |
| `/health/readiness` | Healthcheck (8082) | None | Readiness probe |

These GET endpoints are called multiple times per round, scaling with the number of chat completion targets (minimum 5 repetitions).

### GET-only mode

To isolate non-inference endpoint leak detection from model inference workload, use `test-live-memleak-get`. This runs only the GET endpoints at higher volume (default 50 repetitions per round):

```bash
make test-live-memleak-get CONFIG=llm-retry                              # default 50 reps/round
make test-live-memleak-get CONFIG=llm-retry MEMLEAK_GET_REPETITIONS=100  # custom repetitions
```

### Local services (started by the test runner)

```bash
make test-live-memleak CONFIG=llm-retry                        # all prompts
make test-live-memleak CONFIG=llm-retry PROMPT=short-response  # specific prompt
make test-live-memleak CONFIG=llm-retry VERBOSE=1              # with service logs
make test-live-memleak CONFIG=llm-retry MEMLEAK_ROUNDS=10      # custom rounds
```

### External services (e.g. port-forwarded from cluster)

Forward hub-service (8080), ops (8081), and healthcheck/pprof (8082) ports to localhost, then:

```bash
make test-live-memleak OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080 HEALTHCHECK_URL=http://localhost:8082
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `MEMLEAK_ROUNDS` | 5 | Number of measurement rounds |
| `MEMLEAK_GET_REPETITIONS` | 50 | Number of GET endpoint calls per round (GET-only mode) |
| `LEAK_HEAP_THRESHOLD_PCT` | 20 | Max heap growth (%) before flagging a leak |
| `LEAK_HEAP_MIN_BYTES` | 5242880 | Min absolute heap growth (bytes) required before % threshold applies |
| `LEAK_GOROUTINE_THRESHOLD` | 5 | Max goroutine increase before flagging a leak |
| `LEAK_THREAD_THRESHOLD` | 100 | Max OS thread increase before flagging a leak |
| `HEALTHCHECK_URL` | — | Required for external services (pprof endpoint) |

### Reading the Output

The snapshot summary table shows round-over-round deltas to help identify leaks:

```
=== Snapshot Summary ===
  Round    Heap InUse  Heap Delta  Objects  Goroutines  Gor Delta  Threads
  1          27.62 MB           —    62308          63          —      201
  2          22.61 MB   -5.02 MB    78345          63         +0      216
  3          16.87 MB   -5.74 MB    45590          63         +0      221
```

- **Constant goroutine/heap deltas across rounds** indicate a leak (linear growth)
- **Fluctuating deltas** are normal heap noise
- **Thread growth that plateaus** is expected (Go runtime behavior, not a leak)

### Offline Analysis

Saved pprof profiles can be compared with `go tool pprof`:

```bash
# Heap diff between first and last round
go tool pprof -base /tmp/live-test-memleak-<id>-round1-heap.prof /tmp/live-test-memleak-<id>-round5-heap.prof

# Goroutine diff
go tool pprof -base /tmp/live-test-memleak-<id>-round1-goroutine.prof /tmp/live-test-memleak-<id>-round5-goroutine.prof
```

The exact commands with full paths are printed at the end of each test run.

## Output Files

Each test run generates files in `/tmp/` with the pattern:

```
/tmp/live-test-{uuid}-{test_name}.curl.sh         # Reproducible curl command
/tmp/live-test-{uuid}-{test_name}.resp.headers     # Response headers
/tmp/live-test-{uuid}-{test_name}.resp.body.json   # Response body
```

On test failure, the curl command is logged for easy reproduction.

## Adding New Configurations

Create a new directory under `configs/` with two files:

```bash
mkdir configs/my-new-config
# Add env.genai-hub-service and env.genai-gateway-ops files
```

The new configuration will automatically be included in the test matrix.

## Adding New Prompts

Create a new directory under `prompts/` with three files:

```bash
mkdir prompts/my-new-prompt
# Add system-prompt, user-prompt, and embeddings-input files
```

The new prompt will automatically be included in the test matrix.

**Note:** The `voice/` prompt is specifically designed for WebRTC audio tests — it uses a system prompt that instructs the model to listen to spoken audio and respond in English. The `PROMPT=voice` filter is used automatically by `test-live-webrtc`.

## Templates

| Template | Used By | Description |
|----------|---------|-------------|
| `chat-completion.json.tmpl` | OpenAI (non-streaming) | Messages with system + user prompts |
| `chat-completion-stream.json.tmpl` | OpenAI (streaming) | Same + `"stream": true` |
| `chat-completion-google.json.tmpl` | Google (non-streaming) | Messages + `model` field |
| `chat-completion-google-stream.json.tmpl` | Google (streaming) | Same + `"stream": true` |
| `converse.json.tmpl` | Anthropic/Amazon (both) | Bedrock converse format (streaming uses different URL, same body) |
| `embeddings-openai.json.tmpl` | OpenAI/Google embeddings | `"input"` field |
| `embeddings-amazon.json.tmpl` | Amazon Titan embeddings | `"inputText"` field |
| `embeddings-amazon-nova.json.tmpl` | Amazon Nova embeddings | `"singleEmbeddingParams"` format |
