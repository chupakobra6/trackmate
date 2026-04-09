import asyncio
from datetime import UTC, datetime

import structlog
from aiogram import Bot
from aiogram.client.default import DefaultBotProperties
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.application.progress import publish_pending_progress_events
from trackmate.application.today import run_daily_task_transitions
from trackmate.config import get_settings
from trackmate.db.session import create_engine
from trackmate.logging import configure_logging
from trackmate.worker.jobs.dispatch_alerts import run as dispatch_alerts_run
from trackmate.worker.jobs.seal_material_batches import run as seal_material_batches_run

logger = structlog.get_logger(__name__)
WORKER_LOCK_KEY = 3_842_001


async def _try_acquire_worker_lock(connection) -> bool:
    if connection.dialect.name != "postgresql":
        return True
    result = await connection.execute(
        text("SELECT pg_try_advisory_lock(:lock_key)"),
        {"lock_key": WORKER_LOCK_KEY},
    )
    return bool(result.scalar())


async def _release_worker_lock(connection) -> None:
    if connection.dialect.name != "postgresql":
        return
    await connection.execute(
        text("SELECT pg_advisory_unlock(:lock_key)"),
        {"lock_key": WORKER_LOCK_KEY},
    )


async def main() -> None:
    settings = get_settings()
    configure_logging(settings.log_level)
    engine = create_engine(settings)
    bot = Bot(token=settings.bot_token, default=DefaultBotProperties(parse_mode="HTML"))
    while True:
        async with engine.connect() as connection:
            lock_acquired = False
            session = AsyncSession(bind=connection, expire_on_commit=False)
            try:
                lock_acquired = await _try_acquire_worker_lock(connection)
                if not lock_acquired:
                    logger.info("worker.tick_skipped_lock_not_acquired")
                else:
                    await run_daily_task_transitions(session, now_utc=datetime.now(UTC))
                    await dispatch_alerts_run(session, bot)
                    await publish_pending_progress_events(session, bot)
                    await seal_material_batches_run(
                        session,
                        bot,
                        batch_timeout_seconds=settings.material_batch_timeout_seconds,
                    )
                    await session.commit()
            except Exception:
                await session.rollback()
                logger.exception("worker.tick_failed")
            finally:
                if lock_acquired:
                    await _release_worker_lock(connection)
                await session.close()
        await asyncio.sleep(settings.worker_tick_seconds)


if __name__ == "__main__":
    asyncio.run(main())
