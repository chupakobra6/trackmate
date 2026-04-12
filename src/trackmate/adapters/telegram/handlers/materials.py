from __future__ import annotations

import structlog
from aiogram import Bot, F, Router
from aiogram.dispatcher.event.bases import UNHANDLED
from aiogram.types import CallbackQuery, Message
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import (
    MaterialRepository,
    PendingInputRepository,
    WorkspaceRepository,
)
from trackmate.adapters.telegram.formatters import format_material_card
from trackmate.adapters.telegram.handlers.helpers import display_name
from trackmate.adapters.telegram.keyboards import material_progress_keyboard
from trackmate.adapters.telegram.message_ops import (
    edit_message_like_safe,
    edit_message_text_safe,
    reply_message_logged,
    send_message_logged,
)
from trackmate.adapters.telegram.rich_text import (
    message_input_html,
    message_input_kind,
    message_input_text,
)
from trackmate.application.materials import (
    mark_material_read,
    register_material_message,
    submit_material_artifact,
)
from trackmate.config import Settings
from trackmate.domain.enums import PendingInputKind, TopicKey

router = Router(name="materials")
logger = structlog.get_logger(__name__)


def _artifact_feedback_text(*, is_applied: bool, submitted: bool) -> str:
    if submitted:
        return "🚀 <b>Внедрение отмечено.</b>" if is_applied else "📝 <b>Заметка сохранена.</b>"
    return "Нельзя добавить второе внедрение." if is_applied else "Нельзя добавить вторую заметку."


def _read_feedback_text(*, created: bool) -> str:
    return "" if created else "Материал уже прочитан."


async def _refresh_material_card(
    *,
    bot: Bot,
    chat_id: int,
    batch_id: int,
    materials_repo: MaterialRepository,
    notice: str | None = None,
) -> None:
    batch = await materials_repo.get_batch(batch_id)
    if batch is None or batch.tracking_card_message_id is None:
        return
    progresses = await materials_repo.list_progresses(batch_id)
    await edit_message_text_safe(
        bot=bot,
        chat_id=chat_id,
        message_id=batch.tracking_card_message_id,
        text=format_material_card(batch, progresses, notice=notice),
        reply_markup=material_progress_keyboard(batch_id),
    )


def _extract_forward_metadata(message: Message) -> tuple[int | None, int | None]:
    origin = getattr(message, "forward_origin", None)
    if origin is None:
        return None, None
    chat = getattr(origin, "chat", None)
    message_id = getattr(origin, "message_id", None)
    return getattr(chat, "id", None), message_id


def _looks_like_new_material(
    *,
    message: Message,
    is_materials_topic: bool,
    forwarded_from_chat_id: int | None,
) -> bool:
    return is_materials_topic and (
        forwarded_from_chat_id is not None
        or message.media_group_id is not None
    )


@router.message()
async def material_or_pending_input_handler(
    message: Message,
    session: AsyncSession,
    settings: Settings,
) -> None:
    if message.chat.type not in {"supergroup", "group"}:
        return UNHANDLED
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(message.chat.id)
    if workspace is None:
        return UNHANDLED
    bindings = await workspace_repo.list_topic_bindings(workspace.id)
    materials_binding = bindings.get(TopicKey.MATERIALS)
    is_materials_topic = materials_binding is not None and message.message_thread_id == materials_binding.thread_id
    forwarded_from_chat_id, forwarded_from_message_id = _extract_forward_metadata(message)
    pending = await pending_repo.get(workspace.id, message.from_user.id)
    if pending is not None and pending.kind not in {
        PendingInputKind.MATERIAL_NOTE.value,
        PendingInputKind.MATERIAL_APPLIED.value,
    }:
        return UNHANDLED
    if pending and pending.kind in {PendingInputKind.MATERIAL_NOTE.value, PendingInputKind.MATERIAL_APPLIED.value}:
        if _looks_like_new_material(
            message=message,
            is_materials_topic=is_materials_topic,
            forwarded_from_chat_id=forwarded_from_chat_id,
        ):
            await pending_repo.clear(workspace.id, message.from_user.id)
        else:
            prompt_message_id = pending.payload.get("prompt_message_id")
            is_applied = pending.kind == PendingInputKind.MATERIAL_APPLIED.value
            artifact_text = message_input_text(message)
            artifact_html = message_input_html(message)
            artifact_content_kind = message_input_kind(message)
            if artifact_text is None and artifact_html is None:
                return UNHANDLED
            logger.info(
                "telegram.material_artifact_received",
                chat_id=message.chat.id,
                thread_id=message.message_thread_id,
                user_id=message.from_user.id,
                username=message.from_user.username,
                batch_id=int(pending.payload["batch_id"]),
                is_applied=is_applied,
                text=artifact_text or "",
            )
            submitted = await submit_material_artifact(
                session,
                workspace_id=workspace.id,
                user_id=message.from_user.id,
                username=message.from_user.username,
                display_name=display_name(message.from_user),
                batch_id=int(pending.payload["batch_id"]),
                artifact_html=artifact_html or artifact_text or "",
                artifact_content_kind=artifact_content_kind,
                is_applied=is_applied,
            )
            await pending_repo.clear(workspace.id, message.from_user.id)
            if submitted:
                await _refresh_material_card(
                    bot=message.bot,
                    chat_id=message.chat.id,
                    batch_id=int(pending.payload["batch_id"]),
                    materials_repo=MaterialRepository(session),
                )
            edited = await edit_message_like_safe(
                message=message,
                message_id=prompt_message_id,
                text=_artifact_feedback_text(is_applied=is_applied, submitted=submitted),
            )
            if not edited:
                await send_message_logged(
                    bot=message.bot,
                    chat_id=message.chat.id,
                    message_thread_id=message.message_thread_id,
                    text=_artifact_feedback_text(is_applied=is_applied, submitted=submitted),
                )
            return

    if materials_binding is None or message.message_thread_id != materials_binding.thread_id:
        return UNHANDLED
    batch_id = await register_material_message(
        session,
        workspace_id=workspace.id,
        materials_thread_id=materials_binding.thread_id,
        media_group_id=message.media_group_id,
        source_message_id=message.message_id,
        source_chat_id=message.chat.id,
        source_thread_id=message.message_thread_id,
        content_type=message.content_type,
        forwarded_from_chat_id=forwarded_from_chat_id,
        forwarded_from_message_id=forwarded_from_message_id,
        batch_timeout_seconds=settings.material_batch_timeout_seconds,
    )
    logger.info(
        "telegram.material_message_registered",
        chat_id=message.chat.id,
        thread_id=message.message_thread_id,
        message_id=message.message_id,
        user_id=message.from_user.id,
        username=message.from_user.username,
        batch_id=batch_id,
        media_group_id=message.media_group_id,
        content_type=message.content_type,
        text=message.text,
        caption=message.caption,
    )


