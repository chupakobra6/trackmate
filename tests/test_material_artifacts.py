import pytest

from trackmate.adapters.persistence.repositories import (
    MaterialRepository,
    ProgressRepository,
    WorkspaceRepository,
)
from trackmate.application.materials import mark_material_read, submit_material_artifact


@pytest.mark.asyncio
async def test_submit_material_artifact_rejects_duplicate_note(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(6006, "Group", "Europe/Moscow")
    participant = await workspace_repo.register_participant(workspace.id, 77, "igor", "Igor")

    batch = await MaterialRepository(session).create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        sender_id=participant.user_id,
        media_group_id=None,
        upload_session_key="batch",
    )
    batch.preview_text = "Полезный материал"

    first = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        text="Первая заметка",
        is_applied=False,
    )
    second = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        text="Вторая заметка",
        is_applied=False,
    )

    events = await ProgressRepository(session).list_pending_events()

    assert first is True
    assert second is False
    assert len(events) == 1


@pytest.mark.asyncio
async def test_submit_material_artifact_rejects_duplicate_applied(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(7007, "Group", "Europe/Moscow")
    participant = await workspace_repo.register_participant(workspace.id, 88, "igor", "Igor")

    batch = await MaterialRepository(session).create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        sender_id=participant.user_id,
        media_group_id=None,
        upload_session_key="batch",
    )
    batch.preview_text = "Полезный материал"

    first = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        text="Первое внедрение",
        is_applied=True,
    )
    second = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        text="Второе внедрение",
        is_applied=True,
    )

    events = await ProgressRepository(session).list_pending_events()

    assert first is True
    assert second is False
    assert len(events) == 1


@pytest.mark.asyncio
async def test_mark_material_read_reports_repeat_reads(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(8008, "Group", "Europe/Moscow")
    participant = await workspace_repo.register_participant(workspace.id, 89, "igor", "Igor")

    batch = await MaterialRepository(session).create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        sender_id=participant.user_id,
        media_group_id=None,
        upload_session_key="batch",
    )

    _, first_created = await mark_material_read(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
    )
    _, second_created = await mark_material_read(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
    )

    assert first_created is True
    assert second_created is False


@pytest.mark.asyncio
async def test_submit_material_artifact_includes_material_link(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1001234567890, "Group", "Europe/Moscow")
    participant = await workspace_repo.register_participant(workspace.id, 90, "igor", "Igor")

    batch = await MaterialRepository(session).create_batch(
        workspace_id=workspace.id,
        materials_thread_id=281,
        sender_id=participant.user_id,
        media_group_id=None,
        upload_session_key="batch",
    )
    batch.tracking_card_message_id = 319

    created = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        text='<b>Первая</b> <a href="https://example.com">заметка</a>',
        is_applied=False,
    )

    events = await ProgressRepository(session).list_pending_events()

    assert created is True
    assert events[0].payload["material_link"] == "https://t.me/c/1234567890/319?thread=281"
    assert events[0].payload["text"] == '<b>Первая</b> <a href="https://example.com">заметка</a>'
