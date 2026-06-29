# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-019`
- status: `готово`
- objective: Проверить production bug со сбитым pending input 26.06, удалить показанные chat messages и уточнить voice report text.
- requirement IDs: `REQ-038`
- owned paths: `.project-loop/`
- validation: prod logs/schema probe: pass; Harvest delete/dump verification: pass; DB verification task `160`/`172`/progress `192`: pass; Telegram messages `3361`/`3649` edited: pass
- done criteria: root cause for message `3467` not saving is identified; production schema no longer has the legacy pending constraint and accepts topic-scoped pending inputs; chat messages `3464`/`3465` are deleted; task `172` is restored; voice report `3386` is transcribed enough to store a readable summary in task `160` and progress `192`.

## Фокус Ревью
- Это production data-fix, не кодовый deploy.
- Не трогать routine local fixes и не выкатывать локальную пачку.
- Не создавать новых Telegram posts, только удалить/отредактировать существующие сообщения по показанному кейсу.

## Примечания
- Логи production 2026-06-26 показывают падение `today:add` на `duplicate key value violates unique constraint "uq_pending_inputs_workspace_group_id"` перед сообщением Егора `3467`.
- На production старый constraint уже отсутствует; rollback probe с двумя pending inputs одного пользователя в разных thread проходит.
- Удалены chat messages `3464` и `3465` через Telegram Harvest main profile.
- Backup перед voice-текстом: `/opt/trackmate/backups/trackmate_manual_voice_transcript_20260629T121722Z.dump`.
- ASR для voice `3386` шумный, поэтому в БД и видимых сообщениях сохранена вычитанная смысловая сводка, а не сырой transcript.
