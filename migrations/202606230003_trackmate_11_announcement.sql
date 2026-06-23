-- +goose Up
INSERT INTO progress_events (workspace_group_id, event_type, publish_status, payload, created_at)
SELECT
    wg.id,
    'custom_update'::progresseventtype,
    'pending'::progresspublishstatus,
    jsonb_build_object(
        'slug', 'trackmate-1.1',
        'title', 'Встречайте: Trackmate 1.1',
        'body', 'Добавили два новых раздела: «Рутины» для повторяемых действий и «Цели» для сезонных ориентиров. «Сегодня» остается главным фокусом дня, а «Прогресс» — общей лентой результатов.',
        'items', jsonb_build_array(
            'добавили «Рутины»: ежедневная карточка отметок, причины и итог дня',
            'добавили «Цели»: цели на сезон, недельные обзоры и итог периода',
            'усилили «Сегодня»: одна главная задача дня и мягкие напоминания о целях',
            'сохранили задачи, участников, напоминания и историю прогресса; миграция добавляет новые таблицы без удаления старых данных',
            'итоговый diff: 43 файла, +3574 строки, -165 строк'
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
      AND pe.payload->>'slug' = 'trackmate-1.1'
);

-- +goose Down
DELETE FROM progress_events
WHERE event_type = 'custom_update'::progresseventtype
  AND payload->>'slug' = 'trackmate-1.1'
  AND publish_status IN ('pending', 'failed');
