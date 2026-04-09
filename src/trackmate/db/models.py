from __future__ import annotations

from datetime import UTC, date, datetime
from typing import Any

from sqlalchemy import (
    JSON,
    BigInteger,
    Date,
    DateTime,
    ForeignKey,
    Integer,
    String,
    Text,
    UniqueConstraint,
)
from sqlalchemy import Enum as SqlEnum
from sqlalchemy.orm import Mapped, mapped_column, relationship

from trackmate.db.base import Base, utcnow
from trackmate.domain.enums import (
    AlertDispatchStatus,
    AlertKind,
    DailyTaskStatus,
    GroupSetupStatus,
    MaterialBatchStatus,
    MaterialHighestState,
    ProgressEventType,
    ProgressPublishStatus,
    TopicKey,
)


def enum_column(enum_cls):
    return SqlEnum(
        enum_cls,
        values_callable=lambda enum_items: [item.value for item in enum_items],
    )


class WorkspaceGroup(Base):
    __tablename__ = "workspace_groups"

    id: Mapped[int] = mapped_column(primary_key=True)
    chat_id: Mapped[int] = mapped_column(BigInteger, unique=True, index=True)
    title: Mapped[str | None] = mapped_column(String(255))
    timezone: Mapped[str] = mapped_column(String(64), default="UTC")
    setup_message_id: Mapped[int | None] = mapped_column(Integer)
    setup_status: Mapped[GroupSetupStatus] = mapped_column(
        enum_column(GroupSetupStatus),
        default=GroupSetupStatus.PENDING,
    )
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)

    topics: Mapped[list[TopicBinding]] = relationship(back_populates="workspace", cascade="all, delete-orphan")
    participants: Mapped[list[Participant]] = relationship(back_populates="workspace", cascade="all, delete-orphan")


class TopicBinding(Base):
    __tablename__ = "topic_bindings"
    __table_args__ = (UniqueConstraint("workspace_group_id", "topic_key"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    topic_key: Mapped[TopicKey] = mapped_column(enum_column(TopicKey))
    thread_id: Mapped[int] = mapped_column(index=True)
    topic_title: Mapped[str] = mapped_column(String(255))
    intro_message_id: Mapped[int | None] = mapped_column(Integer)
    control_message_id: Mapped[int | None] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)

    workspace: Mapped[WorkspaceGroup] = relationship(back_populates="topics")


class Participant(Base):
    __tablename__ = "participants"
    __table_args__ = (UniqueConstraint("workspace_group_id", "user_id"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int] = mapped_column(BigInteger, index=True)
    username: Mapped[str | None] = mapped_column(String(255))
    display_name: Mapped[str] = mapped_column(String(255))
    is_active: Mapped[bool] = mapped_column(default=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)

    workspace: Mapped[WorkspaceGroup] = relationship(back_populates="participants")
    daily_tasks: Mapped[list[DailyTask]] = relationship(back_populates="participant")
    material_progresses: Mapped[list[MaterialParticipantProgress]] = relationship(back_populates="participant")


class MaterialBatch(Base):
    __tablename__ = "material_batches"

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    materials_thread_id: Mapped[int] = mapped_column(index=True)
    media_group_id: Mapped[str | None] = mapped_column(String(255), index=True)
    batch_status: Mapped[MaterialBatchStatus] = mapped_column(
        enum_column(MaterialBatchStatus),
        default=MaterialBatchStatus.OPEN,
    )
    batch_size: Mapped[int] = mapped_column(default=0)
    source_anchor_message_id: Mapped[int | None] = mapped_column(Integer)
    tracking_card_message_id: Mapped[int | None] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    last_message_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    sealed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))

    items: Mapped[list[MaterialItem]] = relationship(back_populates="batch", cascade="all, delete-orphan")
    progresses: Mapped[list[MaterialParticipantProgress]] = relationship(back_populates="batch", cascade="all, delete-orphan")


class MaterialItem(Base):
    __tablename__ = "material_items"

    id: Mapped[int] = mapped_column(primary_key=True)
    material_batch_id: Mapped[int] = mapped_column(ForeignKey("material_batches.id", ondelete="CASCADE"), index=True)
    source_message_id: Mapped[int] = mapped_column(index=True)
    source_chat_id: Mapped[int] = mapped_column(BigInteger, index=True)
    source_thread_id: Mapped[int | None] = mapped_column(Integer)
    position: Mapped[int] = mapped_column(Integer)
    content_type: Mapped[str] = mapped_column(String(64))
    forwarded_from_chat_id: Mapped[int | None] = mapped_column(BigInteger)
    forwarded_from_message_id: Mapped[int | None] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)

    batch: Mapped[MaterialBatch] = relationship(back_populates="items")


