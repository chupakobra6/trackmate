from __future__ import annotations

import asyncio
import hashlib
from datetime import UTC, datetime

import structlog
from sqlalchemy import text as sql_text
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import (
    MaterialRepository,
    ProgressRepository,
    WorkspaceRepository,
)
from trackmate.domain.enums import MaterialHighestState, ProgressEventType
from trackmate.domain.rules import derive_material_highest_state

logger = structlog.get_logger(__name__)
_MATERIAL_BATCH_LOCKS: dict[tuple[int, int, str | None], asyncio.Lock] = {}


def _material_batch_db_lock_key(*, materials_thread_id: int, upload_session_key: str) -> int:
    digest = hashlib.blake2b(
        f"{materials_thread_id}:{upload_session_key}".encode(),
        digest_size=4,
    ).digest()
    return int.from_bytes(digest, byteorder="big", signed=True)


def _fallback_upload_session_key(
    *,
    workspace_id: int,
    materials_thread_id: int,
    now_utc: datetime,
    timeout_seconds: int,
) -> str:
    bucket = int(now_utc.timestamp()) // timeout_seconds
    return f"{workspace_id}:{materials_thread_id}:{bucket}"


def _material_batch_lock(
    *,
    workspace_id: int,
    materials_thread_id: int,
    media_group_id: str | None,
) -> asyncio.Lock:
    key = (workspace_id, materials_thread_id, media_group_id)
    lock = _MATERIAL_BATCH_LOCKS.get(key)
    if lock is None:
        lock = asyncio.Lock()
        _MATERIAL_BATCH_LOCKS[key] = lock
    return lock


async def _acquire_material_batch_db_lock(
    session: AsyncSession,
    *,
    workspace_id: int,
    materials_thread_id: int,
    upload_session_key: str,
) -> None:
    bind = session.get_bind()
    if bind.dialect.name != "postgresql":
        return
    await session.execute(
        sql_text("SELECT pg_advisory_xact_lock(:workspace_id, :lock_key)"),
        {
            "workspace_id": workspace_id,
            "lock_key": _material_batch_db_lock_key(
                materials_thread_id=materials_thread_id,
                upload_session_key=upload_session_key,
            ),
        },
    )


def _material_message_link(
    *,
    chat_id: int,
    message_id: int | None,
    thread_id: int | None,
) -> str | None:
    if message_id is None:
        return None
    chat_id_text = str(chat_id)
    if not chat_id_text.startswith("-100"):
        return None
    link = f"https://t.me/c/{chat_id_text[4:]}/{message_id}"
    if thread_id is not None:
        return f"{link}?thread={thread_id}"
    return link


async def register_material_message(
    session: AsyncSession,
    *,
    workspace_id: int,
    materials_thread_id: int,
    media_group_id: str | None,
    source_message_id: int,
    source_chat_id: int,
    source_thread_id: int | None,
    content_type: str,
    forwarded_from_chat_id: int | None,
    forwarded_from_message_id: int | None,
    batch_timeout_seconds: int,
) -> int:
    repo = MaterialRepository(session)
    lock = _material_batch_lock(
        workspace_id=workspace_id,
        materials_thread_id=materials_thread_id,
        media_group_id=media_group_id,
    )
    now_utc = datetime.now(UTC)
    upload_session_key = (
        media_group_id
        or _fallback_upload_session_key(
            workspace_id=workspace_id,
            materials_thread_id=materials_thread_id,
            now_utc=now_utc,
            timeout_seconds=batch_timeout_seconds,
        )
    )
    async with lock:
        await _acquire_material_batch_db_lock(
            session,
            workspace_id=workspace_id,
            materials_thread_id=materials_thread_id,
            upload_session_key=upload_session_key,
        )
        batch = await repo.get_open_batch(
            workspace_id=workspace_id,
            materials_thread_id=materials_thread_id,
            media_group_id=media_group_id,
            timeout_seconds=batch_timeout_seconds,
            now_utc=now_utc,
        )
        created_new_batch = batch is None
        if batch is None:
            batch = await repo.create_batch(
                workspace_id=workspace_id,
                materials_thread_id=materials_thread_id,
                media_group_id=media_group_id,
            )
        await repo.append_item(
            batch=batch,
            source_message_id=source_message_id,
            source_chat_id=source_chat_id,
            source_thread_id=source_thread_id,
            content_type=content_type,
            forwarded_from_chat_id=forwarded_from_chat_id,
            forwarded_from_message_id=forwarded_from_message_id,
        )
    logger.info(
        "telegram.material_batch_item_registered",
        batch_id=batch.id,
        created_new_batch=created_new_batch,
        workspace_id=workspace_id,
        thread_id=materials_thread_id,
        source_message_id=source_message_id,
        media_group_id=media_group_id,
        batch_size=batch.batch_size,
    )
    return batch.id


