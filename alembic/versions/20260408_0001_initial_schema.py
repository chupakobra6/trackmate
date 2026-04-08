"""initial schema"""

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "20260408_0001"
down_revision: str | None = None
branch_labels: Sequence[str] | None = None
depends_on: Sequence[str] | None = None


group_setup_status = sa.Enum("pending", "requirements_failed", "ready", name="groupsetupstatus")
topic_key = sa.Enum("materials", "today", "progress", name="topickey")
material_batch_status = sa.Enum("open", "publishing", "sealed", name="materialbatchstatus")
material_highest_state = sa.Enum("none", "read", "note", "applied", name="materialhigheststate")
daily_task_status = sa.Enum(
    "active",
    "awaiting_report",
    "done",
    "partial",
    "failed",
    name="dailytaskstatus",
)
alert_kind = sa.Enum(
    "day_closed_pending_report",
    "overdue_task_failed",
    name="alertkind",
)
progress_event_type = sa.Enum(
    "material_note_added",
    "material_applied",
    "daily_task.closed",
    "daily_task.auto_failed",
    "system_alert",
    name="progresseventtype",
)
progress_publish_status = sa.Enum("pending", "publishing", "published", "failed", name="progresspublishstatus")
alert_dispatch_status = sa.Enum("pending", "dispatching", "sent", name="alertdispatchstatus")


