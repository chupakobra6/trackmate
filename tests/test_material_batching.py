import asyncio
from datetime import UTC, datetime, timedelta

import pytest
from sqlalchemy import select
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from trackmate.adapters.persistence.repositories import MaterialRepository, WorkspaceRepository
from trackmate.application.materials import register_material_message
from trackmate.db.base import Base
from trackmate.db.models import MaterialBatch
from trackmate.domain.enums import MaterialBatchStatus, TopicKey
from trackmate.worker.jobs import seal_material_batches


@pytest.mark.asyncio
async def test_get_open_batch_reuses_recent_batch_even_with_different_sender(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(8008, "Group", "Europe/Moscow")

    repo = MaterialRepository(session)
    batch = await repo.create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
    )
    batch.last_message_at = datetime.now(UTC) - timedelta(seconds=2)
    await session.flush()

    reopened = await repo.get_open_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
        timeout_seconds=5,
        now_utc=datetime.now(UTC),
    )

    assert reopened is not None
    assert reopened.id == batch.id


@pytest.mark.asyncio
async def test_merge_batches_combines_parallel_open_batches(session) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(8009, "Group", "Europe/Moscow")

    repo = MaterialRepository(session)
    first = await repo.create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
    )
    second = await repo.create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
    )

    await repo.append_item(
        batch=first,
        source_message_id=1,
        source_chat_id=workspace.chat_id,
        source_thread_id=10,
        content_type="text",
        forwarded_from_chat_id=None,
        forwarded_from_message_id=None,
    )
    await repo.append_item(
        batch=second,
        source_message_id=2,
        source_chat_id=workspace.chat_id,
        source_thread_id=10,
        content_type="text",
        forwarded_from_chat_id=None,
        forwarded_from_message_id=None,
    )

    mergeable = await repo.list_mergeable_open_batches(first)
    await repo.merge_batches(first, mergeable[1:])

    merged = await repo.get_batch(first.id)

    assert merged is not None
    assert merged.batch_size == 2
    assert await repo.get_batch(second.id) is None


@pytest.mark.asyncio
async def test_register_material_message_serializes_parallel_batch_updates(tmp_path) -> None:
    db_path = tmp_path / "materials.db"
    engine = create_async_engine(f"sqlite+aiosqlite:///{db_path}")
    async with engine.begin() as connection:
        await connection.run_sync(Base.metadata.create_all)
    session_factory = async_sessionmaker(engine, expire_on_commit=False)

    async with session_factory() as session:
        workspace_repo = WorkspaceRepository(session)
        workspace = await workspace_repo.get_or_create_workspace(8010, "Group", "Europe/Moscow")
        await session.commit()

    async def register(index: int) -> int:
        async with session_factory() as session:
            batch_id = await register_material_message(
                session,
                workspace_id=workspace.id,
                materials_thread_id=10,
                media_group_id=None,
                source_message_id=1000 + index,
                source_chat_id=workspace.chat_id,
                source_thread_id=10,
                content_type="text",
                forwarded_from_chat_id=None,
                forwarded_from_message_id=None,
                batch_timeout_seconds=30,
            )
            await session.commit()
            return batch_id

    batch_ids = await asyncio.gather(*(register(index) for index in range(10)))

    async with session_factory() as session:
        batches = list((await session.execute(select(MaterialBatch))).scalars().all())

    assert len(set(batch_ids)) == 1
    assert len(batches) == 1
    assert batches[0].batch_size == 10

    await engine.dispose()


@pytest.mark.asyncio
async def test_seal_material_batches_reopens_batch_when_send_fails(session, monkeypatch) -> None:
    workspace_repo = WorkspaceRepository(session)
    workspace = await workspace_repo.get_or_create_workspace(-1002002002002, "Group", "UTC")
    await workspace_repo.upsert_topic_binding(workspace.id, TopicKey.MATERIALS, 10, "Материалы")

    repo = MaterialRepository(session)
    batch = await repo.create_batch(
        workspace_id=workspace.id,
        materials_thread_id=10,
        media_group_id=None,
    )
    await repo.append_item(
        batch=batch,
        source_message_id=1,
        source_chat_id=workspace.chat_id,
        source_thread_id=10,
        content_type="voice",
        forwarded_from_chat_id=None,
        forwarded_from_message_id=None,
    )
    batch.last_message_at = datetime.now(UTC) - timedelta(seconds=30)
    await session.commit()

    async def raising_send_message_logged(**kwargs):
        raise RuntimeError("network down")

    monkeypatch.setattr(seal_material_batches, "send_message_logged", raising_send_message_logged)

    await seal_material_batches.run(session, bot=object(), batch_timeout_seconds=15)

    refreshed_batch = await repo.get_batch(batch.id)
    assert refreshed_batch is not None
    assert refreshed_batch.batch_status is MaterialBatchStatus.OPEN
