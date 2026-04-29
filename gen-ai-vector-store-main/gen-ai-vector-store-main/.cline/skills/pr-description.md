# PR Description Generator

## Description
Analyzes changes in the current branch compared to `main` and generates a concise PR description suitable for code review.

## Trigger
When user asks to:
- "prepare PR description"
- "generate PR summary"
- "write PR description"
- "create pull request description"
- "summarize my changes for PR"
- "what did I change"

## Steps

1. **Identify the current branch**
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```

2. **Get list of commits on current branch (not in main)**
   ```bash
   git log main..HEAD --oneline --no-merges
   ```

3. **Get the diff summary (files changed)**
   ```bash
   git diff main...HEAD --stat
   ```

4. **Get the full diff for analysis**
   ```bash
   git diff main...HEAD
   ```

5. **Analyze the changes and produce PR description**

   Based on the git output, write a concise PR description using this structure:

   ### What
   A short (1-3 sentence) summary of what was changed and why.

   ### Changes
   A bullet-point list of the key changes grouped by area:
   - `cmd/service/` - description of change
   - `cmd/ops/` - description of change
   - `cmd/background/` - description of change
   - `cmd/middleware/` - description of change
   - `internal/` - description of change
   - `src/integTest/` - description of test coverage added/modified

## Output Format

The PR description must:
- Be concise and factual - describe **what was done**, not what should be done next
- Use bullet points for individual changes
- Group related changes together by package or concern
- Reference affected files or modules where relevant
- **NOT** include next steps, deployment instructions, follow-up tasks, or TODOs
- Use past tense ("Added", "Fixed", "Refactored", "Extracted", etc.)

## Notes
- If there are no commits ahead of `main`, report that the branch is up-to-date with `main`
- Focus on the intent and impact of changes, not line-by-line details
- Keep the description readable for a human code reviewer, not an automated tool
