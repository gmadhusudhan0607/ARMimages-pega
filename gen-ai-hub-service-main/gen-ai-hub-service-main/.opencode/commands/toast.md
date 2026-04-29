---
description: Start implementing a user story end-to-end following the canonical agent workflow
agent: build
---

Before doing anything else, print the following ASCII art exactly as shown:

```
 ___________      `
|\   ((#####)\    ` \
| \ ==))###(= \     \ //)
|  \ ||#####|_ \     ((#(
|[> \___________\     ))#|
| | |            |    ||#/
 \  |            |    ||/
  \ |            |
   \|____________|

```

Then implement the spec on `.opencode/specs` - if not provided or you cant find in your context, ask what spec we should work on. List the specs you find so i can pick. Then you:
1. **First**, dispatch @git-committer to make sure the branch is up to date or need to be rebased because of changes on the origin.
2. **Then**, based on the specification, dispatch the appropriate developer agents:
   - @go-developer for application code (handlers, middleware, business logic in `cmd/` and `internal/`) with a TDD approach
   - @go-infra-engineer for infrastructure changes (Terraform, Helm, SCE definitions, model specs, metadata)
3. **Then**, dispatch @go-test-developer to write or update tests (unit, integration, live tests) based on the changes made.
4. **Then**, dispatch @qa-tester to run the build and unit tests (`make build && make test`) and fix any failures.
5. **Then**, dispatch @qa-integration-tester to run relevant integration tests and fix any failures.
6. **Then**, dispatch @qa-test-live to run the relevant live tests, including memory leak checks.
7. **Then**, dispatch @reviewer to perform a final code review checking for duplicated code, unnecessary complexity, and adherence to the user story requirements and ADRs.
8. **Finally**, report a summary of all changes made, tests passing, and any remaining items.

Important rules:
- All changes MUST be forward and backward compatible (zero-downtime requirement).
- Consult `SCE_TO_PRODUCT_MAPPING.md` before any infrastructure changes.
- **Always** consider that this does not affect Launchpad or UAS, so UasAuthentication and SaxEnrichment are most probably out of scope. Ask if in doubt.
- No dead code — remove unused functions, types, and variables.
- Follow existing patterns in the codebase.
- Read the relevant guide from `docs/guides/` before working on each area.
