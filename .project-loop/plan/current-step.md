# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Активный Шаг
- id: `STEP-041`
- status: `в работе`
- objective: Сделать автора и исходные цели явными в итоговой карточке сезона.
- requirement IDs: `REQ-071`, `REQ-072`, `VAL-019`
- source IDs: `S044`

## Подтверждённый Контракт
- Постоянный итог сохраняет текущую оценку и ссылки на все части ответа.
- Заголовок повторяет визуальную грамматику сезонной карточки: период в bold, затем `· Имя`.
- Новый final prompt создаётся reply к source goal message и позже редактируется в итог на том же месте.
- Missing reply target не блокирует сезонный lifecycle: выполняется один fallback send без reply.
- Уже отправленному Telegram message нельзя сменить reply target; существующие cards редактируются на месте без дублей.

## Область
- `internal/messages/messages.md`, `internal/ui/formatters.go` и tests;
- `internal/app/goals` reply dispatch/fallback и tests;
- `internal/bot/goals` participant-aware saved formatter и tests;
- production cards `6686`,`6687`, health/queues/logs и handoff.

## Критерий Готовности
- exact saved title показывает Игоря/Ярослава/Егора по общей name-normalization;
- future final request replies to source when source exists and safely falls back when missing;
- существующие летние cards обновлены без новых сообщений;
- focused/full checks и production verification проходят.
