# Model Lifecycle & Deprecation

## Overview

This guide explains how model deprecation and lifecycle data flows through the Gateway
and is consumed by downstream services (Autopilot/Assistant).

## Architecture: Two Metadata Systems

### 1. Helm Model Metadata (RUNTIME SOURCE OF TRUTH)

**File**: `distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml`

This is a Kubernetes ConfigMap template deployed at `/models-metadata/model-metadata.yaml`.
It is the **only** source of deprecation/lifecycle data that reaches the API response.

Each model entry can contain:
- `lifecycle` — One of: `Generally Available`, `Nearing Deprecation`, `Deprecated`, `Preview`
- `deprecation_date` — Format `'YYYY-MM-DD'`, the vendor shutdown date
- `alternate_model_info` — Replacement model (name, provider, creator) for auto-fallback

When a model should automatically transition from `Generally Available` to
`Nearing Deprecation`/`Deprecated` based on `deprecation_date`, leave `lifecycle`
empty in Helm metadata so runtime inference can apply.

### 2. Spec YAMLs (NOT used for deprecation)

**Files**: `internal/models/specs/**/*.yaml`

Some spec YAMLs contain `deprecationDate` or `lifecycle` blocks. **These do NOT flow to the
API response.** The loader (`internal/models/loader/`) does not copy lifecycle fields, and
`EnhancedModelConfig.ToModel()` in `config/types.go` skips them entirely.

Spec YAML lifecycle fields are for reference/documentation only. A banner comment in each
affected spec file explains this.

## Data Flow

```
Helm model-metadata.yaml
    → K8s ConfigMap (/models-metadata/model-metadata.yaml)
    → LoadModelMetadataFromFile()  [cmd/service/api/models.go]
    → enrichWithMetadata()         [cmd/service/api/models.go]
    → API response fields:
        - model.Lifecycle
        - model.DeprecationDate
        - model.DeprecationInfo.IsDeprecated
        - model.DeprecationInfo.ScheduledDeprecationDate
```

## Key Functions

### `enrichWithMetadata()` (cmd/service/api/models.go)

Populates API response fields from Helm metadata:
- `model.Lifecycle = meta.Lifecycle` (or inferred via `inferLifecycleFromDate()`)
- `model.DeprecationDate = meta.DeprecationDate`
- `model.DeprecationInfo.IsDeprecated = (meta.Lifecycle == "Deprecated")`
- `model.DeprecationInfo.ScheduledDeprecationDate = meta.DeprecationDate`

### `inferLifecycleFromDate()` (cmd/service/api/models.go)

When `lifecycle` is empty in Helm metadata, it is inferred from `deprecation_date`:
- No date or `"NA"` → `"Generally Available"`
- Date in the past → `"Deprecated"`
- Date within 3 months → `"Nearing Deprecation"`
- Date further out → `"Generally Available"`

If `lifecycle` is set explicitly in Helm metadata, that explicit value wins and date-based
inference does not run.

## Downstream Consumer: Autopilot/Assistant

Autopilot **only** uses the `deprecation_info` object from the API response:
- `is_deprecated: bool`
- `scheduled_deprecation_date: Optional[date]`

Autopilot does **NOT** use the `lifecycle` field.

### Autopilot Behavior

In `check_model_deprecation_and_get_fallback()`:
1. If `is_deprecated == True` AND `scheduled_deprecation_date < today`:
   → Auto-switch to `alternate_model_info` replacement
2. If deprecated but date hasn't passed yet: warn but continue using model
3. `_find_gpt4o_fallback()` also checks `is_deprecated` to prefer non-deprecated versions

## Lifecycle Values

Valid values (defined in `modelMetadataSchema.json`):
| Value | Meaning |
|---|---|
| `Generally Available` | Active, fully supported |
| `Preview` | Available but not GA; may change |
| `Nearing Deprecation` | Will be deprecated soon (within 3 months) |
| `Deprecated` | End of life; Autopilot may auto-switch |

## How to Add/Update Deprecation Dates

1. Edit `distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml`
2. Find the model key (e.g., `claude-3-7-sonnet`)
3. Add `deprecation_date: 'YYYY-MM-DD'` at the end of the model's block
4. Only set `lifecycle:` explicitly when you want to override runtime inference; otherwise
   leave it empty so the date can drive lifecycle transitions
5. If a replacement model exists, ensure `alternate_model_info` is set

## Vendor Sources

Use the official vendor lifecycle pages when adding or updating shutdown dates:

- AWS Bedrock: `https://docs.aws.amazon.com/bedrock/latest/userguide/model-lifecycle.html`
- Azure OpenAI: `https://learn.microsoft.com/en-us/azure/ai-services/openai/concepts/model-retirements`
- GCP Vertex AI: `https://cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versions`

Vendor shutdown dates can change close to retirement, so re-validate them periodically.

**Do NOT** edit spec YAMLs for deprecation purposes — those fields have no runtime effect.

## Common Pitfalls

- **Editing spec YAMLs for deprecation**: Has no effect on API. Always edit Helm metadata.
- **Forgetting alternate_model_info**: Without it, Autopilot can't auto-switch when a model is deprecated.
- **Gemini model_name collisions**: Multiple Gemini versions share `model_name` (e.g., `Gemini-Flash`).
  Ensure you edit the correct version key (e.g., `gemini-2.0-flash` vs `gemini-2.5-flash`).
- **Lifecycle without date**: If you set `lifecycle: Deprecated` without a `deprecation_date`,
  Autopilot won't auto-switch (it checks both `is_deprecated` AND `scheduled_deprecation_date`).
