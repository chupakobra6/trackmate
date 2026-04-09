from datetime import UTC, datetime

import pytest
from aiogram.types import Chat, Message, Update, User
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from trackmate.adapters.telegram.middleware import DbSessionMiddleware


@pytest.mark.asyncio
async def test_db_session_middleware_passes_original_update_to_handler() -> None:
    engine = create_async_engine("sqlite+aiosqlite:///:memory:")
    session_factory = async_sessionmaker(engine, expire_on_commit=False)
    middleware = DbSessionMiddleware(session_factory)
    update = Update(
        update_id=1,
        message=Message(
            message_id=10,
            date=datetime.now(UTC),
            chat=Chat(id=1, type="private"),
            from_user=User(id=2, is_bot=False, first_name="Igor"),
            text="/setup",
        ),
    )
    seen: dict[str, object] = {}

    async def handler(event, data):
        seen["event"] = event
        seen["session"] = data["session"]
        return "ok"

    result = await middleware(handler, update, {})

    assert result == "ok"
    assert seen["event"] is update
    assert isinstance(seen["session"], AsyncSession)

    await engine.dispose()
