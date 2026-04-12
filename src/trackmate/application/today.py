from __future__ import annotations

from datetime import UTC, datetime
from zoneinfo import ZoneInfo

from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import (
    ProgressRepository,
    TodayRepository,
    WorkspaceRepository,
)
from trackmate.db.models import Participant
from trackmate.domain.enums import AlertKind, DailyTaskStatus, ProgressEventType, TopicKey
from trackmate.domain.rules import next_daily_task_transition


def _message_link(*, chat_id: int, message_id: int | None, thread_id: int | None) -> str | None:
    if message_id is None:
        return None
    chat_id_text = str(chat_id)
    if not chat_id_text.startswith("-100"):
        return None
    link = f"https://t.me/c/{chat_id_text[4:]}/{message_id}"
    if thread_id is not None:
        return f"{link}?thread={thread_id}"
    return link


def local_task_date(timezone_name: str, now_utc: datetime | None = None):
    now = now_utc or datetime.now(UTC)
    return now.astimezone(ZoneInfo(timezone_name)).date()


async def create_daily_task(
    session: AsyncSession,
    *,
    workspace_id: int,
    timezone_name: str,
    user_id: int,
    username: str | None,
    display_name: str,
    task_html: str,
    today_card_message_id: int,
) -> tuple[bool, int | None]:
    workspace_repo = WorkspaceRepository(session)
    today_repo = TodayRepository(session)
    participant = await workspace_repo.register_participant(workspace_id, user_id, username, display_name)
    task_date = local_task_date(timezone_name)
    existing_task = await today_repo.get_task_for_date(workspace_id, participant.id, task_date)
    if existing_task is not None:
        return False, existing_task.id

    try:
        task = await today_repo.create_daily_task(
            workspace_id=workspace_id,
            participant_id=participant.id,
            owner_user_id=user_id,
            task_date=task_date,
            text=task_html,
            today_card_message_id=today_card_message_id,
        )
    except IntegrityError:
        await session.rollback()
        existing_task = await today_repo.get_task_for_date(workspace_id, participant.id, task_date)
        return False, existing_task.id if existing_task is not None else None
    await session.flush()
    return True, task.id


async def submit_daily_task_report(
    session: AsyncSession,
    *,
    task_id: int,
    owner_user_id: int,
    status: DailyTaskStatus,
    report_html: str,
    display_name: str,
) -> bool:
    today_repo = TodayRepository(session)
    progress_repo = ProgressRepository(session)
    workspace_repo = WorkspaceRepository(session)
    task = await today_repo.get_task(task_id)
    if task is None:
        return False
    if task.owner_user_id != owner_user_id:
        return False
    if task.status not in {DailyTaskStatus.ACTIVE, DailyTaskStatus.AWAITING_REPORT}:
        return False
    if status not in {DailyTaskStatus.DONE, DailyTaskStatus.PARTIAL, DailyTaskStatus.FAILED}:
        return False
    participant = await session.get(Participant, task.participant_id)
    workspace = await workspace_repo.get_workspace_by_id(task.workspace_group_id)
    today_binding = await workspace_repo.get_topic_binding(task.workspace_group_id, TopicKey.TODAY)
    task.status = status
    task.report_status = status
    task.report_text = report_html
    task.reported_at = datetime.now(UTC)
    await progress_repo.create_event(
        workspace_group_id=task.workspace_group_id,
        participant_id=task.participant_id,
        daily_task_id=task.id,
        event_type=ProgressEventType.DAILY_TASK_CLOSED,
        payload={
            "status": status.value,
            "report_html": report_html,
            "user_id": participant.user_id if participant else owner_user_id,
            "display_name": display_name,
            "username": participant.username if participant else None,
            "task_html": task.text,
            "task_link": _message_link(
                chat_id=workspace.chat_id,
                message_id=task.today_card_message_id,
                thread_id=today_binding.thread_id if today_binding is not None else None,
            )
            if workspace is not None
            else None,
        },
    )
    await session.flush()
    return True


async def run_daily_task_transitions(session: AsyncSession, *, now_utc: datetime) -> None:
    today_repo = TodayRepository(session)
    workspace_repo = WorkspaceRepository(session)
    progress_repo = ProgressRepository(session)

    tasks = await today_repo.list_tasks_for_transition(
        [DailyTaskStatus.ACTIVE, DailyTaskStatus.AWAITING_REPORT]
    )
    for task in tasks:
        workspace = await workspace_repo.get_workspace_by_id(task.workspace_group_id)
        if workspace is None:
            continue
        transition = next_daily_task_transition(
            task_date=task.task_date,
            workspace_timezone=workspace.timezone,
            current_status=task.status,
            now_utc=now_utc,
        )
        if transition.new_status is None:
            continue
        participant = await session.get(Participant, task.participant_id)

        task.status = transition.new_status
        if transition.new_status is DailyTaskStatus.AWAITING_REPORT:
            task.awaiting_report_at = now_utc
            await today_repo.get_or_create_alert(task.id, AlertKind.DAY_CLOSED_PENDING_REPORT)
        elif transition.new_status is DailyTaskStatus.FAILED:
            task.failed_at = now_utc
            await today_repo.get_or_create_alert(task.id, AlertKind.OVERDUE_TASK_FAILED)
            today_binding = await workspace_repo.get_topic_binding(task.workspace_group_id, TopicKey.TODAY)
            await progress_repo.create_event(
                workspace_group_id=task.workspace_group_id,
                participant_id=task.participant_id,
                daily_task_id=task.id,
                event_type=ProgressEventType.DAILY_TASK_AUTO_FAILED,
                payload={
                    "task_html": task.text,
                    "user_id": participant.user_id if participant else task.owner_user_id,
                    "display_name": participant.display_name if participant else str(task.owner_user_id),
                    "username": participant.username if participant else None,
                    "task_link": _message_link(
                        chat_id=workspace.chat_id,
                        message_id=task.today_card_message_id,
                        thread_id=today_binding.thread_id if today_binding is not None else None,
                    ),
                },
            )
    await session.flush()
