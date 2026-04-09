from datetime import UTC, datetime
from types import SimpleNamespace

import pytest
from aiogram.types import MessageEntity

from trackmate.adapters.persistence.repositories import TodayRepository, WorkspaceRepository
from trackmate.adapters.telegram.handlers.today import (
    _pending_input_html,
    _pending_input_text,
    _today_task_conflict_notice,
)
from trackmate.application.today import create_daily_task, local_task_date
from trackmate.domain.enums import DailyTaskStatus


def test_local_task_date_uses_workspace_timezone() -> None:
    assert local_task_date(
        "Europe/Moscow",
        now_utc=datetime(2026, 4, 7, 21, 30, tzinfo=UTC),
    ).isoformat() == "2026-04-08"


def test_pending_input_text_supports_voice_and_documents() -> None:
    voice_message = SimpleNamespace(text=None, caption=None, content_type="voice")
    document_message = SimpleNamespace(
        text=None,
        caption=None,
        content_type="document",
        document=SimpleNamespace(file_name="report.pdf"),
    )

    assert _pending_input_text(voice_message) == "Голосовое сообщение"
    assert _pending_input_text(document_message) == "Документ: report.pdf"


def test_pending_input_html_preserves_telegram_entities() -> None:
    text_message = SimpleNamespace(
        text="OpenAI",
        html_text='<a href="https://openai.com">OpenAI</a>',
        caption=None,
        caption_entities=None,
    )
    caption_message = SimpleNamespace(
        text=None,
        html_text="",
        caption="read docs",
        caption_entities=[
            MessageEntity(type="text_link", offset=5, length=4, url="https://platform.openai.com/docs")
        ],
    )

    assert _pending_input_html(text_message) == '<a href="https://openai.com">OpenAI</a>'
    assert _pending_input_html(caption_message) == 'read <a href="https://platform.openai.com/docs">docs</a>'


def test_today_task_conflict_notice_prefers_same_day_task_over_generic_open_task() -> None:
    assert (
        _today_task_conflict_notice(today_task_exists=True, open_task_exists=True)
        == "Задача на сегодня уже зафиксирована."
    )


@pytest.mark.asyncio
async def test_create_daily_task_rejects_second_task_for_same_day_even_if_first_closed(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(5005, "Group", "Europe/Moscow")

    created, task_id = await create_daily_task(
        session,
        workspace_id=workspace.id,
        timezone_name=workspace.timezone,
        user_id=77,
        username="igor",
        display_name="Igor",
        task_html="Первая задача",
        today_card_message_id=1,
    )

    assert created is True
    assert task_id is not None

    task = await TodayRepository(session).get_task(task_id)
    task.status = DailyTaskStatus.DONE

    created_again, same_task_id = await create_daily_task(
        session,
        workspace_id=workspace.id,
        timezone_name=workspace.timezone,
        user_id=77,
        username="igor",
        display_name="Igor",
        task_html="Вторая задача",
        today_card_message_id=2,
    )

    assert created_again is False
    assert same_task_id == task_id
