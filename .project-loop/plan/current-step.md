# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-24

## Активный Шаг
- id: `STEP-037`
- status: `в работе`
- objective: Подтвердить owner protection для routine reminder и auto-close, затем безопасно развернуть callback fix.
- requirement IDs: `REQ-062`, `VAL-015`
- source IDs: `S038`

## Проверка
- PostgreSQL `TestRunCheckinTransitionsRemindsAndAutoCloses`: оба routine dismiss callbacks содержат `notice:dismiss:42`.
- PostgreSQL `TestPersonalCallbacksRejectOtherParticipantsWithoutMutation`: user `43` не может выполнить `notice:dismiss:42`, Telegram/DB не меняются.
- DB-backed `make check`, `git diff --check`, Project Loop validation.
- Полный Telegram E2E не запускать.

## Deploy Gate
- push only after focused checks pass;
- read-only production preflight;
- `make docker-db-backup-stop` and archive verification;
- `git pull --ff-only`, `docker compose up -d --build`;
- verify production commit, services, migrations, stale claims, advisory waiters and recent logs.

## Pre-Deploy Evidence
- `TestRunCheckinTransitionsRemindsAndAutoCloses`: pass; reminder и auto-close имеют `notice:dismiss:42`.
- Foreign user regression на exact `notice:dismiss:42`: pass; ноль Telegram/DB mutations.
- Authorized notice dismiss: pass.
- DB-backed `make check`: pass.
- Production preflight: `/opt/trackmate` clean на `a9ad102`, services healthy, Docker context `default`.
