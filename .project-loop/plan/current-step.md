# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Завершенный Шаг
- id: `STEP-033`
- status: `готово`
- objective: Привести routine auto-close к общему контекстному reply-alert lifecycle без standalone сообщения и потери карточки.
- requirement IDs: `REQ-057`, `VAL-011`
- owned paths: `internal/app/routine/`, `internal/storage/postgres/routines.go`, `internal/ui/`, `internal/messages/messages.md`, focused tests/E2E, docs, `.project-loop/`.
- validation: formatter + clean-schema PostgreSQL integration; retry/idempotency checks; lint/vet; только focused live routine auto-close E2E по правилу `AGENTS.md`.
- done criteria: итоговая routine card остается видимой и без кнопок; alert является reply к ней, содержит адресата и source-link, закрывается через `Понял`; ошибка Telegram не теряет будущий retry и успешный tick не дублирует notice.

## Фокус Ревью
- Проверить границу DB transition → Telegram projection → notice delivery и повторный worker tick.
- Не запускать полный Telegram E2E; проверить только измененный routine auto-close flow.

## Примечания
- Screenshot S034 показывает standalone `⏰ Рутина за 28.07 закрыта` без адресата/reply/source-link.
- Текущий код удаляет routine card перед отправкой notice, поэтому корректный reply target уничтожается самим lifecycle.
- Production deploy по-прежнему требует отдельного approval.
- Focused live scenario завершен: при пересечении periodic/manual tick отправлен ровно один alert `851`; после `Понял` alert исчез, финальная карточка `850` осталась.
- Полный Telegram E2E не запускался.
