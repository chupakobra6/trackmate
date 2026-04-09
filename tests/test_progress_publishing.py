import pytest

from trackmate.adapters.persistence.repositories import ProgressRepository, WorkspaceRepository
from trackmate.application import progress as progress_module
from trackmate.application.progress import publish_pending_progress_events
from trackmate.domain.enums import ProgressEventType, ProgressPublishStatus, TopicKey


@pytest.mark.asyncio
async def test_publish_pending_progress_events_requeues_event_when_send_fails(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    progress_repo = ProgressRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1003003003003, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.PROGRESS, 20, "Прогресс")
    participant = await workspace_repo.register_participant(workspace.id, 42, "igor", "Igor")
    event = await progress_repo.create_event(
        workspace_group_id=workspace.id,
        participant_id=participant.id,
        event_type=ProgressEventType.MATERIAL_NOTE_ADDED,
        payload={
            "username": "igor",
            "display_name": "Igor",
            "material_link": "https://t.me/c/123/1?thread=20",
            "html": "text",
        },
    )
    await session.commit()

    async def raising_send_message_logged(**kwargs):
        raise RuntimeError("network down")

    monkeypatch.setattr(progress_module, "send_message_logged", raising_send_message_logged)

    await publish_pending_progress_events(session, bot=object())

    refreshed_event = await session.get(type(event), event.id)
    assert refreshed_event is not None
    assert refreshed_event.publish_status is ProgressPublishStatus.PENDING
