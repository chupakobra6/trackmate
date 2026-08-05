# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Завершенный Шаг
- id: `STEP-032`
- status: `готово`
- objective: Подтвердить routine correction и сделать setup prompts целей/рутин восстанавливаемыми без дублирования.
- requirement IDs: `REQ-055`, `REQ-056`, `VAL-010`
- owned paths: `internal/bot/goals.go`, `internal/bot/routines.go`, shared setup lifecycle, `internal/telegram/`, tests, docs, `.project-loop/`.
- validation: production read-only DB/Telegram evidence; missing/transient Telegram error classification; clean-schema PostgreSQL integration; full tests/lint/Project Loop validation; live Goals callback replay.
- done criteria: выполнены локально. Routine state/card совпадают без лишней mutation; живой prompt переиспользуется, отсутствующий восстанавливается одним send, transient failure не создает replacement; pipeline целей подготовлен пользователю с отмеченной открытой product semantics для unanswered reviews.

## Фокус Ревью
- Проверить общий lifecycle для Routine/Goals без копирования двух recovery path.
- Не менять периодичность/TTL goal weekly reviews без решения пользователя.

## Примечания
- Routine check-in `1220772` и Telegram message `6010` уже корректны; production write не нужен.
- Goals callback отправил message `6037`, DB pending `739` жив, но message отсутствует в fresh MTProto dump.
- STEP-031 завершен и закоммичен как `8f1140c`; production deploy по-прежнему требует отдельного approval.
- Live Goals replay: два configure callback оставили один setup message `833`; source `834` сохранен; confirmation `835`; pending inputs очищены.
