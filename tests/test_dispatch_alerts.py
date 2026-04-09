from datetime import date

import pytest

from trackmate.adapters.persistence.repositories import TodayRepository, WorkspaceRepository
from trackmate.domain.enums import AlertDispatchStatus, AlertKind
from trackmate.worker.jobs import dispatch_alerts
from trackmate.worker.jobs.dispatch_alerts import _alert_text


def test_alert_text_uses_consistent_daily_task_wording() -> None:
    assert _alert_text(AlertKind.DAY_CLOSED_PENDING_REPORT) == (
        "🔔 День уже закончился, а отчет по задаче так и не появился."
    )
    assert _alert_text(AlertKind.OVERDUE_TASK_FAILED) == (
        "⏰ Время вышло — задача автоматически отмечена как не выполненная."
    )


@pytest.mark.asyncio
async def test_dispatch_alerts_requeues_alert_when_send_fails(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1001001001001, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 42, "igor", "Igor")
    task = await TodayRepository(session).create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=date(2026, 4, 9),
        text="Task",
        today_card_message_id=99,
    )
    alert = await TodayRepository(session).get_or_create_alert(task.id, AlertKind.DAY_CLOSED_PENDING_REPORT)
    await session.commit()

    async def raising_send_message_logged(**kwargs):
        raise RuntimeError("network down")

    monkeypatch.setattr(dispatch_alerts, "send_message_logged", raising_send_message_logged)

    await dispatch_alerts.run(session, bot=object())

    refreshed_alert = await session.get(type(alert), alert.id)
    assert refreshed_alert is not None
    assert refreshed_alert.dispatch_status is AlertDispatchStatus.PENDING
