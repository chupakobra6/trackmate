# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-006`
- status: `готово`
- objective: Применить финальное ревью стиля: старый тон закрепов, длинные тире, понятные долгосрочные цели, конкретный недельный обзор и итог периода.
- requirement IDs: `REQ-021`
- owned paths: `.project-loop/`, `internal/ui/`, `internal/bot/`, `internal/app/`, `e2e/telegram/`, tests
- validation: `go test ./internal/ui ./internal/domain ./internal/bot ./internal/app/goals ./internal/app/routine`; `make test`; `make lint`; `loopctl.py validate`
- done criteria: тексты из S007 применены; видимые списки используют `—`; parser routine принимает `—`; E2E-ожидания обновлены; тесты проходят; prod остается без deploy до approval.

## Фокус Ревью
- Вернуть закрепам спокойный стиль старых текстов: `Здесь у каждого...`, `Здесь живут...`, `Здесь собирается...`.
- Заменить видимые маркеры списков с `•` на длинное тире `—`.
- Сделать `Цели` понятнее: долгосрочные цели на сезон, например лето.
- Заменить пример целей на описание полей формата.
- Переписать вопросы недельного обзора и итог периода на конкретные формулировки.

## Примечания
- STEP-006 закрыт: применены тексты из S007, обновлены E2E-ожидания, routine parser принимает `—`.
- Проверки STEP-006: `go test ./internal/ui ./internal/domain ./internal/bot ./internal/app/goals ./internal/app/routine`, `make test`, `make lint`, `loopctl.py validate`.
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- Docker доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Предыдущий live E2E оставил видимые примеры в тестовой группе; текущий шаг может потребовать нового прогона только если пользователь попросит свежий визуальный Telegram review.
- Источник ревью: attachment `/Users/igor/.codex/attachments/55bd8dc4-45f5-4892-8186-d844e12eeb30/pasted-text.txt`.
- STEP-005 уже закрыт: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine ./internal/storage/postgres`, `make test`, `make lint`.
