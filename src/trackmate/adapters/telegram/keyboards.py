from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup


def setup_keyboard() -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="🔄 Проверить снова", callback_data="setup:check")],
            [InlineKeyboardButton(text="✨ Оформить группу", callback_data="setup:start")],
        ]
    )


def today_control_keyboard() -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="➕ Добавить задачу", callback_data="today:add")],
        ]
    )


def material_progress_keyboard(batch_id: int) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [
                InlineKeyboardButton(text="👀 Прочитал", callback_data=f"material:read:{batch_id}"),
                InlineKeyboardButton(text="📝 Заметка", callback_data=f"material:note:{batch_id}"),
                InlineKeyboardButton(text="🚀 Внедрил", callback_data=f"material:applied:{batch_id}"),
            ]
        ]
    )


def daily_task_keyboard(task_id: int) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="🏁 Отчитаться", callback_data=f"task:report:{task_id}")],
        ]
    )


def daily_task_status_keyboard(task_id: int) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [
                InlineKeyboardButton(text="✅ Выполнена", callback_data=f"task:status:{task_id}:done"),
                InlineKeyboardButton(text="🔸 Выполнена частично", callback_data=f"task:status:{task_id}:partial"),
                InlineKeyboardButton(text="❌ Не выполнена", callback_data=f"task:status:{task_id}:failed"),
            ]
        ]
    )


def alert_keyboard(task_id: int, alert_id: int) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="🏁 Отчитаться", callback_data=f"task:report:{task_id}")],
            [InlineKeyboardButton(text="👀 Понял", callback_data=f"alert:ack:{alert_id}")],
        ]
    )
