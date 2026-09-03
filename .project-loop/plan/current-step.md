# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Активный Шаг
- id: `STEP-038`
- status: `в работе`
- objective: Исправить goals lifecycle на границе сезонов и восстановить летние production-итоги.
- requirement IDs: `REQ-063..REQ-066`, `VAL-016`
- source IDs: `S039`

## Подтвержденная Причина
- `summer-2026` корректно хранит `period_ends_on=2026-09-01`.
- weekly retry `6688`, отправленный 24.08, на 72-м часе больше не удалялся Telegram из-за 48-hour limit.
- `advanceWeeklyReview` возвращал error до `skipped_at`, поэтом final review и все последующие worker deliveries не выполнялись.
- Итоги Игоря/Ярослава из `7207`/`7225` поэтому поглотили старые weekly pending; вторая часть Игоря `7208` не была сохранена.

## План Поставки
1. Сделать weekly cleanup неблокирующим и попадающим в Telegram delete window.
2. Закрепить calendar-date границы для всех timezone.
3. Добавить накопление нескольких final messages и явное завершение.
4. Пройти focused tests, DB-backed `make check`, migration и focused Telegram E2E.
5. Сделать focused commit, production backup, deploy, восстановление итогов и проверку очередей/логов.

## Текущая Валидация
- DB-backed `make check`: pass.
- additive migration `202609040001` применена на локальной PostgreSQL, новые колонки прочитаны обратно.
- focused Goals Telegram E2E не стартовал: сохраненная MTProto-сессия runner требует повторного login; ни один шаг сценария не выполнялся.
- exact multi-message/status/finish/source-link path покрыт PostgreSQL bot integration test.
