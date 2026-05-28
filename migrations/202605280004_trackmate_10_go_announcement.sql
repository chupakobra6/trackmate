-- +goose Up
INSERT INTO progress_events (workspace_group_id, event_type, publish_status, payload, created_at)
SELECT
    wg.id,
    'custom_update'::progresseventtype,
    'pending'::progresspublishstatus,
    jsonb_build_object(
        'slug', 'trackmate-go-1.0',
        'title', 'Встречайте: Trackmate 1.0 на Go',
        'body', 'Мы переехали на новый Go runtime. Для участников рабочий поток остался тем же: одна задача дня в «Сегодня» и общая лента результатов в «Прогресс».',
        'items', jsonb_build_array(
            'сохранили задачи, отчеты, участников, напоминания и историю прогресса',
            'заменили Python/aiogram runtime на Go API poller, worker и goose migrations',
            'удалили старый Materials из кода и схемы',
            'в миграции около 5.8k строк нового Go/runtime-кода и около 6.2k строк старого Python/legacy-кода удалено'
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
      AND pe.payload->>'slug' = 'trackmate-go-1.0'
);

-- +goose Down
DELETE FROM progress_events
WHERE event_type = 'custom_update'::progresseventtype
  AND payload->>'slug' = 'trackmate-go-1.0'
  AND publish_status IN ('pending', 'failed');
