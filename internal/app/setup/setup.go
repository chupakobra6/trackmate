package setup

import (
	"context"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
)

var TopicTitles = map[domain.TopicKey]string{
	domain.TopicToday:    "Сегодня",
	domain.TopicProgress: "Прогресс",
}

type Prerequisites struct {
	IsSupergroup    bool
	IsForum         bool
	BotIsAdmin      bool
	CanManageTopics bool
	CanReadMessages bool
}

func (p Prerequisites) IsReady() bool {
	return p.IsSupergroup && p.IsForum && p.BotIsAdmin && p.CanManageTopics && p.CanReadMessages
}

type Service struct {
	Store           *postgres.Store
	Telegram        telegram.API
	BotID           int64
	DefaultTimezone string
}

func (s *Service) CheckPrerequisites(ctx context.Context, chatID int64) (Prerequisites, error) {
	chat, err := s.Telegram.GetChat(ctx, chatID)
	if err != nil {
		return Prerequisites{}, err
	}
	member, err := s.Telegram.GetChatMember(ctx, chatID, s.BotID)
	if err != nil {
		return Prerequisites{}, err
	}
	isOwner := member.Status == "creator" || member.Status == "owner"
	isAdmin := isOwner || member.Status == "administrator"
	return Prerequisites{
		IsSupergroup:    chat.Type == "supergroup",
		IsForum:         chat.IsForum,
		BotIsAdmin:      isAdmin,
		CanManageTopics: isOwner || member.CanManageTopics,
		CanReadMessages: isAdmin,
	}, nil
}

func (s *Service) IsGroupAdmin(ctx context.Context, chatID int64, userID int64) (bool, error) {
	member, err := s.Telegram.GetChatMember(ctx, chatID, userID)
	if err != nil {
		return false, err
	}
	return member.Status == "creator" || member.Status == "owner" || member.Status == "administrator", nil
}

func (s *Service) EnsureWorkspaceTopics(ctx context.Context, chatID int64, title string, timezoneName string) (map[domain.TopicKey]int64, bool, error) {
	threadIDs := map[domain.TopicKey]int64{}
	changed := false
	err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		workspace, err := q.GetOrCreateWorkspace(ctx, chatID, title, timezoneName)
		if err != nil {
			return err
		}
		existing, err := q.ListTopicBindings(ctx, workspace.ID)
		if err != nil {
			return err
		}
		for _, key := range []domain.TopicKey{domain.TopicToday, domain.TopicProgress} {
			threadID, topicChanged, err := s.ensureTopicBinding(ctx, q, workspace.ID, existing, chatID, key)
			if err != nil {
				return err
			}
			threadIDs[key] = threadID
			changed = changed || topicChanged
		}
		return q.MarkWorkspaceReady(ctx, workspace.ID)
	})
	return threadIDs, changed, err
}

func (s *Service) ensureTopicBinding(ctx context.Context, q *postgres.Queries, workspaceID int64, existing map[domain.TopicKey]postgres.TopicBinding, chatID int64, key domain.TopicKey) (int64, bool, error) {
	title := TopicTitles[key]
	binding, ok := existing[key]
	if !ok {
		topic, err := s.Telegram.CreateForumTopic(ctx, telegram.CreateForumTopicRequest{ChatID: chatID, Name: title})
		if err != nil {
			return 0, false, err
		}
		if _, err := q.UpsertTopicBinding(ctx, workspaceID, key, topic.MessageThreadID, title); err != nil {
			return 0, false, err
		}
		return topic.MessageThreadID, true, nil
	}
	if err := s.Telegram.EditForumTopic(ctx, telegram.EditForumTopicRequest{ChatID: chatID, MessageThreadID: binding.ThreadID, Name: title}); err != nil {
		if !telegram.IsMissingThreadError(err) {
			return 0, false, err
		}
		topic, err := s.Telegram.CreateForumTopic(ctx, telegram.CreateForumTopicRequest{ChatID: chatID, Name: title})
		if err != nil {
			return 0, false, err
		}
		if _, err := q.UpsertTopicBinding(ctx, workspaceID, key, topic.MessageThreadID, title); err != nil {
			return 0, false, err
		}
		if err := q.SetTopicMessages(ctx, workspaceID, key, nil, nil, true, true); err != nil {
			return 0, false, err
		}
		return topic.MessageThreadID, true, nil
	}
	if binding.TopicTitle != title {
		if _, err := q.UpsertTopicBinding(ctx, workspaceID, key, binding.ThreadID, title); err != nil {
			return 0, false, err
		}
		return binding.ThreadID, true, nil
	}
	return binding.ThreadID, false, nil
}
