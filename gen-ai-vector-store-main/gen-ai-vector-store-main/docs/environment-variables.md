# Environment Variables Documentation

This document provides a comprehensive reference for all environment variables used in the GenAI Vector Store project across the Service, Ops, and Background services.

**⚠️ IMPORTANT**: When adding, removing, or renaming environment variables, you MUST update this document to keep it current.

## Table of Contents

- [Database Configuration](#database-configuration)
- [Connection Pooling](#connection-pooling)
- [Cloud Provider Settings](#cloud-provider-settings)
- [SAX Authentication](#sax-authentication)
- [GenAI Integration](#genai-integration)
- [Service Ports](#service-ports)
- [Feature Flags](#feature-flags)
- [Runtime Configuration](#runtime-configuration)
- [Performance Tuning](#performance-tuning)
- [Logging & Monitoring](#logging--monitoring)
- [HTTP Client Configuration](#http-client-configuration)
- [Development & Testing](#development--testing)

---

## Database Configuration

### DB_LOCAL
- **Description**: Enables local database mode (non-cloud)
- **Used by**: All services
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `DB_LOCAL=true`

### DB_HOST
- **Description**: Database host address
- **Used by**: All services
- **Required**: Yes
- **Example**: `DB_HOST=localhost` or `DB_HOST=postgres.example.com`

### DB_PORT
- **Description**: Database port number
- **Used by**: All services
- **Required**: Yes
- **Default**: `5432`
- **Example**: `DB_PORT=5432`

### DB_NAME
- **Description**: Database name
- **Used by**: All services
- **Required**: Yes
- **Example**: `DB_NAME=vectorstore`

### DB_USR
- **Description**: Database username (for local mode or GCP local credentials)
- **Used by**: All services
- **Required**: Yes (in local mode)
- **Example**: `DB_USR=postgres`

### DB_PWD
- **Description**: Database password (for local mode or GCP local credentials)
- **Used by**: All services
- **Required**: Yes (in local mode)
- **Example**: `DB_PWD=secretpassword`

### DB_SECRET
- **Description**: ARN or path to cloud secret containing database credentials
- **Used by**: All services (AWS/GCP)
- **Required**: Yes (in cloud mode)
- **Example**: `DB_SECRET=arn:aws:secretsmanager:us-east-1:123456789:secret:db-creds`

### DB_INSTANCE
- **Description**: GCP database instance identifier
- **Used by**: All services (GCP only)
- **Required**: Yes (in GCP mode)
- **Example**: `DB_INSTANCE=my-project:us-central1:my-instance`

### DB_WITH_PRIVATE_IP
- **Description**: Use private IP for GCP database connection
- **Used by**: All services (GCP only)
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `DB_WITH_PRIVATE_IP=true`

### DB_USE_LEGACY_ATTRIBUTE_IDS
- **Description**: Use legacy attribute ID format for backward compatibility
- **Used by**: All services
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `DB_USE_LEGACY_ATTRIBUTE_IDS=false`

### DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION
- **Description**: Override minimum required schema version check
- **Used by**: All services
- **Required**: No
- **Default**: Empty (no override)
- **Example**: `DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION=2.5.0`

### DATABASE_ENGINE_VERSION
- **Description**: PostgreSQL engine version from DBInstance SCE service. This value is automatically set by the DBInstance service and is used to trigger Background Service pod restarts when PostgreSQL is upgraded. The Background Service detects actual PostgreSQL version changes from the database and runs ANALYZE automatically to rebuild statistics. ANALYZE runs at most once per PostgreSQL version.
- **Used by**: Background
- **Required**: No (automatically set by DBInstance SCE)
- **Default**: Empty string
- **Example**: `DATABASE_ENGINE_VERSION=17.2`
- **Note**: This parameter is automatically populated from the DBInstance SCE output. When the DatabaseEngineVersion changes in DBInstance (e.g., during a PostgreSQL upgrade), it triggers a pod restart, allowing the Background Service to detect the new PostgreSQL version and run ANALYZE if needed.

---

## Connection Pooling

### MAX_CONNS
- **Description**: Maximum database connections for the main pool
- **Used by**: Service, Background
- **Required**: No
- **Default**: `20`
- **Example**: `MAX_CONNS=50`

### MAX_CONNS_GENERIC
- **Description**: Maximum connections for generic operations pool
- **Used by**: Service
- **Required**: No
- **Default**: `10`
- **Example**: `MAX_CONNS_GENERIC=15`

### MAX_CONNS_INGESTION
- **Description**: Maximum connections for ingestion operations pool
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `30`
- **Example**: `MAX_CONNS_INGESTION=40`

### MAX_CONNS_SEARCH
- **Description**: Maximum connections for search operations pool
- **Used by**: Service
- **Required**: No
- **Default**: `20`
- **Example**: `MAX_CONNS_SEARCH=25`

---

## Cloud Provider Settings

### CLOUD_PROVIDER
- **Description**: Cloud provider type
- **Used by**: All services
- **Required**: No
- **Default**: `aws`
- **Values**: `aws` | `gcp`
- **Example**: `CLOUD_PROVIDER=gcp`

### REGION
- **Description**: Cloud region (GCP) or AWS region (derived from DB_SECRET ARN for AWS)
- **Used by**: All services (cloud mode)
- **Required**: Yes (GCP), Derived (AWS)
- **Example**: `REGION=us-central1`

### ACCOUNT
- **Description**: GCP account/project identifier
- **Used by**: All services (GCP only)
- **Required**: Yes (GCP)
- **Example**: `ACCOUNT=my-gcp-project`

---

## SAX Authentication

### SAX_DISABLED
- **Description**: Disable SAX authentication (uses mock validator)
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `SAX_DISABLED=false`

### SAX_AUDIENCE
- **Description**: Expected audience in SAX tokens
- **Used by**: Service, Ops
- **Required**: Yes (when SAX enabled)
- **Example**: `SAX_AUDIENCE=genai-vector-store`

### SAX_ISSUER
- **Description**: Expected issuer of SAX tokens
- **Used by**: Service, Ops
- **Required**: Yes (when SAX enabled)
- **Example**: `SAX_ISSUER=https://auth.pega.io`

### SAX_JWKS_ENDPOINT
- **Description**: JWKS endpoint for SAX token validation
- **Used by**: Service, Ops
- **Required**: Yes (when SAX enabled)
- **Example**: `SAX_JWKS_ENDPOINT=https://auth.pega.io/.well-known/jwks.json`

### SAX_CLIENT_DISABLED
- **Description**: Disable SAX client for outbound requests
- **Used by**: Service, Ops, Background
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `SAX_CLIENT_DISABLED=false`

### SAX_CLIENT_ID
- **Description**: Client ID for SAX authentication
- **Used by**: All services (when SAX client enabled)
- **Required**: Yes (when SAX client enabled)
- **Example**: `SAX_CLIENT_ID=genai-vector-store-client`

### SAX_CLIENT_SECRET
- **Description**: Secret name/ARN for SAX client private key
- **Used by**: All services (when SAX client enabled)
- **Required**: Yes (when SAX client enabled)
- **Example**: `SAX_CLIENT_SECRET=arn:aws:secretsmanager:region:account:secret:sax-key`

### SAX_CLIENT_SCOPES
- **Description**: Comma-separated list of OAuth scopes for SAX client
- **Used by**: All services (when SAX client enabled)
- **Required**: Yes (when SAX client enabled)
- **Example**: `SAX_CLIENT_SCOPES=read,write`

### SAX_CLIENT_TOKEN_ENDPOINT
- **Description**: Token endpoint for SAX client authentication
- **Used by**: All services (when SAX client enabled)
- **Required**: Yes (when SAX client enabled)
- **Example**: `SAX_CLIENT_TOKEN_ENDPOINT=https://auth.pega.io/oauth/token`

### SAX_CLIENT_DEV_TOKEN
- **Description**: Development token for SAX client (bypasses normal auth)
- **Used by**: All services (development only)
- **Required**: No
- **Example**: `SAX_CLIENT_DEV_TOKEN=dev-token-12345`

### SAX_TOKEN_CACHE_ENABLED
- **Description**: Enable JWT token caching to improve performance
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `true`
- **Values**: `true` | `false`
- **Example**: `SAX_TOKEN_CACHE_ENABLED=true`

### SAX_TOKEN_CACHE_MAX_TTL
- **Description**: Maximum cache TTL for JWT tokens (security limit)
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `50m`
- **Example**: `SAX_TOKEN_CACHE_MAX_TTL=10m`

### SAX_TOKEN_CACHE_MAX_SIZE
- **Description**: Maximum number of JWT tokens to cache
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `10000`
- **Example**: `SAX_TOKEN_CACHE_MAX_SIZE=5000`

### SAX_TOKEN_CACHE_CLEANUP_INTERVAL
- **Description**: Interval for cleaning up expired cache entries
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `5m`
- **Example**: `SAX_TOKEN_CACHE_CLEANUP_INTERVAL=3m`

---

## GenAI Integration

### GENAI_GATEWAY_SERVICE_URL
- **Description**: Base URL for GenAI Gateway service
- **Used by**: Service, Ops, Background
- **Required**: Yes
- **Example**: `GENAI_GATEWAY_SERVICE_URL=https://genai-gateway.example.com`

### GENAI_GATEWAY_CUSTOM_CONFIG
- **Description**: Custom configuration override for GenAI Gateway
- **Used by**: Service, Ops, Background
- **Required**: No
- **Example**: `GENAI_GATEWAY_CUSTOM_CONFIG={"timeout": 30}`

### GENAI_SMART_CHUNKING_SERVICE_URL
- **Description**: URL for smart chunking service
- **Used by**: Service
- **Required**: Yes (when smart chunking enabled)
- **Example**: `GENAI_SMART_CHUNKING_SERVICE_URL=https://chunking.example.com`

### DEFAULT_EMBEDDING_PROFILE
- **Description**: Default embedding profile to use when not specified
- **Used by**: Service, Ops, Background
- **Required**: No
- **Default**: `openai-text-embedding-ada-002`
- **Example**: `DEFAULT_EMBEDDING_PROFILE=openai-text-embedding-ada-002`

### GEN_AI_API_KEY
- **Description**: API key for GenAI services (alternative authentication)
- **Used by**: All services
- **Required**: No
- **Example**: `GEN_AI_API_KEY=sk-12345abcde`

---

## Usage Metrics Configuration

### USAGE_METRICS_ENABLED
- **Description**: Enable usage metrics export for semantic search requests
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `USAGE_METRICS_ENABLED=true`

### USAGE_METRICS_UPLOAD_INTERVAL_SECONDS
- **Description**: Upload interval in seconds for sending batched metrics to usage data endpoints
- **Used by**: Service
- **Required**: No
- **Default**: `3600`
- **Example**: `USAGE_METRICS_UPLOAD_INTERVAL_SECONDS=1800`

### USAGE_METRICS_MAX_PAYLOAD_SIZE
- **Description**: Maximum payload size in bytes for usage metrics uploads (triggers chunking)
- **Used by**: Service
- **Required**: No
- **Default**: `819200` (800KB)
- **Example**: `USAGE_METRICS_MAX_PAYLOAD_SIZE=819200`

### USAGE_METRICS_RETRY_COUNT
- **Description**: Number of retry attempts for failed usage metrics uploads
- **Used by**: Service
- **Required**: No
- **Default**: `3`
- **Example**: `USAGE_METRICS_RETRY_COUNT=5`

### USAGE_METRICS_REQUEST_TIMEOUT_SECS
- **Description**: HTTP request timeout in seconds for usage metrics uploads
- **Used by**: Service
- **Required**: No
- **Default**: `30`
- **Example**: `USAGE_METRICS_REQUEST_TIMEOUT_SECS=45`

---

## DB Metrics to PDC Configuration

**Note**: DB metrics uploads are controlled by `USAGE_METRICS_ENABLED` and reuse all `USAGE_METRICS_*` configuration settings (retry count, payload size, request timeout) except for the upload interval which is configured separately.

### DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS
- **Description**: Upload interval in seconds for sending database metrics to PDC endpoints
- **Used by**: Background
- **Required**: No
- **Default**: `3600` (1 hour)
- **Example**: `DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS=1800`
- **Note**: This is independent from `USAGE_METRICS_UPLOAD_INTERVAL_SECONDS` which controls semantic search metrics upload interval

---

## Service Ports

### SERVICE_PORT
- **Description**: Main service port for the Service
- **Used by**: Service
- **Required**: No
- **Default**: `8080`
- **Example**: `SERVICE_PORT=8080`

### SERVICE_HEALTHCHECK_PORT
- **Description**: Health check endpoint port for the Service
- **Used by**: Service
- **Required**: No
- **Default**: `8082`
- **Example**: `SERVICE_HEALTHCHECK_PORT=8082`

### OPS_PORT
- **Description**: Main service port for Ops
- **Used by**: Ops
- **Required**: No
- **Default**: `8080`
- **Example**: `OPS_PORT=8080`

### OPS_HEALTHCHECK_PORT
- **Description**: Health check endpoint port for Ops
- **Used by**: Ops
- **Required**: No
- **Default**: `8082`
- **Example**: `OPS_HEALTHCHECK_PORT=8082`

### BKG_HEALTHCHECK_PORT
- **Description**: Health check endpoint port for Background
- **Used by**: Background
- **Required**: No
- **Default**: `8082`
- **Example**: `BKG_HEALTHCHECK_PORT=8082`

---

## Feature Flags

### READ_ONLY_MODE
- **Description**: Enable read-only mode (blocks write operations)
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `READ_ONLY_MODE=true`

### TROUBLESHOOTING_MODE
- **Description**: Enable troubleshooting mode (enables pprof endpoints)
- **Used by**: Service, Ops, Background
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `TROUBLESHOOTING_MODE=true`

### ISOLATION_AUTO_CREATION
- **Description**: Enable automatic isolation creation
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `ISOLATION_AUTO_CREATION=true`

### ISOLATION_AUTO_CREATION_MAX_STORAGE_SIZE
- **Description**: Maximum storage size for auto-created isolations
- **Used by**: Service
- **Required**: No
- **Default**: `100GB`
- **Example**: `ISOLATION_AUTO_CREATION_MAX_STORAGE_SIZE=500GB`

### ISOLATION_ID_VERIFICATION_DISABLED
- **Description**: Disable isolation ID verification in JWT tokens. When set to true, the service will not validate that the isolation ID in the JWT token matches the isolation ID in the request.
- **Used by**: Service
- **Required**: No
- **Default**: `true`
- **Values**: `true` | `false`
- **Example**: `ISOLATION_ID_VERIFICATION_DISABLED=true`

### ENCOURAGE_SEM_SEARCH_INDEX_USE
- **Description**: Encourage use of semantic search indexes
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `ENCOURAGE_SEM_SEARCH_INDEX_USE=true`

---

## Runtime Configuration

### ENABLE_RUNTIME_HEADER_CONFIG
- **Description**: Enable runtime configuration via HTTP headers
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `ENABLE_RUNTIME_HEADER_CONFIG=true`

### EMULATION_MODE
- **Description**: Enable emulation mode (simulates processing delays)
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `EMULATION_MODE=true`

### EMULATION_MIN_TIME
- **Description**: Minimum delay in milliseconds for emulation mode
- **Used by**: Service
- **Required**: No
- **Default**: `100`
- **Example**: `EMULATION_MIN_TIME=50`

### EMULATION_MAX_TIME
- **Description**: Maximum delay in milliseconds for emulation mode
- **Used by**: Service
- **Required**: No
- **Default**: `1000`
- **Example**: `EMULATION_MAX_TIME=2000`

---

## Performance Tuning

### PGVECTOR_DISTANCE_PRECISION
- **Description**: Precision for pgvector distance calculations
- **Used by**: Service
- **Required**: No
- **Default**: `0`
- **Example**: `PGVECTOR_DISTANCE_PRECISION=3`

### PGVECTOR_HNSW_BUILD_MAINTENANCE_MEMORY_MB
- **Description**: Memory limit in MB for HNSW index building
- **Used by**: Service
- **Required**: No
- **Default**: `2048` (2GB)
- **Example**: `PGVECTOR_HNSW_BUILD_MAINTENANCE_MEMORY_MB=4096`

### DOCUMENT_SEMANTIC_SEARCH_MULTIPLIER
- **Description**: Multiplier applied to the CTE embedding scan limit during document semantic search. When searching for documents, the system scans `limit * multiplier` embeddings before deduplicating to unique documents. A higher value increases the chance of returning the requested number of documents when documents have multiple chunks (embeddings).
- **Used by**: Service
- **Required**: No
- **Default**: `10`
- **Example**: `DOCUMENT_SEMANTIC_SEARCH_MULTIPLIER=20`

### HTTP_REQUEST_TIMEOUT
- **Description**: Timeout duration for HTTP requests
- **Used by**: Service
- **Required**: No
- **Default**: `25s`
- **Example**: `HTTP_REQUEST_TIMEOUT=30s`

### HTTP_REQUEST_BACKGROUND_TIMEOUT
- **Description**: Timeout duration for async/background processing
- **Used by**: Service
- **Required**: No
- **Default**: `60s`
- **Example**: `HTTP_REQUEST_BACKGROUND_TIMEOUT=120s`

### EMBEDDING_QUEUE_MAX_RETRY_COUNT
- **Description**: Maximum retry count for embedding queue processing
- **Used by**: Service
- **Required**: No
- **Default**: `300`
- **Example**: `EMBEDDING_QUEUE_MAX_RETRY_COUNT=500`

### DB_METRICS_UPDATE_PERIOD_SEC
- **Description**: Update period in seconds for database metrics collection
- **Used by**: Service, Background
- **Required**: No
- **Default**: `300` (5 minutes)
- **Example**: `DB_METRICS_UPDATE_PERIOD_SEC=600`

### DB_CONFIG_PULL_INTERVAL_SEC
- **Description**: Interval in seconds for pulling database configuration
- **Used by**: Background
- **Required**: No
- **Default**: `600` (10 minutes)
- **Example**: `DB_CONFIG_PULL_INTERVAL_SEC=300`

### TRUNCATE_MAX_LENGTH
- **Description**: Maximum length for truncating log messages
- **Used by**: All services
- **Required**: No
- **Default**: `1024`
- **Example**: `TRUNCATE_MAX_LENGTH=2048`

---

## Logging & Monitoring

### LOG_LEVEL
- **Description**: Logging level
- **Used by**: All services
- **Required**: No
- **Default**: `INFO`
- **Values**: `DEBUG` | `INFO` | `WARN` | `ERROR`
- **Example**: `LOG_LEVEL=DEBUG`

### LOG_PERFORMANCE_TRACE
- **Description**: Enable performance trace logging
- **Used by**: All services
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `LOG_PERFORMANCE_TRACE=true`

### LOG_SERVICE_METRICS
- **Description**: Enable service metrics logging
- **Used by**: Service, Ops
- **Required**: No
- **Default**: `true`
- **Values**: `true` | `false`
- **Example**: `LOG_SERVICE_METRICS=false`

### OTEL_EXPORTER_OTLP_ENDPOINT
- **Description**: OpenTelemetry OTLP exporter endpoint
- **Used by**: Service
- **Required**: No
- **Example**: `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`

---

## HTTP Client Configuration

### HTTP_CLIENT_TIMEOUT
- **Description**: Default timeout for HTTP client requests
- **Used by**: All services
- **Required**: No
- **Default**: Varies by operation
- **Example**: `HTTP_CLIENT_TIMEOUT=30s`

### HTTP_CLIENT_MAX_RETRIES
- **Description**: Maximum number of retries for HTTP client requests
- **Used by**: All services
- **Required**: No
- **Default**: `3`
- **Example**: `HTTP_CLIENT_MAX_RETRIES=5`

---

## Development & Testing

### ENABLE_DB_PROXY
- **Description**: Enable database proxy for development
- **Used by**: Background
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `ENABLE_DB_PROXY=true`

### RANDOM_EMBEDDER_ENABLED
- **Description**: Enable random embedder for testing
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `RANDOM_EMBEDDER_ENABLED=true`

### RANDOM_EMBEDDER_DELAY
- **Description**: Delay in milliseconds for random embedder
- **Used by**: Service
- **Required**: No
- **Example**: `RANDOM_EMBEDDER_DELAY=100`

### INJECT_TEST_HEADERS
- **Description**: Inject test headers in requests (for testing)
- **Used by**: Service
- **Required**: No
- **Default**: `false`
- **Values**: `true` | `false`
- **Example**: `INJECT_TEST_HEADERS=true`

### KEEP
- **Description**: Keep test resources running after integration tests
- **Used by**: Integration tests
- **Required**: No
- **Values**: `true` | duration (e.g., `30s`, `5m`)
- **Example**: `KEEP=true` or `KEEP=30s`

---

## Service-Specific Usage Summary

### Service (cmd/service)
Uses most environment variables including:
- All database configuration
- All connection pooling variables
- SAX authentication
- GenAI integration
- Feature flags (smart chunking, emulation, etc.)
- Performance tuning
- Service ports

### Ops (cmd/ops)
Uses:
- Database configuration
- Connection pooling (generic and ingestion)
- SAX authentication
- GenAI integration (limited)
- Feature flags (read-only, troubleshooting)
- Ops-specific ports

### Background (cmd/background)
Uses:
- Database configuration
- Single connection pool (MAX_CONNS)
- SAX client configuration
- GenAI integration
- Background-specific settings
- DB proxy for development

---

## Configuration Best Practices

1. **Required Variables**: Always set required variables for your deployment environment (local vs cloud)
2. **Cloud Providers**: Set `CLOUD_PROVIDER` appropriately and ensure cloud-specific variables are configured
3. **Security**: Never commit credentials to version control; use cloud secret management
4. **Connection Pools**: Tune connection pool sizes based on your workload and database limits
5. **Timeouts**: Adjust HTTP timeouts based on your network latency and operation complexity
6. **Feature Flags**: Use feature flags to enable/disable functionality without code changes
7. **Development**: Use development-specific variables (like `ENABLE_DB_PROXY`) only in local environments

---

## Updating This Document

**When you add, remove, or rename an environment variable:**

1. Update this document with the new/changed variable information
2. Include the description, default value, and which services use it
3. Update the service-specific usage summary if needed
4. Commit the documentation changes with your code changes

This ensures the documentation stays synchronized with the codebase.
