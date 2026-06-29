package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appsetup "github.com/igor/trackmate/internal/app/setup"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

type CallbackAnswer struct {
	ID   string
	Text string
}

type Service struct {
	Store           *postgres.Store
	Telegram        telegram.API
	Setup           *appsetup.Service
	Logger          *slog.Logger
	DefaultTimezone string
}

func NewService(store *postgres.Store, tg telegram.API, logger *slog.Logger, defaultTimezone string, botID int64) *Service {
	setupSvc := &appsetup.Service{Store: store, Telegram: tg, BotID: botID, DefaultTimezone: defaultTimezone}
	return &Service{Store: store, Telegram: tg, Setup: setupSvc, Logger: logger, DefaultTimezone: defaultTimezone}
}

func (s *Service) HandleUpdate(ctx context.Context, update telegram.Update) (CallbackAnswer, error) {
	if update.MyChatMember != nil {
		return CallbackAnswer{}, s.handleMyChatMember(ctx, *update.MyChatMember)
	}
	if update.Message != nil {
		return CallbackAnswer{}, s.handleMessage(ctx, *update.Message)
	}
	if update.EditedMessage != nil {
		return CallbackAnswer{}, s.handleEditedMessage(ctx, *update.EditedMessage)
	}
	if update.Callback != nil {
		answer, err := s.handleCallback(ctx, *update.Callback)
		answer.ID = update.Callback.ID
		return answer, err
	}
	return CallbackAnswer{}, nil
}

func (s *Service) handleMyChatMember(ctx context.Context, event telegram.ChatMemberUpdated) error {
	if event.Chat.Type != "group" && event.Chat.Type != "supergroup" {
		return nil
	}
	if event.NewChatMember.Status != "member" && event.NewChatMember.Status != "administrator" {
		return nil
	}
	return s.upsertSetupMessage(ctx, event.Chat.ID, event.Chat.Title, nil, "")
}

func (s *Service) handleMessage(ctx context.Context, message telegram.Message) error {
	if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
		return nil
	}
	if isCommand(message.Text, "/setup") {
		return s.upsertSetupMessage(ctx, message.Chat.ID, message.Chat.Title, nil, "")
	}
	if message.From == nil {
		return nil
	}
	return s.handlePendingInputMessage(ctx, message)
}