async def mark_material_read(
    session: AsyncSession,
    *,
    workspace_id: int,
    user_id: int,
    username: str | None,
    display_name: str,
    batch_id: int,
) -> tuple[MaterialHighestState, bool]:
    workspace_repo = WorkspaceRepository(session)
    materials_repo = MaterialRepository(session)
    participant = await workspace_repo.register_participant(workspace_id, user_id, username, display_name)
    progress = await materials_repo.get_progress(batch_id, participant.id)
    if progress is None:
        progress = await materials_repo.create_progress(batch_id, participant.id)
    was_already_read = progress.read_at is not None
    if progress.read_at is None:
        progress.read_at = datetime.now(UTC)
    progress.highest_state = derive_material_highest_state(
        read_at=progress.read_at,
        note_progress_event_id=progress.note_progress_event_id,
        applied_progress_event_id=progress.applied_progress_event_id,
    )
    await session.flush()
    return progress.highest_state, not was_already_read


async def submit_material_artifact(
    session: AsyncSession,
    *,
    workspace_id: int,
    user_id: int,
    username: str | None,
    display_name: str,
    batch_id: int,
    artifact_html: str,
    artifact_content_kind: str = "text",
    is_applied: bool,
) -> bool:
    workspace_repo = WorkspaceRepository(session)
    materials_repo = MaterialRepository(session)
    progress_repo = ProgressRepository(session)

    participant = await workspace_repo.register_participant(workspace_id, user_id, username, display_name)
    batch = await materials_repo.get_batch(batch_id)
    if batch is None:
        return False
    workspace = await workspace_repo.get_workspace_by_id(workspace_id)
    progress = await materials_repo.get_progress(batch_id, participant.id)
    if progress is None:
        progress = await materials_repo.create_progress(batch_id, participant.id)
    if progress.read_at is None:
        progress.read_at = datetime.now(UTC)
    if is_applied and progress.applied_progress_event_id is not None:
        return False
    if not is_applied and progress.note_progress_event_id is not None:
        return False

    event_type = ProgressEventType.MATERIAL_APPLIED if is_applied else ProgressEventType.MATERIAL_NOTE_ADDED
    event = await progress_repo.create_event(
        workspace_group_id=workspace_id,
        participant_id=participant.id,
        material_batch_id=batch_id,
        event_type=event_type,
        payload={
            "html": artifact_html,
            "content_kind": artifact_content_kind,
            "display_name": participant.display_name,
            "username": participant.username,
            "material_link": _material_message_link(
                chat_id=workspace.chat_id,
                message_id=batch.tracking_card_message_id or batch.source_anchor_message_id,
                thread_id=batch.materials_thread_id,
            )
            if workspace is not None
            else None,
        },
    )
    if is_applied:
        progress.applied_progress_event_id = event.id
    else:
        progress.note_progress_event_id = event.id
    progress.highest_state = derive_material_highest_state(
        read_at=progress.read_at,
        note_progress_event_id=progress.note_progress_event_id,
        applied_progress_event_id=progress.applied_progress_event_id,
    )
    await session.flush()
    return True
