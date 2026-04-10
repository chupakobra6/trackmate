from types import SimpleNamespace

import pytest

from trackmate.adapters.persistence.repositories import TodayRepository, WorkspaceRepository
from trackmate.adapters.telegram.handlers import today as today_module
from trackmate.domain.enums import AlertKind, DailyTaskStatus


@pytest.mark.asyncio
async def test_acknowledge_alert_deletes_message_and_marks_alert_acknowledged(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    today_repo = TodayRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1004004004004, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 501, "igor", "Igor")
    task = await today_repo.create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="Task",
        today_card_message_id=11,
    )
    alert = await today_repo.get_or_create_alert(task.id, AlertKind.DAY_CLOSED_PENDING_REPORT)
    alert.telegram_message_id = 93
    await session.commit()

    deleted_message_ids: list[int | None] = []
    answered: list[str | None] = []

    async def fake_delete_message_safe(*, bot, chat_id, message_id):
        deleted_message_ids.append(message_id)

    async def fake_answer(*, text=None):
        answered.append(text)

    monkeypatch.setattr(today_module, "delete_message_safe", fake_delete_message_safe)

    callback = SimpleNamespace(
        data=f"alert:ack:{alert.id}",
        from_user=SimpleNamespace(id=participant.user_id),
        message=SimpleNamespace(
            bot=object(),
            chat=SimpleNamespace(id=workspace.chat_id),
            message_id=93,
        ),
        answer=fake_answer,
    )

    await today_module.acknowledge_alert(callback, session)

    refreshed_alert = await session.get(type(alert), alert.id)
    assert refreshed_alert is not None
    assert refreshed_alert.acknowledged_at is not None
    assert refreshed_alert.telegram_message_id is None
    assert deleted_message_ids == [93]
    assert answered == ["Алерт скрыт."]


@pytest.mark.asyncio
async def test_open_report_flow_for_closed_task_dismisses_stale_alert(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    today_repo = TodayRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1005005005005, "Group", "UTC")
    participant = await workspace_repo.register_participant(workspace.id, 502, "igor", "Igor")
    task = await today_repo.create_daily_task(
        workspace_id=workspace.id,
        participant_id=participant.id,
        owner_user_id=participant.user_id,
        task_date=workspace.created_at.date(),
        text="Task",
        today_card_message_id=12,
    )
    task.status = DailyTaskStatus.DONE
    alert = await today_repo.get_or_create_alert(task.id, AlertKind.DAY_CLOSED_PENDING_REPORT)
    alert.telegram_message_id = 94
    await session.commit()

    deleted_message_ids: list[int | None] = []
    answered: list[str | None] = []

    async def fake_delete_message_safe(*, bot, chat_id, message_id):
        deleted_message_ids.append(message_id)

    async def fake_answer(*, text=None):
        answered.append(text)

    monkeypatch.setattr(today_module, "delete_message_safe", fake_delete_message_safe)

    callback = SimpleNamespace(
        data=f"task:report:{task.id}",
        from_user=SimpleNamespace(id=participant.user_id),
        message=SimpleNamespace(
            bot=object(),
            chat=SimpleNamespace(id=workspace.chat_id),
            message_id=94,
        ),
        answer=fake_answer,
    )

    await today_module.open_report_flow(callback, session)

    refreshed_alert = await session.get(type(alert), alert.id)
    assert refreshed_alert is not None
    assert refreshed_alert.acknowledged_at is not None
    assert refreshed_alert.telegram_message_id is None
    assert deleted_message_ids == [94]
    assert answered == ["Эта задача уже закрыта."]
