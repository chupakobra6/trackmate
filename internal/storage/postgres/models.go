package postgres

import (
	"encoding/json"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

type Workspace struct {
	ID             int64
	ChatID         int64
	Title          *string
	Timezone       string
	SetupStatus    domain.GroupSetupStatus
	SetupMessageID *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TopicBinding struct {
	ID               int64
	WorkspaceGroupID int64
	TopicKey         domain.TopicKey
	ThreadID         int64
	TopicTitle       string
	IntroMessageID   *int64
	ControlMessageID *int64
	CreatedAt        time.Time
}

type Participant struct {
	ID               int64
	WorkspaceGroupID int64
	UserID           int64
	Username         *string
	DisplayName      string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type DailyTask struct {
	ID                    int64
	WorkspaceGroupID      int64
	ParticipantID         int64
	OwnerUserID           int64
	TaskDate              time.Time
	Text                  string
	Status                domain.DailyTaskStatus
	ReportText            *string
	ReportStatus          *domain.DailyTaskStatus
	TodayCardMessageID    *int64
	TaskMessageID         *int64
	TaskMessageThreadID   *int64
	ReportMessageID       *int64
	ReportMessageThreadID *int64
	CreatedAt             time.Time
	ReportedAt            *time.Time
	AwaitingReportAt      *time.Time
	FailedAt              *time.Time
}

type DailyTaskAlert struct {
	ID                int64
	DailyTaskID       int64
	AlertKind         domain.AlertKind
	DispatchStatus    domain.AlertDispatchStatus
	TelegramMessageID *int64
	AcknowledgedAt    *time.Time
	CreatedAt         time.Time
}

type PendingInput struct {
	ID               int64
	WorkspaceGroupID int64
	UserID           int64
	MessageThreadID  int64
	Kind             domain.PendingInputKind
	Payload          map[string]any
	CreatedAt        time.Time
}

type PendingInputContext struct {
	Pending   PendingInput
	Workspace Workspace
}

type ProgressEvent struct {
	ID                 int64
	WorkspaceGroupID   int64
	ParticipantID      *int64
	DailyTaskID        *int64
	EventType          domain.ProgressEventType
	PublishStatus      domain.ProgressPublishStatus
	Payload            map[string]any
	PublishedMessageID *int64
	CreatedAt          time.Time
	PublishedAt        *time.Time
}

type RoutinePlan struct {
	ID               int64
	WorkspaceGroupID int64
	ParticipantID    int64
	OwnerUserID      int64
	Items            []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RoutineCheckin struct {
	ID                       int64
	WorkspaceGroupID         int64
	ParticipantID            int64
	OwnerUserID              int64
	CheckinDate              time.Time
	CardMessageID            *int64
	CardMessageThreadID      *int64
	ReminderMessageID        *int64
	AutoCloseNoticeMessageID *int64
	ReflectionText           *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	ReminderSentAt           *time.Time
	AutoCloseNoticeSentAt    *time.Time
	CompletedAt              *time.Time
	AutoFailedAt             *time.Time
	Items                    []RoutineCheckinItem
}

type RoutineCheckinItem struct {
	ID               int64
	RoutineCheckinID int64
	ItemIndex        int
	Text             string
	Status           *domain.RoutineItemStatus
	ReasonText       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RoutinePlanContext struct {
	Plan        RoutinePlan
	Workspace   Workspace
	Participant Participant
}

type RoutineCheckinContext struct {
	Checkin     RoutineCheckin
	Workspace   Workspace
	Participant Participant
}

type RoutineNoticeContext struct {
	Checkin   RoutineCheckin
	Workspace Workspace
}

type RoutineLeaderboardEntry struct {
	Participant      Participant
	CurrentStreak    int
	MaxStreak        int
	CompletionRate   float64
	RoutineItemCount int
}

type SeasonalGoalSet struct {
	ID                    int64
	WorkspaceGroupID      int64
	ParticipantID         int64
	OwnerUserID           int64
	PeriodKey             string
	PeriodTitle           string
	PeriodStartsOn        time.Time
	PeriodEndsOn          time.Time
	GoalsText             string
	CardMessageID         *int64
	CardMessageThreadID   *int64
	SourceMessageID       *int64
	SourceMessageThreadID *int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type SeasonalGoalSetContext struct {
	GoalSet     SeasonalGoalSet
	Workspace   Workspace
	Participant Participant
}

type GoalWeeklyReview struct {
	ID                    int64
	GoalSetID             int64
	ReviewWeekStart       time.Time
	PromptMessageID       *int64
	PromptMessageThreadID *int64
	ResponseText          *string
	RequestedAt           time.Time
	RespondedAt           *time.Time
}

type GoalFinalReview struct {
	ID                    int64
	GoalSetID             int64
	Status                *domain.GoalFinalStatus
	PromptMessageID       *int64
	PromptMessageThreadID *int64
	SummaryText           *string
	RequestedAt           time.Time
	CompletedAt           *time.Time
}

type GoalNudgeCooldown struct {
	WorkspaceGroupID int64
	ParticipantID    int64
	LastShownAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func int64FromPgInt4(value pgtype.Int4) *int64 {
	if !value.Valid {
		return nil
	}
	v := int64(value.Int32)
	return &v
}

func timeFromPg(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func textFromPg(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func statusFromPg(value pgtype.Text) *domain.DailyTaskStatus {
	if !value.Valid {
		return nil
	}
	status := domain.DailyTaskStatus(value.String)
	return &status
}

func routineStatusFromPg(value pgtype.Text) *domain.RoutineItemStatus {
	if !value.Valid {
		return nil
	}
	status := domain.RoutineItemStatus(value.String)
	return &status
}

func goalFinalStatusFromPg(value pgtype.Text) *domain.GoalFinalStatus {
	if !value.Valid {
		return nil
	}
	status := domain.GoalFinalStatus(value.String)
	return &status
}

func encodePayload(payload map[string]any) ([]byte, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	return json.Marshal(payload)
}

func copyPayload(payload map[string]any) map[string]any {
	copied := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		copied[key] = value
	}
	return copied
}

func decodePayload(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	return payload
}

func decodeStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

func payloadInt64(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func payloadString(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}
