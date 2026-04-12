from trackmate.adapters.telegram.formatters import (
    format_daily_task_card,
    format_material_card,
    format_progress_event,
)
from trackmate.db.models import DailyTask, MaterialBatch, ProgressEvent
from trackmate.domain.enums import DailyTaskStatus, ProgressEventType


def test_format_progress_event_uses_actor_first_titles() -> None:
    event = ProgressEvent(
        event_type=ProgressEventType.MATERIAL_APPLIED,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "material_link": "https://t.me/c/123/319?thread=281",
            "html": '<b>текст</b> <a href="https://example.com">с ссылкой</a>',
            "content_kind": "text",
        },
    )

    formatted = format_progress_event(event)

    assert '🚀 <b><a href="tg://user?id=42">Pheik13</a> внедрил <a href="https://t.me/c/123/319?thread=281">материал</a></b>' in formatted
    assert "<b>Что внедрил:</b>\n\n<blockquote><b>текст</b> <a href=\"https://example.com\">с ссылкой</a></blockquote>" in formatted
    assert "По материалу:" not in formatted


def test_format_progress_event_uses_non_text_material_labels() -> None:
    event = ProgressEvent(
        event_type=ProgressEventType.MATERIAL_NOTE_ADDED,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "material_link": "https://t.me/c/123/319?thread=281",
            "html": "Фото",
            "content_kind": "non_text",
        },
    )

    formatted = format_progress_event(event)

    assert '📝 <b><a href="tg://user?id=42">Pheik13</a> добавил заметку к <a href="https://t.me/c/123/319?thread=281">материалу</a></b>' in formatted
    assert "<b>Формат заметки:</b>\n\n<blockquote>Фото</blockquote>" in formatted
    assert "Текст заметки" not in formatted


def test_format_progress_event_formats_custom_update() -> None:
    event = ProgressEvent(
        event_type=ProgressEventType.CUSTOM_UPDATE,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "html": "Обновили дедлайны и текст кнопок",
            "content_kind": "text",
        },
    )

    formatted = format_progress_event(event)

    assert formatted.startswith('🆕 <b><a href="tg://user?id=42">Pheik13</a> поделился апдейтом</b>')
    assert "<b>Что поменялось:</b>\n\n<blockquote>Обновили дедлайны и текст кнопок</blockquote>" in formatted


def test_format_progress_event_formats_daily_task_with_actor_first_title() -> None:
    event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_CLOSED,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "status": "done",
            "task_html": "сделать бота",
            "report_html": "сделал",
            "task_link": "https://t.me/c/123/545?thread=538",
        },
    )

    formatted = format_progress_event(event)

    assert formatted.startswith('✅ <b><a href="tg://user?id=42">Pheik13</a> выполнил <a href="https://t.me/c/123/545?thread=538">задачу дня</a></b>')
    assert "<b>Что планировал:</b>\n\n<blockquote>сделать бота</blockquote>" in formatted
    assert "<b>Результат:</b>\n\n<blockquote>сделал</blockquote>" in formatted


def test_format_progress_event_uses_consistent_partial_and_auto_failed_wording() -> None:
    partial_event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_CLOSED,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "status": "partial",
            "task_html": "сделать бота",
            "report_html": "частично сделал",
            "task_link": "https://t.me/c/123/545?thread=538",
        },
    )
    auto_failed_event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_AUTO_FAILED,
        payload={
            "user_id": 42,
            "username": "Pheik13",
            "display_name": "Pheik13",
            "task_html": "сделать бота",
            "task_link": "https://t.me/c/123/545?thread=538",
        },
    )

    partial_formatted = format_progress_event(partial_event)
    auto_failed_formatted = format_progress_event(auto_failed_event)

    assert partial_formatted.startswith('🔸 <b><a href="tg://user?id=42">Pheik13</a> частично выполнил <a href="https://t.me/c/123/545?thread=538">задачу дня</a></b>')
    assert auto_failed_formatted.startswith('⏰ <b><a href="tg://user?id=42">Pheik13</a> не выполнил <a href="https://t.me/c/123/545?thread=538">задачу дня</a> вовремя</b>')
    assert "<b>Что планировал:</b>\n\n<blockquote>сделать бота</blockquote>" in auto_failed_formatted


def test_format_material_card_hides_preview_text() -> None:
    batch = MaterialBatch(batch_size=3)

    formatted = format_material_card(batch, [])

    assert "Сообщений в подборке" in formatted
    assert "Коротко:" not in formatted


def test_format_material_card_adds_spacing_before_progress_events() -> None:
    batch = MaterialBatch(batch_size=1)
    progress = type(
        "Progress",
        (),
        {
            "participant": type("Participant", (), {"username": "Pheik13", "display_name": "Pheik13"})(),
            "read_at": object(),
            "note_progress_event_id": None,
            "applied_progress_event_id": None,
        },
    )()

    formatted = format_material_card(batch, [progress])

    assert formatted == "📚 <b>Материал</b>\n\n<blockquote>@Pheik13 прочитал.</blockquote>"


def test_format_daily_task_card_uses_consistent_status_labels() -> None:
    task = DailyTask(text="сделать бота", status=DailyTaskStatus.PARTIAL)

    formatted = format_daily_task_card(task, "Pheik13", "Pheik13")

    assert "<b>Статус:</b> выполнена частично" in formatted


def test_format_daily_task_card_prefers_saved_html_text() -> None:
    task = DailyTask(
        text='Сходить в <a href="https://platform.openai.com/docs">docs</a>\n<blockquote>цитата</blockquote>',
        report_text='Итог: <b>готово</b>',
        status=DailyTaskStatus.DONE,
    )

    formatted = format_daily_task_card(task, "Pheik13", "Pheik13")

    assert 'Сходить в <a href="https://platform.openai.com/docs">docs</a>' in formatted
    assert "<blockquote>цитата</blockquote>" in formatted
    assert "<blockquote>Итог: <b>готово</b></blockquote>" in formatted
