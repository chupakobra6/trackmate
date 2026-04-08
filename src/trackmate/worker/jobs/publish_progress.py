from __future__ import annotations

from aiogram import Bot
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.application.progress import publish_pending_progress_events


async def run(session: AsyncSession, bot: Bot) -> None:
    await publish_pending_progress_events(session, bot)
