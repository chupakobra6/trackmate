from __future__ import annotations

import structlog
from aiogram import Bot
from aiogram.exceptions import TelegramBadRequest
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import ProgressRepository, WorkspaceRepository
from trackmate.adapters.telegram.formatters import format_progress_event
from trackmate.adapters.telegram.message_ops import send_message_logged
from trackmate.domain.enums import TopicKey

logger = structlog.get_logger(__name__)


async def publish_pending_progress_events(session: AsyncSession, bot: Bot) -> None:
    progress_repo = ProgressRepository(session)
    workspace_repo = WorkspaceRepository(session)
    events = await progress_repo.list_pending_events()
    for event in events:
        await progress_repo.claim_event_for_publish(event)
        await session.commit()
        workspace = await workspace_repo.get_workspace_by_id(event.workspace_group_id)
        if workspace is None:
            await progress_repo.mark_event_failed(event)
            await session.commit()
            continue
        bindings = await workspace_repo.list_topic_bindings(workspace.id)
        progress_topic = bindings.get(TopicKey.PROGRESS)
        if progress_topic is None:
            await progress_repo.mark_event_failed(event)
            await session.commit()
            continue
        try:
            message = await send_message_logged(
                bot=bot,
                chat_id=workspace.chat_id,
                message_thread_id=progress_topic.thread_id,
                text=format_progress_event(event),
                disable_web_page_preview=True,
            )
        except TelegramBadRequest:
            await progress_repo.mark_event_failed(event)
            await session.commit()
            continue
        except Exception:
            await progress_repo.requeue_event_for_publish(event)
            await session.commit()
            logger.exception("telegram.progress_event_publish_failed", event_id=event.id)
            continue
        await progress_repo.mark_event_published(
            event,
            published_message_id=message.message_id,
            published_at=message.date,
        )
        await session.commit()
