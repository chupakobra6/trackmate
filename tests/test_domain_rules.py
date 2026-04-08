from datetime import UTC, date, datetime

from aiogram.exceptions import TelegramBadRequest

from trackmate.application.setup import _is_not_modified_error
from trackmate.domain.enums import DailyTaskStatus, MaterialHighestState
from trackmate.domain.rules import (
    derive_material_highest_state,
    next_daily_task_transition,
    should_seal_material_batch,
)


def test_material_highest_state_prefers_applied() -> None:
    state = derive_material_highest_state(
        read_at=datetime(2026, 4, 8, tzinfo=UTC),
        note_progress_event_id=1,
        applied_progress_event_id=2,
    )
    assert state is MaterialHighestState.APPLIED


def test_daily_task_goes_to_awaiting_report_after_midnight() -> None:
    transition = next_daily_task_transition(
        task_date=date(2026, 4, 7),
        workspace_timezone="UTC",
        current_status=DailyTaskStatus.ACTIVE,
        now_utc=datetime(2026, 4, 8, 0, 0, 1, tzinfo=UTC),
    )
    assert transition.new_status is DailyTaskStatus.AWAITING_REPORT


def test_daily_task_goes_to_failed_after_midday() -> None:
    transition = next_daily_task_transition(
        task_date=date(2026, 4, 7),
        workspace_timezone="UTC",
        current_status=DailyTaskStatus.AWAITING_REPORT,
        now_utc=datetime(2026, 4, 8, 12, 0, 1, tzinfo=UTC),
    )
    assert transition.new_status is DailyTaskStatus.FAILED


def test_material_batch_seals_after_timeout() -> None:
    assert should_seal_material_batch(
        last_message_at=datetime(2026, 4, 8, 10, 0, 0, tzinfo=UTC),
        timeout_seconds=30,
        now_utc=datetime(2026, 4, 8, 10, 0, 31, tzinfo=UTC),
    )


def test_topic_not_modified_is_treated_as_non_error() -> None:
    error = TelegramBadRequest(method=None, message="Telegram server says - Bad Request: TOPIC_NOT_MODIFIED")

    assert _is_not_modified_error(error) is True
