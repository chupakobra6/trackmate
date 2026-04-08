import pytest
from sqlalchemy.exc import IntegrityError

from trackmate.adapters.persistence.repositories import TodayRepository, WorkspaceRepository
from trackmate.domain.enums import TopicKey


@pytest.mark.asyncio
async def test_daily_task_uniqueness_constraint(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(1001, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 42, "igor", "Igor")
    today_repo = TodayRepository(session)

    await today_repo.create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="First task",
        today_card_message_id=1,
    )
    await session.commit()

    with pytest.raises(IntegrityError):
        await today_repo.create_daily_task(
            workspace_id=workspace.id,
            participant_id=participant.id,
            owner_user_id=participant.user_id,
            task_date=workspace.created_at.date(),
            text="Second task",
            today_card_message_id=2,
        )


@pytest.mark.asyncio
async def test_topic_binding_is_idempotent_by_logical_key(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(2002, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.TODAY, 10, "Сегодня")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.TODAY, 11, "Сегодня")
    await session.commit()

    bindings = await workspace_repo.list_topic_bindings(workspace.id)
    assert bindings[TopicKey.TODAY].thread_id == 11
    assert len(bindings) == 1
