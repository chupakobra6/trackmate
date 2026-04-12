import asyncio

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties

from trackmate.adapters.telegram.handlers.materials import router as materials_router
from trackmate.adapters.telegram.handlers.progress import router as progress_router
from trackmate.adapters.telegram.handlers.setup import router as setup_router
from trackmate.adapters.telegram.handlers.today import router as today_router
from trackmate.adapters.telegram.middleware import DbSessionMiddleware
from trackmate.config import get_settings
from trackmate.db.session import create_session_factory
from trackmate.logging import configure_logging


async def main() -> None:
    settings = get_settings()
    configure_logging(settings.log_level)
    session_factory = create_session_factory(settings)
    bot = Bot(token=settings.bot_token, default=DefaultBotProperties(parse_mode="HTML"))
    dispatcher = Dispatcher()
    dispatcher.update.middleware(DbSessionMiddleware(session_factory))
    dispatcher["settings"] = settings
    dispatcher.include_router(setup_router)
    dispatcher.include_router(progress_router)
    dispatcher.include_router(today_router)
    dispatcher.include_router(materials_router)
    await dispatcher.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())
