# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Завершенный Шаг
- id: `STEP-038`
- status: `готово`
- objective: Исправить goals lifecycle на границе сезонов и восстановить летние production-итоги.
- requirement IDs: `REQ-063..REQ-066`, `VAL-016`
- source IDs: `S039`

## Причина И Исправление
- `summer-2026` имел правильный `period_ends_on=2026-09-01`; проблема была не в самой московской дате.
- Retry `6688` удалялся ровно через 48 часов после повторной отправки, когда Telegram уже запрещал delete. Error возвращался до `skipped_at` и блокировал final/alerts/progress.
- Retry теперь закрывается через 47 часов после повторной отправки (71 час от первого prompt), а delete failure оставляет inert message и не блокирует БД.
- DB `DATE` читается как calendar day без timezone conversion; setup prompt динамически показывает текущий сезон, а pinned control больше не содержит устаревающую дату.
- Final принимает несколько сообщений, сохраняет их идемпотентно до `✅ Завершить итог` и оставляет короткую карточку со ссылками.

## Валидация
- DB-backed `make check`: pass.
- additive migration `202609040001`: local/prod applied and read back.
- exact bot integration: partial status, empty-complete rejection, messages `801+802`, pending retention, explicit completion, permanent source links: pass.
- focused Telegram runner не начал сценарий из-за истекшей MTProto-сессии; production DB/Harvest verification и exact integration использованы как fallback.

## Production
- code: `7059d88`; target `inferno-nl:/opt/trackmate`.
- backup: `/opt/trackmate/backups/trackmate_20260903T223606Z.dump`, checksum/archive pass, `245029` bytes, mode `600`.
- Igor final: goal set `1`, messages `7207+7208`, permanent card `6686`.
- Yaroslav final: goal set `4`, message `7225`, permanent card `6687`.
- stale weekly `6688` closed; Egor final prompt sent as `7248` linked to goals `4192`.
- stale alerts `150,151,163,164` acknowledged; current alert `165` delivered; `17` progress events published.
- `api`, `worker`, `postgres` healthy; open weekly `0`; unpublished alert/progress `0`; stale claims/idle transactions/waiting advisory locks `0`; fresh errors `0`.

## Следующее Действие
- Егор может выбрать оценку на карточке `7248`, отправить одну или несколько частей и нажать `✅ Завершить итог`.
