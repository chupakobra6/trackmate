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
	Kind             domain.PendingInputKind
	Payload          map[string]any
	CreatedAt        time.Time
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

func encodePayload(payload map[string]any) ([]byte, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	return json.Marshal(payload)
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
