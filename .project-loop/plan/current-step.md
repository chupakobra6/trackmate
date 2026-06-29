# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-022`
- status: `готово`
- objective: Проверить жалобы Ярика и Егора из общего чата, что задачи дня не принимает.
- requirement IDs: `REQ-041`
- owned paths: `.project-loop/`
- validation: prod logs/db: pass; Harvest Chat/Today dumps: pass; service health: pass; no data edit needed
- done criteria: chat complaints are matched to exact Today messages, production logs, DB task rows and progress messages; current production health is known; no unrelated Telegram/DB cleanup is done.

## Фокус Ревью
- Это production verification, не кодовый deploy.
- Не трогать routine local fixes и не выкатывать локальную пачку.
- Если текущая проверка показывает, что сообщения уже засчитаны или это обычный общий чат, не делать лишних Telegram/DB правок.

## Примечания
- Скрин соответствует общему topic `Чат` 2026-06-26: Ярик пишет Игорю в 06:17 MSK, Егор жалуется в 08:54 MSK, Ярик пишет `БРО ГДЕ ЗАДАЧА ВТОРОЙ ДЕНЬ` в 14:18 MSK.
- Задача Ярика за 2026-06-26 была принята: source message `3456`, Trackmate card `3457`, progress message `3487`.
- Жалоба Егора была реальной: `today:add` падал на старый `uq_pending_inputs_workspace_group_id`; это уже восстановлено как task `172` и progress message `3653`, а complaint messages `3464`/`3465` удалены ранее.
- Реплика Ярика `БРО ГДЕ ЗАДАЧА ВТОРОЙ ДЕНЬ` относилась к отсутствующей на тот момент задаче Игоря; Игорь добавил ее через минуту, source message `3476`, card `3477`, progress message `3493`.
- Production сейчас работает: `api`, `worker`, `postgres` healthy; progress outbox clean; Today pending input count = 0.
