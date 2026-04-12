from __future__ import annotations

from aiogram import Bot, Router
from aiogram.dispatcher.event.bases import UNHANDLED
from aiogram.filters import Command
from aiogram.types import Message
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import PendingInputRepository, WorkspaceRepository
from trackmate.adapters.telegram.handlers.helpers import display_name
from trackmate.adapters.telegram.message_ops import (
    edit_message_like_safe,
    reply_message_logged,
    send_message_logged,
)
from trackmate.adapters.telegram.rich_text import message_input_html, message_input_kind
from trackmate.application.progress import create_custom_progress_update
from trackmate.application.setup import is_group_admin
from trackmate.domain.enums import PendingInputKind, TopicKey

router = Router(name="progress")


def _progress_update_confirmation_text() -> str:
    return "✅ <b>Апдейт отправлен.</b>"


@router.message(Command("update"))
async def start_progress_update(
    message: Message,
    bot: Bot,
    session: AsyncSession,
) -> None:
    if message.chat.type not in {"supergroup", "group"}:
        return
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(message.chat.id)
    if workspace is None:
        return
    if not await is_group_admin(bot, message.chat.id, message.from_user.id):
        await message.reply("Отправить апдейт может только администратор.")
        return
    bindings = await workspace_repo.list_topic_bindings(workspace.id)
    progress_binding = bindings.get(TopicKey.PROGRESS)
    if progress_binding is None or message.message_thread_id != progress_binding.thread_id:
        await message.reply("Команду используй в теме Прогресс.")
        return
    existing_pending = await pending_repo.get(workspace.id, message.from_user.id)
    if existing_pending and existing_pending.kind == PendingInputKind.PROGRESS_UPDATE.value:
        await message.reply("Я уже жду текст апдейта.")
        return
    if existing_pending is not None:
        await message.reply("Сначала закончи текущий ввод.")
        return
    prompt_message = await reply_message_logged(
        message=message,
        text="🆕 <b>Пришли один апдейт одним сообщением. Можно текстом, голосовым или медиа.</b>",
    )
    await pending_repo.upsert(
        workspace.id,
        message.from_user.id,
        PendingInputKind.PROGRESS_UPDATE.value,
        {"prompt_message_id": prompt_message.message_id},
    )


@router.message()
async def submit_progress_update(
    message: Message,
    session: AsyncSession,
) -> None:
    if message.chat.type not in {"supergroup", "group"}:
        return UNHANDLED
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(message.chat.id)
    if workspace is None:
        return UNHANDLED
    pending = await pending_repo.get(workspace.id, message.from_user.id)
    if pending is None or pending.kind != PendingInputKind.PROGRESS_UPDATE.value:
        return UNHANDLED
    update_html = message_input_html(message)
    update_content_kind = message_input_kind(message)
    if update_html is None:
        return UNHANDLED
    await create_custom_progress_update(
        session,
        workspace_id=workspace.id,
        user_id=message.from_user.id,
        username=message.from_user.username,
        display_name=display_name(message.from_user),
        html=update_html,
        content_kind=update_content_kind,
    )
    await pending_repo.clear(workspace.id, message.from_user.id)
    edited = await edit_message_like_safe(
        message=message,
        message_id=pending.payload.get("prompt_message_id"),
        text=_progress_update_confirmation_text(),
    )
    if not edited:
        await send_message_logged(
            bot=message.bot,
            chat_id=message.chat.id,
            message_thread_id=message.message_thread_id,
            text=_progress_update_confirmation_text(),
        )
