# AGENTS.md

## Project overview
- This repository contains the product code and local tooling for development.
- Prefer minimal, targeted changes over broad refactors unless explicitly requested.
- Preserve existing architecture unless a task requires a structural change.

## How to work
- Start complex tasks with a plan before writing code.
- For non-trivial changes, explain which files will be touched and why.
- If the implementation starts drifting, stop, restate the plan, and continue from the updated plan.

## Source of truth
- Follow existing patterns in nearby code first.
- Prefer referencing canonical files over inventing new patterns.
- If there are conflicting patterns, choose the one used closest to the edited code.

## Commands
- Install dependencies: `make setup`
- Run app locally: `make dev`
- Run lint: `make lint`
- Run tests: `make test`
- Run a focused test first when possible, then broader checks if needed.

## Code change policy
- Keep diffs small and local.
- Do not rename/move files unless necessary for the task.
- Do not introduce new dependencies without a clear reason.
- Update tests for changed behavior.
- Update docs when public behavior, contracts, or setup changes.

## Verification
- Before finishing, run the narrowest relevant validation.
- If code paths changed materially, run lint + tests relevant to touched files.
- Do not claim success without checking command output.

## Safety
- Ask before destructive actions, schema drops, mass renames, or secret rotation.
- Never hardcode secrets or credentials.
- Prefer mock/stub/local fixtures over calling production systems.

## Context management
- Do not load large irrelevant files into context.
- Search the codebase when the correct file is not obvious.
- For long procedures or rare workflows, consult the relevant skill or local instructions instead of improvising.

## Documentation of learnings
- If the same mistake happens more than once, propose an update to AGENTS.md, a Cursor rule, or a skill.

## Knowledge capture
- Do not write intermediate thoughts or task-specific notes into permanent project files.
- Persist only reusable project knowledge with long-term value.
- Update AGENTS.md only for stable, high-signal rules:
  - repeated mistakes,
  - important invariants,
  - canonical commands,
  - validation requirements,
  - project-specific constraints.
- Put long procedures, operational runbooks, and rare workflows into skills or docs, not into AGENTS.md.
- Put architectural decisions into ADR/docs, not into AGENTS.md.
- Before adding a new rule, check whether it is:
  - reusable,
  - non-obvious,
  - verified,
  - short,
  - non-duplicative.
- Prefer proposing a file update at the end of the task instead of editing permanent guidance continuously during implementation.