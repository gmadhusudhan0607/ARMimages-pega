---
description: Start implementing a user story end-to-end following the canonical agent workflow
agent: build
---

Before doing anything else, print the following ASCII art exactly as shown:

```
     ╦╔═╗╦═╗╦  ╦╦╔═╗
     ║╠═╣╠╦╝╚╗╔╝║╚═╗
    ╚╝╩ ╩╩╚═ ╚╝ ╩╚═╝
    At your service, sir.
```

Then implement user story $ARGUMENTS following the canonical workflow defined in CLAUDE.md:
1. **First**, dispatch @git-committer to create a feature branch following the naming convention `US-{number}/short-description`. Wait for the branch to be created before proceeding. If the user didn't provide enough context for a branch name, ask for one.
2. **Then**, dispatch @rubber-duck to do a design review and create a complete specification. Engage in Socratic dialogue to surface hidden complexity, edge cases, and breaking change risks before any code is written. Separate work into phases to spot complexity early. Aim for simple and clean designs.
3. **Then**, based on the specification, dispatch the appropriate developer agents:
   - @go-developer for application code (handlers, middleware, business logic in `cmd/` and `internal/`) with a TDD approach
   - @go-infra-engineer for infrastructure changes (Terraform, Helm, SCE definitions, model specs, metadata)
4. **Then**, dispatch @go-test-developer to write or update tests (unit, integration, live tests) based on the changes made.
5. **Then**, dispatch @qa-tester to run the build and unit tests (`make build && make test`) and fix any failures.
6. **Then**, dispatch @qa-integration-tester to run relevant integration tests and fix any failures.
7. **Then**, dispatch @qa-test-live to run the relevant live tests, including memory leak checks.
8. **Then**, dispatch @reviewer to perform a final code review checking for duplicated code, unnecessary complexity, and adherence to the user story requirements and ADRs.
9. **Finally**, report a summary of all changes made, tests passing, and any remaining items.

Important rules:
- All changes MUST be forward and backward compatible (zero-downtime requirement).
- Consult `SCE_TO_PRODUCT_MAPPING.md` before any infrastructure changes.
- **Always** consider that this does not affect Launchpad or UAS, so UasAuthentication and SaxEnrichment are most probably out of scope. Ask if in doubt.
- No dead code — remove unused functions, types, and variables.
- Follow existing patterns in the codebase.
- Read the relevant guide from `docs/guides/` before working on each area.
