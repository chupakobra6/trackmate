from __future__ import annotations

from aiogram.types import Message
from aiogram.utils.text_decorations import html_decoration


def message_text_and_html(message: Message) -> tuple[str | None, str | None]:
    if message.text:
        return message.text, message.html_text
    if message.caption:
        return message.caption, html_decoration.unparse(
            text=message.caption,
            entities=message.caption_entities,
        )
    return None, None
