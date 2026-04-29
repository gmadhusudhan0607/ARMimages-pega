# Using the Architecture Reviewer Agent

The `architecture-reviewer` agent is a specialized subagent that reviews code changes against documented Agentic Decision Records (ADRs) to ensure architectural consistency.

## Quick Start

```bash
# In Claude Code, ask:
"Review these changes against our ADRs"
"Check if my test helper implementation follows ADR-0002"
"Is using environment variables for secrets allowed?"
```

The agent will:
1. Read all ADRs in `docs/adr/`
2. Analyze your changes
3. Detect deviations from documented decisions
4. Provide actionable feedback

## When to Use

### ✅ Always Use Before:
- Committing significant architectural changes
- Implementing new patterns or abstractions
- Adding credential handling logic
- Creating new test utilities or helpers
- Making security or infrastructure decisions

### ✅ Use During:
- Code reviews (reviewers can request ADR check)
- Refactoring existing code
- Implementing features that touch multiple components
- When unsure if an approach violates documented patterns

### ✅ Use When:
- Exploring alternatives to documented patterns
- Discovering potential improvements to existing ADRs
- Encountering edge cases not covered by ADRs

## Example Scenarios

### Scenario 1: New Credential Type

**Question**: "I need to add GCP Vertex credentials. Should I use env vars?"

**Agent checks**: ADR-0001 (Use ESO for SAX Credentials)

**Response**:
```markdown
⚠️ Deviation Detected: Level 1 - Credentials via Environment Variables

**ADR**: ADR-0001 - Use External Secrets Operator for SAX Credentials
**Documented Decision**: All credentials must be mounted via ESO as JSON files
**Recommended Pattern**: Mount to `/genai-gcp-vertex-config/credentials.json`

**Actions**:
1. ✅ Align: Implement ESO mount for GCP credentials
2. 📝 Update: Extend ADR-0001 with GCP-specific section
3. ❌ Not recommended: Environment variables (security risk)
```

### Scenario 2: Test Helper Pattern

**Question**: "Should I create `internal/models/modelstest/` for model test helpers?"

**Agent checks**: ADR-0002 (Use Nested Test Packages)

**Response**:
```markdown
✅ Compliant: Follows ADR-0002 Pattern

**ADR**: ADR-0002 - Use Nested Test Packages Following Go Stdlib Convention
**Documented Pattern**: `package/packagetest` structure (like net/http/httptest)
**Your approach**: `internal/models/modelstest/` ✓ Correct

**Recommended Structure**:
```
internal/models/
├── models.go
├── models_test.go
└── modelstest/
    ├── modelstest.go
    └── modelstest_test.go
```

**Next steps**:
1. Add to import validation: Update TestNoTestutilImportInNonTestFiles
2. Follow naming: Use `modelstest` suffix
3. Add tests: Create modelstest_test.go
```

### Scenario 3: Pattern Evolution

**Question**: "I found a better way to handle context than documented in ADR-0002"

**Agent checks**: ADR-0002 and evaluates improvement

**Response**:
```markdown
🔄 Evolution Detected: Level 4 - Potential ADR Supersession

**Current ADR**: ADR-0002 - Nested test packages
**Your proposal**: [describes new approach]
**Evaluation**: [pros/cons compared to current ADR]

**Recommended Actions**:
1. 📋 Create ADR-0003: [New approach title]
   - Status: "Supersedes ADR-0002"
   - Document why new approach is superior
   - Migration path from old to new pattern

2. 📝 Update ADR-0002:
   - Status: "Superseded by ADR-0003"
   - Add link to new ADR
   - Keep for historical reference

3. 📢 Announce: Inform team of pattern change
```

## Deviation Levels

### Level 1: Violation (Critical)
**Definition**: Direct contradiction of accepted ADR

**Examples**:
- Using env vars for secrets (violates ADR-0001)
- AWS SDK runtime fetch for credentials (violates ADR-0001)

**Action**: Must fix or justify with documented exception

### Level 2: Deviation (Warning)
**Definition**: Different approach than documented pattern

**Examples**:
- Flat test package instead of nested (differs from ADR-0002)
- Different file mount path than standard

**Action**: Align with ADR or document as exception

### Level 3: Edge Case (Info)
**Definition**: Valid scenario not covered by existing ADRs

**Examples**:
- New credential type not in ADR-0001
- Test helper for new package type