func (s *Service) handleEditedMessage(ctx context.Context, message telegram.Message) error {
	if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
		return nil
	}
	if message.From == nil {
		return nil
	}
	workspace, err := s.ensureWorkspaceLoaded(ctx, message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return err
	}
	input := telegram.NewMessageInput(message)
	var task postgres.DailyTask
	var progressEvents []postgres.ProgressEvent
	var found bool
	if err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		updated, events, ok, err := q.UpdateTaskTextFromSourceMessage(ctx, workspace.ID, input.Source.UserID, input.Source.MessageID, input.Source.ThreadID, input.TextHTML)
		if err != nil {
			return err
		}
		if ok {
			task = updated
			progressEvents = events
			found = true
			return nil
		}
		updated, events, ok, err = q.UpdateTaskReportFromSourceMessage(ctx, workspace.ID, input.Source.UserID, input.Source.MessageID, input.Source.ThreadID, input.TextHTML)
		if err != nil {
			return err
		}
		if ok {
			task = updated
			progressEvents = events
			found = true
		}
		return nil
	}); err != nil {
		return err
	}
	if !found {
		return nil
	}
	if task.TodayCardMessageID != nil {
		if err := s.editMessageOrQueueProgressAlert(ctx, workspace, telegram.EditMessageTextRequest{
			ChatID:      message.Chat.ID,
			MessageID:   *task.TodayCardMessageID,
			Text:        ui.FormatDailyTaskCard(task, telegram.DisplayName(*message.From), message.From.Username, ""),
			ReplyMarkup: dailyTaskCardKeyboard(task),
		}, messages.Text("progress.edit_failed.target_today_card"), optionalInt64(task.TaskMessageThreadID)); err != nil {
			return err
		}
	}
	for _, event := range progressEvents {
		if event.PublishedMessageID == nil {
			continue
		}
		if err := s.editMessageOrQueueProgressAlert(ctx, workspace, telegram.EditMessageTextRequest{
			ChatID:    message.Chat.ID,
			MessageID: *event.PublishedMessageID,
			Text:      ui.FormatProgressEvent(event),
		}, messages.Text("progress.edit_failed.target_progress_message"), 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) handleCallback(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	if callback.Message == nil {
		return CallbackAnswer{Text: messages.Text("callback.stale_button")}, nil
	}
	parsed, err := domain.ParseCallback(callback.Data)
	if err != nil {
		return CallbackAnswer{Text: messages.Text("callback.stale_button")}, nil
	}
	switch parsed.Kind {
	case domain.CallbackSetupCheck:
		return CallbackAnswer{}, s.upsertSetupMessage(ctx, callback.Message.Chat.ID, callback.Message.Chat.Title, callback.Message, "")
	case domain.CallbackSetupStart:
		return s.handleSetupStart(ctx, callback)
	case domain.CallbackTodayAdd:
		return s.handleTodayAdd(ctx, callback)
	case domain.CallbackTaskReport:
		return s.handleTaskReport(ctx, callback, parsed.TaskID)
	case domain.CallbackTaskStatus:
		return s.handleTaskStatus(ctx, callback, parsed.TaskID, parsed.TaskStatus)
	case domain.CallbackAlertAck:
		return s.handleAlertAck(ctx, callback, parsed.AlertID)
	case domain.CallbackRoutineConfigure:
		return s.handleRoutineConfigure(ctx, callback)
	case domain.CallbackRoutineItem:
		return s.handleRoutineItem(ctx, callback, parsed.RoutineCheckinID, parsed.RoutineItemIndex, parsed.RoutineItemStatus)
	case domain.CallbackGoalsConfigure:
		return s.handleGoalsConfigure(ctx, callback)
	case domain.CallbackGoalFinalStatus:
		return s.handleGoalFinalStatus(ctx, callback, parsed.GoalSetID, parsed.GoalFinalStatus)
	case domain.CallbackNoticeDismiss:
		return s.handleNoticeDismiss(ctx, callback)
	default:
		return CallbackAnswer{Text: messages.Text("callback.stale_button")}, nil
	}
}

func (s *Service) upsertSetupMessage(ctx context.Context, chatID int64, chatTitle string, fallback *telegram.Message, notice string) error {
	var workspace postgres.Workspace
	var prerequisites appsetup.Prerequisites
	if err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		var err error
		workspace, err = q.GetOrCreateWorkspace(ctx, chatID, chatTitle, s.DefaultTimezone)
		if err != nil {
			return err
		}
		prerequisites, err = s.Setup.CheckPrerequisites(ctx, chatID)
		return err
	}); err != nil {
		return err
	}
	text := ui.FormatSetupChecklist(
		prerequisites.IsReady(),
		prerequisites.IsSupergroup,
		prerequisites.IsForum,
		prerequisites.BotIsAdmin,
		prerequisites.CanManageTopics,
		prerequisites.CanReadMessages,
		notice,
	)
	if workspace.SetupMessageID != nil {
		if ok := s.editMessageSafe(ctx, chatID, *workspace.SetupMessageID, text, ui.SetupKeyboard()); ok {
			if fallback != nil && fallback.MessageID != *workspace.SetupMessageID {
				_ = s.Telegram.DeleteMessage(ctx, chatID, fallback.MessageID)
			}
			return nil
		}
	}
	if fallback != nil {
		if ok := s.editMessageSafe(ctx, chatID, fallback.MessageID, text, ui.SetupKeyboard()); ok {
			return s.Store.InTx(ctx, func(q *postgres.Queries) error {
				return q.SetSetupMessageID(ctx, workspace.ID, fallback.MessageID)
			})
		}
	}
	message, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:              chatID,
		Text:                text,
		ReplyMarkup:         ui.SetupKeyboard(),
		DisableNotification: true,
	})
	if err != nil {
		return err
	}
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		return q.SetSetupMessageID(ctx, workspace.ID, message.MessageID)
	})
}

