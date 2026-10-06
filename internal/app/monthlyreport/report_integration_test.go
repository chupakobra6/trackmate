package monthlyreport

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
	"github.com/jackc/pgx/v5/pgxpool"
)

func fixture(t *testing.T) Service {
	t.Helper()
	_, url := testsupport.OpenMigratedStore(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	s := Service{DB: pool, Now: func() time.Time { return time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC) }}
	mustExec(t, s, `INSERT INTO workspace_groups(id,chat_id,title,timezone,setup_status) VALUES(1,-100123,'Test','Asia/Tokyo','ready'),(2,-100456,'Other','UTC','ready')`)
	mustExec(t, s, `INSERT INTO topic_bindings(workspace_group_id,topic_key,thread_id,topic_title) VALUES(1,'progress',7,'Progress'),(2,'progress',8,'Progress')`)
	mustExec(t, s, `INSERT INTO participants(id,workspace_group_id,user_id,display_name) VALUES(1,1,42,'Igor'),(2,2,43,'Other')`)
	return s
}
func mustExec(t *testing.T, s Service, sql string) {
	t.Helper()
	if _, err := s.DB.Exec(context.Background(), sql); err != nil {
		t.Fatal(err)
	}
}
func TestSnapshotCoverage(t *testing.T) {
	s := fixture(t)
	mustExec(t, s, `INSERT INTO daily_tasks(workspace_group_id,participant_id,owner_user_id,task_date,text,status,report_status,report_text,task_message_id,report_message_id) VALUES(1,1,42,'2026-09-01','first','done','done','made it',101,102),(1,1,42,'2026-09-30','auto','failed',NULL,NULL,103,NULL),(1,1,42,'2026-08-31','before','active',NULL,NULL,NULL,NULL),(1,1,42,'2026-10-01','after','active',NULL,NULL,NULL,NULL),(2,2,43,'2026-09-01','other','done','done','other',NULL,NULL)`)
	mustExec(t, s, `INSERT INTO routine_plans(workspace_group_id,participant_id,owner_user_id,items) VALUES(1,1,42,'["plan intention"]')`)
	mustExec(t, s, `INSERT INTO routine_checkins(id,workspace_group_id,participant_id,owner_user_id,checkin_date,reflection_text,source_message_id,auto_failed_at) VALUES(1,1,1,42,'2026-09-20','reflection',201,'2026-09-21'),(2,2,2,43,'2026-09-20','other',202,NULL)`)
	mustExec(t, s, `INSERT INTO routine_checkin_items(routine_checkin_id,item_index,text,status,reason_text) VALUES(1,0,'run','partial','short run'),(2,0,'other','done',NULL)`)
	mustExec(t, s, `INSERT INTO seasonal_goal_sets(id,workspace_group_id,participant_id,owner_user_id,period_key,period_title,period_starts_on,period_ends_on,goals_text,source_message_id) VALUES(1,1,1,42,'summer','summer','2026-06-01','2026-09-01','plan not achievement',301),(2,2,2,43,'fall','fall','2026-09-01','2026-12-01','other',302)`)
	// Exactly local September 1 is included; exactly local October 1 is excluded.
	mustExec(t, s, `INSERT INTO seasonal_goal_weekly_reviews(goal_set_id,review_week_start,response_text,requested_at,responded_at,response_message_id) VALUES(1,'2026-08-24','boundary included','2026-08-25','2026-08-31T15:00:00Z',303),(1,'2026-08-17','before','2026-08-18','2026-08-31T14:59:59Z',304),(1,'2026-10-05','after','2026-10-06','2026-09-30T15:00:00Z',305),(2,'2026-09-07','other','2026-09-08','2026-09-08',306)`)
	mustExec(t, s, `INSERT INTO seasonal_goal_final_reviews(goal_set_id,status,summary_text,requested_at,completed_at,summary_message_ids) VALUES(1,'partial','final evidence','2026-08-31T15:00:00Z','2026-09-02','{307}')`)
	out, err := s.Snapshot(context.Background(), 1, "2026-09")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]int{"participants": 1, "daily_tasks": 2, "routine_checkins": 1, "routine_checkin_items": 1, "seasonal_goal_sets": 1, "seasonal_goal_weekly_reviews": 1, "seasonal_goal_final_reviews": 1, "daily_reported_done": 1, "daily_automatically_failed_without_report": 1} {
		if out.Counts[key] != want {
			t.Errorf("%s=%d want %d", key, out.Counts[key], want)
		}
	}
	var daily []map[string]any
	if err = json.Unmarshal(out.Data["daily_tasks"], &daily); err != nil {
		t.Fatal(err)
	}
	if daily[0]["task_message_id"] != float64(101) || daily[0]["report_text"] != "made it" || daily[1]["report_status"] != nil {
		t.Fatal("lost source evidence", daily)
	}
	if out.Period.Start.UTC() != time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC) {
		t.Fatal(out.Period)
	}
}

type fakeSender struct {
	calls     atomic.Int32
	started   chan struct{}
	release   chan struct{}
	err       error
	mismatch  bool
	trimOuter bool
	afterSend func()
}

