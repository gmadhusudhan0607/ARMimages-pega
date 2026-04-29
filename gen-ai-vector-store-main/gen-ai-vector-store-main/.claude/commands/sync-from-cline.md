Synchronize AI tooling config from Cline skills to Claude Code commands.

Direction: `.cline/skills/` (source) -> `.claude/commands/` (target)

## Steps

1. List all skill files in both directories:
   ```bash
   ls -1 .cline/skills/*.md
   ls -1 .claude/commands/*.md
   ```

2. Build a mapping using convention: same filename maps to same filename, with these exceptions:

   | Cline skill | Claude command |
   |---|---|
   | `.cline/skills/validate-skills.md` | `.claude/commands/validate-commands.md` |
   | `.cline/skills/sync-to-claude.md` | `.claude/commands/sync-from-cline.md` |

   For all other skills: `<name>.md` -> `<name>.md` (same filename in `.claude/commands/`).

   Dynamically detect any new skills not in the exceptions list:
   ```bash
   # Skills that need name mapping (exceptions)
   EXCEPTIONS="validate-skills.md:validate-commands.md sync-to-claude.md:sync-from-cline.md"

   for SKILL in $(ls -1 .cline/skills/*.md | xargs -n1 basename); do
     MAPPED=$(echo "$EXCEPTIONS" | tr ' ' '\n' | grep "^$SKILL:" | cut -d: -f2)
     TARGET="${MAPPED:-$SKILL}"
     echo ".cline/skills/$SKILL -> .claude/commands/$TARGET"
   done
   ```

3. For each Cline skill file:
   a. Read the Cline skill content
   b. Check if the corresponding Claude command exists

   **If Claude command does NOT exist:**
   - Generate a draft by converting the Cline format to Claude format:
     - Strip Title (`# ...`), Description, and Trigger sections
     - Keep Steps and Rules sections
     - Simplify formatting (remove checkboxes, numbered sub-steps -> prose)
   - Write the draft to `.claude/commands/`
   - Report: "Created draft: `.claude/commands/<name>.md`"

   **If Claude command EXISTS:**
   - Read both files
   - Compare the core intent (Steps, Rules, patterns) between them
   - If they are aligned (same steps, same rules, same patterns): report "In sync: `<name>.md`"
   - If they have drifted (different steps, missing rules, different patterns): report the drift details but do NOT overwrite

4. Optionally (if the user asks): compare `.clinerules` with `CLAUDE.md` and flag significant differences.

5. Print a summary report:
   ```
   Sync Report:
   - Created N new drafts
   - Found M files in sync
   - Found K files with drift:
     - <name>.md: <brief description of drift>
   ```

6. After sync is complete, suggest running validation:
   ```
   Sync complete. To validate command quality, run:
     /validate-commands

   This checks for security issues, reliability problems, and clarity improvements.
   ```

## Rules
- NEVER overwrite existing `.claude/commands/` files without explicit user approval.
- NEVER modify `.cline/skills/` files - they are the source of truth.
- Do NOT convert Cline workflows (`.cline/workflows/`) - no Claude equivalent exists.
- Skip comparing these pairs (self-referential or format-specific):
  - sync-to-claude / sync-from-cline
  - validate-skills / validate-commands
- Report what you did clearly so the user can review and approve changes.