func (s *Service) handleSetupStart(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	chat := callback.Message.Chat
	isAdmin, err := s.Setup.IsGroupAdmin(ctx, chat.ID, callback.From.ID)
	if err != nil {
		return CallbackAnswer{}, err
	}
	if !isAdmin {
		return CallbackAnswer{Text: messages.Text("callback.setup.admin_only")}, nil
	}
	prerequisites, err := s.Setup.CheckPrerequisites(ctx, chat.ID)
	if err != nil {
		return CallbackAnswer{}, err
	}
	if !prerequisites.IsReady() {
		return CallbackAnswer{Text: messages.Text("callback.setup.not_ready")}, nil
	}
	workspace, err := s.ensureWorkspace(ctx, chat.ID, chat.Title)
	if err != nil {
		return CallbackAnswer{}, err
	}
	topicIDs, changed, err := s.Setup.EnsureWorkspaceTopics(ctx, chat.ID, chat.Title, workspace.Timezone)
	if err != nil {
		return CallbackAnswer{}, err
	}
	bindings, err := s.Store.Queries().ListTopicBindings(ctx, workspace.ID)
	if err != nil {
		return CallbackAnswer{}, err
	}
	if binding, ok := bindings[domain.TopicToday]; ok {
		if s.ensureTopicMessage(ctx, workspace.ID, chat.ID, topicIDs[domain.TopicToday], domain.TopicToday, binding.ControlMessageID, ui.TodayControlText, ui.TodayControlKeyboard(), true, true) {
			changed = true
		}
	}
	if binding, ok := bindings[domain.TopicRoutine]; ok {
		if s.ensureTopicMessage(ctx, workspace.ID, chat.ID, topicIDs[domain.TopicRoutine], domain.TopicRoutine, binding.ControlMessageID, ui.RoutineControlText, ui.RoutineControlKeyboard(), true, true) {
			changed = true
		}
		if s.ensureTopicMessage(ctx, workspace.ID, chat.ID, topicIDs[domain.TopicRoutine], domain.TopicRoutine, binding.IntroMessageID, ui.FormatRoutineLeaderboard(nil), nil, false, false) {
			changed = true
		}
	}
	if binding, ok := bindings[domain.TopicGoals]; ok {
		if s.ensureTopicMessage(ctx, workspace.ID, chat.ID, topicIDs[domain.TopicGoals], domain.TopicGoals, binding.ControlMessageID, ui.GoalsControlText, ui.GoalsControlKeyboard(), true, true) {
			changed = true
		}
	}
	if binding, ok := bindings[domain.TopicProgress]; ok {
		if s.ensureTopicMessage(ctx, workspace.ID, chat.ID, topicIDs[domain.TopicProgress], domain.TopicProgress, binding.IntroMessageID, ui.ProgressIntroText, nil, false, false) {
			changed = true
		}
	}
	text := ui.SetupReadyText
	if changed {
		text = ui.SetupRepairedText
	}
	_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{ChatID: chat.ID, MessageID: callback.Message.MessageID, Text: text})
	_ = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		return q.SetSetupMessageID(ctx, workspace.ID, callback.Message.MessageID)
	})
	return CallbackAnswer{}, nil
}

func (s *Service) handleTodayAdd(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{}, err
	}
	user := callback.From
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		participant, err := q.RegisterParticipant(ctx, workspace.ID, user.ID, user.Username, telegram.DisplayName(user))
		if err != nil {
			return err
		}
		now, err := q.CurrentNow(ctx, time.Now().UTC())
		if err != nil {
			return err
		}
		taskDate, err := domain.LocalTaskDate(workspace.Timezone, now)
		if err != nil {
			return err
		}
		if _, found, err := q.GetTaskForDate(ctx, workspace.ID, participant.ID, taskDate); err != nil {
			return err
		} else if found {
			answer.Text = messages.Text("callback.today.exists")
			return nil
		}
		if _, found, err := q.GetOpenTask(ctx, workspace.ID, participant.ID); err != nil {
			return err
		} else if found {
			answer.Text = messages.Text("callback.today.close_previous")
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID, callback.Message.MessageThreadID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		nudge, err := s.goalNudge(ctx, q, workspace, participant, "task_text:"+taskDate.Format("2006-01-02"), "")
		if err != nil {
			return err
		}
		prompt, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              callback.Message.Chat.ID,
			MessageThreadID:     callback.Message.MessageThreadID,
			Text:                ui.DailyTaskTextPrompt(nudge),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		_, err = q.UpsertPendingInput(ctx, workspace.ID, user.ID, callback.Message.MessageThreadID, domain.PendingDailyTaskText, map[string]any{
			"thread_id":         callback.Message.MessageThreadID,
			"prompt_message_id": prompt.MessageID,
		})
		return err
	})
	return answer, err
}

