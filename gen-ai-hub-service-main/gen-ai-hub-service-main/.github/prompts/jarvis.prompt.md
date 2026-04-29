---
description: 'Run the canonical end-to-end user-story implementation workflow by invoking the `jarvis` orchestrator agent.'
---

Invoke the `jarvis` agent (defined in `.github/agents/jarvis.md`) with the following target:

$ARGUMENTS

Pass the argument string to the agent as the work-item reference. If the argument is empty, the agent will ask for the work-item ID and description.

The jarvis agent is the single source of truth for the canonical workflow; this prompt is a thin CLI shim so that `/jarvis` in the Copilot CLI dispatches the same orchestrator that the cloud coding agent auto-invokes for new user stories.
