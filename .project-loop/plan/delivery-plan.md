# План Поставки

Проект: trackmate
Обновлено: 2026-06-23

## Этапы
| Шаг | Статус | ID требований | Цель | Ревью | Проверка |
| --- | --- | --- | --- | --- | --- |
| STEP-001 | `готово` | REQ-001..REQ-014 | Локально реализовать `Рутины`, `Цели`, правку `Сегодня`, тесты и подготовить ревью-результат без прод-деплоя. | Сверить diff с чеклистом и raw intake, проверить антиспам, сохранность данных и отсутствие публикации routine в `Прогресс`. | `go test ./internal/...`; `make lint`; `make test`; `loopctl.py validate /Users/igor/projects/trackmate` |
| STEP-002 | `готово` | REQ-015..REQ-017,VAL-004 | Учесть review delta: fair leaderboard, deterministic nudge cooldown, разделение новых доменов, полный Docker/PostgreSQL test pass. | Сверить diff с новой дельтой и убедиться, что поведение Today/Routine/Goals покрыто тестами. | `TRACKMATE_TEST_DATABASE_URL=... go test ./...`; `go test ./... -cover`; `make lint`; `make test`; local `make migrate`; `loopctl.py validate` |
| STEP-003 | `готово` | REQ-018,VAL-005,CON-005 | Выполнить live E2E на тестовом Telegram-боте, исправить ошибки, оставить видимые примеры workflow в темах. | Проверить setup/today/routine/goals/progress/alerts через runner transcripts и визуальные сообщения в Telegram. | `tg-e2e-tool run-scenario`; targeted fixes/tests; no cleanup scenario |
| STEP-004 | `готово` | REQ-019 | Поправить пользовательские тексты и форматирование новых сообщений, убрать англицизмы из видимых терминов. | Сверить UI-тексты и E2E-ожидания, убедиться что новые формулировки сохраняют текущие механики. | `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine`; `make test`; `make lint`; `loopctl.py validate` |
| STEP-005 | `готово` | REQ-020 | Учесть внешнее ревью текстов: термин `задача дня`, компактные карточки, нейтральные вставки, `Таблица рутин`, структурированный шаблон целей. | Сверить UI-тексты и E2E-ожидания, не вернуть англицизмы из конфликтующего примера ревью. | `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine ./internal/storage/postgres`; `make test`; `make lint`; `loopctl.py validate` |
| STEP-006 | `кандидат` | REQ-013 | После пользовательского ревью финализировать prod-миграцию, выполнить DB dry-run на доступной PostgreSQL и получить approval. | Проверить backup/restore/migrate/rollback шаги и что нет destructive SQL для текущих данных. | dry-run локальной миграции + approved command sequence |
| STEP-007 | `отложено` | REQ-013 | После approval выкатить обновление на production. | Подтвердить backup, миграции, запуск сервисов, smoke-check новых topic bindings. | prod command outputs, post-deploy checks |

## Примечания По Порядку
- Шаги достаточно маленькие для цикла: реализация, ревью, исправление, проверка, коммит, handoff.
- Активен один шаг; непрерывное выполнение появляется только по явной инструкции Игоря.
- Для существенной работы используются пары `STEP-N` / `STEP-NR`.
- Человекочитаемые проектные артефакты пишутся на русском.
- Имена файлов описательные; ID источников хранятся в карте источников и чеклисте.
- Текущий запрос Игоря дает явное разрешение на непрерывную локальную реализацию до ревью; prod-действия остаются отложенными до отдельного approval.
