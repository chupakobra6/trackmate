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
        media_group_id=None,
    )

    first = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        artifact_html="Первая заметка",
        is_applied=False,
    )
    second = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        artifact_html="Вторая заметка",
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
        media_group_id=None,
    )

    first = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        artifact_html="Первое внедрение",
        is_applied=True,
    )
    second = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        artifact_html="Второе внедрение",
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
        media_group_id=None,
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
async def test_material_progress_isolated_per_user_when_another_person_clicks(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    materials_repo = MaterialRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(8009, "Group", "Europe/Moscow")
    first = await workspace_repo.register_participant(workspace.id, 89, "igor", "Igor")
    second = await workspace_repo.register_participant(workspace.id, 90, "masha", "Masha")

    batch = await materials_repo.create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
    )

    await mark_material_read(
        session,
        workspace_id=workspace.id,
        user_id=first.user_id,
        username=first.username,
        display_name=first.display_name,
        batch_id=batch.id,
    )
    await mark_material_read(
        session,
        workspace_id=workspace.id,
        user_id=second.user_id,
        username=second.username,
        display_name=second.display_name,
        batch_id=batch.id,
    )

    first_progress = await materials_repo.get_progress(batch.id, first.id)
    second_progress = await materials_repo.get_progress(batch.id, second.id)

    assert first_progress is not None
    assert second_progress is not None
    assert first_progress.participant_id != second_progress.participant_id
    assert first_progress.read_at is not None
    assert second_progress.read_at is not None


@pytest.mark.asyncio
async def test_submit_material_artifact_includes_material_link(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1001234567890, "Group", "Europe/Moscow")
    participant = await workspace_repo.register_participant(workspace.id, 90, "igor", "Igor")

    batch = await MaterialRepository(session).create_batch(
        workspace_id=workspace.id,
        materials_thread_id=281,
        media_group_id=None,
    )
    batch.tracking_card_message_id = 319

    created = await submit_material_artifact(
        session,
        workspace_id=workspace.id,
        user_id=participant.user_id,
        username=participant.username,
        display_name=participant.display_name,
        batch_id=batch.id,
        artifact_html='<b>Первая</b> <a href="https://example.com">заметка</a>',
        is_applied=False,
    )

    events = await ProgressRepository(session).list_pending_events()

    assert created is True
    assert events[0].payload["user_id"] == participant.user_id
    assert events[0].payload["material_link"] == "https://t.me/c/1234567890/319?thread=281"
    assert events[0].payload["html"] == '<b>Первая</b> <a href="https://example.com">заметка</a>'
    assert events[0].payload["content_kind"] == "text"
