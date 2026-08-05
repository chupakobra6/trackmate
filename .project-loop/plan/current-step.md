# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Завершенный Шаг
- id: `STEP-034`
- status: `готово`
- objective: Сделать один повтор неотвеченного двухнедельного вопроса по целям через 24 часа и persisted skip через 72 часа.
- requirement IDs: `REQ-058`, `VAL-012`
- owned paths: `internal/domain/`, `internal/app/goals/`, `internal/app/pending/`, `internal/storage/postgres/`, additive migration, goals docs/E2E expectations, `.project-loop/`.
- validation: domain boundary tests; clean-schema PostgreSQL integration; focused goals/worker/pending tests; lint/vet; только focused live goals-review scenario.
- done criteria: ровно один retry; prompt/pending доступны для ответа до общего 72-часового дедлайна; после дедлайна review persisted как skipped и не возвращается; final review не пересекается с незавершенным weekly review.

## Фокус Ревью
- Проверить границы ровно 24/72 часа, restart-safe timestamps и отсутствие второго retry.
- Проверить, что cleanup не удаляет retry на 48-м часу и не очищает чужой pending.
- Не запускать полный Telegram E2E; проверить только измененный goals-review flow.

## Примечания
- Текущий generic pending cleanup удаляет weekly prompt через 24 часа, а dispatch ищет только новое расписание; поэтому тот же review не возвращается.
- Пользовательский текст prompt не меняется: повтор использует тот же canonical formatter.
- Production deploy по-прежнему требует отдельного approval.
- Focused live подтвержден: initial `855`, retry `856`, active pending на 48-м часу, удаление + `skipped_at` на 72-м часу.
- Полный Telegram E2E не запускался.
