
# **❗ IMPORTANT:**
### This package is the **single source of truth** for model specifications.

# **⚠️ WARNING:**
Do **NOT** include any logic or implementation-specific details here.  
Examples of prohibited content:
- Routing aliases
- Model fallbacks
- Model location or availability
- Any other implementation details

# **📝 NOTE on Deprecation/Lifecycle fields:**
Some spec files contain `deprecationDate` or `lifecycle` blocks. These are for **reference only**
and do **NOT** flow to the API response at runtime. The runtime source of truth for deprecation
data is the Helm `model-metadata.yaml` ConfigMap. See `docs/guides/model_lifecycle.md` for details.


##  Specs Directory Structure
```
models/specs/
  infra/
    provider/
      creator/
        models.yaml
```

####  Specs directory structure example

```

├── models/specs/
    ├── aws/                          # Amazon Web Services
    │   ├── bedrock/
    │   │   ├── anthropic/
    │   │   │   ├── claude-3.yaml  
    │   │   │   └── claude-2.yaml
    │   │   ├── amazon/
    │   │   │   └── titan.yaml
    │   │   └── meta/
    │   │       └── llama.yaml
    │   └── sagemaker/
    │       └── custom/
    │           └── models.yaml
    ├── gcp/                          # Google Cloud Platform
    │   └── vertex/
    │       ├── google/
    │       │   └── gemini.yaml
    │       └── anthropic/
    │           └── claude.yaml
    └── azure/                        # Microsoft Azure
    │   └── openai/
    │       └── openai/
    │           ├── gpt-4.yaml        # GPT-4 family models
    │           ├── gpt-3.5.yaml
    │           └── embeddings.yaml
    └── saas/                         # Software-as-a-Service deployments
        └── custom/
            └── custom-01.yaml        # Custom model configuration
  
```