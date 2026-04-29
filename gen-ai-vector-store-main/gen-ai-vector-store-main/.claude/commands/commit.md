Create a well-formed commit from the currently staged (and unstaged) changes.

## Steps

1. Get the current branch name to extract the work item ID:
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```
   Extract the story/bug ID from the branch name (e.g., `feature/US-736080-description` -> `US-736080`).

2. Check for staged and unstaged changes:
   ```bash
   git status --short
   git diff --stat
   git diff --cached --stat
   ```
   If there are no changes at all, report that there is nothing to commit and stop.

3. Read the diff to understand what changed:
   ```bash
   git diff
   git diff --cached
   ```

4. Review recent commit messages to match the repository's style:
   ```bash
   git log --oneline -10
   ```

5. Stage all modified and new files that are relevant to the change. Prefer staging specific files by name rather than `git add -A`. Do NOT stage:
   - `.env` files or credentials
   - IDE config files (`.idea/`, `.vscode/`)
   - Build artifacts (`bin/`, `build/`, `distribution/`)
   - Vendored dependencies (`vendor/`)
   - Test output files (`*.log`, `*.out`, `coverage.out`)

6. Draft a commit message following these rules:
   - **Prefix** with the work item ID extracted in step 1 (e.g., `US-736080: `). If no ID is found in the branch name, ask the user.
   - **Summary line**: imperative mood, max 72 characters, describes *what* was done (e.g., `US-736080: Add vector index rebuild endpoint`)
   - **Body** (optional, separated by blank line): add only if the summary alone is insufficient - explain *why*, not *what*
   - Use past-tense descriptions in the body if present ("Extracted", "Fixed", "Added")

7. Show the user the proposed commit message and the list of files to be committed. Wait for approval before committing.

8. Create the commit using a HEREDOC for the message.

## Rules
- Never amend an existing commit unless the user explicitly asks for it.
- Never use `--no-verify` or skip hooks.
- Never push to remote - only create the local commit.
- If a pre-commit hook fails, diagnose the issue, fix it, re-stage, and create a NEW commit.
- If the branch name does not contain a recognizable work item ID, ask the user for the prefix to use.
- Copyright header format in this project is block comment `/* ... */`, not line comments `//`.
