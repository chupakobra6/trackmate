#!/usr/bin/env sh
set -eu

echo "Resetting database from current .env..."

uv run python - <<'PY'
import asyncio

from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine

from trackmate.config import get_settings


async def main() -> None:
    settings = get_settings()
    engine = create_async_engine(settings.database_url)
    try:
        async with engine.begin() as conn:
            await conn.execute(text("DROP SCHEMA public CASCADE"))
            await conn.execute(text("CREATE SCHEMA public"))
    finally:
        await engine.dispose()


asyncio.run(main())
PY

uv run alembic upgrade head

echo "Database reset complete."
