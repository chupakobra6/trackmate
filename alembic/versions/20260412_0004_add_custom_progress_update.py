"""add custom progress update event type"""

from collections.abc import Sequence

from alembic import op

revision: str = "20260412_0004"
down_revision: str | None = "20260409_0003"
branch_labels: Sequence[str] | None = None
depends_on: Sequence[str] | None = None


def upgrade() -> None:
    bind = op.get_bind()
    if bind.dialect.name == "postgresql":
        op.execute("ALTER TYPE progresseventtype ADD VALUE IF NOT EXISTS 'custom_update'")


def downgrade() -> None:
    # PostgreSQL enum value removal is intentionally left as a no-op.
    pass
