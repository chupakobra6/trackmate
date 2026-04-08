from __future__ import annotations

from datetime import UTC, datetime

from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.application.today import run_daily_task_transitions


async def run(session: AsyncSession) -> None:
    await run_daily_task_transitions(session, now_utc=datetime.now(UTC))
