# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-023`
- status: `готово`
- objective: Локально изменить routine timing, notification levels и ссылки в `Прогресс`.
- requirement IDs: `REQ-042..REQ-044`
- owned paths: `internal/domain/`, `internal/app/routine/`, `internal/bot/`, `internal/worker/`, `internal/app/progress/`, `internal/storage/postgres/`, `internal/ui/`, `internal/messages/`, `docs/`, `.project-loop/`
- validation: focused Go tests: pass; `go test ./...`: pass; `make test`: pass; `make lint`: pass
- done criteria: routine card is created at 08:00 next day for the previous date; reminder pings at 20:00 and auto-close pings at 00:00; routine plan changes preserve the previous day's old checklist; Progress remains silent and links people/actions/media reports to useful Telegram targets.

## Фокус Ревью
- Это локальная продуктовая правка, не production deploy.
- Не делать Telegram cleanup и не трогать production DB.
- Сохранять тексты в `internal/messages/messages.md`, не размазывать новые русские строки по Go-коду.

## Примечания
- Новая routine-семантика: дата `D` заполняется утром `D+1` в 08:00; reminder в 20:00 `D+1`; auto-close в 00:00 `D+2`.
- Старый список рутины сохраняется для уже наступившей проверки через snapshot check-in перед upsert нового плана.
- `Прогресс` остается message level 1: тихие сообщения без уведомлений.
- Missed/forgotten alerts используют notification level 4: уведомление плюс, где уместно, reply/mention.
