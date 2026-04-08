from collections.abc import Awaitable, Callable
from typing import Any

import structlog
from aiogram import BaseMiddleware
from aiogram.types import CallbackQuery, Message, Update
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

logger = structlog.get_logger(__name__)


class DbSessionMiddleware(BaseMiddleware):
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self.session_factory = session_factory

    async def __call__(
        self,
        handler: Callable[[Any, dict[str, Any]], Awaitable[Any]],
        event: Any,
        data: dict[str, Any],
    ) -> Any:
        if isinstance(event, Update):
            if event.message:
                message = event.message
                logger.info(
                    "telegram.incoming_message",
                    chat_id=message.chat.id,
                    message_id=message.message_id,
                    thread_id=message.message_thread_id,
                    user_id=message.from_user.id if message.from_user else None,
                    username=message.from_user.username if message.from_user else None,
                    text=message.text,
                    caption=message.caption,
                    content_type=message.content_type,
                )
            elif event.callback_query:
                callback = event.callback_query
                logger.info(
                    "telegram.incoming_callback",
                    chat_id=callback.message.chat.id if callback.message else None,
                    message_id=callback.message.message_id if callback.message else None,
                    thread_id=callback.message.message_thread_id if callback.message else None,
                    user_id=callback.from_user.id,
                    username=callback.from_user.username,
                    data=callback.data,
                )
        elif isinstance(event, Message):
            logger.info(
                "telegram.incoming_message",
                chat_id=event.chat.id,
                message_id=event.message_id,
                thread_id=event.message_thread_id,
                user_id=event.from_user.id if event.from_user else None,
                username=event.from_user.username if event.from_user else None,
                text=event.text,
                caption=event.caption,
                content_type=event.content_type,
            )
        elif isinstance(event, CallbackQuery):
            logger.info(
                "telegram.incoming_callback",
                chat_id=event.message.chat.id if event.message else None,
                message_id=event.message.message_id if event.message else None,
                thread_id=event.message.message_thread_id if event.message else None,
                user_id=event.from_user.id,
                username=event.from_user.username,
                data=event.data,
            )
        async with self.session_factory() as session:
            data["session"] = session
            try:
                result = await handler(event, data)
                await session.commit()
                return result
            except Exception:
                await session.rollback()
                raise
