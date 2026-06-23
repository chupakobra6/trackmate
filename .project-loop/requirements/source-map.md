# Карта Источников

Проект: trackmate
Обновлено: 2026-06-23

## Приоритет Источников
1. Текущая прямая инструкция Игоря.
2. Применимые `AGENTS.md` и system/developer instructions.
3. Существующий код, тесты, схемы и канонические проектные документы.
4. Принятое состояние `.project-loop/`.
5. Предыдущий handoff.
6. Исходный intake и внешний research.
7. Вывод модели.

## Источники
| ID | Тип | Дата | Расположение | Статус | Примечания |
| --- | --- | --- | --- | --- | --- |
| S001 | user | 2026-06-22 | `.project-loop/intake/raw/2026-06-23-trackmate-routines-goals.md` | принято | Raw-инпут про новые топики `Рутины` и `Цели`, изменение `Сегодня`, локальное тестирование, безопасную миграцию и запрет потери данных. |
| S002 | repo | 2026-06-23 | `README.md`, `AGENTS.md`, `docs/`, `internal/`, `migrations/` | принято | Текущее Go-only устройство Trackmate: topics `Сегодня`/`Прогресс`, goose migrations, worker transitions, bot callbacks, pending inputs. |
| S003 | user | 2026-06-23 | `.project-loop/intake/user-deltas.md` | принято | Review delta: fair leaderboard metrics, deterministic goal nudge cooldown, domain separation, Docker/full testing. |
| S004 | user | 2026-06-23 | `.project-loop/intake/user-deltas.md` | принято | Live E2E на тестовом боте, исправление найденных ошибок, оставить видимые Telegram-примеры workflow без cleanup. |
| S005 | user | 2026-06-23 | `.project-loop/intake/user-deltas.md` | принято | Review delta: поправить формулировки и визуальное форматирование сообщений, убрать англицизмы из пользовательских терминов. |

## Конфликты
| Источники | Решение | Дата |
| --- | --- | --- |
