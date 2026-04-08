from __future__ import annotations

from dataclasses import dataclass

from aiogram import Bot
from aiogram.exceptions import TelegramBadRequest
from aiogram.types import ChatMemberAdministrator, ChatMemberOwner
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import WorkspaceRepository
from trackmate.domain.enums import TopicKey

TOPIC_TITLES = {
    TopicKey.MATERIALS: "Материалы",
    TopicKey.TODAY: "Сегодня",
    TopicKey.PROGRESS: "Прогресс",
}


@dataclass(frozen=True)
class SetupPrerequisites:
    is_supergroup: bool
    is_forum: bool
    bot_is_admin: bool
    can_manage_topics: bool
    can_read_messages: bool

    @property
    def is_ready(self) -> bool:
        return all(
            [
                self.is_supergroup,
                self.is_forum,
                self.bot_is_admin,
                self.can_manage_topics,
                self.can_read_messages,
            ]
        )


async def check_setup_prerequisites(bot: Bot, chat_id: int) -> SetupPrerequisites:
    chat = await bot.get_chat(chat_id)
    member = await bot.get_chat_member(chat_id, bot.id)
    bot_is_admin = isinstance(member, ChatMemberAdministrator | ChatMemberOwner)
    can_manage_topics = bool(getattr(member, "can_manage_topics", False) or isinstance(member, ChatMemberOwner))
    can_read_messages = bot_is_admin
    return SetupPrerequisites(
        is_supergroup=chat.type == "supergroup",
        is_forum=bool(getattr(chat, "is_forum", False)),
        bot_is_admin=bot_is_admin,
        can_manage_topics=can_manage_topics,
        can_read_messages=can_read_messages,
    )


async def is_group_admin(bot: Bot, chat_id: int, user_id: int) -> bool:
    member = await bot.get_chat_member(chat_id, user_id)
    return isinstance(member, ChatMemberAdministrator | ChatMemberOwner)


def _is_missing_thread_error(error: TelegramBadRequest) -> bool:
    message = str(error).lower()
    return "message thread not found" in message or "topic_id_invalid" in message


def _is_not_modified_error(error: TelegramBadRequest) -> bool:
    message = str(error).lower()
    return "not modified" in message or "topic_not_modified" in message


async def _ensure_topic_binding(
    *,
    repo: WorkspaceRepository,
    workspace_id: int,
    existing: dict[TopicKey, object],
    bot: Bot,
    chat_id: int,
    topic_key: TopicKey,
    ) -> tuple[int, bool]:
    topic_title = TOPIC_TITLES[topic_key]
    binding = existing.get(topic_key)
    if binding is None:
        created = await bot.create_forum_topic(
            chat_id=chat_id,
            name=topic_title,
        )
        await repo.upsert_topic_binding(
            workspace_id=workspace_id,
            topic_key=topic_key,
            thread_id=created.message_thread_id,
            topic_title=topic_title,
        )
        return created.message_thread_id, True

    try:
        await bot.edit_forum_topic(
            chat_id=chat_id,
            message_thread_id=binding.thread_id,
            name=topic_title,
        )
    except TelegramBadRequest as error:
        if _is_missing_thread_error(error):
            try:
                created = await bot.create_forum_topic(
                    chat_id=chat_id,
                    name=topic_title,
                )
            except TelegramBadRequest:
                raise
            await repo.upsert_topic_binding(
                workspace_id=workspace_id,
                topic_key=topic_key,
                thread_id=created.message_thread_id,
                topic_title=topic_title,
            )
            await repo.set_topic_messages(
                workspace_id,
                topic_key,
                reset_intro_message_id=True,
                reset_control_message_id=True,
            )
            return created.message_thread_id, True
        elif not _is_not_modified_error(error):
            raise
    if binding.topic_title != topic_title:
        await repo.upsert_topic_binding(
            workspace_id=workspace_id,
            topic_key=topic_key,
            thread_id=binding.thread_id,
            topic_title=topic_title,
        )
        return binding.thread_id, True
    return binding.thread_id, False


async def ensure_workspace_topics(
    session: AsyncSession,
    bot: Bot,
    *,
    chat_id: int,
    title: str | None,
    timezone_name: str,
) -> tuple[dict[TopicKey, int], bool]:
    repo = WorkspaceRepository(session)
    workspace = await repo.get_or_create_workspace(chat_id, title, timezone_name)
    existing = await repo.list_topic_bindings(workspace.id)
    thread_ids: dict[TopicKey, int] = {}
    changed = False
    for key in TOPIC_TITLES:
        thread_id, topic_changed = await _ensure_topic_binding(
            repo=repo,
            workspace_id=workspace.id,
            existing=existing,
            bot=bot,
            chat_id=chat_id,
            topic_key=key,
        )
        thread_ids[key] = thread_id
        changed = changed or topic_changed
    await repo.mark_ready(workspace)
    return thread_ids, changed


async def pin_message(bot: Bot, chat_id: int, message_id: int) -> None:
    try:
        await bot.pin_chat_message(chat_id=chat_id, message_id=message_id)
    except TelegramBadRequest:
        return
