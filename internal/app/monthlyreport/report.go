// Package monthlyreport exposes evidence and sends a prepared, single monthly message.
// It never generates prose and never retries an uncertain delivery.
package monthlyreport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Sender interface {
	SendMessage(context.Context, telegram.SendMessageRequest) (telegram.Message, error)
}
type Service struct {
	DB     *pgxpool.Pool
	Sender Sender
	Now    func() time.Time
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type Period struct {
	Month    string    `json:"month"`
	Timezone string    `json:"timezone"`
	Start    time.Time `json:"start_inclusive"`
	End      time.Time `json:"end_exclusive"`
}

func Month(month, timezone string, now time.Time) (Period, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return Period{}, err
	}
	start, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil || start.Format("2006-01") != month {
		return Period{}, fmt.Errorf("month must be YYYY-MM")
	}
	end := start.AddDate(0, 1, 0)
	if end.After(now) {
		return Period{}, fmt.Errorf("month is not complete in workspace timezone")
	}
	return Period{month, timezone, start, end}, nil
}

// Prepare validates and canonicalizes the message before hashing or HTML escaping.
// Outer whitespace is removed to match Telegram's sendMessage normalization.
func Prepare(text string) (escaped, hash string, err error) {
	prepared, err := prepare(text)
	return prepared.html, prepared.hash, err
}

type preparedMessage struct{ text, html, hash string }

func prepare(text string) (preparedMessage, error) {
	if !utf8.ValidString(text) {
		return preparedMessage{}, fmt.Errorf("report must be valid UTF-8")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return preparedMessage{}, fmt.Errorf("report is empty")
	}
	if len(utf16.Encode([]rune(text))) > 4096 {
		return preparedMessage{}, fmt.Errorf("report exceeds 4096 UTF-16 units")
	}
	sum := sha256.Sum256([]byte(text))
	return preparedMessage{text, html.EscapeString(text), hex.EncodeToString(sum[:])}, nil
}

type Snapshot struct {
	Period         Period                     `json:"period"`
	AsOf           time.Time                  `json:"as_of"`
	Workspace      json.RawMessage            `json:"workspace"`
	Data           map[string]json.RawMessage `json:"data"`
	Counts         map[string]int             `json:"counts"`
	Interpretation []string                   `json:"interpretation"`
}

