---
description: 'Execute an existing specification end-to-end (implementation + tests + QA + review) by invoking the `toast` orchestrator agent.'
---

Invoke the `toast` agent (defined in `.github/agents/toast.md`) with the following target:

$ARGUMENTS

Pass the argument string to the agent as the spec reference (PR number, link, or free-form description of which spec to implement). If the argument is empty, the agent will locate the spec from the current PR body or ask.

The toast agent assumes a specification already exists (typically produced by `brain`). If no spec exists, use `/jarvis` (end-to-end including specification) or `/brain` (produce the spec first).
