from datetime import UTC, date, datetime

import pytest

from trackmate.adapters.persistence.repositories import (
    ProgressRepository,
    TodayRepository,
    WorkspaceRepository,
)
from trackmate.adapters.telegram.handlers.today import _report_rejected_text
from trackmate.application.today import run_daily_task_transitions, submit_daily_task_report
from trackmate.domain.enums import DailyTaskStatus, ProgressEventType, TopicKey


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
        report_html="done",
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
        report_html="done again",
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
        task_date=date(2026, 4, 9),
        text="Task",
        today_card_message_id=11,
    )

    await run_daily_task_transitions(
        session,
        now_utc=datetime(2026, 4, 10, 9, 0, 0, tzinfo=UTC),
    )

    events = await ProgressRepository(session).list_pending_events()
    alerts = await TodayRepository(session).list_pending_alerts()

    assert task.status is DailyTaskStatus.AWAITING_REPORT
    assert len(alerts) == 1
    assert all(event.event_type is not ProgressEventType.SYSTEM_ALERT for event in events)


@pytest.mark.asyncio
async def test_submit_daily_task_report_preserves_rich_text_payloads(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1001234567890, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(
        workspace.id,
        TopicKey.TODAY,
        thread_id=281,
        topic_title="Сегодня",
    )
    participant = await workspace_repo.register_participant(workspace.id, 204, "owner", "Owner")
    task = await TodayRepository(session).create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text='Сходить в <a href="https://platform.openai.com/docs">docs</a>',
        today_card_message_id=12,
    )

    submitted = await submit_daily_task_report(
        session,
        task_id=task.id,
        owner_user_id=participant.user_id,
        status=DailyTaskStatus.DONE,
        report_html='Изучил <b>раздел API</b>',
        display_name="Owner",
    )

    events = await ProgressRepository(session).list_pending_events()

    assert submitted is True
    assert events[0].payload["user_id"] == participant.user_id
    assert task.report_text == 'Изучил <b>раздел API</b>'
    assert events[0].payload["task_html"] == 'Сходить в <a href="https://platform.openai.com/docs">docs</a>'
    assert events[0].payload["report_html"] == 'Изучил <b>раздел API</b>'
    assert events[0].payload["task_link"] == "https://t.me/c/1234567890/12?thread=281"


def test_report_rejected_text_is_informative_for_closed_task() -> None:
    assert _report_rejected_text(DailyTaskStatus.FAILED) == "Отчет не принят: задача уже закрыта."
