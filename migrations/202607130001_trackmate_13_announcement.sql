-- +goose Up
INSERT INTO progress_events (workspace_group_id, event_type, publish_status, payload, created_at)
SELECT
    wg.id,
    'custom_update'::progresseventtype,
    'pending'::progresspublishstatus,
    jsonb_build_object(
        'slug', 'trackmate-1.3',
        'title', 'Встречайте: Trackmate 1.3',
        'body', 'После 1.2 довели ежедневный контур до конца дня: «Сегодня» больше не предлагает составлять план задним числом, а «Рутины» сохраняют понятную историю.',
        'items', jsonb_build_array(
            'добавили «Итог дня»: после 20:00 та же кнопка в «Сегодня» вместо новой задачи открывает итог; выбери «Хорошо», «Средне» или «Плохо» и опиши главное за день',
            'итог остаётся отдельной записью: он виден в «Сегодня» и «Прогрессе», не меняет статистику задач; правка текста обновляет обе карточки без нового сообщения',
            'если итог не записан вовремя, Trackmate закрывает его как невыполненный и снимает устаревшие кнопки',
            'в «Рутинах» новые карточки и завершённые результаты ведут к исходному списку; при смене списка текущая проверка остаётся на прежних пунктах, а новый список применяется со следующего дня',
            'завершённые рутинные карточки остаются в теме, подписи таблицы и серий стали понятнее',
            'поправили напоминания: кнопка «Понял» больше не остаётся активной, даже если Telegram не даёт удалить сообщение',
            'итоговый diff: 11 коммитов, 40 файлов, +1 655 строк и −330 строк; прикладной код и миграции: 16 файлов, +705 и −133 строки'
        )
    ),
    now()
FROM workspace_groups wg
WHERE EXISTS (
    SELECT 1
    FROM topic_bindings tb
    WHERE tb.workspace_group_id = wg.id
      AND tb.topic_key = 'progress'::topickey
)
AND NOT EXISTS (
    SELECT 1
    FROM progress_events pe
    WHERE pe.workspace_group_id = wg.id
      AND pe.event_type = 'custom_update'::progresseventtype
      AND pe.payload->>'slug' = 'trackmate-1.3'
);

-- +goose Down
DELETE FROM progress_events
WHERE event_type = 'custom_update'::progresseventtype
  AND payload->>'slug' = 'trackmate-1.3'
  AND publish_status IN ('pending', 'failed');