**Action**: Extend existing ADR or create new one

### Level 4: Evolution (Suggestion)
**Definition**: Better approach discovered after ADR

**Examples**:
- Superior Go stdlib pattern emerges
- Better tooling available

**Action**: Supersede ADR with new decision

## ADR Update Workflows

### When to Supersede an ADR

Create new ADR that supersedes old one when:
- ✅ Better approach discovered (with clear advantages)
- ✅ Technology/tooling has evolved
- ✅ Original decision assumptions no longer valid
- ✅ Migration path is clear and documented

**Process**:
1. Create `docs/adr/NNNN-new-title.md` with `Status: Supersedes ADR-XXXX`
2. Update old ADR: `Status: Superseded by ADR-NNNN`
3. Document migration path in new ADR
4. Update `.github/copilot-instructions.md` if pattern changes

### When to Add Exception

Add exception to existing ADR when:
- ✅ Valid edge case that doesn't fit standard pattern
- ✅ Temporary deviation with clear timeline
- ✅ Platform-specific requirement
- ✅ Legacy code that can't be migrated yet

**Process**:
1. Edit existing ADR
2. Add "Exceptions" section
3. Document: scenario, justification, alternative used, timeline (if temporary)

### When to Deprecate an ADR

Mark ADR as deprecated when:
- ⚠️ Pattern no longer recommended but still in use
- ⚠️ Migration to new pattern is optional or gradual
- ⚠️ Existing code still follows old pattern

**Process**:
1. Update ADR: `Status: Deprecated`
2. Add deprecation note with date
3. Link to recommended alternative (or new ADR)
4. Document migration path

## Best Practices

### 1. Review Early and Often
Don't wait until code review - check during implementation.

### 2. Use Specific Questions
Instead of "check my code", ask:
- "Does this credential handling follow ADR-0001?"
- "Is my test package structure compliant with ADR-0002?"

### 3. Trust the Agent, But Verify
Agent may not have full context. If you have valid reasons for deviation:
- Document them clearly
- Request ADR update
- Get team consensus

### 4. Keep ADRs Updated
When agent flags same issue repeatedly, ADR needs updating:
- Add exception section
- Clarify ambiguous guidance
- Document edge cases

### 5. Use for Learning
New team members can ask:
- "What's our pattern for handling secrets?"
- "How should I structure test helpers?"
- "What's our approach to credential injection?"

## Common Questions

### Q: Will this slow down development?

**A**: No. Quick checks take seconds. Catching violations early prevents:
- Lengthy code review discussions
- Rework after merge
- Architectural drift

### Q: What if I disagree with an ADR?

**A**: Great! That's how we evolve:
1. Agent flags deviation
2. You explain rationale for different approach
3. If valid, we supersede the ADR
4. Everyone benefits from improved pattern

### Q: Can I skip ADR review?

**A**: For trivial changes (docs, tests, typos), yes. For architectural changes, strongly recommended.

### Q: What if agent gives false positive?

**A**: Document the exception in the ADR or clarify the ADR wording. False positives indicate ADR needs improvement.

## Integration with Workflow

```
┌─────────────────────────────────────────┐
│ 1. Implement Feature                    │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ 2. Ask: "Review against ADRs"           │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ 3. Agent Checks All ADRs                │
│    - Detects patterns                   │
│    - Compares with decisions            │
│    - Flags deviations                   │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ 4. Review Feedback                      │
│    ✅ Compliant? → Commit               │
│    ⚠️ Deviation? → Choose action:       │
│       - Align with ADR                  │
│       - Supersede ADR                   │
│       - Document exception              │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ 5. Update ADR if Needed                 │
└─────────────────────────────────────────┘
```

## Troubleshooting

### Agent not finding ADRs
**Check**: Are ADRs in `docs/adr/NNNN-*.md` format?

### Agent reports false positive
**Solution**: Add exception section to ADR documenting valid edge case

### Agent misses violation
**Solution**: ADR may be ambiguous. Clarify the documented pattern.

### Want to disable for certain files
**Solution**: Document in ADR that certain paths are exempt (e.g., test fixtures)

## See Also

- [ADR Index](README.md) - List of all ADRs
- [ADR Template](0000-template.md) - Template for new ADRs
- [.github/copilot-instructions.md](../../.github/copilot-instructions.md) - Project guidelines and patterns