func (f *fakeSender) SendMessage(ctx context.Context, r telegram.SendMessageRequest) (telegram.Message, error) {
	f.calls.Add(1)
	if f.started != nil {
		close(f.started)
	}
	if f.release != nil {
		<-f.release
	}
	if f.err != nil {
		return telegram.Message{}, f.err
	}
	m := telegram.Message{MessageID: 999, Chat: telegram.Chat{ID: r.ChatID}, MessageThreadID: r.MessageThreadID, Text: html.UnescapeString(r.Text)}
	if f.trimOuter {
		m.Text = strings.TrimSpace(m.Text)
	}
	if f.mismatch {
		m.Text = "wrong"
	}
	if f.afterSend != nil {
		f.afterSend()
	}
	return m, nil
}
func TestPublishIdempotencyConcurrentAndStatus(t *testing.T) {
	s := fixture(t)
	f := &fakeSender{started: make(chan struct{}), release: make(chan struct{})}
	s.Sender = f
	r, err := s.Validate(context.Background(), 1, "2026-09", "report <&> 😀")
	if err != nil || r.ChatID != -100123 || r.ThreadID != 7 || r.Status != "not_published" {
		t.Fatal(r, err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var first Receipt
	var firstErr error
	go func() {
		defer wg.Done()
		first, firstErr = s.Publish(context.Background(), 1, "2026-09", "report <&> 😀")
	}()
	<-f.started
	state, err := s.Status(context.Background(), 1, "2026-09")
	if err != nil || state.Status != "publishing" || state.ClaimedAt == nil {
		t.Fatal(state, err)
	}
	if _, err = s.Publish(context.Background(), 1, "2026-09", "report <&> 😀"); err == nil {
		t.Fatal("concurrent send accepted")
	}
	close(f.release)
	wg.Wait()
	if firstErr != nil || first.Status != "published" || first.MessageID == nil || first.Link == "" {
		t.Fatal(first, firstErr)
	}
	again, err := s.Publish(context.Background(), 1, "2026-09", "report <&> 😀")
	if err != nil || *again.MessageID != 999 || f.calls.Load() != 1 {
		t.Fatal(again, err, f.calls.Load())
	}
	if _, err = s.Publish(context.Background(), 1, "2026-09", "changed"); err == nil {
		t.Fatal("changed message accepted")
	}
}
func TestPublishAmbiguousNeverRetries(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "receipt"}[mismatch], func(t *testing.T) {
			s := fixture(t)
			f := &fakeSender{mismatch: mismatch}
			if !mismatch {
				f.err = errors.New("timeout")
			}
			s.Sender = f
			r, err := s.Publish(context.Background(), 1, "2026-09", "report")
			if err == nil || r.Status != "uncertain" {
				t.Fatal(r, err)
			}
			for range 2 {
				if _, err = s.Publish(context.Background(), 1, "2026-09", "report"); err == nil {
					t.Fatal("uncertain retried")
				}
			}
			if f.calls.Load() != 1 {
				t.Fatal(f.calls.Load())
			}
		})
	}
}

func TestPublishPersistsAfterCallerCancellation(t *testing.T) {
	s := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeSender{afterSend: cancel}
	s.Sender = f
	r, err := s.Publish(ctx, 1, "2026-09", "accepted before cancellation")
	if err != nil || r.Status != "published" || r.MessageID == nil {
		t.Fatal(r, err)
	}
}
func TestPublishPersistenceFailureDoesNotResend(t *testing.T) {
	s := fixture(t)
	f := &fakeSender{afterSend: func() {
		mustExec(t, s, `CREATE FUNCTION reject_monthly_receipt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture persistence failure'; END $$; CREATE TRIGGER reject_monthly_receipt BEFORE UPDATE ON monthly_reports FOR EACH ROW EXECUTE FUNCTION reject_monthly_receipt()`)
	}}
	s.Sender = f
	if _, err := s.Publish(context.Background(), 1, "2026-09", "accepted without persisted receipt"); err == nil {
		t.Fatal("missing persistence error")
	}
	state, err := s.Status(context.Background(), 1, "2026-09")
	if err != nil || state.Status != "publishing" {
		t.Fatal(state, err)
	}
	if _, err = s.Publish(context.Background(), 1, "2026-09", "accepted without persisted receipt"); err == nil {
		t.Fatal("resent publishing")
	}
	if f.calls.Load() != 1 {
		t.Fatal(f.calls.Load())
	}
}

func TestPublishCanonicalTextSurvivesTelegramTrimming(t *testing.T) {
	s := fixture(t)
	f := &fakeSender{trimOuter: true}
	s.Sender = f
	text := "\n\t report <&> 😀\n\n  keep interior spaces  \nlast line\n "
	canonical := "report <&> 😀\n\n  keep interior spaces  \nlast line"
	_, hash, _ := Prepare(canonical)
	validated, err := s.Validate(context.Background(), 1, "2026-09", text)
	if err != nil || validated.Text != canonical || validated.Hash != hash {
		t.Fatal(validated, err)
	}
	published, err := s.Publish(context.Background(), 1, "2026-09", text)
	if err != nil || published.Status != "published" || published.Text != canonical || published.Hash != hash {
		t.Fatal(published, err)
	}
	persisted, err := s.Status(context.Background(), 1, "2026-09")
	if err != nil || persisted.Text != canonical || persisted.Hash != hash || persisted.Status != "published" {
		t.Fatal(persisted, err)
	}
	again, err := s.Publish(context.Background(), 1, "2026-09", canonical+"\n\t")
	if err != nil || again.Status != "published" || f.calls.Load() != 1 {
		t.Fatal(again, err, f.calls.Load())
	}
}
