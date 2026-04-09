from __future__ import annotations

from contextlib import suppress

import structlog
from aiogram import Bot
from aiogram.exceptions import TelegramBadRequest
from aiogram.types import InlineKeyboardMarkup, Message

logger = structlog.get_logger(__name__)


def is_not_modified_error(error: TelegramBadRequest) -> bool:
    return "not modified" in str(error).lower()


async def edit_message_text_safe(
    *,
    bot: Bot,
    chat_id: int,
    message_id: int,
    text: str,
    reply_markup: InlineKeyboardMarkup | None = None,
) -> bool:
    try:
        await bot.edit_message_text(
            chat_id=chat_id,
            message_id=message_id,
            text=text,
            reply_markup=reply_markup,
        )
        logger.info(
            "telegram.outgoing_message_edited",
            chat_id=chat_id,
            message_id=message_id,
            text=text,
        )
        return True
    except TelegramBadRequest as error:
        if is_not_modified_error(error):
            logger.info(
                "telegram.outgoing_message_unchanged",
                chat_id=chat_id,
                message_id=message_id,
            )
            return True
        logger.warning(
            "telegram.outgoing_message_edit_failed",
            chat_id=chat_id,
            message_id=message_id,
            error=str(error),
        )
        return False


async def edit_message_like_safe(
    *,
    message: Message,
    message_id: int | None,
    text: str,
    reply_markup: InlineKeyboardMarkup | None = None,
) -> bool:
    if message_id is None:
        return False
    return await edit_message_text_safe(
        bot=message.bot,
        chat_id=message.chat.id,
        message_id=message_id,
        text=text,
        reply_markup=reply_markup,
    )


async def send_message_logged(
    *,
    bot: Bot,
    chat_id: int,
    text: str,
    message_thread_id: int | None = None,
    reply_markup: InlineKeyboardMarkup | None = None,
    reply_to_message_id: int | None = None,
    disable_notification: bool = True,
    disable_web_page_preview: bool | None = None,
) -> Message:
    message = await bot.send_message(
        chat_id=chat_id,
        text=text,
        message_thread_id=message_thread_id,
        reply_markup=reply_markup,
        reply_to_message_id=reply_to_message_id,
        disable_notification=disable_notification,
        disable_web_page_preview=disable_web_page_preview,
    )
    logger.info(
        "telegram.outgoing_message_sent",
        chat_id=chat_id,
        message_id=message.message_id,
        message_thread_id=message_thread_id,
        reply_to_message_id=reply_to_message_id,
        disable_notification=disable_notification,
        disable_web_page_preview=disable_web_page_preview,
        text=text,
    )
    return message


async def reply_message_logged(
    *,
    message: Message,
    text: str,
    reply_markup: InlineKeyboardMarkup | None = None,
) -> Message:
    return await send_message_logged(
        bot=message.bot,
        chat_id=message.chat.id,
        message_thread_id=message.message_thread_id,
        text=text,
        reply_markup=reply_markup,
    )


async def delete_message_safe(*, bot: Bot, chat_id: int, message_id: int | None) -> None:
    if message_id is None:
        return
    with suppress(TelegramBadRequest):
        await bot.delete_message(chat_id=chat_id, message_id=message_id)
        logger.info(
            "telegram.outgoing_message_deleted",
            chat_id=chat_id,
            message_id=message_id,
        )


async def delete_current_message_safe(message: Message) -> None:
    with suppress(TelegramBadRequest):
        await message.delete()
        logger.info(
            "telegram.outgoing_message_deleted",
            chat_id=message.chat.id,
            message_id=message.message_id,
        )
