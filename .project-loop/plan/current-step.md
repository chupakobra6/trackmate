# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-24

## Завершенный Шаг
- id: `STEP-036`
- status: `готово`
- objective: Запретить чужим участникам нажимать персональные callback-кнопки через единый deny-by-default access gate.
- requirement IDs: `REQ-061`, `VAL-014`
- source IDs: `S037`

## Owned Paths
- `internal/domain/callback.go`
- `internal/ui/keyboards.go`
- `internal/bot/callback_authorization.go`, callback router/callers и tests
- `internal/app/routine/routine.go`
- `internal/messages/messages.md`
- matching architecture/E2E contract docs
- `.project-loop/`

## Валидация
- domain/UI parser tests: pass;
- focused DB-backed callback authorization tests: pass;
- `TRACKMATE_TEST_DATABASE_URL=... make check`: pass;
- `git diff --check` и Project Loop validation: pass;
- полный Telegram E2E и production deploy не выполнялись.

## Итог
- Все callbacks проходят один access gate; неизвестные и новые kinds по умолчанию запрещены.
- Task/alert/routine/goal проверяют владельца и workspace до mutating handler.
- Stateless notices используют `notice:dismiss:<owner_user_id>`; старый безадресный callback больше не принимается.
- Чужой клик показывает `Эту кнопку может нажать только адресат` и не меняет Telegram/БД.