func (s *Service) handlePendingInputMessage(ctx context.Context, message telegram.Message) error {
	workspace, err := s.ensureWorkspaceLoaded(ctx, message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return err
	}
	pending, found, err := s.Store.Queries().GetPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID)
	if err != nil || !found {
		return err
	}
	switch pending.Kind {
	case domain.PendingDailyTaskText:
		return s.consumeDailyTaskText(ctx, workspace, message)
	case domain.PendingDailyTaskReport:
		return s.consumeDailyTaskReport(ctx, workspace, message)
	case domain.PendingRoutinePlan:
		return s.consumeRoutinePlan(ctx, workspace, message, pending)
	case domain.PendingRoutineReason:
		return s.consumeRoutineReason(ctx, workspace, message)
	case domain.PendingSeasonalGoals:
		return s.consumeSeasonalGoals(ctx, workspace, message, pending)
	case domain.PendingGoalWeeklyReview:
		return s.consumeGoalWeeklyReview(ctx, workspace, message)
	case domain.PendingGoalFinalReflection:
		return s.consumeGoalFinalReflection(ctx, workspace, message)
	default:
		return nil
	}
}

func (s *Service) consumeDailyTaskText(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingDailyTaskText)
		if err != nil || !ok {
			return err
		}
		_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"))
		user := *message.From
		participant, err := q.RegisterParticipant(ctx, workspace.ID, user.ID, user.Username, telegram.DisplayName(user))
		if err != nil {
			return err
		}
		now, err := q.CurrentNow(ctx, time.Now().UTC())
		if err != nil {
			return err
		}
		taskDate, err := domain.LocalTaskDate(workspace.Timezone, now)
		if err != nil {
			return err
		}
		input := telegram.NewMessageInput(message)
		task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, user.ID, taskDate, input.TextHTML, input.Source.MessageID, input.Source.ThreadID)
		if err != nil {
			return err
		}
		if !created {
			text := "⚠️ <b>" + messages.Text("callback.today.exists") + "</b>"
			if task.ID == 0 || task.TaskDate.Format("2006-01-02") != taskDate.Format("2006-01-02") {
				text = "⚠️ <b>" + messages.Text("callback.today.close_previous") + "</b>"
			}
			_, _ = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
				ChatID:              message.Chat.ID,
				MessageThreadID:     message.MessageThreadID,
				Text:                text,
				ReplyMarkup:         ui.DismissKeyboard(),
				DisableNotification: true,
			})
			return nil
		}
		card, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              message.Chat.ID,
			MessageThreadID:     message.MessageThreadID,
			Text:                ui.FormatDailyTaskCard(task, telegram.DisplayName(user), user.Username, ""),
			ReplyMarkup:         ui.DailyTaskKeyboard(task.ID),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		return q.SetDailyTaskCardMessageID(ctx, task.ID, card.MessageID)
	})
}

