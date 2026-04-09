from __future__ import annotations

from html import escape

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


def message_input_kind(message: Message) -> str:
    plain_text, html_text = message_text_and_html(message)
    if plain_text or html_text:
        return "text"
    return "non_text"


def _message_content_type_label(message: Message) -> str:
    content_type = getattr(message, "content_type", None)
    if content_type == "voice":
        return "Голосовое сообщение"
    if content_type == "video_note":
        return "Видео-кружок"
    if content_type == "video":
        return "Видео"
    if content_type == "photo":
        return "Фото"
    if content_type == "audio":
        audio = getattr(message, "audio", None)
        title = getattr(audio, "title", None)
        performer = getattr(audio, "performer", None)
        if title and performer:
            return f"Аудио: {performer} - {title}"
        if title:
            return f"Аудио: {title}"
        return "Аудио"
    if content_type == "document":
        document = getattr(message, "document", None)
        file_name = getattr(document, "file_name", None)
        return f"Документ: {file_name}" if file_name else "Документ"
    if content_type == "animation":
        return "Анимация"
    if content_type == "sticker":
        sticker = getattr(message, "sticker", None)
        emoji = getattr(sticker, "emoji", None)
        return f"Стикер {emoji}" if emoji else "Стикер"
    if content_type == "contact":
        contact = getattr(message, "contact", None)
        first_name = getattr(contact, "first_name", None)
        phone_number = getattr(contact, "phone_number", None)
        if first_name and phone_number:
            return f"Контакт: {first_name} ({phone_number})"
        if first_name:
            return f"Контакт: {first_name}"
        return "Контакт"
    if content_type == "location":
        location = getattr(message, "location", None)
        latitude = getattr(location, "latitude", None)
        longitude = getattr(location, "longitude", None)
        if latitude is not None and longitude is not None:
            return f"Локация: {latitude}, {longitude}"
        return "Локация"
    if content_type == "venue":
        venue = getattr(message, "venue", None)
        title = getattr(venue, "title", None)
        address = getattr(venue, "address", None)
        if title and address:
            return f"Место: {title}, {address}"
        if title:
            return f"Место: {title}"
        return "Место"
    if content_type == "poll":
        poll = getattr(message, "poll", None)
        question = getattr(poll, "question", None)
        return f"Опрос: {question}" if question else "Опрос"
    if content_type == "dice":
        dice = getattr(message, "dice", None)
        emoji = getattr(dice, "emoji", None)
        value = getattr(dice, "value", None)
        if emoji and value is not None:
            return f"Кубик {emoji}: {value}"
        return "Кубик"
    if content_type == "game":
        game = getattr(message, "game", None)
        title = getattr(game, "title", None)
        return f"Игра: {title}" if title else "Игра"
    if content_type == "invoice":
        invoice = getattr(message, "invoice", None)
        title = getattr(invoice, "title", None)
        return f"Счет: {title}" if title else "Счет"
    if content_type == "story":
        return "История"
    if content_type == "paid_media":
        return "Платный медиа-контент"
    if content_type:
        return f"Сообщение типа: {content_type}"
    return "Сообщение"


def message_input_text(message: Message) -> str | None:
    plain_text, _ = message_text_and_html(message)
    if plain_text:
        return plain_text
    return _message_content_type_label(message)


def message_input_html(message: Message) -> str | None:
    _, html_text = message_text_and_html(message)
    if html_text:
        return html_text
    fallback = message_input_text(message)
    return escape(fallback) if fallback else None
