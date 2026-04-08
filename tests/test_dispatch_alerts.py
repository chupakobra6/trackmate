from trackmate.domain.enums import AlertKind
from trackmate.worker.jobs.dispatch_alerts import _alert_text


def test_alert_text_uses_consistent_daily_task_wording() -> None:
    assert _alert_text(AlertKind.DAY_CLOSED_PENDING_REPORT) == (
        "🔔 День уже закончился, а отчет по задаче так и не появился."
    )
    assert _alert_text(AlertKind.OVERDUE_TASK_FAILED) == (
        "⏰ Время вышло — задача автоматически отмечена как не выполненная."
    )