func (s *Service) consumeDailyTaskReport(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingDailyTaskReport)
		if err != nil || !ok {
			return err
		}
		taskID := payloadInt64(pending.Payload, "task_id")
		status := domain.DailyTaskStatus(payloadString(pending.Payload, "status"))
		input := telegram.NewMessageInput(message)
		submitted, err := q.SubmitTaskReport(ctx, taskID, message.From.ID, status, input.TextHTML, telegram.DisplayName(*message.From), input.Source.MessageID, input.Source.ThreadID)
		if err != nil {
			return err
		}
		if !submitted {
			text := messages.Text("task.report.rejected")
			if task, found, err := q.GetTask(ctx, taskID); err == nil && found && !task.Status.IsOpen() {
				text = messages.Text("task.report.rejected_closed")
			}
			if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, ui.DismissKeyboard()) {
				_, _ = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{ChatID: message.Chat.ID, MessageThreadID: message.MessageThreadID, Text: text, ReplyMarkup: ui.DismissKeyboard(), DisableNotification: true})
			}
			return nil
		}
		if err := s.dismissTaskAlerts(ctx, q, message.Chat.ID, taskID); err != nil {
			return err
		}
		task, found, err := q.GetTask(ctx, taskID)
		if err != nil {
			return err
		}
		if found {
			_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
				ChatID:    message.Chat.ID,
				MessageID: optionalInt64(task.TodayCardMessageID),
				Text:      ui.FormatDailyTaskCard(task, telegram.DisplayName(*message.From), message.From.Username, ""),
			})
		}
		text := messages.Text("task.report.saved")
		if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, ui.DismissKeyboard()) {
			_, _ = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{ChatID: message.Chat.ID, MessageThreadID: message.MessageThreadID, Text: text, ReplyMarkup: ui.DismissKeyboard(), DisableNotification: true})
		}
		return nil
	})
}

func (s *Service) handleTaskReport(ctx context.Context, callback telegram.CallbackQuery, taskID int64) (CallbackAnswer, error) {
	var answer CallbackAnswer
	err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		task, found, err := q.GetTask(ctx, taskID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = messages.Text("callback.task.not_found")
			return nil
		}
		if callback.From.ID != task.OwnerUserID {
			answer.Text = messages.Text("callback.task.author_only")
			return nil
		}
		if !task.Status.IsOpen() {
			if err := s.dismissTaskAlerts(ctx, q, callback.Message.Chat.ID, taskID); err != nil {
				return err
			}
			answer.Text = messages.Text("callback.task.closed")
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, task.WorkspaceGroupID, callback.From.ID, callback.Message.MessageThreadID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		_, err = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              callback.Message.Chat.ID,
			MessageThreadID:     callback.Message.MessageThreadID,
			Text:                messages.Text("task.status.prompt"),
			ReplyMarkup:         ui.DailyTaskStatusKeyboard(taskID),
			DisableNotification: true,
		})
		return err
	})
	return answer, err
}

func (s *Service) handleTaskStatus(ctx context.Context, callback telegram.CallbackQuery, taskID int64, status domain.DailyTaskStatus) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: messages.Text("callback.workspace_missing")}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		task, found, err := q.GetTask(ctx, taskID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = messages.Text("callback.task.not_found")
			return nil
		}
		if callback.From.ID != task.OwnerUserID {
			answer.Text = messages.Text("callback.task.author_only")
			return nil
		}
		if !task.Status.IsOpen() {
			answer.Text = messages.Text("callback.task.closed")
			return nil
		}
		if previous, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID); err != nil {
			return err
		} else if found && previous.Kind == domain.PendingDailyTaskReport {
			_ = s.Telegram.DeleteMessage(ctx, callback.Message.Chat.ID, payloadInt64(previous.Payload, "prompt_message_id"))
		} else if found {
			answer.Text = pendingBusyText(previous.Kind)
			return nil
		}
		participant, _, err := q.GetParticipantByID(ctx, task.ParticipantID)
		if err != nil {
			return err
		}
		nudge, err := s.goalNudge(ctx, q, workspace, participant, fmt.Sprintf("task_status:%d:%s", task.ID, status), string(status))
		if err != nil {
			return err
		}
		_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:    callback.Message.Chat.ID,
			MessageID: callback.Message.MessageID,
			Text:      ui.DailyTaskReportPrompt(nudge),
		})
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, domain.PendingDailyTaskReport, map[string]any{
			"task_id":           taskID,
			"status":            string(status),
			"prompt_message_id": callback.Message.MessageID,
			"thread_id":         callback.Message.MessageThreadID,
		})
		return err
	})
	return answer, err
}

