"""add workspace setup message id"""

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "20260408_0002"
down_revision: str | None = "20260408_0001"
branch_labels: Sequence[str] | None = None
depends_on: Sequence[str] | None = None


def upgrade() -> None:
    op.add_column("workspace_groups", sa.Column("setup_message_id", sa.Integer(), nullable=True))


def downgrade() -> None:
    op.drop_column("workspace_groups", "setup_message_id")
