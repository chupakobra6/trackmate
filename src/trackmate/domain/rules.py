from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, date, datetime, time, timedelta
from zoneinfo import ZoneInfo

from trackmate.domain.enums import DailyTaskStatus, MaterialHighestState


@dataclass(frozen=True)
class DailyTaskTransition:
    new_status: DailyTaskStatus | None
    should_emit_auto_fail: bool = False
    should_emit_awaiting_report: bool = False


def derive_material_highest_state(
    *,
    read_at: datetime | None,
    note_progress_event_id: int | None,
    applied_progress_event_id: int | None,
) -> MaterialHighestState:
    if applied_progress_event_id is not None:
        return MaterialHighestState.APPLIED
    if note_progress_event_id is not None:
        return MaterialHighestState.NOTE
    if read_at is not None:
        return MaterialHighestState.READ
    return MaterialHighestState.NONE


def next_daily_task_transition(
    *,
    task_date: date,
    workspace_timezone: str,
    current_status: DailyTaskStatus,
    now_utc: datetime,
) -> DailyTaskTransition:
    local_now = now_utc.astimezone(ZoneInfo(workspace_timezone))
    midnight_after_task = datetime.combine(task_date + timedelta(days=1), time.min, tzinfo=local_now.tzinfo)
    midday_after_task = datetime.combine(
        task_date + timedelta(days=1),
        time(hour=12),
        tzinfo=local_now.tzinfo,
    )

    if current_status is DailyTaskStatus.ACTIVE and local_now >= midnight_after_task:
        return DailyTaskTransition(
            new_status=DailyTaskStatus.AWAITING_REPORT,
            should_emit_awaiting_report=True,
        )
    if current_status is DailyTaskStatus.AWAITING_REPORT and local_now >= midday_after_task:
        return DailyTaskTransition(
            new_status=DailyTaskStatus.FAILED,
            should_emit_auto_fail=True,
        )
    return DailyTaskTransition(new_status=None)


def should_seal_material_batch(*, last_message_at: datetime, timeout_seconds: int, now_utc: datetime) -> bool:
    if last_message_at.tzinfo is None:
        last_message_at = last_message_at.replace(tzinfo=UTC)
    if now_utc.tzinfo is None:
        now_utc = now_utc.replace(tzinfo=UTC)
    return now_utc >= last_message_at + timedelta(seconds=timeout_seconds)