func (s *Service) handleAlertAck(ctx context.Context, callback telegram.CallbackQuery, alertID int64) (CallbackAnswer, error) {
	var answer CallbackAnswer
	err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		alert, found, err := q.GetAlert(ctx, alertID)
		if err != nil {
			return err
		}
		if !found {
			_ = s.Telegram.DeleteMessage(ctx, callback.Message.Chat.ID, callback.Message.MessageID)
			return nil
		}
		task, taskFound, err := q.GetTask(ctx, alert.DailyTaskID)
		if err != nil {
			return err
		}
		if taskFound && task.OwnerUserID != callback.From.ID {
			return nil
		}
		if err := s.dismissAlertMessage(ctx, q, callback.Message.Chat.ID, alert); err != nil {
			return err
		}
		answer.Text = messages.Text("callback.alert_hidden")
		return nil
	})
	return answer, err
}

func (s *Service) handleNoticeDismiss(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	_ = s.Telegram.DeleteMessage(ctx, callback.Message.Chat.ID, callback.Message.MessageID)
	return CallbackAnswer{}, nil
}

func (s *Service) dismissTaskAlerts(ctx context.Context, q *postgres.Queries, chatID int64, taskID int64) error {
	alerts, err := q.ListAlertsForTask(ctx, taskID)
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		if err := s.dismissAlertMessage(ctx, q, chatID, alert); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) dismissAlertMessage(ctx context.Context, q *postgres.Queries, chatID int64, alert postgres.DailyTaskAlert) error {
	_ = s.Telegram.DeleteMessage(ctx, chatID, optionalInt64(alert.TelegramMessageID))
	if err := q.ClearAlertMessage(ctx, alert.ID); err != nil {
		return err
	}
	if alert.AcknowledgedAt == nil {
		return q.AcknowledgeAlert(ctx, alert.ID, time.Now().UTC())
	}
	return nil
}

func (s *Service) ensureWorkspace(ctx context.Context, chatID int64, title string) (postgres.Workspace, error) {
	var workspace postgres.Workspace
	err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		var err error
		workspace, err = q.GetOrCreateWorkspace(ctx, chatID, title, s.DefaultTimezone)
		return err
	})
	return workspace, err
}

func (s *Service) ensureWorkspaceLoaded(ctx context.Context, chatID int64) (postgres.Workspace, error) {
	workspace, found, err := s.Store.Queries().GetWorkspaceByChatID(ctx, chatID)
	if err != nil || !found {
		return postgres.Workspace{}, err
	}
	return workspace, nil
}

func (s *Service) ensureTopicMessage(ctx context.Context, workspaceID int64, chatID int64, threadID int64, topicKey domain.TopicKey, currentMessageID *int64, text string, keyboard *telegram.InlineKeyboardMarkup, isControl bool, pin bool) bool {
	if currentMessageID != nil {
		if ok := s.editMessageSafe(ctx, chatID, *currentMessageID, text, keyboard); ok {
			return false
		}
	}
	message, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:              chatID,
		MessageThreadID:     threadID,
		Text:                text,
		ReplyMarkup:         keyboard,
		DisableNotification: true,
	})
	if err != nil {
		return false
	}
	_ = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if isControl {
			return q.SetTopicMessages(ctx, workspaceID, topicKey, nil, &message.MessageID, false, false)
		}
		return q.SetTopicMessages(ctx, workspaceID, topicKey, &message.MessageID, nil, false, false)
	})
	if pin {
		_ = s.Telegram.PinChatMessage(ctx, chatID, message.MessageID)
	}
	return true
}

func (s *Service) editMessageSafe(ctx context.Context, chatID int64, messageID int64, text string, keyboard *telegram.InlineKeyboardMarkup) bool {
	if messageID == 0 {
		return false
	}
	if err := s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{ChatID: chatID, MessageID: messageID, Text: text, ReplyMarkup: keyboard}); err != nil {
		return false
	}
	return true
}

