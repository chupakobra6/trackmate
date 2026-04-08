import asyncio

from trackmate.bootstrap.app import create_bot, create_dispatcher
from trackmate.config import get_settings
from trackmate.db.session import create_session_factory
from trackmate.logging import configure_logging


async def main() -> None:
    settings = get_settings()
    configure_logging(settings.log_level)
    session_factory = create_session_factory(settings)
    bot = create_bot(settings)
    dispatcher = create_dispatcher(settings=settings, session_factory=session_factory)
    await dispatcher.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())
