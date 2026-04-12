from __future__ import annotations

from contextlib import suppress

from aiogram import Bot, F, Router
from aiogram.exceptions import TelegramBadRequest
from aiogram.filters import Command
from aiogram.types import CallbackQuery, ChatMemberUpdated, Message
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import WorkspaceRepository
from trackmate.adapters.telegram.formatters import format_setup_checklist
from trackmate.adapters.telegram.keyboards import setup_keyboard, today_control_keyboard
from trackmate.adapters.telegram.message_ops import edit_message_text_safe, send_message_logged
from trackmate.application.setup import (
    check_setup_prerequisites,
    ensure_workspace_topics,
    is_group_admin,
    pin_message,
)
from trackmate.config import Settings
from trackmate.domain.enums import TopicKey

router = Router(name="setup")

TODAY_CONTROL_TEXT = (
    "🎯 <b>Сегодня</b>\n"
    "Здесь у каждого одна главная задача на день.\n"
    "Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.\n\n"
    "Как это работает:\n"
    "• ты формулируешь одну задачу на день;\n"
    "• я закрепляю ее в отдельной карточке;\n"
    "• вечером в этой же карточке можно оставить результат."
)
MATERIALS_INTRO_TEXT = (
    "📚 <b>Материалы</b>\n"
    "Сюда удобно пересылать полезные посты, заметки и разборы.\n\n"
    "Если подряд прилетит несколько сообщений, я соберу их в одну карточку.\n"
    "Дальше по кнопкам можно отметить, что материал прочитан, по нему есть заметка или уже получилось что-то внедрить."
)
PROGRESS_INTRO_TEXT = (
    "✨ <b>Прогресс</b>\n"
    "Здесь будет собираться все важное в аккуратную общую ленту.\n\n"
    "Что появится здесь:\n"
    "• заметки по материалам;\n"
    "• результаты внедрения;\n"
    "• закрытые задачи дня;\n"
    "• кастомные апдейты через команду <code>/update</code>.\n\n"
    "Так всегда видно, кто что прочитал, сделал и довел до результата."
)
SETUP_READY_TEXT = (
    "✅ <b>Все на месте.</b>\n"
    "Темы и стартовые сообщения уже в порядке. Ничего восстанавливать не пришлось."
)
SETUP_REPAIRED_TEXT = (
    "✨ <b>Готово!</b>\n"
    "Я проверил пространство и восстановил все, чего не хватало.\n\n"
    "Что дальше:\n"
    "• в теме <b>Сегодня</b> каждый фиксирует одну задачу на день;\n"
    "• в теме <b>Материалы</b> можно пересылать полезные материалы;\n"
    "• в теме <b>Прогресс</b> будут появляться результаты и заметки."
)


async def _try_edit_setup_message(
    *,
    bot: Bot,
    chat_id: int,
    message_id: int,
    text: str,
) -> bool:
    return await edit_message_text_safe(
        bot=bot,
        chat_id=chat_id,
        message_id=message_id,
        text=text,
        reply_markup=setup_keyboard(),
    )


async def _send_setup_message(*, bot: Bot, chat_id: int, text: str) -> Message:
    return await send_message_logged(
        bot=bot,
        chat_id=chat_id,
        text=text,
        reply_markup=setup_keyboard(),
    )


async def _ensure_topic_message(
    *,
    bot: Bot,
    repo: WorkspaceRepository,
    workspace_id: int,
    chat_id: int,
    thread_id: int,
    topic_key: TopicKey,
    current_message_id: int | None,
    text: str,
    reply_markup=None,
    is_control: bool = False,
    pin_after_send: bool = False,
) -> bool:
    if current_message_id is not None:
        updated = await edit_message_text_safe(
            bot=bot,
            chat_id=chat_id,
            message_id=current_message_id,
            text=text,
            reply_markup=reply_markup,
        )
        if updated:
            return False
    message = await send_message_logged(
        bot=bot,
        chat_id=chat_id,
        message_thread_id=thread_id,
        text=text,
        reply_markup=reply_markup,
    )
    await repo.set_topic_messages(
        workspace_id,
        topic_key,
        control_message_id=message.message_id if is_control else None,
        intro_message_id=None if is_control else message.message_id,
    )
    if pin_after_send:
        await pin_message(bot, chat_id, message.message_id)
    return True


