# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-021`
- status: `готово`
- objective: Проверить, работает ли production `Сегодня` после скрина 28.06 и почему у Игоря нет задачи 29.06.
- requirement IDs: `REQ-040`
- owned paths: `.project-loop/`
- validation: prod services/logs/db: pass; Harvest Today/Chat/all dumps: pass; no data edit needed
- done criteria: current production health is known; screenshot messages are matched to exact production logs; current Today topic is clean; Igor's 29.06 morning messages are classified correctly; no unrelated data is changed.

## Фокус Ревью
- Это production data-fix, не кодовый deploy.
- Не трогать routine local fixes и не выкатывать локальную пачку.
- Если текущая проверка показывает, что сообщения уже удалены/засчитаны, не делать лишних Telegram/DB правок.

## Примечания
- Скрин соответствует 2026-06-28 22:25 MSK: `today:add` Егора падал на старый `uq_pending_inputs_workspace_group_id`, отправлял prompts `3556`/`3558`, затем шли messages `3557`/`3559`/`3560`.
- В актуальном Harvest dump prompts `3556`/`3558` и messages `3557`/`3559`/`3560` уже отсутствуют.
- Отчет Игоря `3542` за 2026-06-27 уже засчитан: task `169`, progress message `3543`.
- На 2026-06-29 в `Сегодня` нет задачи Игоря; его сообщения `3642`/`3643`/`3647`/`3648` были в общем `Чате`, а не в топике `Сегодня`, и не являются текстом задачи дня.
- Production сейчас работает: task `171` Ярика создан 2026-06-29, card `3652`; `api`, `worker`, `postgres` healthy; progress outbox чистый.
