from __future__ import annotations

from datetime import UTC, date, datetime, timedelta

from sqlalchemy import and_, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from trackmate.db.models import (
    DailyTask,
    DailyTaskAlert,
    MaterialBatch,
    MaterialItem,
    MaterialParticipantProgress,
    Participant,
    PendingInput,
    ProgressEvent,
    TopicBinding,
    WorkspaceGroup,
)
from trackmate.domain.enums import (
    AlertDispatchStatus,
    AlertKind,
    DailyTaskStatus,
    GroupSetupStatus,
    MaterialBatchStatus,
    ProgressPublishStatus,
    TopicKey,
)


class WorkspaceRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_or_create_workspace(self, chat_id: int, title: str | None, timezone: str) -> WorkspaceGroup:
        result = await self.session.execute(select(WorkspaceGroup).where(WorkspaceGroup.chat_id == chat_id))
        workspace = result.scalar_one_or_none()
        if workspace is None:
            workspace = WorkspaceGroup(chat_id=chat_id, title=title, timezone=timezone)
            self.session.add(workspace)
            await self.session.flush()
        else:
            if title and workspace.title != title:
                workspace.title = title
            if timezone and workspace.timezone != timezone and workspace.timezone in {"UTC", "Etc/UTC"}:
                workspace.timezone = timezone
        return workspace

    async def get_workspace_by_chat_id(self, chat_id: int) -> WorkspaceGroup | None:
        result = await self.session.execute(select(WorkspaceGroup).where(WorkspaceGroup.chat_id == chat_id))
        return result.scalar_one_or_none()

    async def get_workspace_by_id(self, workspace_id: int) -> WorkspaceGroup | None:
        result = await self.session.execute(select(WorkspaceGroup).where(WorkspaceGroup.id == workspace_id))
        return result.scalar_one_or_none()

    async def upsert_topic_binding(
        self,
        workspace_id: int,
        topic_key: TopicKey,
        thread_id: int,
        topic_title: str,
    ) -> TopicBinding:
        result = await self.session.execute(
            select(TopicBinding).where(
                TopicBinding.workspace_group_id == workspace_id,
                TopicBinding.topic_key == topic_key,
            )
        )
        binding = result.scalar_one_or_none()
        if binding is None:
            binding = TopicBinding(
                workspace_group_id=workspace_id,
                topic_key=topic_key,
                thread_id=thread_id,
                topic_title=topic_title,
            )
            self.session.add(binding)
        else:
            binding.thread_id = thread_id
            binding.topic_title = topic_title
        await self.session.flush()
        return binding

    async def list_topic_bindings(self, workspace_id: int) -> dict[TopicKey, TopicBinding]:
        result = await self.session.execute(
            select(TopicBinding).where(TopicBinding.workspace_group_id == workspace_id)
        )
        return {item.topic_key: item for item in result.scalars().all()}

    async def get_topic_binding(self, workspace_id: int, topic_key: TopicKey) -> TopicBinding | None:
        result = await self.session.execute(
            select(TopicBinding).where(
                TopicBinding.workspace_group_id == workspace_id,
                TopicBinding.topic_key == topic_key,
            )
        )
        return result.scalar_one_or_none()

    async def set_topic_messages(
        self,
        workspace_id: int,
        topic_key: TopicKey,
        *,
        intro_message_id: int | None = None,
        control_message_id: int | None = None,
        reset_intro_message_id: bool = False,
        reset_control_message_id: bool = False,
    ) -> None:
        result = await self.session.execute(
            select(TopicBinding).where(
                TopicBinding.workspace_group_id == workspace_id,
                TopicBinding.topic_key == topic_key,
            )
        )
        binding = result.scalar_one_or_none()
        if binding is None:
            return
        if reset_intro_message_id:
            binding.intro_message_id = None
        elif intro_message_id is not None:
            binding.intro_message_id = intro_message_id
        if reset_control_message_id:
            binding.control_message_id = None
        elif control_message_id is not None:
            binding.control_message_id = control_message_id
        await self.session.flush()

    async def mark_ready(self, workspace: WorkspaceGroup) -> None:
        workspace.setup_status = GroupSetupStatus.READY
        await self.session.flush()

    async def set_setup_message_id(self, workspace_id: int, message_id: int) -> None:
        workspace = await self.get_workspace_by_id(workspace_id)
        if workspace is None:
            return
        workspace.setup_message_id = message_id
        await self.session.flush()

    async def register_participant(
        self,
        workspace_id: int,
        user_id: int,
        username: str | None,
        display_name: str,
    ) -> Participant:
        result = await self.session.execute(
            select(Participant).where(
                Participant.workspace_group_id == workspace_id,
                Participant.user_id == user_id,
            )
        )
        participant = result.scalar_one_or_none()
        if participant is None:
            participant = Participant(
                workspace_group_id=workspace_id,
                user_id=user_id,
                username=username,
                display_name=display_name,
            )
            self.session.add(participant)
            await self.session.flush()
        else:
            participant.username = username
            participant.display_name = display_name
        return participant


class MaterialRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_open_batch(
        self,
        *,
        workspace_id: int,
        materials_thread_id: int,
        media_group_id: str | None,
        timeout_seconds: int,
        now_utc: datetime,
    ) -> MaterialBatch | None:
        conditions = [
            MaterialBatch.workspace_group_id == workspace_id,
            MaterialBatch.materials_thread_id == materials_thread_id,
            MaterialBatch.batch_status == MaterialBatchStatus.OPEN,
        ]
        if media_group_id:
            conditions.append(MaterialBatch.media_group_id == media_group_id)
        else:
            conditions.append(MaterialBatch.media_group_id.is_(None))
            conditions.append(MaterialBatch.last_message_at >= now_utc - timedelta(seconds=timeout_seconds))
        result = await self.session.execute(
            select(MaterialBatch).where(and_(*conditions)).order_by(MaterialBatch.id.desc())
        )
        return result.scalars().first()

    async def create_batch(
        self,
        *,
        workspace_id: int,
        materials_thread_id: int,
        media_group_id: str | None,
    ) -> MaterialBatch:
        batch = MaterialBatch(
            workspace_group_id=workspace_id,
            materials_thread_id=materials_thread_id,
            media_group_id=media_group_id,
        )
        self.session.add(batch)
        await self.session.flush()
        return batch

    async def append_item(
        self,
        *,
        batch: MaterialBatch,
        source_message_id: int,
        source_chat_id: int,
        source_thread_id: int | None,
        content_type: str,
        forwarded_from_chat_id: int | None,
        forwarded_from_message_id: int | None,
    ) -> None:
        item = MaterialItem(
            material_batch_id=batch.id,
            source_message_id=source_message_id,
            source_chat_id=source_chat_id,
            source_thread_id=source_thread_id,
            position=batch.batch_size + 1,
            content_type=content_type,
            forwarded_from_chat_id=forwarded_from_chat_id,
            forwarded_from_message_id=forwarded_from_message_id,
        )
        self.session.add(item)
        batch.batch_size += 1
        batch.last_message_at = datetime.now(UTC)
        batch.source_anchor_message_id = batch.source_anchor_message_id or source_message_id
        await self.session.flush()

    async def list_sealable_batches(self, *, timeout_seconds: int, now_utc: datetime) -> list[MaterialBatch]:
        threshold = now_utc - timedelta(seconds=timeout_seconds)
        result = await self.session.execute(
            select(MaterialBatch)
            .where(
                MaterialBatch.batch_status == MaterialBatchStatus.OPEN,
                MaterialBatch.last_message_at <= threshold,
            )
            .order_by(MaterialBatch.id.asc())
        )
        return list(result.scalars().all())

    async def list_mergeable_open_batches(self, batch: MaterialBatch) -> list[MaterialBatch]:
        conditions = [
            MaterialBatch.workspace_group_id == batch.workspace_group_id,
            MaterialBatch.materials_thread_id == batch.materials_thread_id,
            MaterialBatch.batch_status == MaterialBatchStatus.OPEN,
        ]
        if batch.media_group_id:
            conditions.append(MaterialBatch.media_group_id == batch.media_group_id)
        else:
            conditions.append(MaterialBatch.media_group_id.is_(None))
        result = await self.session.execute(
            select(MaterialBatch).where(and_(*conditions)).order_by(MaterialBatch.id.asc())
        )
        return list(result.scalars().all())

    async def merge_batches(self, primary: MaterialBatch, sources: list[MaterialBatch]) -> None:
        for source in sources:
            items_result = await self.session.execute(
                select(MaterialItem)
                .where(MaterialItem.material_batch_id == source.id)
                .order_by(MaterialItem.position.asc(), MaterialItem.id.asc())
            )
            items = list(items_result.scalars().all())
            next_position = primary.batch_size
            for item in items:
                next_position += 1
                item.material_batch_id = primary.id
                item.position = next_position
            primary.batch_size += len(items)
            primary.source_anchor_message_id = primary.source_anchor_message_id or source.source_anchor_message_id
            primary.last_message_at = max(primary.last_message_at, source.last_message_at)
            await self.session.delete(source)
        await self.session.flush()

    async def claim_batch_for_publish(self, batch: MaterialBatch) -> None:
        batch.batch_status = MaterialBatchStatus.PUBLISHING
        await self.session.flush()

    async def seal_batch(self, batch: MaterialBatch, tracking_card_message_id: int) -> None:
        batch.batch_status = MaterialBatchStatus.SEALED
        batch.tracking_card_message_id = tracking_card_message_id
        batch.sealed_at = datetime.now(UTC)
        await self.session.flush()

    async def get_batch(self, batch_id: int) -> MaterialBatch | None:
        result = await self.session.execute(select(MaterialBatch).where(MaterialBatch.id == batch_id))
        return result.scalar_one_or_none()

    async def get_progress(
        self,
        batch_id: int,
        participant_id: int,
    ) -> MaterialParticipantProgress | None:
        result = await self.session.execute(
            select(MaterialParticipantProgress).where(
                MaterialParticipantProgress.material_batch_id == batch_id,
                MaterialParticipantProgress.participant_id == participant_id,
            )
        )
        return result.scalar_one_or_none()

    async def list_progresses(self, batch_id: int) -> list[MaterialParticipantProgress]:
        result = await self.session.execute(
            select(MaterialParticipantProgress)
            .options(selectinload(MaterialParticipantProgress.participant))
            .where(MaterialParticipantProgress.material_batch_id == batch_id)
        )
        return list(result.scalars().all())

    async def create_progress(self, batch_id: int, participant_id: int) -> MaterialParticipantProgress:
        progress = MaterialParticipantProgress(material_batch_id=batch_id, participant_id=participant_id)
        self.session.add(progress)
        await self.session.flush()
        return progress


class TodayRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_open_task(self, workspace_id: int, participant_id: int) -> DailyTask | None:
        result = await self.session.execute(
            select(DailyTask)
            .where(
                DailyTask.workspace_group_id == workspace_id,
                DailyTask.participant_id == participant_id,
                DailyTask.status.in_([DailyTaskStatus.ACTIVE, DailyTaskStatus.AWAITING_REPORT]),
            )
            .order_by(DailyTask.id.desc())
        )
        return result.scalars().first()

    async def get_task_for_date(
        self,
        workspace_id: int,
        participant_id: int,
        task_date: date,
    ) -> DailyTask | None:
        result = await self.session.execute(
            select(DailyTask).where(
                DailyTask.workspace_group_id == workspace_id,
                DailyTask.participant_id == participant_id,
                DailyTask.task_date == task_date,
            )
        )
        return result.scalar_one_or_none()

    async def create_daily_task(
        self,
        *,
        workspace_id: int,
        participant_id: int,
        owner_user_id: int,
        task_date: date,
        text: str,
        today_card_message_id: int,
    ) -> DailyTask:
        task = DailyTask(
            workspace_group_id=workspace_id,
            participant_id=participant_id,
            owner_user_id=owner_user_id,
            task_date=task_date,
            text=text,
            today_card_message_id=today_card_message_id,
        )
        self.session.add(task)
        await self.session.flush()
        return task

    async def get_task(self, task_id: int) -> DailyTask | None:
        result = await self.session.execute(select(DailyTask).where(DailyTask.id == task_id))
        return result.scalar_one_or_none()

    async def list_tasks_for_transition(self, statuses: list[DailyTaskStatus]) -> list[DailyTask]:
        result = await self.session.execute(select(DailyTask).where(DailyTask.status.in_(statuses)))
        return list(result.scalars().all())

    async def get_or_create_alert(self, task_id: int, kind: AlertKind) -> DailyTaskAlert:
        result = await self.session.execute(
            select(DailyTaskAlert).where(
                DailyTaskAlert.daily_task_id == task_id,
                DailyTaskAlert.alert_kind == kind,
            )
        )
        alert = result.scalar_one_or_none()
        if alert is None:
            alert = DailyTaskAlert(daily_task_id=task_id, alert_kind=kind)
            self.session.add(alert)
            await self.session.flush()
        return alert

    async def list_pending_alerts(self) -> list[DailyTaskAlert]:
        result = await self.session.execute(
            select(DailyTaskAlert).where(
                DailyTaskAlert.dispatch_status == AlertDispatchStatus.PENDING,
                DailyTaskAlert.acknowledged_at.is_(None),
            )
        )
        return list(result.scalars().all())

    async def claim_alert_dispatch(self, alert: DailyTaskAlert) -> None:
        alert.dispatch_status = AlertDispatchStatus.DISPATCHING
        await self.session.flush()

    async def mark_alert_sent(self, alert: DailyTaskAlert, telegram_message_id: int) -> None:
        alert.dispatch_status = AlertDispatchStatus.SENT
        alert.telegram_message_id = telegram_message_id
        await self.session.flush()


class ProgressRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def create_event(
        self,
        *,
        workspace_group_id: int,
        event_type,
        payload: dict,
        participant_id: int | None = None,
        material_batch_id: int | None = None,
        daily_task_id: int | None = None,
    ) -> ProgressEvent:
        event = ProgressEvent(
            workspace_group_id=workspace_group_id,
            participant_id=participant_id,
            material_batch_id=material_batch_id,
            daily_task_id=daily_task_id,
            event_type=event_type,
            payload=payload,
        )
        self.session.add(event)
        await self.session.flush()
        return event

    async def list_pending_events(self) -> list[ProgressEvent]:
        result = await self.session.execute(
            select(ProgressEvent)
            .where(ProgressEvent.publish_status == ProgressPublishStatus.PENDING)
            .order_by(ProgressEvent.id.asc())
        )
        return list(result.scalars().all())

    async def claim_event_for_publish(self, event: ProgressEvent) -> None:
        event.publish_status = ProgressPublishStatus.PUBLISHING
        await self.session.flush()

    async def mark_event_published(
        self,
        event: ProgressEvent,
        *,
        published_message_id: int,
        published_at: datetime,
    ) -> None:
        event.published_message_id = published_message_id
        event.publish_status = ProgressPublishStatus.PUBLISHED
        event.published_at = published_at
        await self.session.flush()

    async def mark_event_failed(self, event: ProgressEvent) -> None:
        event.publish_status = ProgressPublishStatus.FAILED
        await self.session.flush()


class PendingInputRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get(self, workspace_id: int, user_id: int) -> PendingInput | None:
        result = await self.session.execute(
            select(PendingInput).where(
                PendingInput.workspace_group_id == workspace_id,
                PendingInput.user_id == user_id,
            )
        )
        return result.scalar_one_or_none()

    async def upsert(self, workspace_id: int, user_id: int, kind: str, payload: dict) -> PendingInput:
        pending = await self.get(workspace_id, user_id)
        if pending is None:
            pending = PendingInput(workspace_group_id=workspace_id, user_id=user_id, kind=kind, payload=payload)
            self.session.add(pending)
        else:
            pending.kind = kind
            pending.payload = payload
        await self.session.flush()
        return pending

    async def clear(self, workspace_id: int, user_id: int) -> None:
        pending = await self.get(workspace_id, user_id)
        if pending is not None:
            await self.session.delete(pending)
