import asyncio

from sqlalchemy import text

from trackmate.config import get_settings
from trackmate.db.session import create_session_factory


async def main() -> None:
    session_factory = create_session_factory(get_settings())
    async with session_factory() as session:
        await session.execute(text("select 1"))


if __name__ == "__main__":
    asyncio.run(main())
