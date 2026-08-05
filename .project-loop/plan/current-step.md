# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Завершенный Шаг
- id: `STEP-031`
- status: `готово`
- objective: Устранить stacking status prompts и ложное acknowledgement Today alerts через единый идемпотентный Telegram message lifecycle.
- requirement IDs: `REQ-054`, `VAL-009`
- owned paths: `internal/bot/`, `internal/messages/`, `internal/telegram/` при необходимости, `docs/architecture.md`, `e2e/telegram/`, `.project-loop/`.
- validation: unit + clean-schema PostgreSQL integration на replay/ack/fallback/failure; полный `go test ./...`; lint; Project Loop validation; live test-bot alert/report lifecycle.
- done criteria: выполнены локально. Повторные callbacks редактируют одно сообщение; `Понял` удаляет или явно закрывает alert без кнопок; Telegram failure не очищает DB преждевременно; старый null message ID восстанавливается из callback; live message `827` прошел весь lifecycle in place.

## Фокус Ревью
- Проверить повторную доставку callback, порядок Telegram transition → DB acknowledgement и отсутствие новых status messages.
- Не менять доменную модель задач/итогов, Progress events и расписание worker alerts.

## Примечания
- Production evidence: `task:report:244` пришел трижды для message `5828`; alert `114` получил повторные `alert:ack` после DB acknowledgement.
- Root cause закрыт единым helper-слоем message lifecycle и in-place report transition.
- Видимый fallback copy: раньше старый alert оставался на экране с кнопкой; теперь `👀 Уведомление закрыто` без кнопок.
- Production не менялся; deploy требует отдельного approval по `CON-004`.
