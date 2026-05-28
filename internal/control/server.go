package control

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/worker"
)

type Server struct {
	Store  *postgres.Store
	Worker *worker.Runner
	Logger *slog.Logger
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /control/reset", s.handleReset)
	mux.HandleFunc("GET /control/topics", s.handleTopics)
	mux.HandleFunc("POST /control/clock", s.handleClock)
	mux.HandleFunc("POST /control/tick", s.handleTick)
	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if s.Logger != nil {
		s.Logger.InfoContext(ctx, "control_server_starting", "addr", addr)
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	chatID, ok := int64Param(w, r, "chat_id")
	if !ok {
		return
	}
	var result postgres.ResetWorkspaceResult
	err := s.Store.InTx(r.Context(), func(q *postgres.Queries) error {
		var err error
		result, err = q.ResetWorkspaceForE2E(r.Context(), chatID)
		return err
	})
	writeJSON(w, result, err)
}

func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	chatID, ok := int64Param(w, r, "chat_id")
	if !ok {
		return
	}
	bindings, err := s.Store.Queries().ActiveTopicBindings(r.Context(), chatID)
	writeJSON(w, bindings, err)
}

func (s *Server) handleClock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Now string `json:"now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var value *time.Time
	if body.Now != "" {
		parsed, err := time.Parse(time.RFC3339, body.Now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		parsed = parsed.UTC()
		value = &parsed
	}
	err := s.Store.InTx(r.Context(), func(q *postgres.Queries) error {
		return q.SetClockOverride(r.Context(), value)
	})
	writeJSON(w, map[string]any{"ok": err == nil, "now": body.Now}, err)
}

func (s *Server) handleTick(w http.ResponseWriter, r *http.Request) {
	err := s.Worker.Tick(r.Context(), time.Now().UTC())
	writeJSON(w, map[string]any{"ok": err == nil}, err)
}

func int64Param(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := r.URL.Query().Get(key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s query parameter is required", key), http.StatusBadRequest)
		return 0, false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}