func (s Service) Snapshot(ctx context.Context, workspace int64, month string) (Snapshot, error) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Snapshot{}, err
	}
	defer tx.Rollback(ctx)
	var out Snapshot
	var timezone string
	err = tx.QueryRow(ctx, `SELECT timezone,to_jsonb(w) FROM workspace_groups w WHERE id=$1`, workspace).Scan(&timezone, &out.Workspace)
	if err != nil {
		return out, err
	}
	out.Period, err = Month(month, timezone, s.now())
	if err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `SELECT transaction_timestamp()`).Scan(&out.AsOf); err != nil {
		return out, err
	}
	out.Data = map[string]json.RawMessage{}
	out.Counts = map[string]int{}
	out.Interpretation = []string{"Plans are intentions, not achievements. Only explicit reports/reviews support achievements.", "Daily status=failed with report_status=null is automatic failure, not a participant's reported outcome.", "Routine failed items may be auto-closed; inspect reason_text and auto_failed_at, not status alone.", "This is current database state, including late reports and edits; it is not a historical month-end reconstruction.", "Routine plans are current context only; checkin item text preserves the recorded routine."}
	queries := []struct{ name, sql string }{
		{"participants", `SELECT to_jsonb(p) FROM participants p WHERE workspace_group_id=$1 ORDER BY id`},
		{"daily_tasks", `SELECT to_jsonb(d) FROM daily_tasks d WHERE workspace_group_id=$1 AND task_date >= $2::date AND task_date < $3::date ORDER BY task_date,id`},
		{"routine_plans_context", `SELECT to_jsonb(p) FROM routine_plans p WHERE workspace_group_id=$1 ORDER BY id`},
		{"routine_checkins", `SELECT to_jsonb(c) FROM routine_checkins c WHERE workspace_group_id=$1 AND checkin_date >= $2::date AND checkin_date < $3::date ORDER BY checkin_date,id`},
		{"routine_checkin_items", `SELECT to_jsonb(i) FROM routine_checkin_items i JOIN routine_checkins c ON c.id=i.routine_checkin_id WHERE c.workspace_group_id=$1 AND c.checkin_date >= $2::date AND c.checkin_date < $3::date ORDER BY c.checkin_date,c.id,i.item_index`},
		{"seasonal_goal_sets", `SELECT to_jsonb(g) FROM seasonal_goal_sets g WHERE workspace_group_id=$1 AND ((period_starts_on < $3::date AND period_ends_on >= $2::date) OR EXISTS (SELECT 1 FROM seasonal_goal_weekly_reviews r WHERE r.goal_set_id=g.id AND ((r.review_week_start < $3::date AND r.review_week_start+7 > $2::date) OR (r.responded_at >= $4 AND r.responded_at < $5) OR (r.requested_at >= $4 AND r.requested_at < $5))) OR EXISTS (SELECT 1 FROM seasonal_goal_final_reviews r WHERE r.goal_set_id=g.id AND ((r.completed_at >= $4 AND r.completed_at < $5) OR (r.requested_at >= $4 AND r.requested_at < $5)))) ORDER BY id`},
		{"seasonal_goal_weekly_reviews", `SELECT to_jsonb(r) FROM seasonal_goal_weekly_reviews r JOIN seasonal_goal_sets g ON g.id=r.goal_set_id WHERE g.workspace_group_id=$1 AND ((r.review_week_start < $3::date AND r.review_week_start+7 > $2::date) OR (r.responded_at >= $4 AND r.responded_at < $5) OR (r.requested_at >= $4 AND r.requested_at < $5)) ORDER BY r.id`},
		{"seasonal_goal_final_reviews", `SELECT to_jsonb(r) FROM seasonal_goal_final_reviews r JOIN seasonal_goal_sets g ON g.id=r.goal_set_id WHERE g.workspace_group_id=$1 AND ((r.completed_at >= $4 AND r.completed_at < $5) OR (r.requested_at >= $4 AND r.requested_at < $5) OR (g.period_ends_on >= $2::date AND g.period_ends_on < $3::date)) ORDER BY r.id`},
	}
	for _, q := range queries {
		// Parameter count follows query's highest placeholder; date strings avoid driver/session timezone coercion.
		args := []any{workspace}
		if strings.Contains(q.sql, "$5") {
			args = append(args, out.Period.Start.Format("2006-01-02"), out.Period.End.Format("2006-01-02"), out.Period.Start, out.Period.End)
		} else if strings.Contains(q.sql, "$3") {
			args = append(args, out.Period.Start.Format("2006-01-02"), out.Period.End.Format("2006-01-02"))
		}
		rows, e := tx.Query(ctx, q.sql, args...)
		if e != nil {
			return out, e
		}
		values := []json.RawMessage{}
		for rows.Next() {
			var v json.RawMessage
			if e = rows.Scan(&v); e != nil {
				rows.Close()
				return out, e
			}
			values = append(values, v)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		out.Counts[q.name] = len(values)
		for _, value := range values {
			var item map[string]any
			if e = json.Unmarshal(value, &item); e != nil {
				return out, e
			}
			switch q.name {
			case "daily_tasks":
				outcome, _ := item["report_status"].(string)
				if outcome != "" {
					out.Counts["daily_reported_"+outcome]++
				} else if item["status"] == "failed" {
					out.Counts["daily_automatically_failed_without_report"]++
				} else {
					out.Counts["daily_without_report"]++
				}
			case "routine_checkin_items":
				outcome, _ := item["status"].(string)
				if outcome == "" {
					outcome = "unmarked"
				}
				out.Counts["routine_item_"+outcome]++
			case "routine_checkins":
				if item["auto_failed_at"] != nil {
					out.Counts["routine_auto_closed"]++
				}
				if item["reflection_text"] != nil {
					out.Counts["routine_reflections"]++
				}
			case "seasonal_goal_weekly_reviews":
				if item["responded_at"] != nil {
					out.Counts["weekly_responses"]++
				}
			case "seasonal_goal_final_reviews":
				if item["completed_at"] != nil {
					out.Counts["final_responses"]++
				}
			}
		}
		out.Data[q.name], err = json.Marshal(values)
		if err != nil {
			return out, err
		}
	}
	err = tx.Commit(ctx)
	return out, err
}

