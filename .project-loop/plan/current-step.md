# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-028`
- status: `готово`
- objective: Production-сброс рутинных данных и мусора перед повторной настройкой рутин участниками.
- requirement IDs: `REQ-050`
- owned paths: `.project-loop/`; production DB workspace `Haru`
- validation: backup: pass; prod SQL `routine_reset_verify|0|0|0|0`: pass; `api`/`worker`/`postgres` running
- done criteria: routine plans/checkins/items/pending inputs in the main production workspace are zero; routine leaderboard is reset; future update message reminder is recorded.

## Фокус Ревью
- Это production data operation, не deploy локального кода.
- Не трогать Today/Goals/Progress данные без отдельного конкретного запроса.
- Перед destructive DB changes нужен backup.

## Примечания
- STEP-027 с полным live E2E текущего head отложен и остается обязательным перед будущим production deploy локальной пачки.
- В следующем update message попросить участников заново настроить рутины.
- Рутинные сообщения `3655`, `3656`, `3657` уже отсутствовали в Telegram на момент чистки.
