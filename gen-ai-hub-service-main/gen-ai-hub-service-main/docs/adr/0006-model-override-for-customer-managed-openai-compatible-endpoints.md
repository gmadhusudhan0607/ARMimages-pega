# ADR-0006: Model Override for Customer-Managed OpenAI-Compatible Endpoints

- **Status**: Proposed
- **Date**: 2026-04-16
- **Deciders**: Lukasz Korzen (Gateway), Santosh Hegde (Autopilot), Andrzej Lassak, Julien Etienne (VP GenAI Engineering)
- **Related ADRs**: [ADR-0001](0001-use-eso-for-sax-credentials.md) (credential management pattern), [ADR-0004](0004-api-version-governance-with-webrtc-bypass.md) (Azure-specific API governance)

## Context

Some customers need to route GenAI traffic through customer-managed LLM infrastructure rather than Pega-managed endpoints. The main drivers are regulatory, compliance, security, and traffic-control constraints rather than model quality alone.

For Pega Cloud, Gateway already sits in the request path between Autopilot and the underlying model providers. Any model-override solution for Gateway must preserve that architecture, keep routing explicit, and avoid introducing a second agentic loop in Infinity or Autopilot.

This ADR covers the Gateway-side architectural decision only. On-premises deployments, where Gateway is not part of the runtime stack, require a companion decision in the Autopilot repository.

### Problem Statement

How should Gateway support customer-managed LLM endpoints in Pega Cloud so that customers can use their own OpenAI-compatible chat-completion endpoints without:

- duplicating the agentic loop,
- hiding customer-managed routing inside Azure-specific behavior,
- exposing customer credentials or metadata incorrectly, or
- breaking coexistence with Pega-managed models?

### Requirements

- **Gateway scope**: This ADR applies only to Pega Cloud, where Gateway participates in the runtime call path.
- **Additive behavior**: Customer-managed models must coexist with Pega-managed models. Customers may also choose a customer-managed-only setup, but that must remain an explicit support tradeoff rather than the default assumption.
- **Single loop**: The solution must not introduce a second agentic loop or move orchestration into Infinity.
- **Initial protocol scope**: The first iteration supports only OpenAI-compatible chat-completion endpoints.
- **Explicit routing contract**: Customer-managed models must be clearly distinguishable from Pega-managed models in model discovery and request routing, including cases where model names overlap.
- **Stable routing identity**: The model-discovery contract must provide a routing identity for customer-managed models that is independent of the display or model name so overlapping names remain unambiguous.
- **Default selection compatibility**: Default-model selection must be able to target a customer-managed model.
- **Credential safety**: Customer credentials must remain server-side and follow the project's existing secret-management approach. They must never appear in `/models` responses or other client-visible configuration payloads.
- **Provider agnosticism behind the contract**: If a customer's backend is not natively OpenAI-compatible, the customer is responsible for supplying a translation layer that presents the supported contract to Gateway.
- **Metadata responsibility**: Customers must provide the metadata required for model discovery and selection. Gateway must not infer arbitrary customer-model metadata.

### Assumptions

- Pega-managed models remain the recommended and fully-supported path.
- Model override exists for customers who cannot use Pega-managed models for business or regulatory reasons.
- Customer-managed model behavior, compatibility, and performance remain the customer's responsibility.
- Any companion Autopilot changes will align with the Gateway decision documented here rather than redefining it.

## Decision

For Pega Cloud, Gateway will support model override as an additive capability for customer-managed, OpenAI-compatible chat-completion endpoints.

The decision includes the following architectural constraints:

1. **Gateway remains in the Pega Cloud path.** Model override does not bypass Gateway in Pega Cloud.
2. **Customer-managed models are a distinct routing class.** They are surfaced distinctly from Pega-managed models, with a stable routing identity separate from display name, so that routing and model selection remain explicit even when names overlap.
3. **Customer-managed traffic uses a dedicated contract separate from Azure-specific behavior.** Model override is not treated as a hidden special case of Azure routing.
4. **The initial feature scope is narrow by design.** This ADR covers chat completions only. Other operations are out of scope unless documented by a later decision.
5. **Gateway owns only the supported contract, not the customer's backend implementation.** Customers may use any underlying provider if they present the required OpenAI-compatible interface to Gateway.
6. **Credential handling stays server-side.** Customer endpoint credentials follow existing secret-management decisions and are not part of client-visible model metadata.
7. **On-premises support is out of scope for this ADR.** That path is handled outside Gateway.

