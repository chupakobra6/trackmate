"""cleanup unused material columns"""

from collections.abc import Sequence

from alembic import op

revision: str = "20260409_0003"
down_revision: str | None = "20260408_0002"
branch_labels: Sequence[str] | None = None
depends_on: Sequence[str] | None = None


def upgrade() -> None:
    op.drop_index("ix_material_batches_created_by_user_id", table_name="material_batches")
    op.drop_index("ix_material_batches_sender_id", table_name="material_batches")
    op.drop_index("ix_material_batches_upload_session_key", table_name="material_batches")
    op.drop_column("material_batches", "created_by_user_id")
    op.drop_column("material_batches", "sender_id")
    op.drop_column("material_batches", "upload_session_key")
    op.drop_column("material_batches", "preview_text")
    op.drop_column("material_items", "text_preview")


def downgrade() -> None:
    raise NotImplementedError("Downgrade is not supported for cleanup migration 20260409_0003.")
