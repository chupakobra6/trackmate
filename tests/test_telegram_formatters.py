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
            "username": "Pheik13",
            "display_name": "Pheik13",
            "material_link": "https://t.me/c/123/319?thread=281",
            "text": "текст",
        },
    )

    formatted = format_progress_event(event)

    assert '🚀 <b>@Pheik13 внедрил по <a href="https://t.me/c/123/319?thread=281">материалу</a></b>' in formatted
    assert "<b>Что внедрил:</b>" in formatted
    assert "По материалу:" not in formatted


def test_format_progress_event_formats_daily_task_with_actor_first_title() -> None:
    event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_CLOSED,
        payload={
            "username": "Pheik13",
            "display_name": "Pheik13",
            "status": "done",
            "task_text": "сделать бота",
            "text": "сделал",
        },
    )

    formatted = format_progress_event(event)

    assert formatted.startswith("✅ <b>@Pheik13 выполнил задачу дня</b>")
    assert "<b>Что планировал:</b>" in formatted
    assert "<b>Результат:</b>" in formatted


def test_format_progress_event_uses_consistent_partial_and_auto_failed_wording() -> None:
    partial_event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_CLOSED,
        payload={
            "username": "Pheik13",
            "display_name": "Pheik13",
            "status": "partial",
            "task_text": "сделать бота",
            "text": "частично сделал",
        },
    )
    auto_failed_event = ProgressEvent(
        event_type=ProgressEventType.DAILY_TASK_AUTO_FAILED,
        payload={
            "username": "Pheik13",
            "display_name": "Pheik13",
            "task_text": "сделать бота",
        },
    )

    partial_formatted = format_progress_event(partial_event)
    auto_failed_formatted = format_progress_event(auto_failed_event)

    assert partial_formatted.startswith("🔸 <b>@Pheik13 частично выполнил задачу дня</b>")
    assert auto_failed_formatted.startswith("⏰ <b>@Pheik13 не выполнил задачу дня вовремя</b>")


def test_format_material_card_hides_preview_text() -> None:
    batch = MaterialBatch(batch_size=3, preview_text="Ненужное превью")

    formatted = format_material_card(batch, [])

    assert "Сообщений в подборке" in formatted
    assert "Коротко:" not in formatted


def test_format_daily_task_card_uses_consistent_status_labels() -> None:
    task = DailyTask(text="сделать бота", status=DailyTaskStatus.PARTIAL)

    formatted = format_daily_task_card(task, "Pheik13", "Pheik13")

    assert "<b>Статус:</b> выполнена частично" in formatted
