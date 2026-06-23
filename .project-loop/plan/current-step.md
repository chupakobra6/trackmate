# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-004`
- status: `готово`
- objective: Поправить пользовательские сообщения и форматирование новых сценариев, убрать англицизмы из видимых терминов.
- requirement IDs: `REQ-019`
- owned paths: `.project-loop/`, `internal/ui/`, `internal/bot/`, `internal/app/`, `e2e/telegram/`, tests
- validation: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine`; `make test`; `make lint`; `loopctl.py validate`
- done criteria: видимые сообщения стали лаконичнее и полностью русскими по терминам; E2E-ожидания обновлены; тесты проходят; prod остается без deploy до approval.

## Фокус Ревью
- Убрать из пользовательского текста `check-in`, `leaderboard`, `review`, `daily`, `weekly`, `outcome`, `стрик` и похожие термины.
- Не переименовывать технические callback/data/API/enum контракты ради текста.
- Сохранить смысл текущих сценариев: Today, Routine, Goals, Progress, alerts.
- Сделать карточки более ровными: короткие заголовки, понятные блоки, меньше лишних точек и тяжелых формулировок.

## Примечания
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- Docker доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Предыдущий live E2E оставил видимые примеры в тестовой группе; текущий шаг может потребовать нового прогона только если пользователь попросит свежий визуальный Telegram review.
- Проверки STEP-004 прошли: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine`, `make test`, `make lint`.
