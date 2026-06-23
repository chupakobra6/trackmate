# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-005`
- status: `готово`
- objective: Учесть внешнее ревью текстов: термин `задача дня`, компактные карточки, нейтральные вставки, `Таблица рутин`, структурированный шаблон целей.
- requirement IDs: `REQ-020`
- owned paths: `.project-loop/`, `internal/ui/`, `internal/bot/`, `internal/app/`, `e2e/telegram/`, tests
- validation: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine`; `make test`; `make lint`; `loopctl.py validate`
- done criteria: рекомендации внешнего ревью применены; конфликтующий англицизм из примера целей не возвращен; E2E-ожидания обновлены; тесты проходят; prod остается без deploy до approval.

## Фокус Ревью
- Закрепить `задача дня` вместо `цель-задача дня`, чтобы не путать ежедневный сценарий с сезонными целями.
- Уплотнить карточки `Сегодня`/`Прогресс`: заголовок `План:` сразу перед `<blockquote>`, без лишней пустой строки.
- Смягчить вставки про цели: убрать `провал`, `двигает тебя` и похожую назидательность.
- Переименовать видимый заголовок routine table в `Таблица рутин`.
- Структурировать шаблон целей через маркированные поля, но не возвращать `оффер Go/backend`.

## Примечания
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- Docker доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Предыдущий live E2E оставил видимые примеры в тестовой группе; текущий шаг может потребовать нового прогона только если пользователь попросит свежий визуальный Telegram review.
- Источник ревью: attachment `/Users/igor/.codex/attachments/94d461f5-f57c-4b41-89ca-515bccdee362/pasted-text.txt`.
- Проверки STEP-005 прошли: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine ./internal/storage/postgres`, `make test`, `make lint`.
