package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/igor/trackmate/internal/app/monthlyreport"
	"github.com/igor/trackmate/internal/config"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "Usage: trackmate-report <snapshot|status|validate|publish> --workspace ID --month YYYY-MM [--file PATH|-] [--dry-run]\nReports accept one plain UTF-8 message, up to 4096 UTF-16 units. publish --dry-run is validate. Publishing or uncertain status requires manual reconciliation; never resend automatically.")
		return nil
	}
	command := args[0]
	switch command {
	case "snapshot", "status", "validate", "publish":
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(errOut)
	workspace := flags.Int64("workspace", 0, "database workspace id")
	month := flags.String("month", "", "completed calendar month YYYY-MM in workspace timezone")
	path := flags.String("file", "", "plain UTF-8 text file; - reads stdin")
	dry := flags.Bool("dry-run", false, "validate without sending")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *workspace <= 0 || *month == "" {
		return fmt.Errorf("--workspace ID and --month YYYY-MM are required; no positional arguments")
	}
	if (command == "snapshot" || command == "status") && (*path != "" || *dry) {
		return fmt.Errorf("--file and --dry-run apply only to validate/publish")
	}
	var text string
	if command == "validate" || command == "publish" {
		if *path == "" {
			return fmt.Errorf("--file is required")
		}
		var reader io.Reader = in
		var file *os.File
		if *path != "-" {
			var err error
			file, err = os.Open(*path)
			if err != nil {
				return err
			}
			defer file.Close()
			reader = file
		}
		data, err := io.ReadAll(io.LimitReader(reader, 65537))
		if err != nil {
			return err
		}
		if len(data) > 65536 {
			return fmt.Errorf("report input too large")
		}
		text = string(data)
		if _, _, err := monthlyreport.Prepare(text); err != nil {
			return err
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("cannot configure database")
	}
	defer pool.Close()
	service := monthlyreport.Service{DB: pool}
	if command == "publish" && !*dry {
		// Repeated publication can return its persisted receipt without credentials.
		state, statusErr := service.Status(ctx, *workspace, *month)
		if statusErr != nil {
			return statusErr
		}
		if state.Status == "not_published" {
			if err = cfg.RequireBotToken(); err != nil {
				return err
			}
			service.Sender = singleSend{telegram.NewClient(cfg.BotToken, nil)}
		}
	}
	var result any
	switch command {
	case "snapshot":
		result, err = service.Snapshot(ctx, *workspace, *month)
	case "status":
		result, err = service.Status(ctx, *workspace, *month)
	case "validate":
		result, err = service.Validate(ctx, *workspace, *month, text)
	case "publish":
		if *dry {
			result, err = service.Validate(ctx, *workspace, *month, text)
		} else {
			result, err = service.Publish(ctx, *workspace, *month, text)
		}
	}
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

// singleSend uses the existing transport without implicit send retries.
type singleSend struct{ client *telegram.Client }

func (s singleSend) SendMessage(ctx context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	return s.client.SendMessageOnce(ctx, request)
}
