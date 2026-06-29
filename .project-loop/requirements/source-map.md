# Карта Источников

Проект: trackmate
Обновлено: 2026-06-29

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
| S010 | user | 2026-06-24 | `.project-loop/intake/user-deltas.md` | принято | Прод-наблюдение: карточка рутины с датой выглядит двусмысленно; нужно объяснить, за какой день отмечать рутину, прямо в карточке. |
| S011 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-c88a6f65-65d8-4919-851d-90137a4b96e9.png` | принято | Прод-наблюдение: prompt причины рутины выглядит иначе, чем обычная карточка; нужно унифицировать вид и заменить первый emoji рутины. |
| S012 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-0914828b-48af-41a5-ac70-9785d8f1a187.png` | принято | Прод-наблюдение/UX-решение: причина по `Нет`/`Частично` должна спрашиваться отдельным временным сообщением, основная карточка рутины только отмечает пункты, финальный итог рутины убирается. |
| S013 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-4dc107f3-ea8e-4b59-9bc4-379941454acb.png` | принято | Production bug: сообщение Егора `3386` не прикрепилось к задаче `160`, потому что старый уникальный constraint `pending_inputs(workspace_group_id,user_id)` остался после topic-scoped миграции и блокировал pending в другом топике. |
| S014 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-3e1a8117-5364-45ec-8663-82c0445b0536.png` | принято | Copy/UX delta: routine example with dashes only, simpler style like `Сегодня`, all bot copy imported from one document, problem messages dismissible and cleaned after action/fix. |
| S015 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-f9ad8f12-c2b9-4608-a6da-5d9195757bf0.png` | принято | Routine topic cleanup delta: no persistent routine save confirmation, delete routine setup input, accept numbered routine lists, delete completed/autoclosed routine cards so topic keeps only table plus active temporary messages. |
| S016 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-07dd5469-5de7-46db-86f1-b1868f3329fc.png` | принято | Routine reminder delta: reminder copy must be shorter and clearer, routine alerts need dismiss/action button, reminder disappears on routine close or after about 24h, auto-close notice is restored as a temporary 24h alert. |
| S017 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-ae9fb7f7-38dc-49e9-9f94-2e7767bd7cb8.png` | принято | Production correction: verify Egor voice report `3386`/task `160`, fix missing DB representation so the done task has a `daily_task.closed` progress event. |
| S018 | user | 2026-06-29 | `.project-loop/intake/user-deltas.md`; screenshot `/var/folders/70/xq5yx2813j1c27f2xf1mjkxw0000gn/T/codex-clipboard-4a1ae09c-c9e7-4078-b721-7fade5ba8c2a.png` | принято | Production follow-up: verify whether the "task does not save" bug is fixed, delete the chat complaint messages with Telegram Harvest, and use Harvest/Vosk transcription for the related voice report. |

## Конфликты
| Источники | Решение | Дата |
| --- | --- | --- |
| S008 vs S009 | S009 заменяет сброс setup-черновиков между `Рутины`/`Цели`: теперь разные топики изолированы, wrong-topic сообщения игнорируются, а stale cleanup через 24 часа чистит старые pending молча. | 2026-06-24 |
| S001/CON-001/REQ-005 vs S012 | S012 разрешает отдельные временные reason prompts для routine `Нет`/`Частично`, потому что они удаляются после ответа; final reflection рутины удаляется, чтобы не дублировать `Сегодня`. | 2026-06-29 |
| S001 vs S014 vs S015 | S014 заменил широкий routine parser на дефисный формат; S015 вернул поддержку нумерации `1.`/`2)` как префикса пункта. Свободные строки и прочие маркеры по-прежнему не принимаются, пример в prompt остается дефисным. | 2026-06-29 |
| S014 vs S015 | S014 делал routine save confirmation dismissable; S015 сильнее: после сохранения рутины prompt и input удаляются, отдельное confirmation-сообщение не остается. | 2026-06-29 |
| S015 vs S016 | S015 убирал auto-close notice полностью; S016 сильнее: auto-close notice возвращается, но как временный dismissable alert с TTL около суток, чтобы пользователь видел причину закрытия без постоянного мусора в topic. | 2026-06-29 |
| S013 vs S017 | S013 исправил саму daily task `160`, но без нового progress event; S017 добавляет недостающий `daily_task.closed` event и публикацию в `Прогресс`, чтобы БД и видимая история совпали с фактическим итогом. | 2026-06-29 |
| S013/S017 vs S018 | S018 не меняет продуктовый код: он подтверждает, что root cause уже снят на production schema, чистит показанные chat messages и улучшает сохраненный текст voice report `3386` через Harvest/Vosk. | 2026-06-29 |
