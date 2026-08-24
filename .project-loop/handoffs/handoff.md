# Handoff

Проект: trackmate
Обновлено: 2026-08-24

## Цель
- Персональные Telegram callback-кнопки может выполнить только их адресат; новые callback kinds закрыты deny-by-default до явной политики доступа.

## Завершенный Шаг
- step: `STEP-036`
- status: `готово`
- requirements: `REQ-061`, `VAL-014`
- source: `S037`

## Реализация
- `internal/bot/callback_authorization.go` стал единым access gate перед dispatch.
- Public self-scoped controls, admin setup и persisted owner callbacks имеют явные ветви; default запрещает неизвестный kind.
- Task/alert/routine/goal callbacks сверяют immutable owner и workspace callback-message до mutating handler.
- `DismissKeyboard(ownerUserID)` компиляционно требует адресата; callback имеет вид `notice:dismiss:<owner_user_id>`.
- Старый ownerless `notice:dismiss` намеренно не поддерживается: его владельца достоверно определить нельзя, поэтому он получает stale answer и ничего не удаляет.
- Чужой клик получает `Эту кнопку может нажать только адресат`; подпись кнопки `👀 Понял` и нормальный owner-flow не менялись.
- Инвариант сохранен в `AGENTS.md` и `docs/architecture.md`; E2E comments указывают границу single-user live scenario и multi-user PostgreSQL regression.

## Проверка
- `TRACKMATE_TEST_DATABASE_URL=... go test ./internal/bot -run 'Test(PersonalCallbacksRejectOtherParticipantsWithoutMutation|OwnedCallbackRejectsResourceFromAnotherWorkspace|NoticeDismiss)' -count=1 -v`: pass.
- DB-backed regression проверил task report/status, alert ack, routine item, goal final и notice dismiss: ноль Telegram send/edit/delete/markup mutations; task/alert/routine/goal DB state не изменен.
- Cross-workspace callback того же владельца: stale answer, ноль mutations.
- `TRACKMATE_TEST_DATABASE_URL=... make check`: pass.
- `git diff --check`: pass.
- `loopctl.py validate`: pass.
- Полный Telegram E2E не запускался по project rule; production/push не выполнялись.

## Review
- Все `CallbackKind` из parser/dispatch имеют явную ветвь access gate.
- Все вызовы `DismissKeyboard` передают доменного владельца или автора пользовательского ввода.
- Handler/storage owner conditions сохранены как второй барьер вокруг записей.
- Residual: уже отправленные старой версией ownerless dismiss-кнопки после будущего deploy станут безопасно неактивными; добавлять неаутентифицируемый compatibility path нельзя.

## Следующее Действие
- По отдельному запросу: push/deploy текущего local commit и focused smoke владельца + второго участника на тестовом/production Telegram без полного E2E.

## Обновленные Источники Правды
- `.project-loop/requirements/source-map.md`
- `.project-loop/requirements/checklist.md`
- `.project-loop/plan/delivery-plan.md`
- `.project-loop/plan/current-step.md`
- `.project-loop/intake/user-deltas.md`
- `.project-loop/handoffs/handoff.md`
- `AGENTS.md`
- `docs/architecture.md`