type Receipt struct {
	Workspace   int64      `json:"workspace"`
	Month       string     `json:"month"`
	Status      string     `json:"status"`
	ChatID      int64      `json:"chat_id"`
	ThreadID    int64      `json:"thread_id"`
	Text        string     `json:"text"`
	HTML        string     `json:"html,omitempty"`
	Hash        string     `json:"content_hash"`
	MessageID   *int64     `json:"message_id,omitempty"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Link        string     `json:"link,omitempty"`
	LastError   *string    `json:"last_error,omitempty"`
}

func (s Service) target(ctx context.Context, workspace int64, month string) (Receipt, error) {
	r := Receipt{Workspace: workspace, Month: month, Status: "not_published"}
	var timezone string
	err := s.DB.QueryRow(ctx, `SELECT w.chat_id,t.thread_id,w.timezone FROM workspace_groups w JOIN topic_bindings t ON t.workspace_group_id=w.id AND t.topic_key='progress' WHERE w.id=$1 AND w.setup_status='ready'`, workspace).Scan(&r.ChatID, &r.ThreadID, &timezone)
	if err != nil {
		return r, fmt.Errorf("ready workspace and progress topic required: %w", err)
	}
	_, err = Month(month, timezone, s.now())
	if r.ThreadID <= 0 && err == nil {
		err = fmt.Errorf("progress topic must have a positive thread id")
	}
	return r, err
}
func (s Service) Status(ctx context.Context, workspace int64, month string) (Receipt, error) {
	r, err := s.target(ctx, workspace, month)
	if err != nil {
		return r, err
	}
	err = s.DB.QueryRow(ctx, `SELECT status,chat_id,thread_id,text,content_hash,message_id,claimed_at,published_at,last_error FROM monthly_reports WHERE workspace_group_id=$1 AND month=$2::date`, workspace, month+"-01").Scan(&r.Status, &r.ChatID, &r.ThreadID, &r.Text, &r.Hash, &r.MessageID, &r.ClaimedAt, &r.PublishedAt, &r.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	if r.MessageID != nil {
		r.Link = postgres.MessageLink(r.ChatID, *r.MessageID, r.ThreadID)
	}
	return r, err
}
func (s Service) Validate(ctx context.Context, workspace int64, month, text string) (Receipt, error) {
	prepared, err := prepare(text)
	if err != nil {
		return Receipt{}, err
	}
	r, err := s.Status(ctx, workspace, month)
	if err != nil {
		return r, err
	}
	if r.Status != "not_published" && r.Hash != prepared.hash {
		return r, fmt.Errorf("month already reserved with different content; reconciliation required")
	}
	r.Text = prepared.text
	r.HTML = prepared.html
	r.Hash = prepared.hash
	return r, nil
}
func (s Service) Publish(ctx context.Context, workspace int64, month, text string) (Receipt, error) {
	r, err := s.Validate(ctx, workspace, month, text)
	if err != nil {
		return r, err
	}
	if r.Status == "published" {
		return r, nil
	}
	if r.Status != "not_published" {
		return r, fmt.Errorf("delivery %s; reconciliation required, do not resend", r.Status)
	}
	if s.Sender == nil {
		return r, fmt.Errorf("sender required")
	}
	tag, err := s.DB.Exec(ctx, `INSERT INTO monthly_reports(workspace_group_id,month,status,content_hash,text,chat_id,thread_id) VALUES($1,$2::date,'publishing',$3,$4,$5,$6) ON CONFLICT DO NOTHING`, workspace, month+"-01", r.Hash, r.Text, r.ChatID, r.ThreadID)
	if err != nil {
		return r, err
	}
	if tag.RowsAffected() != 1 {
		expectedHash := r.Hash
		r, e := s.Status(ctx, workspace, month)
		if e != nil {
			return r, e
		}
		if r.Status == "published" && r.Hash == expectedHash {
			return r, nil
		}
		return r, fmt.Errorf("month reserved concurrently; reconciliation required")
	}
	msg, sendErr := s.Sender.SendMessage(ctx, telegram.SendMessageRequest{ChatID: r.ChatID, MessageThreadID: r.ThreadID, Text: r.HTML, ParseMode: "HTML"})
	status := "published"
	var failure any
	if sendErr != nil {
		status = "uncertain"
		failure = "send failed; verify Telegram and reconcile manually"
	} else if msg.MessageID <= 0 || msg.Chat.ID != r.ChatID || msg.MessageThreadID != r.ThreadID || msg.Text != r.Text {
		status = "uncertain"
		sendErr = fmt.Errorf("Telegram receipt does not match recipient, topic, text and message id")
		failure = sendErr.Error()
	}
	// Persist even if the caller's deadline expired after send. Never automatically reclaim a publishing row.
	saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var messageID any
	if msg.MessageID > 0 && msg.Chat.ID == r.ChatID && msg.MessageThreadID == r.ThreadID {
		messageID = msg.MessageID
	}
	saved, saveErr := s.DB.Exec(saveCtx, `UPDATE monthly_reports SET status=$3,message_id=$4,last_error=$5,published_at=CASE WHEN $3='published' THEN now() ELSE NULL END WHERE workspace_group_id=$1 AND month=$2::date AND status='publishing'`, workspace, month+"-01", status, messageID, failure)
	if saveErr != nil || saved.RowsAffected() != 1 {
		return r, fmt.Errorf("delivery receipt persistence failed; reconciliation required")
	}
	r, err = s.Status(saveCtx, workspace, month)
	if sendErr != nil {
		return r, fmt.Errorf("delivery uncertain; reconciliation required")
	}
	return r, err
}