def upgrade() -> None:
    op.create_table(
        "workspace_groups",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("chat_id", sa.BigInteger(), nullable=False),
        sa.Column("title", sa.String(length=255), nullable=True),
        sa.Column("timezone", sa.String(length=64), nullable=False),
        sa.Column("setup_status", group_setup_status, nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("chat_id"),
    )
    op.create_index("ix_workspace_groups_chat_id", "workspace_groups", ["chat_id"])

    op.create_table(
        "topic_bindings",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("topic_key", topic_key, nullable=False),
        sa.Column("thread_id", sa.Integer(), nullable=False),
        sa.Column("topic_title", sa.String(length=255), nullable=False),
        sa.Column("intro_message_id", sa.Integer(), nullable=True),
        sa.Column("control_message_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("workspace_group_id", "topic_key"),
    )
    op.create_index("ix_topic_bindings_workspace_group_id", "topic_bindings", ["workspace_group_id"])
    op.create_index("ix_topic_bindings_thread_id", "topic_bindings", ["thread_id"])

    op.create_table(
        "participants",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.BigInteger(), nullable=False),
        sa.Column("username", sa.String(length=255), nullable=True),
        sa.Column("display_name", sa.String(length=255), nullable=False),
        sa.Column("is_active", sa.Boolean(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("workspace_group_id", "user_id"),
    )
    op.create_index("ix_participants_workspace_group_id", "participants", ["workspace_group_id"])
    op.create_index("ix_participants_user_id", "participants", ["user_id"])

    op.create_table(
        "material_batches",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("materials_thread_id", sa.Integer(), nullable=False),
        sa.Column("created_by_user_id", sa.BigInteger(), nullable=False),
        sa.Column("sender_id", sa.BigInteger(), nullable=False),
        sa.Column("media_group_id", sa.String(length=255), nullable=True),
        sa.Column("upload_session_key", sa.String(length=255), nullable=True),
        sa.Column("batch_status", material_batch_status, nullable=False),
        sa.Column("preview_text", sa.Text(), nullable=True),
        sa.Column("batch_size", sa.Integer(), nullable=False),
        sa.Column("source_anchor_message_id", sa.Integer(), nullable=True),
        sa.Column("tracking_card_message_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("last_message_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("sealed_at", sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
    )
    for column in [
        "workspace_group_id",
        "materials_thread_id",
        "created_by_user_id",
        "sender_id",
        "media_group_id",
        "upload_session_key",
    ]:
        op.create_index(f"ix_material_batches_{column}", "material_batches", [column])

    op.create_table(
        "material_items",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("material_batch_id", sa.Integer(), nullable=False),
        sa.Column("source_message_id", sa.Integer(), nullable=False),
        sa.Column("source_chat_id", sa.BigInteger(), nullable=False),
        sa.Column("source_thread_id", sa.Integer(), nullable=True),
        sa.Column("position", sa.Integer(), nullable=False),
        sa.Column("content_type", sa.String(length=64), nullable=False),
        sa.Column("text_preview", sa.Text(), nullable=True),
        sa.Column("forwarded_from_chat_id", sa.BigInteger(), nullable=True),
        sa.Column("forwarded_from_message_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["material_batch_id"], ["material_batches.id"], ondelete="CASCADE"),
    )
    for column in ["material_batch_id", "source_message_id", "source_chat_id"]:
        op.create_index(f"ix_material_items_{column}", "material_items", [column])

    op.create_table(
        "daily_tasks",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("participant_id", sa.Integer(), nullable=False),
        sa.Column("owner_user_id", sa.BigInteger(), nullable=False),
        sa.Column("task_date", sa.Date(), nullable=False),
        sa.Column("text", sa.Text(), nullable=False),
        sa.Column("status", daily_task_status, nullable=False),
        sa.Column("report_text", sa.Text(), nullable=True),
        sa.Column("report_status", daily_task_status, nullable=True),
        sa.Column("today_card_message_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("reported_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("awaiting_report_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("failed_at", sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["participant_id"], ["participants.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("workspace_group_id", "participant_id", "task_date"),
    )
    for column in ["workspace_group_id", "participant_id", "owner_user_id", "status"]:
        op.create_index(f"ix_daily_tasks_{column}", "daily_tasks", [column])

    op.create_table(
        "progress_events",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("participant_id", sa.Integer(), nullable=True),
        sa.Column("material_batch_id", sa.Integer(), nullable=True),
        sa.Column("daily_task_id", sa.Integer(), nullable=True),
        sa.Column("event_type", progress_event_type, nullable=False),
        sa.Column("publish_status", progress_publish_status, nullable=False),
        sa.Column("payload", sa.JSON(), nullable=False),
        sa.Column("published_message_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("published_at", sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["participant_id"], ["participants.id"], ondelete="SET NULL"),
        sa.ForeignKeyConstraint(["material_batch_id"], ["material_batches.id"], ondelete="SET NULL"),
        sa.ForeignKeyConstraint(["daily_task_id"], ["daily_tasks.id"], ondelete="SET NULL"),
    )
    for column in ["workspace_group_id", "participant_id", "material_batch_id", "daily_task_id", "event_type"]:
        op.create_index(f"ix_progress_events_{column}", "progress_events", [column])

    op.create_table(
        "material_participant_progresses",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("material_batch_id", sa.Integer(), nullable=False),
        sa.Column("participant_id", sa.Integer(), nullable=False),
        sa.Column("highest_state", material_highest_state, nullable=False),
        sa.Column("read_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("note_progress_event_id", sa.Integer(), nullable=True),
        sa.Column("applied_progress_event_id", sa.Integer(), nullable=True),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["material_batch_id"], ["material_batches.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["participant_id"], ["participants.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["note_progress_event_id"], ["progress_events.id"], ondelete="SET NULL"),
        sa.ForeignKeyConstraint(["applied_progress_event_id"], ["progress_events.id"], ondelete="SET NULL"),
        sa.UniqueConstraint("material_batch_id", "participant_id"),
    )
    op.create_index(
        "ix_material_participant_progresses_material_batch_id",
        "material_participant_progresses",
        ["material_batch_id"],
    )
    op.create_index(
        "ix_material_participant_progresses_participant_id",
        "material_participant_progresses",
        ["participant_id"],
    )

    op.create_table(
        "daily_task_alerts",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("daily_task_id", sa.Integer(), nullable=False),
        sa.Column("alert_kind", alert_kind, nullable=False),
        sa.Column("dispatch_status", alert_dispatch_status, nullable=False),
        sa.Column("telegram_message_id", sa.Integer(), nullable=True),
        sa.Column("acknowledged_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["daily_task_id"], ["daily_tasks.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("daily_task_id", "alert_kind"),
    )
    op.create_index("ix_daily_task_alerts_daily_task_id", "daily_task_alerts", ["daily_task_id"])

    op.create_table(
        "pending_inputs",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("workspace_group_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.BigInteger(), nullable=False),
        sa.Column("kind", sa.String(length=64), nullable=False),
        sa.Column("payload", sa.JSON(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["workspace_group_id"], ["workspace_groups.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("workspace_group_id", "user_id"),
    )
    op.create_index("ix_pending_inputs_workspace_group_id", "pending_inputs", ["workspace_group_id"])
    op.create_index("ix_pending_inputs_user_id", "pending_inputs", ["user_id"])


def downgrade() -> None:
    for table in [
        "pending_inputs",
        "daily_task_alerts",
        "material_participant_progresses",
        "progress_events",
        "daily_tasks",
        "material_items",
        "material_batches",
        "participants",
        "topic_bindings",
        "workspace_groups",
    ]:
        op.drop_table(table)

