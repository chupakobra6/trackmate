package domain

type GroupSetupStatus string

const (
	GroupSetupPending            GroupSetupStatus = "pending"
	GroupSetupRequirementsFailed GroupSetupStatus = "requirements_failed"
	GroupSetupReady              GroupSetupStatus = "ready"
)

type TopicKey string

const (
	TopicToday    TopicKey = "today"
	TopicProgress TopicKey = "progress"
	TopicRoutine  TopicKey = "routine"
	TopicGoals    TopicKey = "goals"
)

type DailyTaskStatus string

const (
	DailyTaskActive         DailyTaskStatus = "active"
	DailyTaskAwaitingReport DailyTaskStatus = "awaiting_report"
	DailyTaskDone           DailyTaskStatus = "done"
	DailyTaskPartial        DailyTaskStatus = "partial"
	DailyTaskFailed         DailyTaskStatus = "failed"
)

func (s DailyTaskStatus) IsOpen() bool {
	return s == DailyTaskActive || s == DailyTaskAwaitingReport
}

func (s DailyTaskStatus) IsFinalReport() bool {
	return s == DailyTaskDone || s == DailyTaskPartial || s == DailyTaskFailed
}

type AlertKind string

const (
	AlertDayClosedPendingReport AlertKind = "day_closed_pending_report"
	AlertOverdueTaskFailed      AlertKind = "overdue_task_failed"
)

type AlertDispatchStatus string

const (
	AlertDispatchPending     AlertDispatchStatus = "pending"
	AlertDispatchDispatching AlertDispatchStatus = "dispatching"
	AlertDispatchSent        AlertDispatchStatus = "sent"
)

type ProgressEventType string

const (
	ProgressDailyTaskClosed   ProgressEventType = "daily_task.closed"
	ProgressDailyTaskAutoFail ProgressEventType = "daily_task.auto_failed"
	ProgressSystemAlert       ProgressEventType = "system_alert"
	ProgressCustomUpdate      ProgressEventType = "custom_update"
)

type ProgressPublishStatus string

const (
	ProgressPublishPending    ProgressPublishStatus = "pending"
	ProgressPublishPublishing ProgressPublishStatus = "publishing"
	ProgressPublishPublished  ProgressPublishStatus = "published"
	ProgressPublishFailed     ProgressPublishStatus = "failed"
)

type PendingInputKind string

const (
	PendingDailyTaskText       PendingInputKind = "daily_task_text"
	PendingDailyTaskReport     PendingInputKind = "daily_task_report"
	PendingRoutinePlan         PendingInputKind = "routine_plan"
	PendingRoutineReason       PendingInputKind = "routine_reason"
	PendingSeasonalGoals       PendingInputKind = "seasonal_goals"
	PendingGoalWeeklyReview    PendingInputKind = "goal_weekly_review"
	PendingGoalFinalReflection PendingInputKind = "goal_final_reflection"
)

type RoutineItemStatus string

const (
	RoutineItemDone    RoutineItemStatus = "done"
	RoutineItemPartial RoutineItemStatus = "partial"
	RoutineItemFailed  RoutineItemStatus = "failed"
)

func (s RoutineItemStatus) IsValid() bool {
	return s == RoutineItemDone || s == RoutineItemPartial || s == RoutineItemFailed
}

type GoalFinalStatus string

const (
	GoalFinalDone    GoalFinalStatus = "done"
	GoalFinalPartial GoalFinalStatus = "partial"
	GoalFinalFailed  GoalFinalStatus = "failed"
)

func (s GoalFinalStatus) IsValid() bool {
	return s == GoalFinalDone || s == GoalFinalPartial || s == GoalFinalFailed
}
