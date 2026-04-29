---
description: 'Produce a specification for a user story (no implementation) by invoking the `brain` orchestrator agent.'
---

Invoke the `brain` agent (defined in `.github/agents/brain.md`) with the following target:

$ARGUMENTS

Pass the argument string to the agent as the work-item reference. If the argument is empty, the agent will ask for the work-item ID and summary.

The brain agent produces a specification in the PR body and, if warranted, a new ADR at `docs/adr/NNNN-title.md`. It does NOT implement code. Use `/jarvis` for end-to-end implementation, or follow up with `/toast` after brain produces the spec.
