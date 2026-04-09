from __future__ import annotations

from datetime import UTC, datetime

import structlog
from aiogram import Bot
from sqlalchemy.ext.asyncio import AsyncSession

from trackmate.adapters.persistence.repositories import MaterialRepository, WorkspaceRepository
from trackmate.adapters.telegram.formatters import format_material_card
from trackmate.adapters.telegram.keyboards import material_progress_keyboard
from trackmate.adapters.telegram.message_ops import send_message_logged
from trackmate.domain.enums import MaterialBatchStatus, TopicKey

logger = structlog.get_logger(__name__)


async def run(session: AsyncSession, bot: Bot, *, batch_timeout_seconds: int) -> None:
    materials_repo = MaterialRepository(session)
    workspace_repo = WorkspaceRepository(session)
    batches = await materials_repo.list_sealable_batches(
        timeout_seconds=batch_timeout_seconds,
        now_utc=datetime.now(UTC),
    )
    for batch in batches:
        fresh_batch = await materials_repo.get_batch(batch.id)
        if fresh_batch is None or fresh_batch.batch_status is not MaterialBatchStatus.OPEN:
            continue
        mergeable_batches = await materials_repo.list_mergeable_open_batches(fresh_batch)
        if not mergeable_batches:
            continue
        primary_batch = mergeable_batches[0]
        if fresh_batch.id != primary_batch.id:
            continue
        if len(mergeable_batches) > 1:
            await materials_repo.merge_batches(primary_batch, mergeable_batches[1:])
            logger.info(
                "telegram.material_batches_merged",
                primary_batch_id=primary_batch.id,
                merged_batch_ids=[source.id for source in mergeable_batches[1:]],
                batch_size=primary_batch.batch_size,
                workspace_id=primary_batch.workspace_group_id,
                thread_id=primary_batch.materials_thread_id,
            )
        batch = primary_batch
        await materials_repo.claim_batch_for_publish(batch)
        await session.commit()
        workspace = await workspace_repo.get_workspace_by_id(batch.workspace_group_id)
        if workspace is None:
            continue
        bindings = await workspace_repo.list_topic_bindings(workspace.id)
        materials_binding = bindings.get(TopicKey.MATERIALS)
        if materials_binding is None:
            continue
        progresses = await materials_repo.list_progresses(batch.id)
        logger.info(
            "telegram.material_batch_publishing",
            batch_id=batch.id,
            batch_size=batch.batch_size,
            workspace_id=batch.workspace_group_id,
            thread_id=materials_binding.thread_id,
        )
        message = await send_message_logged(
            bot=bot,
            chat_id=workspace.chat_id,
            message_thread_id=materials_binding.thread_id,
            text=format_material_card(batch, progresses),
            reply_to_message_id=batch.source_anchor_message_id,
            reply_markup=material_progress_keyboard(batch.id),
        )
        await materials_repo.seal_batch(batch, message.message_id)
        await session.commit()
