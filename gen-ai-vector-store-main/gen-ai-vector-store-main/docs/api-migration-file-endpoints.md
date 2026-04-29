# Vector Store API Migration Guide: `/file` and `/file-text` Endpoints

> **Scope**: `PUT /v1/{isolationID}/collections/{collectionName}/file` and `PUT /v1/{isolationID}/collections/{collectionName}/file/text`

---

## Summary

Both `/file` and `/file-text` endpoints now delegate all document processing (extraction, chunking, indexing) to the **Smart Chunking** service asynchronously. Vector Store validates the request, forwards it to Smart Chunking, and returns `202 Accepted`.

---

## What Changed

### 1. `consistencyLevel` Query Parameter — Removed

All requests are now asynchronous. Remove `?consistencyLevel=...` from all calls.

### 2. New `documentMetadata` Fields

The `documentMetadata` object extends the existing schema with three new fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enableSmartAttribution` | boolean | `false` | Enable smart attribution extraction (regions, dates, versions) per chunk. |
| `embedSmartAttributes` | boolean | `false` | When `true` **and** `enableSmartAttribution` is also `true`, auto-merges resolved attribute names into `embeddingAttributes`. Ignored when `enableSmartAttribution` is `false`. |
| `enableOCR` | boolean | `false` | Enable OCR for scanned PDFs and images. When disabled, only native text is extracted. |

Example `documentMetadata`:
```json
{
  "embeddingAttributes": ["version"],
  "enableSmartAttribution": true,
  "embedSmartAttributes": false,
  "enableOCR": true
}
```

### 3. Response — `202 Accepted` Only

`201 Created` is no longer returned. All successful submissions return `202 Accepted` with an informational JSON body:

```json
{
  "documentID": "my-document",
  "status": "IN_PROGRESS"
}
```

The response body is optional to parse — `documentID` echoes the request value and `status` is always `"IN_PROGRESS"`.

### 4. Response Headers — Reduced

Only `X-Genai-Vectorstore-Request-Duration-Ms` is returned. The following headers have been removed (metrics are no longer available at submission time):

- `X-Genai-Vectorstore-Db-Query-Time-Ms`
- `X-Genai-Vectorstore-Documents-Count`
- `X-Genai-Vectorstore-Vectors-Count`
- `X-Genai-Vectorstore-Model-Id`
- `X-Genai-Vectorstore-Model-Version`
- `X-Genai-Vectorstore-Embedding-Time-Ms`
- `X-Genai-Vectorstore-Embedding-Calls-Count`

### 5. `/file-text` Content-Type — Now `application/json`

The server now expects `application/json` for `/file-text`, aligning with existing client behavior.

```json
{
  "documentID": "my-text-doc",
  "documentContent": "Plain text content of the document.",
  "documentAttributes": [
    { "name": "version", "type": "string", "value": ["8.8"] }
  ],
  "documentMetadata": {
    "embeddingAttributes": ["version"],
    "enableSmartAttribution": true,
    "embedSmartAttributes": false,
    "enableOCR": false
  }
}
```

---

## Migration Checklist

- [ ] **Remove `consistencyLevel` query parameter** from all `/file` and `/file-text` requests
- [ ] **Adopt new metadata fields** — set `enableSmartAttribution`, `embedSmartAttributes`, `enableOCR` in `documentMetadata`
- [ ] **Handle `202 Accepted` as the only success code** — remove handling for `201 Created`
- [ ] **Remove reliance on removed response headers** if any are present
