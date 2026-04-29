# Commit Message Generator

## Description
Creates a well-formed git commit from current changes, auto-prefixing with the work item ID extracted from the branch name.

## Trigger
When user asks to:
- "commit"
- "create a commit"
- "commit my changes"
- "stage and commit"

## Steps

1. **Extract work item ID from branch name**
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```
   Parse the story/bug ID (e.g., `feature/US-736080-description` -> `US-736080`).

2. **Check for changes**
   ```bash
   git status --short
   git diff --stat
   git diff --cached --stat
   ```
   If no changes, report nothing to commit and stop.

3. **Read the diff to understand what changed**
   ```bash
   git diff
   git diff --cached
   ```

4. **Review recent commits for style**
   ```bash
   git log --oneline -10
   ```

5. **Stage relevant files**
   - Stage specific files by name (not `git add -A`)
   - Exclude: `.env`, credentials, IDE config, build artifacts (`bin/`, `build/`, `distribution/`), vendored deps, `*.log`, `*.out`

6. **Draft commit message**
   - Prefix: `<WORK-ITEM-ID>: <Summary>` (e.g., `US-736080: Add vector index rebuild endpoint`)
   - Summary: imperative mood, max 72 characters
   - Body (optional): only if summary is insufficient, explain *why*
   - If no work item ID found in branch name, ask the user

7. **Show proposed message and file list - wait for user approval**

8. **Create commit** using HEREDOC format

## Rules
- Never amend unless explicitly asked
- Never use `--no-verify`
- Never push to remote
- If pre-commit hook fails: diagnose, fix, re-stage, create NEW commit
- If branch has no work item ID, ask the user for the prefix

## Notes
- Follows project convention: `US-1234: Short Message` (from CLAUDE.md)
- Copyright header format: block comment `/* ... */` (not `//` line comments)
- Matches existing commit style in the repository
