# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-020`
- status: `готово`
- objective: Дочистить production topic `Сегодня` после broken flow и опубликовать пропущенные progress итоги.
- requirement IDs: `REQ-039`
- owned paths: `.project-loop/`
- validation: prod backup: pass; worker publish log/db: pass; Harvest Today/Progress dumps: pass; service health: pass
- done criteria: Today topic has no duplicate prompt/service-noise messages in the checked range; missed manual-restored progress events are visible in `Прогресс`; no Today pending inputs remain; progress outbox has no pending/publishing/failed events.

## Фокус Ревью
- Это production data-fix, не кодовый deploy.
- Не трогать routine local fixes и не выкатывать локальную пачку.
- Новые Telegram posts допустимы только через штатный progress worker для уже существующих missed outbox events.

## Примечания
- Backup перед cleanup: `/opt/trackmate/backups/trackmate_manual_today_cleanup_20260629T122958Z.dump`.
- `progress_events.id=193` и `194` были `published` без `published_message_id`; их вернули в `pending`, worker опубликовал messages `3653` и `3654`.
- Через Harvest удалены old service confirmations `3351,3356,3364,3394,3399,3437,3448,3485,3491,3519,3530,3541,3572`.
- Финальный dump `Сегодня` не содержит `Напиши главную задачу` и `Итог сохранен`.
- Ретроактивно вставить bot-card на старое место вместо raw message `3467` невозможно; raw source message оставлен как source link для восстановленной задачи.
