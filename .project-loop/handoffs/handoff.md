# Handoff

Проект: trackmate
Обновлено: 2026-08-24

## Цель
- Подтвердить защиту routine reminder/auto-close от чужого `Понял` и развернуть общий callback fix в production.

## Активный Шаг
- step: `STEP-037`
- status: `в работе`
- requirements: `REQ-062`, `VAL-015`
- source: `S038`

## Подтверждено До Deploy
- Routine transition integration генерирует `notice:dismiss:42` и для reminder, и для auto-close alert.
- Bot authorization regression нажимает exact callback от user `43`: callback answer сообщает `только адресат`, Telegram send/edit/delete/markup не вызываются, БД не меняется.
- Owner user `42` по-прежнему закрывает notice обычным lifecycle.
- Ownerless legacy `notice:dismiss` отклоняется без удаления.
- DB-backed `make check` прошел; полный Telegram E2E намеренно не запускался.

## Production Preflight
- target: `inferno-nl:/opt/trackmate`
- production commit: `a9ad102`
- worktree: clean, `main...origin/main`
- Docker context: `default`
- `api`, `worker`, `postgres`: healthy до deploy.

## Следующее Действие
- Закоммитить Project Loop delta, push `main`, выполнить `make docker-db-backup-stop`, pull/build/start и post-deploy DB/health/log checks.

## Обновленные Источники Правды
- `.project-loop/requirements/source-map.md`
- `.project-loop/requirements/checklist.md`
- `.project-loop/plan/delivery-plan.md`
- `.project-loop/plan/current-step.md`
- `.project-loop/intake/user-deltas.md`
- `.project-loop/handoffs/handoff.md`
