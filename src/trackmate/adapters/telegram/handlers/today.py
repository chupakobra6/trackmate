from __future__ import annotations

from contextlib import suppress

from aiogram import F, Router
from aiogram.dispatcher.event.bases import UNHANDLED
from aiogram.exceptions import TelegramBadRequest
from aiogram.types import CallbackQuery, Message
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import (
    PendingInputRepository,
    TodayRepository,
    WorkspaceRepository,
)
from trackmate.adapters.telegram.formatters import format_daily_task_card
from trackmate.adapters.telegram.handlers.helpers import display_name
from trackmate.adapters.telegram.keyboards import daily_task_keyboard, daily_task_status_keyboard
from trackmate.adapters.telegram.message_ops import (
    delete_message_safe,
    edit_message_like_safe,
    reply_message_logged,
    send_message_logged,
)
from trackmate.adapters.telegram.rich_text import message_input_html, message_input_text
from trackmate.application.today import create_daily_task, local_task_date, submit_daily_task_report
from trackmate.db.models import DailyTaskAlert
from trackmate.domain.enums import DailyTaskStatus, PendingInputKind

router = Router(name="today")


def _content_type_label(message: Message) -> str:
    return message_input_text(message) or "Сообщение"


def _pending_input_text(message: Message) -> str | None:
    return message_input_text(message)


def _pending_input_html(message: Message) -> str | None:
    return message_input_html(message)


def _report_confirmation_text() -> str:
    return "✅ <b>Отчет сохранен.</b>"


def _today_task_conflict_text(*, same_day: bool) -> str:
    if same_day:
        return "Задача на сегодня уже зафиксирована."
    return "Сначала закрой предыдущую задачу."


def _today_task_conflict_notice(*, today_task_exists: bool, open_task_exists: bool) -> str | None:
    if today_task_exists:
        return _today_task_conflict_text(same_day=True)
    if open_task_exists:
        return _today_task_conflict_text(same_day=False)
    return None


def _report_rejected_text(task_status: DailyTaskStatus | None) -> str:
    if task_status in {DailyTaskStatus.DONE, DailyTaskStatus.PARTIAL, DailyTaskStatus.FAILED}:
        return "Отчет не принят: задача уже закрыта."
    return "Отчет не принят."


@router.callback_query(F.data == "today:add")
async def add_today_task_callback(
    callback: CallbackQuery,
    session: AsyncSession,
) -> None:
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    today_repo = TodayRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(callback.message.chat.id)
    if workspace is None:
        await callback.answer()
        return
    participant = await workspace_repo.register_participant(
        workspace.id,
        callback.from_user.id,
        callback.from_user.username,
        display_name(callback.from_user),
    )
    today_date = local_task_date(workspace.timezone)
    today_task = await today_repo.get_task_for_date(workspace.id, participant.id, today_date)
    open_task = await today_repo.get_open_task(workspace.id, participant.id)
    conflict_notice = _today_task_conflict_notice(
        today_task_exists=today_task is not None,
        open_task_exists=open_task is not None,
    )
    if conflict_notice is not None:
        await callback.answer(text=conflict_notice)
        return
    existing_pending = await pending_repo.get(workspace.id, callback.from_user.id)
    if existing_pending and existing_pending.kind == PendingInputKind.DAILY_TASK_TEXT.value:
        await callback.answer(text="Я уже жду формулировку задачи.")
        return
    if existing_pending and existing_pending.kind == PendingInputKind.DAILY_TASK_REPORT.value:
        await callback.answer(text="Сначала закончи текущий отчет.")
        return
    prompt_message = await reply_message_logged(
        message=callback.message,
        text="✍️ <b>Напиши одну главную задачу дня одним сообщением. Можно текстом, голосовым или медиа.</b>",
    )
    await pending_repo.upsert(
        workspace.id,
        callback.from_user.id,
        PendingInputKind.DAILY_TASK_TEXT.value,
        {
            "thread_id": callback.message.message_thread_id,
            "prompt_message_id": prompt_message.message_id,
        },
    )
    await callback.answer()