func (s *Service) editMessageOrQueueProgressAlert(ctx context.Context, workspace postgres.Workspace, request telegram.EditMessageTextRequest, target string, threadID int64) error {
	if err := s.Telegram.EditMessageText(ctx, request); err != nil {
		if telegram.IsNotModifiedError(err) {
			return nil
		}
		messageLink := postgres.MessageLink(request.ChatID, request.MessageID, threadID)
		_, createErr := s.Store.Queries().CreateProgressEvent(ctx, workspace.ID, domain.ProgressSystemAlert, map[string]any{
			"kind":       "edit_failed",
			"target":     target,
			"message_id": request.MessageID,
			"message":    messageLink,
			"error":      truncateProgressAlertError(err.Error()),
		}, nil, nil)
		return createErr
	}
	return nil
}

func truncateProgressAlertError(text string) string {
	const maxLen = 300
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func dailyTaskCardKeyboard(task postgres.DailyTask) *telegram.InlineKeyboardMarkup {
	if task.Status.IsOpen() {
		return ui.DailyTaskKeyboard(task.ID)
	}
	return nil
}

func isCommand(text string, command string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return false
	}
	token := strings.SplitN(fields[0], "@", 2)[0]
	return token == command
}

func payloadInt64(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	case json.Number:
		result, _ := value.Int64()
		return result
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

func (s *Service) refreshPendingInputActivity(ctx context.Context, workspaceID int64, userID int64, threadID int64, pending postgres.PendingInput, userMessageID int64) error {
	payload := appendPayloadInt64(pending.Payload, "user_message_ids", userMessageID)
	_, err := s.Store.Queries().UpsertPendingInput(ctx, workspaceID, userID, threadID, pending.Kind, payload)
	return err
}

func appendPayloadInt64(payload map[string]any, key string, value int64) map[string]any {
	result := make(map[string]any, len(payload)+1)
	for existingKey, existingValue := range payload {
		result[existingKey] = existingValue
	}
	var values []int64
	switch raw := payload[key].(type) {
	case []int64:
		values = append(values, raw...)
	case []any:
		for _, item := range raw {
			switch typed := item.(type) {
			case float64:
				values = append(values, int64(typed))
			case int64:
				values = append(values, typed)
			case int:
				values = append(values, int64(typed))
			case json.Number:
				parsed, _ := typed.Int64()
				values = append(values, parsed)
			}
		}
	}
	for _, existing := range values {
		if existing == value {
			result[key] = values
			return result
		}
	}
	result[key] = append(values, value)
	return result
}

func (s *Service) deletePendingUserMessages(ctx context.Context, chatID int64, payload map[string]any) {
	for _, messageID := range payloadInt64Slice(payload, "user_message_ids") {
		_ = s.Telegram.DeleteMessage(ctx, chatID, messageID)
	}
}

func payloadInt64Slice(payload map[string]any, key string) []int64 {
	switch values := payload[key].(type) {
	case []int64:
		return values
	case []any:
		result := make([]int64, 0, len(values))
		for _, value := range values {
			switch typed := value.(type) {
			case float64:
				result = append(result, int64(typed))
			case int64:
				result = append(result, typed)
			case int:
				result = append(result, int64(typed))
			case json.Number:
				parsed, _ := typed.Int64()
				result = append(result, parsed)
			}
		}
		return result
	default:
		return nil
	}
}

func messagePlainText(message telegram.Message) string {
	if message.Text != "" {
		return message.Text
	}
	return message.Caption
}

func pendingBusyText(kind domain.PendingInputKind) string {
	switch kind {
	case domain.PendingDailyTaskText:
		return messages.Text("pending.daily_task_text")
	case domain.PendingDailyTaskReport:
		return messages.Text("pending.daily_task_report")
	case domain.PendingRoutinePlan:
		return messages.Text("pending.routine_plan")
	case domain.PendingRoutineReason:
		return messages.Text("pending.routine_reason")
	case domain.PendingSeasonalGoals:
		return messages.Text("pending.seasonal_goals")
	case domain.PendingGoalWeeklyReview:
		return messages.Text("pending.goal_weekly_review")
	case domain.PendingGoalFinalReflection:
		return messages.Text("pending.goal_final_reflection")
	default:
		return messages.Text("pending.default")
	}
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func (s *Service) DebugSummary() string {
	return fmt.Sprintf("trackmate bot default_timezone=%s", s.DefaultTimezone)
}
