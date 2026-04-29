---
description: Start specifying a work unit end-to-end following the agent workflow below
agent: build
---

Before doing anything else, print the following ASCII art exactly as shown:

```
                                 ,.   '\'\    ,---.
Quiet, Pinky; I'm pondering.    | \\  l\\l_ //    |   Err ... right,
       _              _         |  \\/ `/  `.|    |   Brain!  Narf!
     /~\\   \        //~\       | Y |   |   ||  Y |
     |  \\   \      //  |       |  \|   |   |\ /  |   /
     [   ||        ||   ]       \   |  o|o  | >  /   /
    ] Y  ||        ||  Y [       \___\_--_ /_/__/
    |  \_|l,------.l|_/  |       /.-\(____) /--.\
    |   >'          `<   |       `--(______)----'
    \  (/~`--____--'~\)  /           U// U / \
     `-_>-__________-<_-'            / \  / /|
         /(_#(__)#_)\               ( .) / / ]
         \___/__\___/                `.`' /   [
          /__`--'__\                  |`-'    |
       /\(__,>-~~ __)                 |       |__
    /\//\\(  `--~~ )                 _l       |--:.
    '\/  <^\      /^>               |  `   (  <   \\
         _\ >-__-< /_             ,-\  ,-~~->. \   `:.___,/
        (___\    /___)           (____/    (____)    `---'

```

Then lets dig into analyzing all possible impacts of user story "$ARGUMENTS" and create an specification file on .`opencode/specs` directory:
1. **First**, dispatch @git-committer to create a feature branch following the naming convention `{type}/{ticket number}-short-description`. `type` would be `bugfix` for BUG-#### tickets or `feature` for `ENHANCEMENT-###`. Wait for the branch to be created before proceeding. If the user didn't provide enough context for a branch name, ask for one.
2. **Then**, dispatch @rubber-duck to do a design review and create a complete specification. Engage in Socratic dialogue to surface hidden complexity, edge cases, and breaking change risks before any code is written. Separate work into phases to spot complexity early. Aim for simple and clean designs.
3. **Then**, dispatch @reviewer to perform an architecture review checking to assess complexity, adherence to the user story requirements and ADRs.
4. **Then**, dispatch @qa-integration-tester to discern about the write level (unit, integration, manual) test for this, options to automate the testing process and other ideas of how to produce a validation that can be assessed post implementation.
5. **Then** with the main agent, lets look into breaking the effort into tasks that fit the agents we have at hand in .opencode/agents that are more suitable to each task.
9. **Finally**, create an specification document describing the problem, the tasks identified and the validations that should be done to decide if work is complete.

Important rules:
- The approach MUST be forward and backward compatible (zero-downtime requirement).
- Consult `SCE_TO_PRODUCT_MAPPING.md` for how infrastructure is configured.
- **Always** consider that this does not affect Launchpad or UAS, so UasAuthentication and SaxEnrichment are most probably out of scope. Ask if in doubt.
- Read the relevant guide from `docs/guides/` before working on each area.
