from types import SimpleNamespace

from trackmate.adapters.telegram.rich_text import message_input_html, message_input_text


def test_message_input_text_supports_contact_and_location() -> None:
    contact_message = SimpleNamespace(
        text=None,
        html_text=None,
        caption=None,
        caption_entities=None,
        content_type="contact",
        contact=SimpleNamespace(first_name="Igor", phone_number="+79990000000"),
    )
    location_message = SimpleNamespace(
        text=None,
        html_text=None,
        caption=None,
        caption_entities=None,
        content_type="location",
        location=SimpleNamespace(latitude=55.7558, longitude=37.6173),
    )

    assert message_input_text(contact_message) == "Контакт: Igor (+79990000000)"
    assert message_input_text(location_message) == "Локация: 55.7558, 37.6173"


def test_message_input_html_falls_back_to_generic_content_type_label() -> None:
    message = SimpleNamespace(
        text=None,
        html_text=None,
        caption=None,
        caption_entities=None,
        content_type="forum_topic_created",
    )

    assert message_input_text(message) == "Сообщение типа: forum_topic_created"
    assert message_input_html(message) == "Сообщение типа: forum_topic_created"
