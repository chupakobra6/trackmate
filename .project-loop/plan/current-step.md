# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-24

## Активный Шаг
- id: `STEP-009`
- status: `готово`
- objective: Починить UX настройки рутин/целей: сброс черновиков между топиками, короткое сохранение целей, явное время рутины.
- requirement IDs: `REQ-022..REQ-024`
- owned paths: `.project-loop/`, `internal/ui/`, `internal/bot/`, `internal/app/`, `e2e/telegram/`, tests
- validation: `go test ./internal/bot ./internal/ui ./internal/storage/postgres ./internal/domain ./internal/app/routine ./internal/app/goals`; `make test`; `make lint`; `loopctl.py validate`
- done criteria: черновики `routine_plan`/`seasonal_goals` сбрасываются при переходе в другой setup-топик; старые prompt и wrong-topic user message удаляются; goals setup больше не эхоит полный текст целей; routine check-in явно после 09:00; тесты проходят.

## Фокус Ревью
- Pending setup cleanup между `Рутины` и `Цели`.
- Единый короткий confirmation при сохранении целей без отдельной карточки с raw goals.
- Точное объяснение времени рутины: после 09:00 local workspace time.

## Примечания
- STEP-009 закрыт: pending setup cleanup, короткий goals confirmation и время routine check-in реализованы.
- Проверки STEP-009: `go test ./internal/bot ./internal/ui ./internal/storage/postgres ./internal/domain ./internal/app/routine ./internal/app/goals`, `make test`, `make lint`, `loopctl.py validate`.
- Production уже на `v1.1`; текущий шаг пока локальная правка.
- Docker доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Routine dispatch time is defined by `domain.RoutineCheckinHour = 9`.
