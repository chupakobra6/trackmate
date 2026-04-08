from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from trackmate.adapters.telegram.handlers.materials import router as materials_router
from trackmate.adapters.telegram.handlers.setup import router as setup_router
from trackmate.adapters.telegram.handlers.today import router as today_router
from trackmate.adapters.telegram.middleware import DbSessionMiddleware
from trackmate.config import Settings


def create_bot(settings: Settings) -> Bot:
    return Bot(token=settings.bot_token, default=DefaultBotProperties(parse_mode="HTML"))


def create_dispatcher(
    *,
    settings: Settings,
    session_factory: async_sessionmaker[AsyncSession],
) -> Dispatcher:
    dispatcher = Dispatcher()
    dispatcher.update.middleware(DbSessionMiddleware(session_factory))
    dispatcher["settings"] = settings
    dispatcher.include_router(setup_router)
    dispatcher.include_router(today_router)
    dispatcher.include_router(materials_router)
    return dispatcher
