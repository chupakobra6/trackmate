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
- Run local API: `make api`
- Run local worker: `make worker`
- Run local Docker stack: `make docker-up`
- Run lint: `make lint`
- Run tests: `make test`
- Run a focused test first when possible, then broader checks if needed.

## Documentation boundaries
- Use `README.md` and `docs/*.md` for tracked, reusable project guidance.
- Do not track or read `private-docs/`; keep machine-specific operational context outside the repository.
- Default assumption: local `.env` plus local Docker is development; a VPS running the long-lived bot is production unless the user says otherwise.
- The tracked Docker default binds PostgreSQL to `127.0.0.1:5432`; do not broaden that host bind unless the user explicitly needs remote database access.
- Trackmate is Go-only. Keep tracked docs focused on the current runtime; do not add migration-era Python/Alembic guides back to the public documentation.
- Product-owned Telegram E2E scenarios live in `e2e/telegram/`; rendered scenarios stay under ignored `tmp/`.
- User-visible Telegram copy lives in `internal/messages/messages.md`; Go code should use `internal/messages` and `internal/ui` instead of embedding Russian message text directly.
- When public Telegram behavior changes, update the matching product docs and E2E scenario comments or expectations in the same pass: timing, topic flow, parser rules, notification level, visible messages, and Progress links.

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