@router.message()
async def today_pending_input_handler(
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
    if pending is None:
        return UNHANDLED
    if pending.kind not in {
        PendingInputKind.DAILY_TASK_TEXT.value,
        PendingInputKind.DAILY_TASK_REPORT.value,
    }:
        return UNHANDLED
    if pending.kind == PendingInputKind.DAILY_TASK_TEXT.value:
        task_html = _pending_input_html(message)
        await delete_message_safe(
            bot=message.bot,
            chat_id=message.chat.id,
            message_id=pending.payload.get("prompt_message_id"),
        )
        placeholder = await reply_message_logged(
            message=message,
            text="⏳ <b>Собираю карточку задачи...</b>",
        )
        created, task_id = await create_daily_task(
            session,
            workspace_id=workspace.id,
            timezone_name=workspace.timezone,
            user_id=message.from_user.id,
            username=message.from_user.username,
            display_name=display_name(message.from_user),
            task_html=task_html or "Сообщение",
            today_card_message_id=placeholder.message_id,
        )
        if not created:
            task = await TodayRepository(session).get_task(task_id) if task_id is not None else None
            text = (
                "⚠️ <b>Задача на сегодня уже зафиксирована.</b>"
                if task is not None and task.task_date == local_task_date(workspace.timezone)
                else "⚠️ <b>Сначала закрой предыдущую задачу.</b>"
            )
            await edit_message_like_safe(message=message, message_id=placeholder.message_id, text=text)
        else:
            task = await TodayRepository(session).get_task(task_id)
            await edit_message_like_safe(
                message=message,
                message_id=placeholder.message_id,
                text=format_daily_task_card(task, display_name(message.from_user), message.from_user.username),
                reply_markup=daily_task_keyboard(task.id),
            )
        await pending_repo.clear(workspace.id, message.from_user.id)
        return

    if pending.kind == PendingInputKind.DAILY_TASK_REPORT.value:
        report_html = _pending_input_html(message)
        prompt_message_id = pending.payload.get("prompt_message_id")
        task_id = int(pending.payload["task_id"])
        status = DailyTaskStatus(pending.payload["status"])
        submitted = await submit_daily_task_report(
            session,
            task_id=task_id,
            owner_user_id=message.from_user.id,
            status=status,
            report_html=report_html or "Сообщение",
            display_name=display_name(message.from_user),
        )
        if not submitted:
            task = await TodayRepository(session).get_task(task_id)
            await pending_repo.clear(workspace.id, message.from_user.id)
            rejection_text = _report_rejected_text(task.status if task is not None else None)
            edited = await edit_message_like_safe(
                message=message,
                message_id=prompt_message_id,
                text=rejection_text,
            )
            if not edited:
                await send_message_logged(
                    bot=message.bot,
                    chat_id=message.chat.id,
                    message_thread_id=message.message_thread_id,
                    text=rejection_text,
                )
            return
        task = await TodayRepository(session).get_task(task_id)
        if task:
            await edit_message_like_safe(
                message=message,
                message_id=task.today_card_message_id,
                text=format_daily_task_card(task, display_name(message.from_user), message.from_user.username),
            )
        await pending_repo.clear(workspace.id, message.from_user.id)
        edited = await edit_message_like_safe(
            message=message,
            message_id=prompt_message_id,
            text=_report_confirmation_text(),
        )
        if not edited:
            await send_message_logged(
                bot=message.bot,
                chat_id=message.chat.id,
                message_thread_id=message.message_thread_id,
                text=_report_confirmation_text(),
            )


@router.callback_query(F.data.startswith("task:report:"))
async def open_report_flow(callback: CallbackQuery, session: AsyncSession) -> None:
    _, _, raw_task_id = callback.data.split(":")
    task_id = int(raw_task_id)
    repo = TodayRepository(session)
    task = await repo.get_task(task_id)
    if task is None:
        await callback.answer(text="Задача не найдена.")
        return
    if callback.from_user.id != task.owner_user_id:
        await callback.answer(text="Отчитаться может только автор задачи.")
        return
    if task.status not in {DailyTaskStatus.ACTIVE, DailyTaskStatus.AWAITING_REPORT}:
        await callback.answer(text="Эта задача уже закрыта.")
        return
    existing_pending = await PendingInputRepository(session).get(task.workspace_group_id, callback.from_user.id)
    if existing_pending and existing_pending.kind == PendingInputKind.DAILY_TASK_REPORT.value:
        await callback.answer(text="Я уже жду короткий результат.")
        return
    await reply_message_logged(
        message=callback.message,
        text="🧾 <b>Выбери итог дня.</b>",
        reply_markup=daily_task_status_keyboard(task_id),
    )
    await callback.answer()


@router.callback_query(F.data.startswith("task:status:"))
async def choose_report_status(callback: CallbackQuery, session: AsyncSession) -> None:
    _, _, raw_task_id, raw_status = callback.data.split(":")
    task_id = int(raw_task_id)
    workspace_repo = WorkspaceRepository(session)
    pending_repo = PendingInputRepository(session)
    today_repo = TodayRepository(session)
    workspace = await workspace_repo.get_workspace_by_chat_id(callback.message.chat.id)
    if workspace is None:
        await callback.answer(text="Не получилось найти настройки группы.")
        return
    task = await today_repo.get_task(task_id)
    if task is None:
        await callback.answer(text="Задача не найдена.")
        return
    if callback.from_user.id != task.owner_user_id:
        await callback.answer(text="Отчитаться может только автор задачи.")
        return
    if task.status not in {DailyTaskStatus.ACTIVE, DailyTaskStatus.AWAITING_REPORT}:
        await callback.answer(text="Эта задача уже закрыта.")
        return
    previous_pending = await pending_repo.get(workspace.id, callback.from_user.id)
    if previous_pending and previous_pending.kind == PendingInputKind.DAILY_TASK_REPORT.value:
        await delete_message_safe(
            bot=callback.message.bot,
            chat_id=callback.message.chat.id,
            message_id=previous_pending.payload.get("prompt_message_id"),
        )
    with suppress(TelegramBadRequest):
        await callback.message.edit_text(
            "✍️ <b>Теперь напиши короткий результат одним сообщением. Можно текстом, голосовым или медиа.</b>"
        )
    await pending_repo.upsert(
        workspace.id,
        callback.from_user.id,
        PendingInputKind.DAILY_TASK_REPORT.value,
        {
            "task_id": task_id,
            "status": raw_status,
            "prompt_message_id": callback.message.message_id,
        },
    )
    await callback.answer()


@router.callback_query(F.data.startswith("alert:ack:"))
async def acknowledge_alert(callback: CallbackQuery, session: AsyncSession) -> None:
    _, _, raw_alert_id = callback.data.split(":")
    alert_id = int(raw_alert_id)
    alert = await session.get(DailyTaskAlert, alert_id)
    if alert is not None:
        task = await TodayRepository(session).get_task(alert.daily_task_id)
        if task is not None and task.owner_user_id != callback.from_user.id:
            await callback.answer()
            return
    if alert is not None and alert.acknowledged_at is None:
        from datetime import UTC, datetime

        alert.acknowledged_at = datetime.now(UTC)
    await callback.answer()