### Acceptance Criteria

- `GET /models` can advertise one or more customer-managed models without breaking or hiding existing Pega-managed models.
- The model-discovery contract makes customer-managed models distinguishable from Pega-managed models for routing and selection.
- A customer-managed model can participate in default-model selection.
- Gateway can proxy customer-managed chat-completion requests without Azure-specific assumptions such as Azure-only API governance.
- The feature remains additive and backward-compatible for existing Gateway behavior.
- The design does not require Pega to expose customer credentials, infer arbitrary customer-model metadata, or guarantee compatibility with the customer's backend implementation.

## Alternatives Considered

### Alternative 1: Implement model override inside Infinity or through a second agentic loop

**Description**: Move model-override behavior into Infinity or create a separate orchestration path outside the existing Autopilot loop.

**Pros**:
- Reduces Gateway involvement
- Gives customers direct control over their custom integration point

**Cons**:
- Creates a second agentic loop to maintain
- Splits feature behavior across multiple implementations
- Increases long-term product and support complexity

**Rejected because**: Maintaining more than one agentic loop is operationally unacceptable and conflicts with the goal of keeping orchestration behavior centralized.

### Alternative 2: Let Autopilot bypass Gateway in Pega Cloud

**Description**: Keep Gateway out of the custom-model call path and let Autopilot call customer-managed endpoints directly in Pega Cloud.

**Pros**:
- One fewer network hop
- Concentrates logic in a single upstream service

**Cons**:
- Breaks the "Gateway is the Pega Cloud entry point for model routing" architecture
- Expands the credential and routing surface outside Gateway
- Makes coexistence with Gateway-driven model discovery and traffic handling less clear

**Rejected because**: For Pega Cloud, bypassing Gateway weakens the existing service boundary instead of extending it cleanly.

### Alternative 3: Reuse Azure routing with mid-flight interception

**Description**: Handle customer-managed traffic as a special case inside the existing Azure-specific routing flow.

**Pros**:
- Lower short-term implementation effort
- Reuses an existing request path

**Cons**:
- Blurs the boundary between generic customer-managed endpoints and Azure-specific behavior
- Makes routing intent harder to understand and reason about
- Increases the risk that Azure-only assumptions leak into model-override traffic

**Rejected because**: Model override is a separate architectural concern and needs an explicit routing contract rather than an implicit branch inside Azure handling.

### Alternative 4: Deploy full Gateway for on-premises model override

**Description**: Use the same Gateway-centered runtime pattern for on-premises deployments.

**Pros**:
- Higher architectural symmetry between cloud and on-prem
- Reuses the same high-level concept across environments

**Cons**:
- Introduces unnecessary runtime footprint for on-prem deployments
- Pulls Gateway into an environment where it is not otherwise required
- Mixes a separate deployment decision into this Gateway-specific ADR

**Rejected because**: On-premises support is a separate deployment and ownership decision and does not belong in the Gateway ADR.

## Consequences

### Positive

- Gateway remains the single architectural entry point for model-override traffic in Pega Cloud.
- Customer-managed and Pega-managed models can coexist without hiding routing intent.
- Customers retain provider choice behind an OpenAI-compatible contract.
- Support, ownership, and credential boundaries stay explicit.
- The design preserves a path for default-model remapping without redefining the existing override concept.

### Negative

- The first iteration is intentionally narrow and does not imply support for embeddings, image generation, or other operations.
- Customers may need to operate a translation layer to satisfy the supported contract.
- Pega cannot guarantee model quality, latency, availability, or feature completeness for customer-managed endpoints.
- If customers disable all Pega-managed models, some out-of-the-box GenAI features may no longer behave as expected.

### Neutral

- Exact provider labels, route names, metadata schema, observability details, rollout steps, and file-level implementation choices are intentionally left to implementation artifacts and should not be treated as part of this ADR unless they become durable architectural constraints.

## References

- [ADR-0001](0001-use-eso-for-sax-credentials.md)
- [ADR-0004](0004-api-version-governance-with-webrtc-bypass.md)
- [Using the Architecture Reviewer Agent](USING_ARCHITECTURE_REVIEWER.md)
