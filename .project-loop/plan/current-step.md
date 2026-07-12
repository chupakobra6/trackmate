# Текущий Шаг

Проект: trackmate
Обновлено: 2026-07-12

## Активный Шаг
- id: `STEP-029`
- status: `готово`
- objective: Исправить production bug кнопки `Понял` у routine reminder и развернуть проверенное исправление.
- requirement IDs: `REQ-052`
- owned paths: `internal/telegram/`, `internal/bot/`, `.project-loop/`
- validation: production callback/DB check: pass; focused tests: pass; `make test`: pass; `make lint`: pass; local Docker healthy; production `2a25305`, services healthy, migrations applied, `pending_inputs=0`
- done criteria: ошибка удаления не маскируется; при запрете удаления клавиатура снимается; production обновлен и healthy.

## Фокус Ревью
- Проверить только обработку `notice:dismiss`, классификацию Telegram delete errors и fallback снятия клавиатуры.
- Пользовательские тексты не менять.

## Примечания
- STEP-028 production routine reset уже завершен ранее; в будущем update message все еще нужно попросить участников заново настроить рутины.
- В E2E reset добавлена очистка `goal_nudge_cooldowns`, иначе сценарий `14` зависел от старого cooldown.
