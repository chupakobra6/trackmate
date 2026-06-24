# Карта Источников

Проект: trackmate
Обновлено: 2026-06-24

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
| S006 | user | 2026-06-23 | `.project-loop/intake/user-deltas.md`; attachment `/Users/igor/.codex/attachments/94d461f5-f57c-4b41-89ca-515bccdee362/pasted-text.txt` | принято | Внешнее ревью текстов: `задача дня` вместо `цель-задача`, компактные карточки, нейтральные вставки про цели, `Таблица рутин`, структурированный шаблон целей. |
| S007 | user | 2026-06-23 | `.project-loop/intake/user-deltas.md`; attachment `/Users/igor/.codex/attachments/55bd8dc4-45f5-4892-8186-d844e12eeb30/pasted-text.txt` | принято | Ревью текстов: стиль старых закрепов, длинные тире вместо маркеров, понятное описание долгосрочных целей, описание полей, новые вопросы недельного обзора и конкретный итог периода. |
| S008 | user | 2026-06-24 | `.project-loop/intake/user-deltas.md` | принято | UX-правки после production: сбрасывать незаконченный ввод рутин/целей при переходе в другой топик, удалять лишние сообщения, не эхоить цели большим полотном, уточнить время routine check-in. |
| S009 | user | 2026-06-24 | `.project-loop/intake/user-deltas.md` | принято | Новая правка UX: pending inputs изолированы по topic thread, чужие топики не влияют друг на друга, stale pending старше суток чистится молча, routine check-in вечером с напоминанием и автозакрытием на следующий день. |

## Конфликты
| Источники | Решение | Дата |
| --- | --- | --- |
| S008 vs S009 | S009 заменяет сброс setup-черновиков между `Рутины`/`Цели`: теперь разные топики изолированы, wrong-topic сообщения игнорируются, а stale cleanup через 24 часа чистит старые pending молча. | 2026-06-24 |
