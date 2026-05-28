# ADR 0001: Delete Materials From Trackmate 2.0

## Status

Accepted.

## Context

The old Trackmate runtime had a Materials topic, material batching, material progress buttons, material progress events, and material-specific database tables.

For Trackmate 2.0, Materials is not part of the useful product surface. The project needs to preserve daily-task work history, reports, alerts, participants, workspaces, and non-material progress history. It does not need to preserve material data or callback compatibility.

Keeping Materials as legacy state would add:

- unused enum values;
- unused tables and foreign keys;
- legacy callback parsing;
- formatter branches for events the product no longer creates;
- tests for behavior the product no longer wants.

## Decision

Trackmate 2.0 deletes Materials from runtime and schema.

The Go runtime must not:

- create or repair a Materials topic;
- store Materials topic bindings;
- parse `material:*` callbacks as first-class callbacks;
- create material batches, items, pending inputs, or progress rows;
- format material progress events;
- run material batching worker code.

Migration `202605280002_drop_materials.sql` intentionally removes material rows and material schema objects while preserving Today, alerts, and non-material progress data.

## Consequences

- Old Telegram material cards may still exist visually in old groups, but their buttons are treated as stale unknown callbacks.
- Material database data is removed during migration.
- The data model is smaller and easier to reason about.
- Future resource/library workflows must be designed as new product work, not as a revival of the old Materials implementation.
