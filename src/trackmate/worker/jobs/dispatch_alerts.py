from __future__ import annotations

from aiogram import Bot
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import TodayRepository, WorkspaceRepository
from trackmate.adapters.telegram.keyboards import alert_keyboard
from trackmate.adapters.telegram.message_ops import send_message_logged
from trackmate.domain.enums import AlertKind


def _alert_text(alert_kind: AlertKind) -> str:
    if alert_kind is AlertKind.DAY_CLOSED_PENDING_REPORT:
        return "🔔 День уже закончился, а отчет по задаче так и не появился."
    return "⏰ Время вышло — задача автоматически отмечена как не выполненная."


async def run(session: AsyncSession, bot: Bot) -> None:
    today_repo = TodayRepository(session)
    workspace_repo = WorkspaceRepository(session)
    alerts = await today_repo.list_pending_alerts()
    for alert in alerts:
        await today_repo.claim_alert_dispatch(alert)
        await session.commit()
        task = await today_repo.get_task(alert.daily_task_id)
        if task is None:
            continue
        workspace = await workspace_repo.get_workspace_by_id(task.workspace_group_id)
        if workspace is None:
            continue
        message = await send_message_logged(
            bot=bot,
            chat_id=workspace.chat_id,
            text=_alert_text(alert.alert_kind),
            reply_to_message_id=task.today_card_message_id,
            reply_markup=alert_keyboard(task.id, alert.id),
        )
        await today_repo.mark_alert_sent(alert, message.message_id)
        await session.commit()
