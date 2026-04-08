import asyncio

import structlog
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.bootstrap.app import create_bot
from trackmate.config import get_settings
from trackmate.db.session import create_engine
from trackmate.logging import configure_logging
from trackmate.worker.jobs import (
    dispatch_alerts,
    publish_progress,
    seal_material_batches,
    transition_daily_tasks,
)

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
    bot = create_bot(settings)
    while True:
        async with engine.connect() as connection:
            lock_acquired = False
            session = AsyncSession(bind=connection, expire_on_commit=False)
            try:
                lock_acquired = await _try_acquire_worker_lock(connection)
                if not lock_acquired:
                    logger.info("worker.tick_skipped_lock_not_acquired")
                else:
                    await transition_daily_tasks.run(session)
                    await dispatch_alerts.run(session, bot)
                    await publish_progress.run(session, bot)
                    await seal_material_batches.run(
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
