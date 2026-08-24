# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-24

## Завершенный Шаг
- id: `STEP-037`
- status: `готово`
- objective: Подтвердить owner protection для routine reminder и auto-close, затем безопасно развернуть callback fix.
- requirement IDs: `REQ-062`, `VAL-015`
- source IDs: `S038`

## Валидация
- Routine reminder и auto-close callback data: `notice:dismiss:42`, focused PostgreSQL test pass.
- Foreign user `43` на exact callback владельца `42`: no Telegram/DB mutations, callback answer owner-only.
- Authorized owner dismiss и legacy ownerless rejection: pass.
- DB-backed `make check`: pass; full Telegram E2E не запускался.

## Production
- pushed head: `ec6c31b` (`03a4c78` содержит product code).
- backup: `/opt/trackmate/backups/trackmate_20260824T114142Z.dump`; checksum и `pg_restore --list` pass.
- production updated `a9ad102 -> ec6c31b`; `api`, `worker`, `postgres` healthy.
- schema version `202608050002`; stale alert/progress claims `0`; unpublished progress `0`; idle transactions `0`; advisory waiters `0`; suspicious fresh logs `0`.
- один активный old-version routine auto-close notice `6641` точечно получил addressed callback `notice:dismiss:1747674822`.

## Итог
- Чужой участник больше не может снять ни routine reminder, ни routine auto-close alert.
- Владелец продолжает закрывать свой alert кнопкой `👀 Понял`.
- Уже существовавший активный routine alert также приведен к новому защищенному контракту.
