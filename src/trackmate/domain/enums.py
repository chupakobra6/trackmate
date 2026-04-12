from enum import StrEnum


class GroupSetupStatus(StrEnum):
    PENDING = "pending"
    REQUIREMENTS_FAILED = "requirements_failed"
    READY = "ready"


class TopicKey(StrEnum):
    MATERIALS = "materials"
    TODAY = "today"
    PROGRESS = "progress"


class MaterialBatchStatus(StrEnum):
    OPEN = "open"
    PUBLISHING = "publishing"
    SEALED = "sealed"


class MaterialHighestState(StrEnum):
    NONE = "none"
    READ = "read"
    NOTE = "note"
    APPLIED = "applied"


class DailyTaskStatus(StrEnum):
    ACTIVE = "active"
    AWAITING_REPORT = "awaiting_report"
    DONE = "done"
    PARTIAL = "partial"
    FAILED = "failed"


class AlertKind(StrEnum):
    DAY_CLOSED_PENDING_REPORT = "day_closed_pending_report"
    OVERDUE_TASK_FAILED = "overdue_task_failed"


class ProgressEventType(StrEnum):
    MATERIAL_NOTE_ADDED = "material_note_added"
    MATERIAL_APPLIED = "material_applied"
    DAILY_TASK_CLOSED = "daily_task.closed"
    DAILY_TASK_AUTO_FAILED = "daily_task.auto_failed"
    CUSTOM_UPDATE = "custom_update"
    SYSTEM_ALERT = "system_alert"


class ProgressPublishStatus(StrEnum):
    PENDING = "pending"
    PUBLISHING = "publishing"
    PUBLISHED = "published"
    FAILED = "failed"


class AlertDispatchStatus(StrEnum):
    PENDING = "pending"
    DISPATCHING = "dispatching"
    SENT = "sent"


class PendingInputKind(StrEnum):
    MATERIAL_NOTE = "material_note"
    MATERIAL_APPLIED = "material_applied"
    DAILY_TASK_TEXT = "daily_task_text"
    DAILY_TASK_REPORT = "daily_task_report"
    PROGRESS_UPDATE = "progress_update"
