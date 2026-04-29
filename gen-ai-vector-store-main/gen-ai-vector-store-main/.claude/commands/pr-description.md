Analyze changes in the current branch compared to `main` and generate a PR description.

## Steps

1. Identify the current branch:
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```

2. Get commits not in main:
   ```bash
   git log main..HEAD --oneline --no-merges
   ```

3. Get file change summary:
   ```bash
   git diff main...HEAD --stat
   ```

4. Get the full diff:
   ```bash
   git diff main...HEAD
   ```

5. If there are no commits ahead of main, report that the branch is up-to-date and stop.

6. Analyze the changes and produce a PR description in this format:

   ### What
   1-3 sentence summary of what was changed and why.

   ### Changes
   Bullet-point list of key changes grouped by area:
   - `cmd/service/` - description of change
   - `cmd/ops/` - description of change
   - `cmd/background/` - description of change
   - `cmd/middleware/` - description of change
   - `internal/` - description of change
   - `src/integTest/` - description of test coverage added/modified

## Rules
- Be concise and factual - describe **what was done**, not what should be done next.
- Use past tense ("Added", "Fixed", "Refactored", "Extracted").
- Group related changes together by package or concern.
- Reference affected files or modules where relevant.
- Do NOT include next steps, deployment instructions, follow-up tasks, or TODOs.
- Focus on the intent and impact of changes, not line-by-line details.
