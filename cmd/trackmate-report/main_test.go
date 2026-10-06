package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/igor/trackmate/internal/app/monthlyreport"
	"github.com/igor/trackmate/internal/testsupport"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReadOnlyCommandsWithoutToken(t *testing.T) {
	_, url := testsupport.OpenMigratedStore(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, sql := range []string{`INSERT INTO workspace_groups(id,chat_id,title,timezone,setup_status) VALUES(1,-100123,'CLI','UTC','ready')`, `INSERT INTO topic_bindings(workspace_group_id,topic_key,thread_id,topic_title) VALUES(1,'progress',7,'Progress')`} {
		if _, err = pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TRACKMATE__DATABASE_URL", url)
	t.Setenv("TRACKMATE__BOT_TOKEN", "replace-me")
	for _, command := range []string{"snapshot", "status", "validate", "publish"} {
		t.Run(command, func(t *testing.T) {
			args := []string{command, "--workspace", "1", "--month", "2026-09"}
			if command == "validate" || command == "publish" {
				args = append(args, "--file", "-")
				if command == "publish" {
					args = append(args, "--dry-run")
				}
			}
			var out, errOut bytes.Buffer
			if err := run(args, strings.NewReader(" \nmonthly <evidence> 😀\n\t"), &out, &errOut); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err = json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err, out.String())
			}
			if command != "snapshot" && (result["status"] != "not_published" || result["thread_id"] != float64(7)) {
				t.Fatal(result)
			}
			if (command == "validate" || command == "publish") && result["text"] != "monthly <evidence> 😀" {
				t.Fatal("CLI returned noncanonical text", result)
			}
		})
	}
	var count int
	if err = pool.QueryRow(context.Background(), `SELECT count(*) FROM monthly_reports`).Scan(&count); err != nil || count != 0 {
		t.Fatal("read-only commands mutated publication", count, err)
	}
	text := "already sent"
	_, hash, _ := monthlyreport.Prepare(text)
	if _, err = pool.Exec(context.Background(), `INSERT INTO monthly_reports(workspace_group_id,month,status,content_hash,text,chat_id,thread_id,message_id,published_at) VALUES(1,'2026-09-01','published',$1,$2,-100123,7,999,now())`, hash, text); err != nil {
		t.Fatal(err)
	}
	var receipt, errOut bytes.Buffer
	if err = run([]string{"publish", "--workspace", "1", "--month", "2026-09", "--file", "-"}, strings.NewReader(text), &receipt, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(receipt.String(), `"status": "published"`) || !strings.Contains(receipt.String(), `"message_id": 999`) {
		t.Fatal(receipt.String())
	}

}
func TestCommandInputValidation(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"publish", "--workspace", "1", "--month", "2026-09"}, {"snapshot", "--workspace", "1", "--month", "2026-09", "--file", "-"}, {"validate", "--workspace", "1", "--month", "2026-09", "--file", "-"}} {
		var out, errOut bytes.Buffer
		if err := run(args, strings.NewReader(""), &out, &errOut); err == nil {
			t.Fatal("accepted", args)
		}
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"--help"}, nil, &out, &errOut); err != nil || !strings.Contains(out.String(), "publish") {
		t.Fatal(out.String(), err)
	}
}
