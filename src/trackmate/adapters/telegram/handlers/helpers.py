from __future__ import annotations

from aiogram.types import Chat, User
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import WorkspaceRepository
from trackmate.db.models import WorkspaceGroup


def display_name(user: User) -> str:
    full = " ".join(part for part in [user.first_name, user.last_name] if part)
    return full or user.username or str(user.id)


async def resolve_workspace(session: AsyncSession, chat: Chat) -> WorkspaceGroup:
    repo = WorkspaceRepository(session)
    workspace = await repo.get_or_create_workspace(chat.id, chat.title, "UTC")
    return workspace
