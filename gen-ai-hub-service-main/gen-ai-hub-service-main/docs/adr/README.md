# Agentic Decision Records (ADR)

This directory contains Agentic Decision Records (ADRs) for the GenAI Hub Service.

## What is an ADR?

An **Agentic Decision Record** is a document that captures an important architectural or design decision made during agentic coding sessions, along with its context and consequences.

### The Pun: "Agentic" = "Architectural"

While maintaining full compatibility with traditional Architecture Decision Records (ADRs) and their tooling, we emphasize that these decisions emerge from **agentic coding workflows** - collaborative sessions between human developers and AI coding agents.

### Purpose: Context Management

ADRs serve a dual purpose in agentic development:

1. **Traditional Role**: Document important decisions for the team and future developers
2. **Agentic Role**: Unclutter the main context window during AI-assisted development

Just as subagents handle specific research or implementation tasks to preserve the main agent's context, ADRs capture decision-making processes and evaluations **outside the main coding context**. This allows:

- Complex architectural evaluations without bloating the main conversation
- Reusable decision documentation across multiple coding sessions
- Clear separation between exploration (ADR creation) and implementation (main context)
- Future AI agents can reference past decisions without re-exploring alternatives

**Think of ADRs as "saved subagent work"** - they contain the research, comparison, and reasoning that would otherwise consume valuable context tokens.

## ADR Format

We use the format proposed by Michael Nygard:

- **Title**: Short noun phrase describing the decision
- **Status**: Proposed | Accepted | Deprecated | Superseded
- **Context**: What is the issue we're seeing that is motivating this decision?
- **Decision**: What is the change we're proposing and/or doing?
- **Consequences**: What becomes easier or more difficult to do because of this change?

## ADR Naming Convention

ADRs are numbered sequentially: `NNNN-title-with-dashes.md`

Example: `0001-record-architecture-decisions.md`

## ADR Lifecycle

1. **Proposed** - Decision under discussion
2. **Accepted** - Decision approved and implemented
3. **Deprecated** - Decision no longer recommended (but still in effect for existing code)
4. **Superseded** - Decision replaced by a new ADR (link to the new one)

## Creating a New ADR

### During Agentic Coding Sessions

When an architectural decision requires extensive research or comparison:

1. **Recognize the need**: Decision involves multiple approaches with trade-offs
2. **Create ADR**: Use template to document context, alternatives, and decision
3. **Keep main context clean**: ADR becomes the "subagent's output"
4. **Reference in code**: Link to ADR from `.github/copilot-instructions.md` or relevant files

### Manual Creation

1. Copy `0000-template.md` to a new file
2. Increment the number (check the highest existing number)
3. Update the title, date, and status
4. Fill in Context, Decision, and Consequences sections
5. Submit for review

### When to Create an ADR

- Comparing 3+ implementation approaches
- Decisions affecting multiple parts of the system
- Security or performance trade-offs
- Pattern selection (when not in Go stdlib or reference codebases)
- Infrastructure or deployment strategy changes
- Breaking changes or major refactorings

## Agentic Workflow Benefits

### Context Window Management

- **Problem**: Complex decisions consume 10k-50k tokens in main conversation
- **Solution**: ADR captures decision in structured document (~2k tokens)
- **Benefit**: Main coding session stays focused on implementation

### Decision Reusability

- **Problem**: Each new AI session must re-learn project decisions
- **Solution**: ADRs provide decision history with full rationale
- **Benefit**: New sessions start with accumulated project knowledge

### Pattern Recognition

- **Problem**: AI may suggest previously-rejected approaches
- **Solution**: ADRs document why alternatives were rejected
- **Benefit**: Consistent decision-making across sessions

## Architecture Reviewer Agent

An `architecture-reviewer` subagent is available to automatically review code changes against documented ADRs.

**Usage**:
```
Ask the architecture-reviewer agent to review changes:
"Review these changes against our ADRs"
"Check if this implementation aligns with documented decisions"
"Is adding env vars for credentials okay?"
```

**What it does**:
- Reads all ADRs and builds decision index
- Analyzes code changes (git diff or specific files)
- Detects deviations from documented patterns
- Classifies violations by severity (Critical/Deviation/Edge Case/Evolution)
- Provides actionable feedback with 3 options: Align/Supersede/Exception

**Agent location**: `.github/agents/architecture-reviewer.md` (project-level)

**When to use**:
- Before committing significant changes
- During implementation of new features
- When uncertain if a change violates documented decisions
- During code reviews

**Deviation levels**:
- **Level 1: Violation** - Direct contradiction (block/warn strongly)
- **Level 2: Deviation** - Different approach (warn, request justification)
- **Level 3: Edge Case** - Scenario not covered by ADR (suggest update)
- **Level 4: Evolution** - Better approach found (recommend superseding)

## Tools

This directory maintains full compatibility with standard ADR tools:

- [adr-tools](https://github.com/npryce/adr-tools) - Command line tools for working with ADRs
- [log4brains](https://github.com/thomvaill/log4brains) - Web UI and CLI for ADRs

To use adr-tools:
```bash
# Install
npm install -g adr-log

# Create new ADR
adr new "Use PostgreSQL for data storage"

# List all ADRs
adr list

# Generate index
adr generate toc > index.md
```

## Index

<!-- adrlog -->

- [ADR-0000](0000-template.md) - ADR Template
- [ADR-0001](0001-use-eso-for-sax-credentials.md) - Use External Secrets Operator for SAX Credentials
- [ADR-0002](0002-use-nested-test-packages.md) - Use Nested Test Packages Following Go Stdlib Convention
- [ADR-0003](0003-native-gemini-generatecontent-endpoint.md) - Use Native Gemini /generateContent Endpoint for Image Generation
- [ADR-0004](0004-api-version-governance-with-webrtc-bypass.md) - API Version Governance with WebRTC Bypass
- [ADR-0005](0005-lazy-on-demand-model-cache.md) - Lazy On-Demand Model Cache with TTL-Based Expiry
- [ADR-0006](0006-model-override-for-customer-managed-openai-compatible-endpoints.md) - Model Override for Customer-Managed OpenAI-Compatible Endpoints

<!-- adrlogstop -->
