from trackmate.adapters.telegram.keyboards import daily_task_status_keyboard


def test_daily_task_status_keyboard_uses_consistent_labels() -> None:
    keyboard = daily_task_status_keyboard(42)

    assert keyboard.inline_keyboard[0][0].text == "✅ Выполнена"
    assert keyboard.inline_keyboard[0][1].text == "🔸 Выполнена частично"
    assert keyboard.inline_keyboard[0][2].text == "❌ Не выполнена"
