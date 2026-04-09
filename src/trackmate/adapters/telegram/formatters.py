from __future__ import annotations

from html import escape

from trackmate.db.models import DailyTask, MaterialBatch, ProgressEvent
from trackmate.domain.enums import DailyTaskStatus, ProgressEventType


def _mark(value: bool) -> str:
    return "✅" if value else "•"


def _task_status_label(status: DailyTaskStatus) -> str:
    return {
        DailyTaskStatus.ACTIVE: "в процессе",
        DailyTaskStatus.AWAITING_REPORT: "ждет отчета",
        DailyTaskStatus.DONE: "выполнена",
        DailyTaskStatus.PARTIAL: "выполнена частично",
        DailyTaskStatus.FAILED: "не выполнена",
    }[status]

def _daily_task_closed_title(status: str, person: str) -> str:
    return {
        "done": f"✅ <b>{person} выполнил задачу дня</b>",
        "partial": f"🔸 <b>{person} частично выполнил задачу дня</b>",
        "failed": f"❌ <b>{person} не выполнил задачу дня</b>",
    }.get(status, f"✅ <b>{person} завершил задачу дня</b>")


def _person_label(username: str | None, display_name: str) -> str:
    if username:
        return f"@{escape(username)}"
    return escape(display_name)


def _participant_label(participant) -> str:
    if participant is None:
        return "без имени"
    if participant.username:
        return f"@{escape(participant.username)}"
    return escape(participant.display_name)


def _append_notice(lines: list[str], notice: str | None) -> list[str]:
    if notice:
        lines.extend(["", f"<blockquote>{escape(notice)}</blockquote>"])
    return lines


def _material_link(payload: dict, label: str) -> str:
    link = payload.get("material_link")
    if not link:
        return label
    return f'<a href="{escape(link)}">{label}</a>'


def format_setup_checklist(
    *,
    ready: bool,
    is_supergroup: bool,
    is_forum: bool,
    is_admin: bool,
    can_manage_topics: bool,
    can_read_messages: bool,
    notice: str | None = None,
) -> str:
    status = (
        "✅ Можно начинать: все условия выполнены."
        if ready
        else "До запуска нужно закрыть несколько пунктов."
    )
    checks = [
        f"{_mark(is_supergroup)} Группа переведена в супергруппу.",
        f"{_mark(is_forum)} Включены темы.",
        f"{_mark(is_admin)} Бот назначен администратором.",
        f"{_mark(can_manage_topics)} У бота есть право управлять темами.",
        f"{_mark(can_read_messages)} Бот видит сообщения участников.",
    ]
    return "\n".join(
        _append_notice(
            [
                "⚙️ <b>Подготовка пространства</b>",
                status,
                "",
                *checks,
                "",
                "Когда все будет готово, можно запускать оформление группы.",
            ],
            notice,
        )
    )


def format_material_card(batch: MaterialBatch, progresses: list, notice: str | None = None) -> str:
    lines = ["📚 <b>Материал</b>"]
    if batch.batch_size > 1:
        lines.append(f"<b>Сообщений в подборке:</b> {batch.batch_size}")

    events: list[str] = []
    for progress in progresses:
        person = _participant_label(progress.participant)
        if progress.read_at:
            events.append(f"<blockquote>{person} прочитал.</blockquote>")
        if progress.note_progress_event_id is not None:
            events.append(f"<blockquote>{person} добавил заметку.</blockquote>")
        if progress.applied_progress_event_id is not None:
            events.append(f"<blockquote>{person} внедрил.</blockquote>")

    if events:
        if len(lines) > 1:
            lines.append("")
        lines.extend(events)
    return "\n".join(_append_notice(lines, notice))


def format_today_control(notice: str | None = None) -> str:
    return "\n".join(
        _append_notice(
            [
                "🎯 <b>Сегодня</b>",
                "Здесь у каждого одна главная задача на день.",
                "Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.",
                "",
                "Как это работает:",
                "• ты формулируешь одну задачу на день;",
                "• я закрепляю ее в отдельной карточке;",
                "• вечером в этой же карточке можно оставить результат.",
            ],
            notice,
        )
    )


def format_daily_task_card(
    task: DailyTask,
    display_name: str,
    username: str | None = None,
    notice: str | None = None,
) -> str:
    person = _person_label(username, display_name)
    lines = [
        f"🎯 <b>Задача дня</b> {person}:",
        "",
        task.text,
        "",
        f"<b>Статус:</b> {_task_status_label(task.status)}",
    ]
    if task.report_text:
        lines.extend(
            [
                "",
                "<b>Результат:</b>",
                task.report_text,
            ]
        )
    return "\n".join(_append_notice(lines, notice))


def format_progress_event(event: ProgressEvent) -> str:
    event_type = event.event_type
    payload = event.payload or {}
    person = _person_label(payload.get("username"), payload.get("display_name", "Без имени"))
    if event_type is ProgressEventType.MATERIAL_NOTE_ADDED:
        material = _material_link(payload, "материалу")
        return "\n".join(
            [
                f"📝 <b>{person} добавил заметку к {material}</b>",
                "",
                "<b>Текст заметки:</b>",
                payload.get("html", ""),
            ]
        )
    if event_type is ProgressEventType.MATERIAL_APPLIED:
        material = _material_link(payload, "материалу")
        return "\n".join(
            [
                f"🚀 <b>{person} внедрил по {material}</b>",
                "",
                "<b>Что внедрил:</b>",
                payload.get("html", ""),
            ]
        )
    if event_type is ProgressEventType.DAILY_TASK_CLOSED:
        status = payload.get("status", "")
        title = _daily_task_closed_title(status, person)
        return "\n".join(
            [
                title,
                "",
                "<b>Что планировал:</b>",
                payload.get("task_html", ""),
                "",
                "<b>Результат:</b>",
                payload.get("report_html", ""),
            ]
        )
    if event_type is ProgressEventType.DAILY_TASK_AUTO_FAILED:
        return "\n".join(
            [
                f"⏰ <b>{person} не выполнил задачу дня вовремя</b>",
                "",
                "<b>Что планировал:</b>",
                payload.get("task_html", ""),
            ]
        )
    return "\n".join(
        [
            "🔔 Системное сообщение",
            str(payload),
        ]
    )
