from dataclasses import dataclass

from trackmate.domain.enums import DailyTaskStatus


@dataclass(frozen=True)
class SetupWorkspaceGroup:
    chat_id: int
    title: str | None
    timezone: str


@dataclass(frozen=True)
class CreateDailyTask:
    workspace_group_id: int
    participant_id: int
    text: str


@dataclass(frozen=True)
class SubmitDailyTaskReport:
    task_id: int
    status: DailyTaskStatus
    text: str


@dataclass(frozen=True)
class RegisterMaterialBatch:
    workspace_group_id: int
    sender_id: int
    materials_thread_id: int
    media_group_id: str | None


@dataclass(frozen=True)
class ApplyMaterialProgress:
    batch_id: int
    participant_id: int
    kind: str
    text: str | None = None
