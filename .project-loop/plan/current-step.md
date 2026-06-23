# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-003`
- status: `готово`
- objective: Выполнить полный live E2E на тестовом Telegram-боте для новых workflow, исправить найденные ошибки и оставить видимые примеры в темах без cleanup.
- requirement IDs: `REQ-018`, `VAL-005`, `CON-005`
- owned paths: `.project-loop/`, `e2e/telegram/`, `internal/`, `migrations/`, `docs/`, tests
- validation: `make docker-up`; `make migrate`; `telegram-bot-e2e-test-tool make doctor/chats/run-scenario`; targeted `go test`; `make lint`; `loopctl.py validate`
- done criteria: Live E2E пройден на тестовом боте; найденные ошибки исправлены и повторно проверены; видимые примеры workflow оставлены в Telegram; локальный commit создан при изменениях; prod остается без deploy до approval.

## Фокус Ревью
- E2E идет через реального Telegram user runner, не через Bot API shortcuts.
- Cleanup/delete сценарии не запускаются после текущего прогона.
- Каждый топик получает пример: `Сегодня`, `Рутины`, `Цели`, `Прогресс`.
- Если сценарий падает, сначала читаются transcript artifacts, затем чинится продукт/сценарий и failing path повторяется.

## Примечания
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- Docker теперь доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Docker compose проверен локально: `postgres`, `api`, `worker` healthy/up; агент может запускать Docker-команды для тестирования.
- Выполнено: `TRACKMATE_TEST_DATABASE_URL=... go test ./...`, `go test ./... -cover`, `make lint`, `TRACKMATE_TEST_DATABASE_URL=... make test`, `TRACKMATE__DATABASE_URL=... make migrate`, `loopctl.py validate`.
- Для S004 не запускать `99-cleanup-visible-messages.jsonl`.
- Live E2E выполнен на `@yaminotoubot` в группе `тестирование trackmate v2`: `Сегодня` topic `10`, `Прогресс` topic `11`, `Рутины` topic `339`, `Цели` topic `340`.
- Пройдены: setup, Today add/create/block/report statuses/wrong-topic/photo dedupe/progress/edit sync/alert ack, Routine configure/check-in/reason/reflection/leaderboard, Goals configure/weekly/final, deterministic goal nudge.
- Найдено и исправлено:
  - scenario wait для уже видимой routine-card заменен на `assert_visible_text`;
  - scenario wait для weekly/final goals prompts заменен на `assert_visible_text`;
  - final review целей теперь сравнивает дату окончания как локальную дату workspace, а не UTC instant.
- Видимые примеры оставлены в Telegram; финальный snapshot: `tmp/e2e-live-logs/98-dump-review-state.log`.
