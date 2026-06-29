# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-30

## Активный Шаг
- id: `STEP-027`
- status: `готово`
- objective: Полный live E2E текущего локального head и исправление найденных deterministic-сбоев.
- requirement IDs: `REQ-049`, `VAL-007`
- owned paths: `internal/`, `cmd/trackmate-worker/`, `e2e/telegram/scenarios/`, `.project-loop/`
- validation: live E2E run `s030-015143`: pass; log scan timeout/error/panic: clean; DB `pending_inputs=0`, unpublished progress `0`; `go test ./...`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate`: pass
- done criteria: scenarios `00`, `01..11`, split `12`, split `13`, `14` pass on the test Telegram bot; found product/test-control issues are fixed; no production deploy performed.

## Фокус Ревью
- Проверить только изменения, найденные E2E: pending time source, routine reason prompt, goal prompt fallback, deterministic `/control/tick`, E2E reset cleanup и progress scenario assertion.
- Production deploy отдельно не выполнялся и требует отдельного решения.

## Примечания
- STEP-028 production routine reset уже завершен ранее; в будущем update message все еще нужно попросить участников заново настроить рутины.
- В E2E reset добавлена очистка `goal_nudge_cooldowns`, иначе сценарий `14` зависел от старого cooldown.
