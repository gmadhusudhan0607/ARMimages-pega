# Sync Skills to Claude Code

## Description
Synchronizes Cline skills to Claude Code commands format. Cline skills are the source of truth - this generates/updates `.claude/commands/` counterparts.

## Trigger
When user asks to:
- "sync skills to claude"
- "update claude commands from cline"
- "generate claude commands"
- "sync cline to claude"

## Steps

1. **List all skill files in both directories**
   ```bash
   ls -1 .cline/skills/*.md
   ls -1 .claude/commands/*.md
   ```

2. **Build mapping** using convention: same filename maps to same filename, with these exceptions:

   | Cline skill | Claude command |
   |---|---|
   | `validate-skills.md` | `validate-commands.md` |
   | `sync-to-claude.md` | `sync-from-cline.md` |

   For all other skills: `<name>.md` → `<name>.md` (same filename in `.claude/commands/`).

   Dynamically detect any new skills not in the exceptions list:
   ```bash
   # Skills that need name mapping (exceptions)
   EXCEPTIONS="validate-skills.md:validate-commands.md sync-to-claude.md:sync-from-cline.md"

   for SKILL in $(ls -1 .cline/skills/*.md | xargs -n1 basename); do
     MAPPED=$(echo "$EXCEPTIONS" | tr ' ' '\n' | grep "^$SKILL:" | cut -d: -f2)
     TARGET="${MAPPED:-$SKILL}"
     echo "$SKILL -> $TARGET"
   done
   ```

3. **For each Cline skill:**

   **If Claude command does NOT exist:**
   - Generate draft by converting Cline -> Claude format:
     - Strip Title, Description, Trigger sections
     - Keep Steps and Rules
     - Simplify formatting
   - Write draft to `.claude/commands/`
   - Report: "Created draft: `<name>.md`"

   **If Claude command EXISTS:**
   - Compare core intent (Steps, Rules, patterns)
   - Aligned -> report "In sync"
   - Drifted -> report drift details, do NOT overwrite

4. **Optionally** (if user asks): compare `.clinerules` with `CLAUDE.md` and flag differences.

5. **Print summary report:**
   ```
   Sync Report:
   - Created N new drafts
   - Found M files in sync
   - Found K files with drift:
     - <name>.md: <brief drift description>
   ```

6. **After sync is complete, suggest running validation:**
   ```
   Sync complete. To validate skill quality, ask Cline:
     "validate skills"

   This checks for security issues, reliability problems, and clarity improvements.
   ```

## Rules
- NEVER overwrite existing `.claude/commands/` without explicit user approval
- NEVER modify `.cline/skills/` - they are the source of truth
- Do NOT convert `.cline/workflows/` - no Claude equivalent
- Skip comparing these pairs (self-referential or format-specific):
  - sync-to-claude / sync-from-cline
  - validate-skills / validate-commands
- Report clearly so user can review

## Notes
- Cline format: Title, Description, Trigger, Steps, Rules, Notes
- Claude format: Description line, Steps, Rules (no Title/Trigger headers)
- The conversion strips Cline-specific metadata and simplifies formatting