async def _upsert_setup_message(
    *,
    bot: Bot,
    session: AsyncSession,
    chat_id: int,
    chat_title: str | None,
    timezone_name: str,
    fallback_message: Message | None = None,
    notice: str | None = None,
) -> None:
    repo = WorkspaceRepository(session)
    workspace = await repo.get_or_create_workspace(chat_id, chat_title, timezone_name)
    prerequisites = await check_setup_prerequisites(bot, chat_id)
    text = format_setup_checklist(
        ready=prerequisites.is_ready,
        is_supergroup=prerequisites.is_supergroup,
        is_forum=prerequisites.is_forum,
        is_admin=prerequisites.bot_is_admin,
        can_manage_topics=prerequisites.can_manage_topics,
        can_read_messages=prerequisites.can_read_messages,
        notice=notice,
    )

    if workspace.setup_message_id is not None:
        updated = await _try_edit_setup_message(
            bot=bot,
            chat_id=chat_id,
            message_id=workspace.setup_message_id,
            text=text,
        )
        if updated:
            if fallback_message and fallback_message.message_id != workspace.setup_message_id:
                with suppress(TelegramBadRequest):
                    await bot.delete_message(chat_id=chat_id, message_id=fallback_message.message_id)
            return

    if fallback_message is not None:
        updated = await _try_edit_setup_message(
            bot=bot,
            chat_id=chat_id,
            message_id=fallback_message.message_id,
            text=text,
        )
        if updated:
            await repo.set_setup_message_id(workspace.id, fallback_message.message_id)
            return
        message = await _send_setup_message(bot=bot, chat_id=chat_id, text=text)
        await repo.set_setup_message_id(workspace.id, message.message_id)
        return

    message = await _send_setup_message(bot=bot, chat_id=chat_id, text=text)
    await repo.set_setup_message_id(workspace.id, message.message_id)


@router.my_chat_member()
async def on_bot_added(event: ChatMemberUpdated, bot: Bot, session: AsyncSession, settings: Settings) -> None:
    if event.chat.type not in {"group", "supergroup"}:
        return
    if event.new_chat_member.status not in {"member", "administrator"}:
        return
    await _upsert_setup_message(
        bot=bot,
        session=session,
        chat_id=event.chat.id,
        chat_title=event.chat.title,
        timezone_name=settings.default_timezone,
    )


@router.message(Command("setup"), F.chat.type.in_({"group", "supergroup"}))
async def setup_command(message: Message, bot: Bot, session: AsyncSession, settings: Settings) -> None:
    await _upsert_setup_message(
        bot=bot,
        session=session,
        chat_id=message.chat.id,
        chat_title=message.chat.title,
        timezone_name=settings.default_timezone,
    )


@router.callback_query(F.data == "setup:check")
async def check_setup_callback(
    callback: CallbackQuery,
    bot: Bot,
    session: AsyncSession,
    settings: Settings,
) -> None:
    await _upsert_setup_message(
        bot=bot,
        session=session,
        chat_id=callback.message.chat.id,
        chat_title=callback.message.chat.title,
        timezone_name=settings.default_timezone,
        fallback_message=callback.message,
    )
    await callback.answer()


@router.callback_query(F.data == "setup:start")
async def start_setup_callback(
    callback: CallbackQuery,
    bot: Bot,
    session: AsyncSession,
    settings: Settings,
) -> None:
    chat = callback.message.chat
    repo = WorkspaceRepository(session)
    if not await is_group_admin(bot, chat.id, callback.from_user.id):
        await callback.answer(text="Оформить группу может только администратор.")
        return
    prerequisites = await check_setup_prerequisites(bot, chat.id)
    if not prerequisites.is_ready:
        await callback.answer(text="Сначала закрой пункты выше, а потом запускай оформление.")
        return

    workspace = await repo.get_or_create_workspace(chat.id, chat.title, settings.default_timezone)
    topic_ids, topics_changed = await ensure_workspace_topics(
        session,
        bot,
        chat_id=chat.id,
        title=chat.title,
        timezone_name=workspace.timezone or settings.default_timezone,
    )
    bindings = await repo.list_topic_bindings(workspace.id)
    changed = topics_changed

    today_binding = bindings[TopicKey.TODAY]
    changed = (
        await _ensure_topic_message(
            bot=bot,
            repo=repo,
            workspace_id=workspace.id,
            chat_id=chat.id,
            thread_id=topic_ids[TopicKey.TODAY],
            topic_key=TopicKey.TODAY,
            current_message_id=today_binding.control_message_id,
            text=TODAY_CONTROL_TEXT,
            reply_markup=today_control_keyboard(),
            is_control=True,
            pin_after_send=True,
        )
        or changed
    )

    materials_binding = bindings[TopicKey.MATERIALS]
    changed = (
        await _ensure_topic_message(
            bot=bot,
            repo=repo,
            workspace_id=workspace.id,
            chat_id=chat.id,
            thread_id=topic_ids[TopicKey.MATERIALS],
            topic_key=TopicKey.MATERIALS,
            current_message_id=materials_binding.intro_message_id,
            text=MATERIALS_INTRO_TEXT,
        )
        or changed
    )
    progress_binding = bindings[TopicKey.PROGRESS]
    changed = (
        await _ensure_topic_message(
            bot=bot,
            repo=repo,
            workspace_id=workspace.id,
            chat_id=chat.id,
            thread_id=topic_ids[TopicKey.PROGRESS],
            topic_key=TopicKey.PROGRESS,
            current_message_id=progress_binding.intro_message_id,
            text=PROGRESS_INTRO_TEXT,
        )
        or changed
    )

    with suppress(TelegramBadRequest):
        await callback.message.edit_text(SETUP_REPAIRED_TEXT if changed else SETUP_READY_TEXT)
    await repo.set_setup_message_id(workspace.id, callback.message.message_id)
    await callback.answer()
