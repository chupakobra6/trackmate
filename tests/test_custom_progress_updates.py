from types import SimpleNamespace

import pytest

from trackmate.adapters.persistence.repositories import (
    PendingInputRepository,
    ProgressRepository,
    WorkspaceRepository,
)
from trackmate.adapters.telegram.handlers import progress as progress_module
from trackmate.application.progress import create_custom_progress_update
from trackmate.domain.enums import PendingInputKind, ProgressEventType, TopicKey


@pytest.mark.asyncio
async def test_create_custom_progress_update_creates_pending_event(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    progress_repo = ProgressRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1006006006006, "Group", "UTC")

    await create_custom_progress_update(
        session,
        workspace_id=workspace.id,
        user_id=42,
        username="igor",
        display_name="Igor",
        html="<b>Обновили поток материалов</b>",
        content_kind="text",
    )

    events = await progress_repo.list_pending_events()

    assert len(events) == 1
    assert events[0].event_type is ProgressEventType.CUSTOM_UPDATE
    assert events[0].payload["user_id"] == 42
    assert events[0].payload["username"] == "igor"
    assert events[0].payload["html"] == "<b>Обновили поток материалов</b>"
    assert events[0].payload["content_kind"] == "text"


@pytest.mark.asyncio
async def test_start_progress_update_requires_progress_topic(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1007007007007, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.PROGRESS, 77, "Прогресс")

    answers: list[str] = []

    async def fake_reply(text):
        answers.append(text)
        return SimpleNamespace(message_id=501)

    async def fake_is_group_admin(bot, chat_id, user_id):
        return True

    monkeypatch.setattr(progress_module, "is_group_admin", fake_is_group_admin)

    message = SimpleNamespace(
        chat=SimpleNamespace(id=workspace.chat_id, type="supergroup"),
        from_user=SimpleNamespace(id=42, username="igor", first_name="Igor", last_name=None),
        message_thread_id=99,
        reply=fake_reply,
    )

    await progress_module.start_progress_update(message, bot=object(), session=session)

    pending = await pending_repo.get(workspace.id, 42)
    assert pending is None
    assert answers == ["Команду используй в теме Прогресс."]


@pytest.mark.asyncio
async def test_start_progress_update_sets_pending_for_admin_in_progress_topic(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1008008008008, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.PROGRESS, 88, "Прогресс")

    async def fake_reply_message_logged(*, message, text, reply_markup=None):
        return SimpleNamespace(message_id=777)

    async def fake_is_group_admin(bot, chat_id, user_id):
        return True

    monkeypatch.setattr(progress_module, "is_group_admin", fake_is_group_admin)
    monkeypatch.setattr(progress_module, "reply_message_logged", fake_reply_message_logged)

    message = SimpleNamespace(
        chat=SimpleNamespace(id=workspace.chat_id, type="supergroup"),
        from_user=SimpleNamespace(id=42, username="igor", first_name="Igor", last_name=None),
        message_thread_id=88,
    )

    await progress_module.start_progress_update(message, bot=object(), session=session)

    pending = await pending_repo.get(workspace.id, 42)
    assert pending is not None
    assert pending.kind == PendingInputKind.PROGRESS_UPDATE.value
    assert pending.payload["prompt_message_id"] == 777
