# Handoff

Проект: trackmate
Обновлено: 2026-09-04

## Цель
- Проверить жалобу на 39 сообщений в `Прогрессе`, не удалить корректную историю и сохранить preventive production workflow.

## Завершенный Шаг
- step: `STEP-039`
- status: `готово`
- requirements: `REQ-067`, `REQ-068`, `VAL-017`
- sources: `S040`, `S041`

## Вывод
- Восстановление goals worker действительно раскрыло backlog, но новых карточек было `17`, а не `39`.
- Production DB даёт точное разложение пользовательского числа: `22` progress events были опубликованы 7–25 августа, `17` — 3 сентября после снятия worker wedge; итого `39`.
- Новые events `304..320` соответствуют отдельным daily tasks за 26.08–03.09 и Telegram messages `7250..7266`.
- Duplicate `(daily_task_id,event_type)` = `0`; duplicate message IDs = `0`; missing task/source links = `0`.
- Harvest подтвердил 17 отдельных пользовательских карточек. Это корректная история, поэтому Telegram cleanup не выполнялся.

## Prevention И Production
- Root cause backlog уже устранён в `7059d88`: недоступное удаление weekly goal prompt больше не блокирует worker stages.
- `AGENTS.md` закрепляет standing checks → commit → push → deploy → live verification и изоляцию worker failures.
- Глобальный `/Users/igor/.codex/AGENTS.md` требует root-cause fix, regression/invariant, bounded recovery и post-deploy queue/log/live-state checks.
- Rules commit `37c1136` находится локально, в `origin/main` и production checkout; services healthy.
- Текущий progress outbox пуст, свежего повторного backlog нет.

## Остаточный Риск
- После будущего длительного outage корректный backlog по-прежнему публикуется отдельными карточками. Coalesced summary — отдельное продуктовое изменение, не необходимое для исправления этого инцидента.

## Следующее Действие
- Нет обязательного действия. Не удалять messages `7250..7266`; при отдельном запросе спроектировать summary-mode для большого backlog.
