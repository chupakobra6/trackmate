from datetime import timedelta

import pytest

from trackmate.adapters.persistence.repositories import (
    ProgressRepository,
    TodayRepository,
    WorkspaceRepository,
)
from trackmate.application.today import run_daily_task_transitions, submit_daily_task_report
from trackmate.domain.enums import DailyTaskStatus, ProgressEventType


@pytest.mark.asyncio
async def test_submit_daily_task_report_rejects_wrong_owner(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(3003, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 101, "owner", "Owner")
    task = await TodayRepository(session).create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="Task",
        today_card_message_id=10,
    )

    submitted = await submit_daily_task_report(
        session,
        task_id=task.id,
        owner_user_id=999,
        status=DailyTaskStatus.DONE,
        text="done",
        display_name="Intruder",
    )

    assert submitted is False


@pytest.mark.asyncio
async def test_submit_daily_task_report_rejects_already_closed_task(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(4004, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 202, "owner", "Owner")
    task = await TodayRepository(session).create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="Task",
        today_card_message_id=11,
    )
    task.status = DailyTaskStatus.DONE

    submitted = await submit_daily_task_report(
        session,
        task_id=task.id,
        owner_user_id=participant.user_id,
        status=DailyTaskStatus.DONE,
        text="done again",
        display_name="Owner",
    )

    assert submitted is False


@pytest.mark.asyncio
async def test_transition_to_awaiting_report_does_not_create_progress_event(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(4005, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 203, "owner", "Owner")
    task = await TodayRepository(session).create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="Task",
        today_card_message_id=11,
    )

    await run_daily_task_transitions(
        session,
        now_utc=workspace.created_at + timedelta(days=1, seconds=1),
    )

    events = await ProgressRepository(session).list_pending_events()
    alerts = await TodayRepository(session).list_pending_alerts()

    assert task.status is DailyTaskStatus.AWAITING_REPORT
    assert len(alerts) == 1
    assert all(event.event_type is not ProgressEventType.SYSTEM_ALERT for event in events)