@router.callback_query(F.data.startswith("material:"))
async def material_progress_callback(
    callback: CallbackQuery,
    bot: Bot,
    session: AsyncSession,
) -> None:
    _, action, raw_batch_id = callback.data.split(":")
    batch_id = int(raw_batch_id)
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    materials_repo = MaterialRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(callback.message.chat.id)
    if workspace is None:
        await callback.answer()
        return

    if action == "read":
        _, created = await mark_material_read(
            session,
            workspace_id=workspace.id,
            user_id=callback.from_user.id,
            username=callback.from_user.username,
            display_name=display_name(callback.from_user),
            batch_id=batch_id,
        )
        await _refresh_material_card(
            bot=bot,
            chat_id=callback.message.chat.id,
            batch_id=batch_id,
            materials_repo=materials_repo,
        )
        if created:
            await callback.answer()
        else:
            await callback.answer(text=_read_feedback_text(created=created))
        return

    pending_kind = PendingInputKind.MATERIAL_NOTE if action == "note" else PendingInputKind.MATERIAL_APPLIED
    existing_pending = await pending_repo.get(workspace.id, callback.from_user.id)
    participant = await workspace_repo.register_participant(
        workspace.id,
        callback.from_user.id,
        callback.from_user.username,
        display_name(callback.from_user),
    )
    progress = await materials_repo.get_progress(batch_id, participant.id)
    if action == "note" and progress and progress.note_progress_event_id is not None:
        await _refresh_material_card(
            bot=bot,
            chat_id=callback.message.chat.id,
            batch_id=batch_id,
            materials_repo=materials_repo,
        )
        await callback.answer(text="Нельзя добавить вторую заметку.")
        return
    if action == "applied" and progress and progress.applied_progress_event_id is not None:
        await _refresh_material_card(
            bot=bot,
            chat_id=callback.message.chat.id,
            batch_id=batch_id,
            materials_repo=materials_repo,
        )
        await callback.answer(text="Нельзя добавить второе внедрение.")
        return
    await pending_repo.upsert(
        workspace.id,
        callback.from_user.id,
        pending_kind.value,
        {
            "batch_id": batch_id,
            "prompt_message_id": (
                existing_pending.payload.get("prompt_message_id")
                if existing_pending and existing_pending.kind in {
                    PendingInputKind.MATERIAL_NOTE.value,
                    PendingInputKind.MATERIAL_APPLIED.value,
                }
                else None
            ),
        },
    )
    prompt = (
        "📝 <b>Добавь заметку одним сообщением. Можно текстом, голосовым или любым медиа.</b>"
        if action == "note"
        else "🚀 <b>Опиши одним сообщением, что удалось внедрить. Можно текстом, голосовым или любым медиа.</b>"
    )
    prompt_message_id = (
        existing_pending.payload.get("prompt_message_id")
        if existing_pending and existing_pending.kind in {
            PendingInputKind.MATERIAL_NOTE.value,
            PendingInputKind.MATERIAL_APPLIED.value,
        }
        else None
    )
    if prompt_message_id is not None:
        edited = await edit_message_text_safe(
            bot=bot,
            chat_id=callback.message.chat.id,
            message_id=prompt_message_id,
            text=prompt,
        )
        if edited:
            await pending_repo.upsert(
                workspace.id,
                callback.from_user.id,
                pending_kind.value,
                {"batch_id": batch_id, "prompt_message_id": prompt_message_id},
            )
            await callback.answer()
            return
    prompt_message = await reply_message_logged(message=callback.message, text=prompt)
    await pending_repo.upsert(
        workspace.id,
        callback.from_user.id,
        pending_kind.value,
        {"batch_id": batch_id, "prompt_message_id": prompt_message.message_id},
    )
    await callback.answer()