class ProgressEvent(Base):
    __tablename__ = "progress_events"

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    participant_id: Mapped[int | None] = mapped_column(ForeignKey("participants.id", ondelete="SET NULL"), index=True)
    material_batch_id: Mapped[int | None] = mapped_column(ForeignKey("material_batches.id", ondelete="SET NULL"), index=True)
    daily_task_id: Mapped[int | None] = mapped_column(ForeignKey("daily_tasks.id", ondelete="SET NULL"), index=True)
    event_type: Mapped[ProgressEventType] = mapped_column(enum_column(ProgressEventType), index=True)
    publish_status: Mapped[ProgressPublishStatus] = mapped_column(
        enum_column(ProgressPublishStatus),
        default=ProgressPublishStatus.PENDING,
    )
    payload: Mapped[dict[str, Any]] = mapped_column(JSON, default=dict)
    published_message_id: Mapped[int | None] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    published_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))


class MaterialParticipantProgress(Base):
    __tablename__ = "material_participant_progresses"
    __table_args__ = (UniqueConstraint("material_batch_id", "participant_id"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    material_batch_id: Mapped[int] = mapped_column(ForeignKey("material_batches.id", ondelete="CASCADE"), index=True)
    participant_id: Mapped[int] = mapped_column(ForeignKey("participants.id", ondelete="CASCADE"), index=True)
    highest_state: Mapped[MaterialHighestState] = mapped_column(
        enum_column(MaterialHighestState),
        default=MaterialHighestState.NONE,
    )
    read_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    note_progress_event_id: Mapped[int | None] = mapped_column(ForeignKey("progress_events.id", ondelete="SET NULL"))
    applied_progress_event_id: Mapped[int | None] = mapped_column(ForeignKey("progress_events.id", ondelete="SET NULL"))
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow, onupdate=utcnow)

    batch: Mapped[MaterialBatch] = relationship(back_populates="progresses")
    participant: Mapped[Participant] = relationship(back_populates="material_progresses")


class DailyTask(Base):
    __tablename__ = "daily_tasks"
    __table_args__ = (UniqueConstraint("workspace_group_id", "participant_id", "task_date"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    participant_id: Mapped[int] = mapped_column(ForeignKey("participants.id", ondelete="CASCADE"), index=True)
    owner_user_id: Mapped[int] = mapped_column(BigInteger, index=True)
    task_date: Mapped[date] = mapped_column(Date)
    text: Mapped[str] = mapped_column(Text)
    status: Mapped[DailyTaskStatus] = mapped_column(
        enum_column(DailyTaskStatus),
        default=DailyTaskStatus.ACTIVE,
        index=True,
    )
    report_text: Mapped[str | None] = mapped_column(Text)
    report_status: Mapped[DailyTaskStatus | None] = mapped_column(enum_column(DailyTaskStatus))
    today_card_message_id: Mapped[int | None] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    reported_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    awaiting_report_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    failed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))

    participant: Mapped[Participant] = relationship(back_populates="daily_tasks")


class DailyTaskAlert(Base):
    __tablename__ = "daily_task_alerts"
    __table_args__ = (UniqueConstraint("daily_task_id", "alert_kind"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    daily_task_id: Mapped[int] = mapped_column(ForeignKey("daily_tasks.id", ondelete="CASCADE"), index=True)
    alert_kind: Mapped[AlertKind] = mapped_column(enum_column(AlertKind))
    dispatch_status: Mapped[AlertDispatchStatus] = mapped_column(
        enum_column(AlertDispatchStatus),
        default=AlertDispatchStatus.PENDING,
    )
    telegram_message_id: Mapped[int | None] = mapped_column(Integer)
    acknowledged_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=utcnow)


class PendingInput(Base):
    __tablename__ = "pending_inputs"
    __table_args__ = (UniqueConstraint("workspace_group_id", "user_id"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    workspace_group_id: Mapped[int] = mapped_column(ForeignKey("workspace_groups.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int] = mapped_column(BigInteger, index=True)
    kind: Mapped[str] = mapped_column(String(64))
    payload: Mapped[dict[str, Any]] = mapped_column(JSON, default=dict)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=lambda: datetime.now(UTC))
