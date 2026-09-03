# Handoff

Проект: trackmate
Обновлено: 2026-09-04

## Цель
- Проверить все удаления Telegram-сообщений, устранить граничные timed deletes и гарантировать, что delete failure не блокирует worker.

## Завершенный Шаг
- step: `STEP-040`
- status: `готово`
- requirements: `REQ-069`, `REQ-070`, `VAL-018`
- sources: `S042`, `S043`

## Контракт
- По официальному Bot API обычное сообщение можно удалить только в возрасте строго меньше `48h`.
- Trackmate планирует удаление не позже `47h`, сохраняя час запаса.
- Weekly deadline `71h` считается от первого prompt: retry через `24h`, затем `47h` до cleanup.
- `GoalNudgeCooldown=72h` не относится к удалениям и не менялся.

## Реализация И Prevention
- `internal/domain` содержит один источник delete limit, margin и target age.
- `internal/app/messagecleanup` выполняет age-aware best-effort delete и inert fallback без worker block.
- Pending, routine notices и weekly goals cleanup переведены на общий контракт.
- Architecture regression test запрещает обход контракта в application code; единственное исключение — компенсация сообщения, отправленного в том же вызове.
- Интерактивные `internal/bot` call sites проаудированы: они user-triggered/immediate и не блокируют worker при delete error.
- `AGENTS.md` закрепляет Telegram invariant и каноническую production backup-команду.

## Проверки И Production
- focused tests, fresh `go test ./... -count=1`, `make check`, `git diff --check`, Project Loop validation: pass.
- code commit `c1e126d` включён в local, `origin/main` и `/opt/trackmate`.
- backup `/opt/trackmate/backups/trackmate_20260903T230430Z.dump` проверен checksum и `pg_restore --list`.
- `api`, `worker`, `postgres` healthy; migrate завершён успешно.
- unpublished progress, outstanding alerts, stale claims, stale generic pending, stale routine notices, weekly past safe deadline и advisory waiters: `0`.
- fresh production error/delete-failure scan: clean.

## Остаточный Риск
- Telegram может отказать в удалении и до 48 часов по другим правилам Bot API или transient причинам; в worker-owned flows это теперь приводит к inert fallback/безопасному завершению, а не к повторному loop.

## Следующее Действие
- Нет обязательного действия.
