package bot

import (
	"context"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/telegram"
)

// authorizeCallback is the single access gate before callback dispatch.
// Unknown and newly added callback kinds are denied until their policy is
// explicitly defined here.
func (s *Service) authorizeCallback(ctx context.Context, callback telegram.CallbackQuery, parsed domain.Callback) (CallbackAnswer, bool, error) {
	if callback.Message.Chat.Type != "group" && callback.Message.Chat.Type != "supergroup" {
		return staleCallbackAnswer(), false, nil
	}

	switch parsed.Kind {
	case domain.CallbackSetupCheck,
		domain.CallbackTodayAdd,
		domain.CallbackRoutineConfigure,
		domain.CallbackGoalsConfigure:
		return CallbackAnswer{}, true, nil
	case domain.CallbackSetupStart:
		isAdmin, err := s.Setup.IsGroupAdmin(ctx, callback.Message.Chat.ID, callback.From.ID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !isAdmin {
			return CallbackAnswer{Text: messages.Text("callback.setup.admin_only")}, false, nil
		}
		return CallbackAnswer{}, true, nil
	case domain.CallbackTaskReport, domain.CallbackTaskStatus:
		task, found, err := s.Store.Queries().GetTask(ctx, parsed.TaskID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !found {
			return staleCallbackAnswer(), false, nil
		}
		return s.authorizeOwnedCallback(ctx, callback, task.WorkspaceGroupID, task.OwnerUserID)
	case domain.CallbackAlertAck:
		alert, found, err := s.Store.Queries().GetAlert(ctx, parsed.AlertID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !found {
			return staleCallbackAnswer(), false, nil
		}
		task, found, err := s.Store.Queries().GetTask(ctx, alert.DailyTaskID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !found {
			return staleCallbackAnswer(), false, nil
		}
		return s.authorizeOwnedCallback(ctx, callback, task.WorkspaceGroupID, task.OwnerUserID)
	case domain.CallbackRoutineItem:
		checkin, found, err := s.Store.Queries().GetRoutineCheckin(ctx, parsed.RoutineCheckinID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !found {
			return staleCallbackAnswer(), false, nil
		}
		return s.authorizeOwnedCallback(ctx, callback, checkin.WorkspaceGroupID, checkin.OwnerUserID)
	case domain.CallbackGoalFinalStatus, domain.CallbackGoalFinalComplete:
		goalSet, found, err := s.Store.Queries().GetSeasonalGoalSet(ctx, parsed.GoalSetID)
		if err != nil {
			return CallbackAnswer{}, false, err
		}
		if !found {
			return staleCallbackAnswer(), false, nil
		}
		return s.authorizeOwnedCallback(ctx, callback, goalSet.WorkspaceGroupID, goalSet.OwnerUserID)
	case domain.CallbackNoticeDismiss:
		if callback.From.ID != parsed.NoticeOwnerUserID {
			return ownerOnlyCallbackAnswer(), false, nil
		}
		return CallbackAnswer{}, true, nil
	default:
		return staleCallbackAnswer(), false, nil
	}
}

func (s *Service) authorizeOwnedCallback(ctx context.Context, callback telegram.CallbackQuery, workspaceID int64, ownerUserID int64) (CallbackAnswer, bool, error) {
	workspace, found, err := s.Store.Queries().GetWorkspaceByChatID(ctx, callback.Message.Chat.ID)
	if err != nil {
		return CallbackAnswer{}, false, err
	}
	if !found || workspace.ID != workspaceID {
		return staleCallbackAnswer(), false, nil
	}
	if callback.From.ID != ownerUserID {
		return ownerOnlyCallbackAnswer(), false, nil
	}
	return CallbackAnswer{}, true, nil
}

func ownerOnlyCallbackAnswer() CallbackAnswer {
	return CallbackAnswer{Text: messages.Text("callback.owner_only")}
}

func staleCallbackAnswer() CallbackAnswer {
	return CallbackAnswer{Text: messages.Text("callback.stale_button")}
}
